# Uncover

Uncover prints the code not covered in a coverage profile.

## Installation

```bash
go install rsc.io/uncover@latest
```

## Usage

First generate a coverage profile with the Go test tool:

```bash
go test -coverprofile=c.out
```

Then run uncover to see the uncovered code:

```bash
uncover [-a] [-l] [-f] [-t N] [-s] c.out
```

## Command Line Flags

- `-a`: Show all uncovered blocks, including those marked as unreachable/untested
- `-l`: Print long (absolute) file names instead of relative ones
- `-f`: List uncovered functions by name instead of showing code blocks
- `-t N`: List top N functions with the lowest coverage percentage
- `-s`: Output in a sortable/diffable format suitable for comparing different coverage runs

## Features

- Lists uncovered code blocks from Go test coverage profiles
- Can identify and skip blocks marked as intentionally unreachable/untested
- Can list uncovered functions by name
- Can report functions with the lowest test coverage percentage
- Formats output in a readable way with proper indentation
- Can produce sortable output for easy comparison between different test runs

For more details, see the [package documentation](https://pkg.go.dev/rsc.io/uncover).