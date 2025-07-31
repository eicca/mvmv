package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
)

// Options holds the configuration for the move operation
type Options struct {
	Workers int
	Buffer  int
	Stats   bool
	Verbose bool
	DryRun  bool
}

// Statistics tracks metrics during the move operation
type Statistics struct {
	DirsChecked     int64
	DirsSkipped     int64
	DirsMoved       int64
	FilesChecked    int64
	FilesSkipped    int64
	FilesMoved      int64
	BytesMoved      int64
	SymlinksSkipped int64
	Errors          int64
	StartTime       time.Time
}

// Job represents a single move operation
type Job struct {
	SourcePath string
	TargetPath string
}

// runMove is the main entry point for the move command
func runMove(cmd *cobra.Command, args []string) error {
	source := cleanPath(args[0])
	target := cleanPath(args[1])

	workers, _ := cmd.Flags().GetInt("workers")
	if workers == 0 {
		workers = runtime.NumCPU()
	}

	buffer, _ := cmd.Flags().GetInt("buffer")
	stats, _ := cmd.Flags().GetBool("stats")
	verbose, _ := cmd.Flags().GetBool("verbose")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	opts := &Options{
		Workers: workers,
		Buffer:  buffer,
		Stats:   stats,
		Verbose: verbose,
		DryRun:  dryRun,
	}

	return performMove(source, target, opts)
}

func cleanPath(p string) string {
	cleaned := filepath.Clean(p)

	if !filepath.IsAbs(cleaned) {
		abs, err := filepath.Abs(cleaned)
		if err == nil {
			cleaned = abs
		}
	}

	return cleaned
}

// performMove executes the parallel move operation
func performMove(source, target string, opts *Options) error {
	// Verify source exists using Lstat to not follow symlinks
	sourceInfo, err := os.Lstat(source)
	if err != nil {
		return fmt.Errorf("source path error: %w", err)
	}
	if !sourceInfo.IsDir() {
		return fmt.Errorf("source must be a directory")
	}
	if sourceInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("source cannot be a symlink")
	}

	// Verify target exists and is a directory
	targetInfo, err := os.Lstat(target)
	if err != nil {
		return fmt.Errorf("target path error: %w", err)
	}
	if !targetInfo.IsDir() {
		return fmt.Errorf("target must be a directory")
	}
	if targetInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("target cannot be a symlink")
	}

	stats := &Statistics{
		StartTime: time.Now(),
	}

	bufferSize := opts.Buffer
	if bufferSize == 0 {
		bufferSize = 100000
	}
	jobs := make(chan Job, bufferSize)

	var jobsWg sync.WaitGroup
	for range opts.Workers {
		go worker(jobs, &jobsWg, stats, opts)
	}

	var statsDone chan struct{}
	if opts.Stats {
		statsDone = make(chan struct{})
		go statsReporter(stats, statsDone)
	}

	jobsWg.Add(1)
	jobs <- Job{SourcePath: source, TargetPath: target}

	jobsWg.Wait()
	close(jobs)

	if opts.Stats {
		close(statsDone)
		printFinalStats(stats)
	}

	if atomic.LoadInt64(&stats.Errors) > 0 {
		return fmt.Errorf("completed with %d errors", stats.Errors)
	}

	return nil
}

func worker(jobs chan Job, jobsWg *sync.WaitGroup, stats *Statistics, opts *Options) {
	for job := range jobs {
		newJobs := processPath(job.SourcePath, job.TargetPath, stats, opts)

		for _, newJob := range newJobs {
			jobsWg.Add(1)
			jobs <- newJob
		}

		jobsWg.Done()
	}
}

func processPath(sourcePath, targetPath string, stats *Statistics, opts *Options) []Job {
	if sourcePath == targetPath {
		return nil
	}

	sourceInfo, err := os.Lstat(sourcePath)
	if err != nil {
		atomic.AddInt64(&stats.Errors, 1)
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "Cannot stat %s: %v\n", sourcePath, err)
		}
		return nil
	}

	if sourceInfo.Mode()&os.ModeSymlink != 0 {
		atomic.AddInt64(&stats.SymlinksSkipped, 1)
		if opts.Verbose {
			fmt.Printf("Skipping symlink: %s\n", sourcePath)
		}
		return nil
	}

	_, err = os.Lstat(targetPath)
	targetExists := err == nil

	if sourceInfo.IsDir() {
		return processDir(sourcePath, targetPath, targetExists, stats, opts)
	}

	processFile(sourcePath, targetPath, targetExists, sourceInfo, stats, opts)
	return nil
}

func processDir(sourcePath, targetPath string, targetExists bool, stats *Statistics, opts *Options) []Job {
	atomic.AddInt64(&stats.DirsChecked, 1)

	if !targetExists {
		if opts.Verbose {
			fmt.Printf("Moving directory: %s -> %s\n", sourcePath, targetPath)
		}

		if !opts.DryRun {
			if err := os.Rename(sourcePath, targetPath); err != nil {
				atomic.AddInt64(&stats.Errors, 1)
				if opts.Verbose {
					fmt.Fprintf(os.Stderr, "Failed to move directory %s: %v\n", sourcePath, err)
				}
			} else {
				atomic.AddInt64(&stats.DirsMoved, 1)
			}
		} else {
			atomic.AddInt64(&stats.DirsMoved, 1)
		}
		return nil
	}

	atomic.AddInt64(&stats.DirsSkipped, 1)

	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		atomic.AddInt64(&stats.Errors, 1)
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "Cannot read directory %s: %v\n", sourcePath, err)
		}
		return nil
	}

	newJobs := make([]Job, 0, len(entries))
	for _, entry := range entries {
		childSource := filepath.Join(sourcePath, entry.Name())
		childTarget := filepath.Join(targetPath, entry.Name())
		newJobs = append(newJobs, Job{SourcePath: childSource, TargetPath: childTarget})
	}

	return newJobs
}

func processFile(sourcePath, targetPath string, targetExists bool, sourceInfo os.FileInfo, stats *Statistics, opts *Options) {
	atomic.AddInt64(&stats.FilesChecked, 1)

	if targetExists {
		atomic.AddInt64(&stats.FilesSkipped, 1)
		if opts.Verbose {
			fmt.Printf("Skipping existing file: %s\n", targetPath)
		}
		return
	}

	if opts.Verbose {
		fmt.Printf("Moving file: %s -> %s\n", sourcePath, targetPath)
	}

	if !opts.DryRun {
		if err := os.Rename(sourcePath, targetPath); err != nil {
			atomic.AddInt64(&stats.Errors, 1)
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "Failed to move file %s: %v\n", sourcePath, err)
			}
		} else {
			atomic.AddInt64(&stats.FilesMoved, 1)
			atomic.AddInt64(&stats.BytesMoved, sourceInfo.Size())
		}
	} else {
		atomic.AddInt64(&stats.FilesMoved, 1)
		atomic.AddInt64(&stats.BytesMoved, sourceInfo.Size())
	}
}

// statsReporter periodically prints statistics during operation
func statsReporter(stats *Statistics, done <-chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			printStats(stats)
		}
	}
}

// printStats prints current statistics
func printStats(stats *Statistics) {
	elapsed := time.Since(stats.StartTime)
	dirsChecked := atomic.LoadInt64(&stats.DirsChecked)
	dirsMoved := atomic.LoadInt64(&stats.DirsMoved)
	filesChecked := atomic.LoadInt64(&stats.FilesChecked)
	filesMoved := atomic.LoadInt64(&stats.FilesMoved)
	bytesMoved := atomic.LoadInt64(&stats.BytesMoved)
	symlinksSkipped := atomic.LoadInt64(&stats.SymlinksSkipped)
	errors := atomic.LoadInt64(&stats.Errors)

	rate := float64(bytesMoved) / elapsed.Seconds() / 1024 / 1024 // MB/s

	fmt.Printf("\r[%s] Dirs: %d/%d, Files: %d/%d, Symlinks skipped: %d, Data: %.2f GB, Rate: %.2f MB/s, Errors: %d",
		formatDuration(elapsed),
		dirsMoved, dirsChecked,
		filesMoved, filesChecked,
		symlinksSkipped,
		float64(bytesMoved)/1024/1024/1024,
		rate,
		errors)
}

// printFinalStats prints final statistics after operation completes
func printFinalStats(stats *Statistics) {
	elapsed := time.Since(stats.StartTime)
	fmt.Printf("\n\nOperation completed in %s\n", formatDuration(elapsed))
	fmt.Printf("Directories: %d moved, %d skipped, %d checked\n",
		stats.DirsMoved, stats.DirsSkipped, stats.DirsChecked)
	fmt.Printf("Files: %d moved, %d skipped, %d checked\n",
		stats.FilesMoved, stats.FilesSkipped, stats.FilesChecked)

	if stats.SymlinksSkipped > 0 {
		fmt.Printf("Symlinks skipped: %d\n", stats.SymlinksSkipped)
	}

	// Only show data stats if we have file-level data
	if stats.BytesMoved > 0 {
		fmt.Printf("Total data moved: %.2f GB (file data only)\n", float64(stats.BytesMoved)/1024/1024/1024)
		if elapsed.Seconds() > 0 {
			fmt.Printf("Average rate: %.2f MB/s\n", float64(stats.BytesMoved)/elapsed.Seconds()/1024/1024)
		}
	}

	if stats.Errors > 0 {
		fmt.Printf("Errors: %d\n", stats.Errors)
	}
}

// formatDuration formats a duration in human-readable format
func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
