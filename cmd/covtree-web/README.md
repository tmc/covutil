# covtree-web

`covtree-web` is a standalone web server for interactive coverage data exploration, providing an enhanced web interface for Go coverage analysis.

## Features

- **Interactive Coverage Explorer**: Modern, responsive web interface for browsing coverage data
- **Real-time Filtering**: Filter packages by name patterns and coverage thresholds
- **Expandable Package View**: Click to expand packages and view function-level coverage
- **Coverage Visualization**: Color-coded coverage indicators and progress bars
- **Auto-browser Opening**: Optionally open browser automatically when server starts
- **Custom Branding**: Set custom titles for different projects
- **RESTful API**: JSON endpoints for programmatic access

## Installation

```bash
go install github.com/tmc/covutil/cmd/covtree-web@latest
```

## Usage

### Basic Usage

```bash
# Start server with coverage data from current directory
covtree-web -i=./coverage

# Use custom port and title
covtree-web -i=./my-project-coverage -http=:9000 -title="My Project Coverage"

# Open browser automatically
covtree-web -i=./coverage -open
```

### Command Line Options

- `-i directory`: Input directory to scan recursively for coverage data (required)
- `-http address`: HTTP server address (default: `:8080`)
- `-title string`: Custom title for the web interface (default: `"Coverage Report"`)
- `-open`: Open browser automatically after starting server

### Examples

```bash
# Development server with auto-open
covtree-web -i=./coverage -http=:8080 -title="Dev Coverage" -open

# Production-style server
covtree-web -i=/var/coverage-data -http=:80 -title="Production Coverage Report"

# Team coverage dashboard
covtree-web -i=./team-coverage -http=:3000 -title="Team Dashboard"
```

## Web Interface

### Main Features

1. **Coverage Summary**: High-level statistics with visual progress indicators
2. **Package Filtering**: 
   - Pattern-based filtering (e.g., `github.com/user/*`)
   - Minimum/maximum coverage percentage filters
   - Real-time search and filtering
3. **Package Explorer**:
   - Expandable package list with function details
   - Color-coded coverage indicators
   - Detailed coverage percentages

### Keyboard Shortcuts

- `Ctrl/Cmd + F`: Focus search filter
- `Ctrl/Cmd + R`: Refresh package list

## API Endpoints

The web server provides several RESTful API endpoints:

### GET /api/summary
Returns overall coverage summary.

**Response:**
```json
{
  "TotalPackages": 42,
  "TotalLines": 1500,
  "CoveredLines": 1200,
  "CoverageRate": 0.8
}
```

### GET /api/packages
Returns filtered list of packages.

**Query Parameters:**
- `pattern`: Package name pattern (e.g., `github.com/user/*`)
- `min_coverage`: Minimum coverage as decimal (e.g., `0.5` for 50%)
- `max_coverage`: Maximum coverage as decimal (e.g., `0.9` for 90%)

**Response:**
```json
[
  {
    "ImportPath": "github.com/user/repo/pkg",
    "TotalLines": 100,
    "CoveredLines": 85,
    "CoverageRate": 0.85,
    "Functions": [...]
  }
]
```

### GET /api/package/{path}
Returns detailed information for a specific package.

### GET /api/health
Returns server health and status information.

**Response:**
```json
{
  "status": "ok",
  "total_packages": 42,
  "title": "Coverage Report",
  "timestamp": 1640995200
}
```

## Coverage Data Format

`covtree-web` works with Go coverage data in the format produced by:

- `go build -cover`
- `go test -coverprofile=coverage.out`
- Various CI/CD coverage collection tools

The tool scans the input directory recursively for coverage data files and automatically processes nested repositories.

## Development

### Running Tests

```bash
go test -v
```

### Building from Source

```bash
git clone https://github.com/tmc/covutil
cd covutil/cmd/covtree-web
go build
```

## Comparison with covtree serve

While `covtree serve` provides basic web functionality, `covtree-web` offers:

- Enhanced UI/UX with modern design
- Better mobile responsiveness  
- Additional API endpoints
- Auto-browser opening
- Custom branding options
- Improved error handling
- Extended keyboard shortcuts

## Troubleshooting

### Common Issues

1. **"must specify input directory"**: Use the `-i` flag to specify the coverage data directory
2. **"input directory does not exist"**: Ensure the path exists and contains coverage data
3. **"failed to load coverage data"**: Check that the directory contains valid Go coverage files
4. **Port already in use**: Use a different port with `-http=:XXXX`

### Debug Mode

For debugging coverage data loading issues, check the console output when starting the server. The tool will log the number of packages loaded and any errors encountered.