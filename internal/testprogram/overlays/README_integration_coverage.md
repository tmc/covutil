# Integration Coverage Overlay

This overlay addresses [Go issue #60182](https://github.com/golang/go/issues/60182) - coverage data collection across multiple binaries in integration tests.

## Problem

When running integration tests that execute separate Go binaries, coverage data is not collected properly:

- Only `covmeta` files are generated, no `covcounters` files
- Tests show "coverage: 0.0% of statements" even with `-coverpkg=./...`
- Executed binaries don't write their coverage data to `GOCOVERDIR`

## Solution

Our overlay system provides three key enhancements:

### 1. Coverage Runtime Enhancement (`runtime/coverage/coverage.go`)

- **Coverage File Snapshots**: Records existing coverage files before test execution
- **Integration Mode Detection**: Activates when `GO_INTEGRATION_COVERAGE=1` is set
- **Forced Coverage Writing**: Ensures executed binaries write coverage data
- **New File Tracking**: Identifies coverage files created during test execution

### 2. Testing Package Enhancement (`testing/testing.go`)

- **`RunWithIntegrationCoverage()`**: Enhanced test runner for integration tests
- **`ExecuteBinaryWithCoverage()`**: Execute binaries with automatic coverage collection
- **Automatic Coverage Merging**: Collects coverage data from executed binaries
- **Per-Test Coverage Directories**: Organizes coverage data by test

### 3. Automatic Environment Setup

- **Environment Variable Management**: Automatically sets `GOCOVERDIR` for executed binaries
- **Coverage Directory Creation**: Creates and manages coverage subdirectories
- **Data Collection Hooks**: Installs exit hooks to ensure coverage data is written

## Usage

### Basic Integration Test with Coverage

```go
func TestIntegrationWithCoverage(t *testing.T) {
    // Build your test binary
    binaryPath := buildTestBinary(t)
    
    // Use the enhanced test runner
    t.RunWithIntegrationCoverage("my_integration_test", func(subT *testing.T) {
        // Execute the binary with automatic coverage collection
        if err := subT.ExecuteBinaryWithCoverage(binaryPath, "arg1", "arg2"); err != nil {
            subT.Fatalf("Binary execution failed: %v", err)
        }
        
        // Coverage data is automatically collected and merged
    })
}
```

### Manual Coverage Collection

```go
func TestManualCoverageCollection(t *testing.T) {
    // Set integration coverage mode
    os.Setenv("GO_INTEGRATION_COVERAGE", "1")
    os.Setenv("GOCOVERDIR", "./coverage_data")
    
    // Run your integration tests...
    
    // Get new coverage files generated during tests
    if newFiles, err := coverage.GetNewCoverageFiles(); err == nil {
        t.Logf("Generated %d new coverage files", len(newFiles))
    }
    
    // Force any remaining coverage data to be written
    coverage.ForceWriteCoverageData()
}
```

### Command Line Usage

```bash
# Generate the integration overlay
go run generate_integration_overlay.go full

# Run tests with integration coverage
export GO_INTEGRATION_COVERAGE=1
export GOCOVERDIR=./coverage
go test -overlay=overlay_integration.json -coverpkg=./... ./...

# Generate coverage report
go tool covdata textfmt -i=./coverage -o=coverage.out
go tool cover -html=coverage.out
```

## Key Features

### Coverage File Snapshots
- Records existing coverage files before test execution
- Only processes new files created during integration tests
- Prevents duplicate coverage data collection

### Binary Coverage Collection
- Automatically sets `GOCOVERDIR` for executed binaries
- Creates temporary coverage directories for each binary
- Merges coverage data back to main directory after execution

### Enhanced Testing Methods
- `RunWithIntegrationCoverage()` for test execution with coverage
- `ExecuteBinaryWithCoverage()` for binary execution with coverage
- Automatic cleanup and data aggregation

### Environment Management
- Automatic environment variable setup for executed binaries
- Coverage directory creation and management
- Proper cleanup after test completion

## How It Solves Go Issue #60182

1. **Addresses Missing Counter Files**: Forces coverage data writing from executed binaries
2. **Proper GOCOVERDIR Propagation**: Automatically sets coverage directory for all executed binaries
3. **Coverage Data Aggregation**: Collects and merges coverage data from multiple binary executions
4. **Integration Test Support**: Provides testing methods specifically designed for integration scenarios

## File Structure

```
overlays/
├── runtime/coverage/
│   ├── coverage.go.in                    # Enhanced coverage runtime
│   └── coverage_integration.go.in        # Integration-specific enhancements
├── testing/
│   ├── testing.go.in                     # Enhanced testing package
│   └── testing_integration.go.in         # Integration testing methods
├── generate_integration_overlay.go       # Overlay generator
├── overlay_integration.json              # Generated overlay configuration
└── README_integration_coverage.md        # This documentation
```

## Benefits

- ✅ **Fixes Coverage Collection**: Solves the core issue of missing coverage data
- ✅ **Backward Compatible**: Works with existing test suites
- ✅ **Automatic Setup**: Minimal configuration required
- ✅ **Detailed Logging**: Provides visibility into coverage collection process
- ✅ **Clean Integration**: Uses Go's overlay system without modifying source code
- ✅ **Testscript Compatible**: Works with scripttest-style integration tests

This overlay system provides a comprehensive solution to Go issue #60182, enabling proper coverage collection in integration test scenarios where multiple binaries are executed.