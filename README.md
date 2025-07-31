# mvmv - Parallel Move Tool for Large Directory Structures

A parallel file move utility for merging large directory structures.

## Overview

`mvmv` merges directory trees by moving files and directories that don't exist in the target location.
It uses parallel processing for better performance on large file systems.

Features:
- Moves only non-existing files/directories to target
- Parallel processing with configurable workers
- Skips symbolic links
- Live progress statistics
- Dry run mode

## Installation

```bash
go install github.com/eicca/mvmv
```

Or build from source:

```bash
git clone https://github.com/eicca/mvmv
cd mvmv
go build -o mvmv
```

## Usage

```bash
mvmv [OPTIONS] SOURCE TARGET
```

### Options

- `--workers N, -w N`: Number of parallel workers (default: number of CPU cores)
- `--buffer N, -b N`: Job queue buffer size (default: 100,000)
- `--stats, -s`: Show statistics during and after operation
- `--verbose, -v`: Enable verbose output
- `--dry-run, -n`: Preview what would be moved without actually moving
- `--help, -h`: Show help message
- `--version`: Show version information

### Examples

```bash
# Basic usage
mvmv /data/source/ /data/target/

# Use 32 parallel workers with statistics
mvmv --workers 32 --stats /data/source/ /data/target/

# Dry run with verbose output
mvmv -v -n /data/source/ /data/target/

# Custom buffer size for very large directories
mvmv --buffer 1000000 /data/source/ /data/target/

# Show statistics during operation
mvmv --stats /data/source/ /data/target/
```

## Algorithm

1. Start multiple worker goroutines
2. Submit the source directory as the initial job
3. For each job, workers:
   - Skip if source and target paths are the same
   - Skip symbolic links
   - For directories:
     - If target doesn't exist, move entire directory tree
     - If target exists, scan contents and create jobs for each entry
   - For files:
     - Skip if file exists in target
     - Move file if it doesn't exist in target
4. Workers recursively process all jobs until complete

Implementation details:
- Workers pull jobs from a shared buffered channel
- Uses sync.WaitGroup to track job completion
- Atomic operations for thread-safe statistics
- OS rename for atomic move operations

## Error Handling

- Continues on non-fatal errors
- Logs errors when verbose mode enabled
- Uses atomic rename operations
- Tracks and reports total error count

## Requirements

- Source and target must be directories
- Source and target cannot be symbolic links
- Filesystem must support atomic rename operations

## Testing

```bash
go test ./...
```

