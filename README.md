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

# Show statistics during operation
mvmv --stats /data/source/ /data/target/
```

## Algorithm

1. If target directory doesn't exist, move entire source directory
2. If target directory exists, recursively check contents
3. Move files that don't exist in target
4. Use parallel workers to process multiple items concurrently

Implementation details:
- Each worker maintains its own job queue
- Workers share jobs when queues become unbalanced
- Uses OS rename operation for atomic moves

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

