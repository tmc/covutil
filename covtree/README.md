# covtree - Advanced Go Coverage Analysis with Metadata Extensions

Package covtree provides functionality for analyzing and visualizing Go coverage data in a hierarchical tree structure, with powerful metadata extensions beyond the standard `go tool covdata` format.

## Overview

covtree is designed to work with Go's native coverage format introduced in Go 1.20, but extends it with metadata that enables advanced use cases:

- **Cross-module coverage tracking**: Track coverage across module boundaries
- **Test attribution**: Know which tests cover which code
- **Coverage evolution**: Build coverage forests showing changes over time
- **Environment tracking**: Capture where and how coverage was collected

## Key Features

### Metadata Extensions

Unlike standard Go coverage tools, covtree supports metadata extensions that provide rich context:

- **GoTestName**: The specific test or test suite that generated the coverage
- **GoModuleName**: The Go module being tested (crucial for cross-module coverage)
- **GoTestPackage**: The package containing the test that generated coverage
- **TestType**: Classification of tests (unit, integration, e2e, cross-module)
- **TestRunID**: Unique identifier for correlating coverage across modules
- **Environment**: Machine, OS, architecture, and build tags used

### File System Abstraction

Load coverage data from various sources:
- Local directories
- Embedded file systems (embed.FS)
- ZIP archives
- Custom virtual file systems
- Depth-limited directory scanning

## Installation

```bash
go get github.com/tmc/covutil/covtree
```

## Usage Examples

### Basic Usage

```go
import "github.com/tmc/covutil/covtree"

// Create and load coverage data
tree := covtree.NewCoverageTree()
err := tree.LoadFromDirectory("/path/to/coverage/data")
if err != nil {
    log.Fatal(err)
}

// Get summary statistics
summary := tree.Summary()
fmt.Printf("Total Coverage: %.1f%%\n", summary.CoverageRate*100)
fmt.Printf("Packages: %d\n", summary.TotalPackages)
```

### Using Metadata Extensions

```go
// Set metadata to track test attribution
tree := covtree.NewCoverageTree()
tree.SetMetadata("GoTestName", "TestAPIIntegration")
tree.SetMetadata("GoModuleName", "github.com/myorg/service-a")
tree.SetMetadata("GoTestPackage", "github.com/myorg/service-a/integration")
tree.SetMetadata("TestType", "integration")

err := tree.LoadFromDirectory("/path/to/coverage/data")
```

### Cross-Module Coverage Tracking

```go
// In your test setup, set environment variables
os.Setenv("COVUTIL_TEST_RUN_ID", "integration-2024-01-15-001")
os.Setenv("COVUTIL_MODULE", "github.com/myorg/service-a")
os.Setenv("COVUTIL_TEST_NAME", "TestServiceIntegration")

// In each module being tested
tree := covtree.NewCoverageTree()
tree.LoadMetadataFromEnv()  // Automatically loads from environment

// Now all coverage data is tagged with the same TestRunID
// allowing correlation across modules
```

### Filtering with Metadata

```go
// Filter packages by metadata
filter := covtree.Filter{
    MaxCoverage: 0.8,
    Metadata: map[string]string{
        "TestType": "unit",
        "GoTestName": "TestUnit*",  // Supports wildcards
    },
}

lowCoverageUnitTests := tree.FilterPackages(filter)
for _, pkg := range lowCoverageUnitTests {
    fmt.Printf("%s: %.1f%% (tested by %s)\n", 
        pkg.ImportPath, 
        pkg.CoverageRate*100,
        pkg.Metadata["GoTestName"])
}
```

### Loading from fs.FS

```go
// Load from embedded files
//go:embed testdata/coverage/*
var coverageFS embed.FS

tree := covtree.NewCoverageTree()
opts := &covtree.LoadOptions{
    MaxDepth: 3,  // Limit scanning depth
}
err := tree.LoadFromFS(coverageFS, "testdata/coverage", opts)
```

## Environment Variables

covtree recognizes these environment variables for automatic metadata:

- `COVUTIL_TEST_RUN_ID`: Unique identifier for test runs
- `COVUTIL_MODULE`: Module being tested
- `COVUTIL_TEST_NAME`: Name of the test
- `COVUTIL_TEST_PACKAGE`: Package containing the test
- `COVUTIL_TEST_TYPE`: Type of test (unit, integration, etc.)
- `GITHUB_RUN_ID`, `GITHUB_REPOSITORY`, `GITHUB_REF`: GitHub Actions metadata
- `JENKINS_BUILD_ID`: Jenkins CI metadata
- `CI`: General CI indicator

## Integration with covforest

covtree works seamlessly with covforest to manage coverage across time and sources:

```go
import (
    "github.com/tmc/covutil/covtree"
    "github.com/tmc/covutil/internal/covforest"
)

forest := covforest.NewForest()

// Add coverage from different test types
unitTree := covtree.NewCoverageTree()
unitTree.SetMetadata("TestType", "unit")
unitTree.LoadFromDirectory("/coverage/unit")

forest.AddTree(&covforest.Tree{
    ID:   "unit-tests-2024-01-15",
    Name: "Unit Tests",
    CoverageTree: unitTree,
})
```

## Use Cases

### 1. Cross-Module Coverage

Track how integration tests in one module cover code in its dependencies:

```bash
# In module A's integration test
export COVUTIL_TEST_RUN_ID="integration-$(date +%s)"
export COVUTIL_MODULE="github.com/org/module-a"
go test -cover ./integration/...

# In module B (dependency), coverage will be tagged
# with the same TEST_RUN_ID, allowing correlation
```

### 2. Test Impact Analysis

Understand which tests cover which code:

```go
// Find all packages covered by integration tests
filter := covtree.Filter{
    Metadata: map[string]string{
        "TestType": "integration",
    },
}
integrationCoverage := tree.FilterPackages(filter)
```

### 3. Coverage Evolution

Track how coverage changes over time by storing metadata:

```go
tree.SetMetadata("CommitSHA", gitCommit)
tree.SetMetadata("BuildTime", time.Now().Format(time.RFC3339))
tree.SetMetadata("Branch", gitBranch)
```

## Compatibility

- Requires Go 1.20+ (uses modern coverage format)
- Compatible with `go test -cover` and `go build -cover`
- Works with custom coverage collection using `runtime/coverage`
- Integrates with covutil's synthetic coverage for non-Go artifacts

## Differences from Standard Tools

| Feature | go tool covdata | covtree |
|---------|----------------|---------|
| Binary format support | ✓ | ✓ |
| Text format support | ✓ | ✓ |
| Hierarchical tree view | ✗ | ✓ |
| Metadata extensions | ✗ | ✓ |
| Cross-module tracking | ✗ | ✓ |
| Test attribution | ✗ | ✓ |
| fs.FS support | ✗ | ✓ |
| Filtering by metadata | ✗ | ✓ |

## Performance Considerations

- Loads all coverage data into memory
- For very large datasets, consider processing in chunks
- Use `LoadOptions.MaxDepth` to limit directory traversal
- Metadata filtering happens in-memory after loading

## Future Enhancements

- Streaming API for large datasets
- Direct support for coverage profiles (.out files)
- Incremental loading and updates
- Coverage diff between trees
- Parallel loading for better performance