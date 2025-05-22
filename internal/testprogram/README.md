# Coverage Testing Program

This directory contains various utilities for testing and demonstrating coverage functionality using the root `covutil` package.

## Structure

### Main Entry Point
- `testprogram.go` - Unified entry point with menu-driven interface

### Command Line Tools
- `cmd/test-apis/` - API testing utilities
  - `test_highlevel_api.go` - High-level API demonstrations
  - `test_simple_api.go` - Simple API demonstrations
- `cmd/synthetic-demo/` - Synthetic coverage generation
  - `synthetic_proper.go` - Synthetic coverage generation demo

### Core Functionality
- `functions.go` - Shared utility functions
- `coverage_json.go` - JSON conversion and comparison utilities
- `coverage_parser.go` - Coverage data parsing utilities
- `demo_json.go` - JSON functionality demonstrations

### Legacy Files
- `legacy_main.go` - Original main function (renamed to avoid conflicts)
- `simple_test_main.go` - Simple test functionality (function renamed)

### Test Files
- `*_test.go` - Various test files for different aspects of coverage functionality

## Usage

### Interactive Mode
```bash
go run testprogram.go functions.go coverage_json.go coverage_parser.go demo_json.go
```

### Command Line Mode
```bash
# Run specific functionality
go run testprogram.go functions.go coverage_json.go coverage_parser.go demo_json.go high-level
go run testprogram.go functions.go coverage_json.go coverage_parser.go demo_json.go synthetic
```

### Direct Tool Usage
```bash
# Run API tests directly
cd cmd/test-apis && go run test_highlevel_api.go
cd cmd/test-apis && go run test_simple_api.go

# Run synthetic demo directly
cd cmd/synthetic-demo && go run synthetic_proper.go
```

## Type Migration

All files have been updated to use the root `covutil` package types instead of internal imports:
- `covutil.Pod` instead of custom types
- `covutil.Profile` instead of custom structures
- `covutil.LoadCoverageSetFromFS` instead of internal parsers
- `covutil.WritePodToDirectory` instead of internal writers

## Testing

Run the tests with:
```bash
go test -v
```

Note: Some tests may require specific coverage data files to be present in the directory.