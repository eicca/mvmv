package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information, set during build
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "mvmv SOURCE TARGET",
	Short: "Parallel move tool for large directory structures",
	Long: `mvmv is a parallel file move utility designed for merging massive
directory structures efficiently.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	Args:    cobra.ExactArgs(2),
	RunE:    runMove,
}

func init() {
	rootCmd.Flags().IntP("workers", "w", 0, "Number of parallel workers (default: number of CPU cores)")
	rootCmd.Flags().IntP("buffer", "b", 100000, "Job queue buffer size")
	rootCmd.Flags().BoolP("stats", "s", false, "Show statistics during and after operation")
	rootCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().BoolP("dry-run", "n", false, "Preview what would be moved without actually moving")
}
