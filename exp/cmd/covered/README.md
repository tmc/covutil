# covered

`covered` analyzes Go coverage profiles and outputs colorized source code with coverage information, similar to `go tool cover -html` but for terminals.

## Features

- **Terminal-based coverage visualization** with ANSI colors
- **Covered lines in green, uncovered lines in red**
- **Context-aware display** showing only relevant lines with configurable context
- **Path filtering** with regular expressions
- **File sorting** by coverage percentage (least covered first)
- **txtar-style headers** showing filename and coverage percentage
- **Partial line coverage** highlighting for precise coverage visualization
- **Respects comment rules** from uncover (skips "// unreachable" and "// untested" patterns)

## Installation

```bash
go install github.com/tmc/covutil/utils/exp/cmd/covered@latest
```

## Usage

Generate a coverage profile first:

```bash
go test -coverprofile=cover.out ./...
```

### Basic Usage

```bash
# Show coverage for all files with default context (5 lines)
covered -c cover.out

# Show coverage with more context lines
covered -C 10 -c cover.out

# Show all lines (no context filtering)
covered -C 0 -c cover.out
```

### Filtering and Sorting

```bash
# Filter files by path pattern
covered -path ".*_test\.go" -c cover.out

# Show files sorted by least covered first
covered -t -c cover.out

# Combine filtering and sorting
covered -path "internal/.*" -t -c cover.out
```

### Output Control

```bash
# Force colors even when piping
covered -color=always -c cover.out | less -R

# Disable colors for plain text output
covered -color=never -c cover.out

# Auto-detect terminal (default behavior)
covered -color=auto -c cover.out
```

## Command Line Flags

- `-c string`: Coverage profile to read (default "cover.out")
- `-path string`: Regular expression to filter file paths
- `-C int`: Number of context lines to show around covered/uncovered blocks (default 5)
- `-t`: Sort files by least covered (ascending coverage percentage)
- `-color string`: Colorize output: auto, always, never (default "auto")
- `-a`: Show all uncovered lines (including those marked as unreachable/untested)

## Example Output

```
-- main.go (85.7% covered) --
   1 + package main
   2 + 
   3 + import (
   4 +     "fmt"
   5 + )
   6 + 
   7 + func main() {
   8 +     fmt.Println("Hello, world!")
   9 -     unusedFunction()
  10 + }
  11 + 
  12 - func unusedFunction() {
  13 -     fmt.Println("This function is not covered by tests")
  14 - }
```

### Color Legend

- **Green lines with `+`**: Covered by tests
- **Red lines with `-`**: Not covered by tests
- **Cyan headers**: File information with coverage percentage
- **Dark gray text**: Comments and import statements
- **Gray `...`**: Indicates skipped lines (when using context mode)

### Non-Color Mode

When colors are disabled (`-color=never`), the output uses Unicode symbols:
- **`✓`**: Covered lines
- **`✗`**: Uncovered lines  
- **`⋯`**: Skipped lines indicator

## Context Display

By default, `covered` shows only:

1. **Uncovered lines** and their surrounding context
2. **Coverage transition points** where coverage changes from one line to the next
3. **Configurable context** around these interesting areas (default: 5 lines)

Set `-C 0` to show all lines, or adjust the context size with `-C N`.

## Comment Rules

Following the `uncover` tradition, `covered` respects certain comment patterns:

- Lines starting with `// unreachable` or `// untested` are considered acceptable to be uncovered
- Lines starting with `// Unreachable` or `// Untested` (capitalized) are also respected

## Use Cases

- **Quick coverage review** during development
- **CI/CD pipeline integration** for coverage reports
- **Code review preparation** to identify untested code paths
- **Terminal-based workflows** where HTML coverage reports are impractical
- **Focused testing** by identifying least-covered files first

## Integration with Other Tools

Works well with:

- `go test -coverprofile=cover.out`
- Standard Go coverage tools
- Terminal multiplexers and editors
- CI/CD systems for automated coverage reporting

## Credits

Inspired by [rsc.io/uncover](https://pkg.go.dev/rsc.io/uncover) and designed to complement the Go toolchain's coverage utilities.