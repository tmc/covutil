# covutil

[![Go Reference](https://pkg.go.dev/badge/github.com/tmc/covutil.svg)](https://pkg.go.dev/github.com/tmc/covutil)
[![Go Report Card](https://goreportcard.com/badge/github.com/tmc/covutil)](https://goreportcard.com/report/github.com/tmc/covutil)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Advanced coverage utilities for Go projects, featuring comprehensive tools for analyzing, tracking, and reporting code coverage across multiple languages and artifact types.

## Overview

`covutil` extends Go's coverage capabilities with advanced analysis tools and synthetic coverage tracking for non-Go artifacts. It enables comprehensive testing coverage measurement across your entire project ecosystem.

**Key capabilities:**
- Advanced Go coverage analysis and visualization
- Coverage tracking for scripts, templates, and configuration files
- Multi-language coverage integration
- Interactive web-based coverage exploration
- Coverage forest management across test runs

## Installation

```bash
go get github.com/tmc/covutil
```

### Command Line Tools

```bash
# Core analysis tools
go install github.com/tmc/covutil/cmd/covtree@latest
go install github.com/tmc/covutil/cmd/covforest@latest
go install github.com/tmc/covutil/cmd/covtree-web@latest

# Experimental tools
go install github.com/tmc/covutil/exp/cmd/covanalyze@latest
go install github.com/tmc/covutil/exp/cmd/covcompare@latest
go install github.com/tmc/covutil/exp/cmd/covdiff@latest
```

## Quick Start

### Basic Coverage Analysis

```bash
# Generate coverage data
go test -coverprofile=coverage.out ./...

# Interactive coverage exploration
covtree coverage.out

# Web-based coverage viewer
covtree-web coverage.out
```

### Synthetic Coverage for Scripts

```go
package main

import (
    "log"
    
    "github.com/tmc/covutil/synthetic"
)

func main() {
    // Create tracker for deployment scripts
    tracker := synthetic.NewScriptTracker(
        synthetic.WithTestName("deployment"),
    )
    
    // Parse deployment script
    deployScript := `#!/bin/bash
echo "Starting deployment..."
kubectl apply -f manifests/
echo "Deployment complete"`
    
    err := tracker.ParseAndTrack(deployScript, "deploy.sh", "bash", "deployment")
    if err != nil {
        log.Fatal(err)
    }
    
    // Track execution (from logs, instrumentation, etc.)
    tracker.TrackExecution("deploy.sh", "deployment", 2) // echo
    tracker.TrackExecution("deploy.sh", "deployment", 3) // kubectl
    
    // Generate coverage report
    report := tracker.GetReport()
    log.Println(report)
}
```

## Features

### Coverage Analysis Tools

| Tool | Description |
|------|-------------|
| `covtree` | Interactive coverage tree visualization |
| `covforest` | Manage coverage data across multiple test runs |
| `covtree-web` | Web-based coverage explorer |
| `covanalyze` | Advanced coverage analysis and metrics |
| `covcompare` | Compare coverage between runs |
| `covdiff` | Show coverage differences and deltas |

### Synthetic Coverage

Track coverage for non-Go artifacts with built-in parsers:

| Language | Extensions | Features |
|----------|------------|----------|
| **Bash** | `.bash`, `bash` | Functions, arrays, here docs, test constructs |
| **Shell** | `.sh`, `shell` | POSIX shell compatibility |
| **Python** | `.py`, `python` | Classes, decorators, imports, context managers |
| **Go Templates** | `.tmpl`, `gotemplate` | Directives, functions, pipelines |
| **Scripttest** | `.txt`, `.txtar` | Complete scripttest command set (300+ patterns) |

## Usage

### Coverage Forest Management

```bash
# Initialize coverage forest
covforest init

# Add coverage from different test runs
covforest add --name="unit-tests" coverage-unit.out
covforest add --name="integration-tests" coverage-integration.out

# View summary across all runs
covforest summary

# List all stored coverage data
covforest list
```

### Integration Testing with Coverage

```go
func TestDeploymentWorkflow(t *testing.T) {
    tracker := synthetic.NewScriptTracker(
        synthetic.WithTestName("deployment"),
        synthetic.WithLabels(map[string]string{
            "environment": "staging",
            "version":     "v1.2.3",
        }),
    )
    
    // Parse deployment artifacts
    err := tracker.ParseAndTrack(deployScript, "deploy.sh", "bash", "deployment")
    require.NoError(t, err)
    
    err = tracker.ParseAndTrack(configTemplate, "app.tmpl", "gotemplate", "deployment")
    require.NoError(t, err)
    
    // Execute deployment
    output := runDeployment()
    
    // Track execution based on logs
    executedLines := parseExecutionLogs(output)
    for _, line := range executedLines {
        tracker.TrackExecution("deploy.sh", "deployment", line)
    }
    
    // Generate coverage pod
    deploymentPod, err := tracker.GeneratePod()
    require.NoError(t, err)
    
    // Combine with Go coverage
    goCoverage, err := covutil.LoadCoverageSetFromDirectory(os.Getenv("GOCOVERDIR"))
    require.NoError(t, err)
    
    combinedCoverage := &covutil.CoverageSet{
        Pods: append(goCoverage.Pods, deploymentPod),
    }
    
    // Assert coverage thresholds
    coverage := calculateOverallCoverage(combinedCoverage)
    assert.Greater(t, coverage, 0.8, "Overall coverage should be >80%")
}
```

### Custom Parser Development

Extend coverage tracking to new file types:

```go
import "github.com/tmc/covutil/synthetic/parsers"

type DockerfileParser struct{}

func (p *DockerfileParser) Name() string {
    return "dockerfile"
}

func (p *DockerfileParser) Extensions() []string {
    return []string{".dockerfile", "Dockerfile"}
}

func (p *DockerfileParser) Description() string {
    return "Docker build files"
}

func (p *DockerfileParser) ParseScript(content string) map[int]string {
    // Implementation for parsing Dockerfile instructions
    // Return map of line numbers to executable instructions
}

func (p *DockerfileParser) IsExecutable(line string) bool {
    // Determine if line is executable (FROM, RUN, COPY, etc.)
}

// Register globally
func init() {
    parsers.Register(&DockerfileParser{})
}
```

## API Reference

### Core Types

```go
// Load coverage data
coverageSet, err := covutil.LoadCoverageSetFromDirectory("coverage-dir")

// Basic tracking
tracker := synthetic.NewBasicTracker(
    synthetic.WithLabels(map[string]string{"test": "my-test"}),
)
tracker.Track("artifact", "location", true)

// Script tracking with auto-parsing
scriptTracker := synthetic.NewScriptTracker()
err := scriptTracker.ParseAndTrack(content, "file.sh", "bash", "test-name")
scriptTracker.TrackExecution("file.sh", "test-name", lineNumber)

// Generate outputs
pod, err := tracker.GeneratePod()           // Binary pod format
profile, err := tracker.GenerateProfile("text") // Text profile format
report := tracker.GetReport()               // Human-readable report
```

### Supported Output Formats

- **Binary Pod**: Compatible with Go coverage tools and covutil ecosystem
- **Text Profile**: Compatible with `go tool cover` and standard Go toolchain
- **JSON**: Machine-readable format for custom tooling (planned)
- **HTML**: Rich interactive coverage reports

## Architecture

```
covutil/
├── cmd/                    # Command-line tools
│   ├── covtree/           # Interactive coverage explorer
│   ├── covforest/         # Coverage forest management
│   └── covtree-web/       # Web-based coverage viewer
├── synthetic/             # Synthetic coverage engine
│   └── parsers/           # Modular parser architecture
│       ├── bash/          # Bash script parser
│       ├── python/        # Python script parser
│       ├── scripttest/    # Scripttest format parser
│       └── defaults/      # Auto-registration
├── internal/              # Core coverage infrastructure
│   └── coverage/          # Adapted from Go's internal package
└── exp/                   # Experimental tools and features
```

## Documentation

- **[Synthetic Coverage Guide](./synthetic/README.md)** - Comprehensive guide for script coverage
- **[API Documentation](https://pkg.go.dev/github.com/tmc/covutil)** - Full Go package documentation
- **[Examples](./synthetic/examples_test.go)** - Working examples and integration patterns

## Requirements

- Go 1.21 or later
- For web interface: Modern browser with JavaScript enabled

## Contributing

We welcome contributions! Please see our guidelines:

1. **Bug Reports**: Use GitHub issues with detailed reproduction steps
2. **Feature Requests**: Describe use case and proposed API
3. **Pull Requests**: Include tests and update documentation
4. **Parser Development**: Follow the `parsers.Parser` interface

For synthetic coverage parsers:
- Implement comprehensive command/syntax recognition  
- Include real-world test cases
- Handle edge cases and error conditions
- Update documentation with usage examples

### Development Setup

```bash
git clone https://github.com/tmc/covutil.git
cd covutil
go mod download
go test ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

The `internal/coverage` package is adapted from the Go standard library and retains its original BSD-style license.

## Credits

- Core coverage infrastructure adapted from [golang/go](https://github.com/golang/go/tree/master/src/internal/coverage)
- Scripttest integration inspired by Go's testing infrastructure
- Web interface built with modern web standards

---

For questions, issues, or contributions, please visit the [GitHub repository](https://github.com/tmc/covutil).