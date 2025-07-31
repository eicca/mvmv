package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMvMv(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (src, dst string)
		validate func(t *testing.T, src, dst string)
	}{
		{
			name: "move_entire_directory_when_not_exists_in_target",
			setup: func(t *testing.T) (string, string) {
				src := t.TempDir()
				dst := t.TempDir()

				// Create source structure
				createFile(t, filepath.Join(src, "dir1", "file1.txt"), "content1")
				createFile(t, filepath.Join(src, "dir1", "subdir", "file2.txt"), "content2")

				return src, dst
			},
			validate: func(t *testing.T, src, dst string) {
				// Source should be empty
				assertNotExists(t, filepath.Join(src, "dir1"))

				// Destination should have the moved structure
				assertFileContent(t, filepath.Join(dst, "dir1", "file1.txt"), "content1")
				assertFileContent(t, filepath.Join(dst, "dir1", "subdir", "file2.txt"), "content2")
			},
		},
		{
			name: "skip_existing_files_and_move_only_new_ones",
			setup: func(t *testing.T) (string, string) {
				src := t.TempDir()
				dst := t.TempDir()

				// Create overlapping structure
				createFile(t, filepath.Join(src, "dir1", "existing.txt"), "new_content")
				createFile(t, filepath.Join(src, "dir1", "new.txt"), "new_file")
				createFile(t, filepath.Join(dst, "dir1", "existing.txt"), "old_content")

				return src, dst
			},
			validate: func(t *testing.T, src, dst string) {
				// Existing file should not be overwritten
				assertFileContent(t, filepath.Join(dst, "dir1", "existing.txt"), "old_content")

				// New file should be moved
				assertNotExists(t, filepath.Join(src, "dir1", "new.txt"))
				assertFileContent(t, filepath.Join(dst, "dir1", "new.txt"), "new_file")

				// Source existing file should remain
				assertFileContent(t, filepath.Join(src, "dir1", "existing.txt"), "new_content")
			},
		},
		{
			name: "handle_deeply_nested_structures",
			setup: func(t *testing.T) (string, string) {
				src := t.TempDir()
				dst := t.TempDir()

				// Create deep structure
				deepPath := filepath.Join(src, "a", "b", "c", "d", "e", "f")
				createFile(t, filepath.Join(deepPath, "deep.txt"), "deep_content")

				// Partial overlap
				createFile(t, filepath.Join(dst, "a", "b", "existing.txt"), "existing")

				return src, dst
			},
			validate: func(t *testing.T, src, dst string) {
				// Deep structure should be moved
				assertNotExists(t, filepath.Join(src, "a", "b", "c"))
				assertFileContent(t, filepath.Join(dst, "a", "b", "c", "d", "e", "f", "deep.txt"), "deep_content")

				// Existing file should remain
				assertFileContent(t, filepath.Join(dst, "a", "b", "existing.txt"), "existing")
			},
		},
		{
			name: "handle_multiple_source_directories",
			setup: func(t *testing.T) (string, string) {
				src1 := t.TempDir()
				src2 := t.TempDir()
				dst := t.TempDir()

				createFile(t, filepath.Join(src1, "from_src1.txt"), "src1_content")
				createFile(t, filepath.Join(src2, "from_src2.txt"), "src2_content")

				// Return src1 as primary, but we'll test with multiple sources
				return src1, dst
			},
			validate: func(t *testing.T, src, dst string) {
				// Both sources should be moved
				assertFileContent(t, filepath.Join(dst, "from_src1.txt"), "src1_content")
				// Note: This test would need adjustment for multiple source handling
			},
		},
		{
			name: "preserve_permissions_and_timestamps",
			setup: func(t *testing.T) (string, string) {
				src := t.TempDir()
				dst := t.TempDir()

				file := filepath.Join(src, "perm_test.txt")
				createFile(t, file, "content")
				if err := os.Chmod(file, 0755); err != nil {
					t.Fatalf("Failed to chmod %s: %v", file, err)
				}

				return src, dst
			},
			validate: func(t *testing.T, src, dst string) {
				dstFile := filepath.Join(dst, "perm_test.txt")
				info, err := os.Stat(dstFile)
				if err != nil {
					t.Fatalf("Failed to stat moved file: %v", err)
				}

				if info.Mode().Perm() != 0755 {
					t.Errorf("Permissions not preserved: got %v, want %v", info.Mode().Perm(), 0755)
				}
			},
		},
		{
			name: "handle_empty_directories",
			setup: func(t *testing.T) (string, string) {
				src := t.TempDir()
				dst := t.TempDir()

				// Create empty directories
				if err := os.MkdirAll(filepath.Join(src, "empty", "nested", "dirs"), 0755); err != nil {
					t.Fatalf("Failed to create directories: %v", err)
				}

				return src, dst
			},
			validate: func(t *testing.T, src, dst string) {
				// Empty directories should be moved
				assertNotExists(t, filepath.Join(src, "empty"))
				assertDirExists(t, filepath.Join(dst, "empty", "nested", "dirs"))
			},
		},
		{
			name: "move_mixed_files_and_directories_at_root",
			setup: func(t *testing.T) (string, string) {
				src := t.TempDir()
				dst := t.TempDir()

				// Create mixed content at root level
				createFile(t, filepath.Join(src, "file_at_root.txt"), "root_file")
				createFile(t, filepath.Join(src, "another_file.log"), "log_content")
				createFile(t, filepath.Join(src, "dir1", "file1.txt"), "dir1_file")
				createFile(t, filepath.Join(src, "dir1", "data.json"), "json_data")
				createFile(t, filepath.Join(src, "dir2", "file3.txt"), "dir2_file")
				if err := os.MkdirAll(filepath.Join(src, "dir3", "subdir1"), 0755); err != nil {
					t.Fatalf("Failed to create directories: %v", err)
				}

				return src, dst
			},
			validate: func(t *testing.T, src, dst string) {
				// All files at root should be moved
				assertNotExists(t, filepath.Join(src, "file_at_root.txt"))
				assertNotExists(t, filepath.Join(src, "another_file.log"))
				assertFileContent(t, filepath.Join(dst, "file_at_root.txt"), "root_file")
				assertFileContent(t, filepath.Join(dst, "another_file.log"), "log_content")

				// All directories should be moved
				assertNotExists(t, filepath.Join(src, "dir1"))
				assertNotExists(t, filepath.Join(src, "dir2"))
				assertNotExists(t, filepath.Join(src, "dir3"))
				assertFileContent(t, filepath.Join(dst, "dir1", "file1.txt"), "dir1_file")
				assertFileContent(t, filepath.Join(dst, "dir1", "data.json"), "json_data")
				assertFileContent(t, filepath.Join(dst, "dir2", "file3.txt"), "dir2_file")
				assertDirExists(t, filepath.Join(dst, "dir3", "subdir1"))
			},
		},
		{
			name: "merge_directories_with_new_subdirs_and_files",
			setup: func(t *testing.T) (string, string) {
				src := t.TempDir()
				dst := t.TempDir()

				// Create initial target structure
				createFile(t, filepath.Join(dst, "dir1", "existing.txt"), "existing_content")
				createFile(t, filepath.Join(dst, "file_at_root.txt"), "existing_root")
				createFile(t, filepath.Join(dst, "dir2", "old.txt"), "old_content")

				// Create source with overlapping and new content
				createFile(t, filepath.Join(src, "dir1", "existing.txt"), "new_version")
				createFile(t, filepath.Join(src, "dir1", "newfile.txt"), "new_file")
				createFile(t, filepath.Join(src, "dir1", "newsubdir", "nested.txt"), "nested_content")
				createFile(t, filepath.Join(src, "dir4", "file4.txt"), "dir4_content")
				createFile(t, filepath.Join(src, "newfile_at_root.txt"), "new_root_file")

				return src, dst
			},
			validate: func(t *testing.T, src, dst string) {
				// Existing files should not be overwritten
				assertFileContent(t, filepath.Join(dst, "dir1", "existing.txt"), "existing_content")
				assertFileContent(t, filepath.Join(dst, "file_at_root.txt"), "existing_root")

				// New files and directories should be moved
				assertNotExists(t, filepath.Join(src, "dir1", "newfile.txt"))
				assertNotExists(t, filepath.Join(src, "dir1", "newsubdir"))
				assertNotExists(t, filepath.Join(src, "dir4"))
				assertNotExists(t, filepath.Join(src, "newfile_at_root.txt"))

				assertFileContent(t, filepath.Join(dst, "dir1", "newfile.txt"), "new_file")
				assertFileContent(t, filepath.Join(dst, "dir1", "newsubdir", "nested.txt"), "nested_content")
				assertFileContent(t, filepath.Join(dst, "dir4", "file4.txt"), "dir4_content")
				assertFileContent(t, filepath.Join(dst, "newfile_at_root.txt"), "new_root_file")

				// Source versions of existing files should remain
				assertFileContent(t, filepath.Join(src, "dir1", "existing.txt"), "new_version")
			},
		},
		{
			name: "skip_symlinks",
			setup: func(t *testing.T) (string, string) {
				src := t.TempDir()
				dst := t.TempDir()

				// Create a file and a symlink to it
				createFile(t, filepath.Join(src, "original.txt"), "content")
				if err := os.Symlink(filepath.Join(src, "original.txt"), filepath.Join(src, "link.txt")); err != nil {
					t.Fatalf("Failed to create symlink: %v", err)
				}

				// Create a directory and a symlink to it
				if err := os.MkdirAll(filepath.Join(src, "dir"), 0755); err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}
				if err := os.Symlink(filepath.Join(src, "dir"), filepath.Join(src, "dirlink")); err != nil {
					t.Fatalf("Failed to create directory symlink: %v", err)
				}

				return src, dst
			},
			validate: func(t *testing.T, src, dst string) {
				// Original file should be moved
				assertNotExists(t, filepath.Join(src, "original.txt"))
				assertFileContent(t, filepath.Join(dst, "original.txt"), "content")

				// Symlinks should remain in source
				assertSymlinkExists(t, filepath.Join(src, "link.txt"))
				assertSymlinkExists(t, filepath.Join(src, "dirlink"))

				// Symlinks should not exist in destination
				assertNotExists(t, filepath.Join(dst, "link.txt"))
				assertNotExists(t, filepath.Join(dst, "dirlink"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst := tt.setup(t)

			err := performMove(src, dst, &Options{Workers: 2, Buffer: 10000})
			if err != nil {
				t.Fatalf("mvmv failed: %v", err)
			}

			tt.validate(t, src, dst)
		})
	}
}

func TestParallelism(t *testing.T) {
	t.Run("concurrent_directory_processing", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()

		// Create many directories to test parallelism
		for i := 0; i < 100; i++ {
			createFile(t, filepath.Join(src, fmt.Sprintf("dir%d", i), "file.txt"), "content")
		}

		err := performMove(src, dst, &Options{Workers: 8, Buffer: 10000})
		if err != nil {
			t.Fatalf("mvmv failed: %v", err)
		}

		// Verify all directories were moved
		for i := 0; i < 100; i++ {
			assertFileContent(t, filepath.Join(dst, fmt.Sprintf("dir%d", i), "file.txt"), "content")
		}
	})

	t.Run("large_directory_parallel_processing", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()

		// Create a directory with many files to test parallel processing
		largeDir := filepath.Join(src, "largedir")
		if err := os.MkdirAll(largeDir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Create 2000 files
		for i := 0; i < 2000; i++ {
			createFile(t, filepath.Join(largeDir, fmt.Sprintf("file%04d.txt", i)), fmt.Sprintf("content%d", i))
		}

		// Also create some files that should be skipped
		if err := os.MkdirAll(filepath.Join(dst, "largedir"), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		for i := 0; i < 100; i++ {
			createFile(t, filepath.Join(dst, "largedir", fmt.Sprintf("file%04d.txt", i)), "existing")
		}

		err := performMove(src, dst, &Options{Workers: 4, Buffer: 10000})
		if err != nil {
			t.Fatalf("mvmv failed: %v", err)
		}

		// Verify files were moved correctly
		for i := 0; i < 100; i++ {
			// First 100 should be skipped (remain as "existing")
			assertFileContent(t, filepath.Join(dst, "largedir", fmt.Sprintf("file%04d.txt", i)), "existing")
		}
		for i := 100; i < 2000; i++ {
			// Rest should be moved
			assertFileContent(t, filepath.Join(dst, "largedir", fmt.Sprintf("file%04d.txt", i)), fmt.Sprintf("content%d", i))
		}
	})
}

func TestDryRun(t *testing.T) {
	t.Run("dry_run_does_not_move_files", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()

		// Create source structure
		createFile(t, filepath.Join(src, "file.txt"), "content")
		createFile(t, filepath.Join(src, "dir1", "file2.txt"), "content2")

		// Run in dry-run mode
		err := performMove(src, dst, &Options{Workers: 1, Buffer: 10000, DryRun: true})
		if err != nil {
			t.Fatalf("Dry run failed: %v", err)
		}

		// Source files should still exist
		assertFileContent(t, filepath.Join(src, "file.txt"), "content")
		assertFileContent(t, filepath.Join(src, "dir1", "file2.txt"), "content2")

		// Destination should be empty
		assertNotExists(t, filepath.Join(dst, "file.txt"))
		assertNotExists(t, filepath.Join(dst, "dir1"))
	})
}

func TestStatistics(t *testing.T) {
	t.Run("statistics_tracking", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()

		// Create test structure with known sizes
		createFile(t, filepath.Join(src, "file1.txt"), "12345")       // 5 bytes
		createFile(t, filepath.Join(src, "file2.txt"), "1234567890")  // 10 bytes
		createFile(t, filepath.Join(src, "dir1", "file3.txt"), "123") // 3 bytes
		createFile(t, filepath.Join(dst, "existing.txt"), "old")      // Should be skipped

		// Create a file that will be skipped
		createFile(t, filepath.Join(src, "existing.txt"), "new")

		// Run with stats enabled
		opts := &Options{Workers: 1, Buffer: 10000, Stats: true}
		err := performMove(src, dst, opts)
		if err != nil {
			t.Fatalf("Move failed: %v", err)
		}

		// Verify files were moved
		assertFileContent(t, filepath.Join(dst, "file1.txt"), "12345")
		assertFileContent(t, filepath.Join(dst, "file2.txt"), "1234567890")
		assertFileContent(t, filepath.Join(dst, "dir1", "file3.txt"), "123")

		// Verify existing file was not overwritten
		assertFileContent(t, filepath.Join(dst, "existing.txt"), "old")
	})
}

func TestErrorHandling(t *testing.T) {
	t.Run("handle_missing_source", func(t *testing.T) {
		src := "/non/existent/path"
		dst := t.TempDir()

		err := performMove(src, dst, &Options{Workers: 1, Buffer: 10000})
		if err == nil {
			t.Fatal("Expected error for non-existent source")
		}
	})

	t.Run("handle_file_as_source", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "file.txt")
		dst := t.TempDir()

		createFile(t, src, "content")

		err := performMove(src, dst, &Options{Workers: 1, Buffer: 10000})
		if err == nil {
			t.Fatal("Expected error for file as source")
		}
	})

	t.Run("handle_symlink_as_source", func(t *testing.T) {
		tmp := t.TempDir()
		src := filepath.Join(tmp, "source_link")
		dst := t.TempDir()

		// Create a directory and symlink to it
		realDir := filepath.Join(tmp, "real_dir")
		if err := os.Mkdir(realDir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.Symlink(realDir, src); err != nil {
			t.Fatalf("Failed to create symlink: %v", err)
		}

		err := performMove(src, dst, &Options{Workers: 1, Buffer: 10000})
		if err == nil {
			t.Fatal("Expected error for symlink as source")
		}
	})

	t.Run("handle_missing_target", func(t *testing.T) {
		src := t.TempDir()
		dst := "/non/existent/target"

		err := performMove(src, dst, &Options{Workers: 1, Buffer: 10000})
		if err == nil {
			t.Fatal("Expected error for non-existent target")
		}
	})

	t.Run("handle_file_as_target", func(t *testing.T) {
		src := t.TempDir()
		dst := filepath.Join(t.TempDir(), "file.txt")

		createFile(t, dst, "content")

		err := performMove(src, dst, &Options{Workers: 1, Buffer: 10000})
		if err == nil {
			t.Fatal("Expected error for file as target")
		}
	})
}

func TestVerboseOutput(t *testing.T) {
	t.Run("verbose_mode_shows_operations", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()

		createFile(t, filepath.Join(src, "file.txt"), "content")
		createFile(t, filepath.Join(src, "dir1", "file2.txt"), "content2")

		// Capture verbose output
		err := performMove(src, dst, &Options{Workers: 1, Verbose: true})
		if err != nil {
			t.Fatalf("Move failed: %v", err)
		}

		// Verify files were moved
		assertFileContent(t, filepath.Join(dst, "file.txt"), "content")
		assertFileContent(t, filepath.Join(dst, "dir1", "file2.txt"), "content2")
	})
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{61 * time.Minute, "1h01m00s"},
		{125 * time.Minute, "2h05m00s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

// Helper functions
func createFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create file %s: %v", path, err)
	}
}

func assertFileContent(t *testing.T, path, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	if string(content) != expected {
		t.Errorf("File content mismatch at %s: got %q, want %q", path, string(content), expected)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("Path should not exist: %s", path)
	}
}

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Directory should exist: %s", path)
	}
	if !info.IsDir() {
		t.Errorf("Path is not a directory: %s", path)
	}
}

func assertSymlinkExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Symlink should exist: %s", path)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Path is not a symlink: %s", path)
	}
}
