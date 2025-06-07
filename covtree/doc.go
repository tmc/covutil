// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package covtree provides functionality for analyzing and visualizing
// Go coverage data in a hierarchical tree structure, with powerful
// metadata extensions beyond the standard go tool covdata format.
//
// The package can load coverage data from directories containing
// coverage metadata and counter files produced by Go's coverage
// instrumentation, and organize this data into a tree structure
// that mirrors the package hierarchy while enriching it with
// additional metadata.
//
// # Overview
//
// covtree is designed to work with Go's native coverage format introduced
// in Go 1.20. It reads binary coverage data files (covmeta and covcounters)
// and provides a rich API for analysis, filtering, and visualization of
// coverage information. Unlike the standard go tool covdata, covtree
// extends the coverage format with metadata that enables advanced
// use cases like cross-module testing, test attribution, and coverage
// tracking across different test types.
//
// # Metadata Extensions
//
// covtree goes beyond the standard coverage format by supporting metadata
// extensions that provide context about how and where coverage was collected:
//
//   - GoTestName: The specific test or test suite that generated the coverage
//   - GoModuleName: The Go module being tested (crucial for cross-module coverage)
//   - GoTestPackage: The package containing the test that generated coverage
//   - TestType: Classification of tests (unit, integration, e2e, cross-module)
//   - TestRunID: Unique identifier for correlating coverage across modules
//   - Environment: Machine, OS, architecture, and build tags used
//
// These extensions enable scenarios like:
//   - Tracking which tests cover which code across module boundaries
//   - Aggregating coverage from different test types and environments
//   - Building coverage forests that show evolution over time
//   - Understanding the true coverage of a module including its use by dependents
//
// # Core Types
//
// The main types provided by this package are:
//
//   - CoverageTree: The root structure that holds all coverage data with metadata
//   - PackageNode: Represents coverage data for a single Go package with extensions
//   - FunctionNode: Represents coverage data for a single function
//   - CoverableUnitNode: Represents a single coverage unit (code block)
//   - DirectoryNode: Represents a directory in the package hierarchy
//   - LoadOptions: Configuration options for loading coverage data
//   - Filter: Criteria for filtering packages based on coverage metrics
//
// # Loading Coverage Data
//
// The package supports multiple ways to load coverage data:
//
//   - LoadFromDirectory: Load from a single coverage directory
//   - LoadFromFS: Load from any io/fs.FS implementation
//   - LoadFromNestedRepository: Recursively scan and load from nested directories
//   - LoadWithMetadata: Load coverage data with custom metadata extensions
//
// # File System Abstraction
//
// As of Go 1.16, the package supports loading coverage data through the io/fs
// interface, enabling advanced use cases:
//
//   - Loading from embedded file systems (embed.FS)
//   - Loading from ZIP archives
//   - Loading from custom virtual file systems
//   - Depth-limited directory scanning
//
// # Filtering and Analysis
//
// The package provides powerful filtering capabilities:
//
//   - Filter by minimum/maximum coverage percentage
//   - Filter by package name patterns
//   - Filter by function name patterns
//   - Filter by metadata (test name, module, test type)
//   - Combine multiple filter criteria
//
// # Example Usage
//
// Basic usage:
//
//	tree := covtree.NewCoverageTree()
//	err := tree.LoadFromDirectory("/path/to/coverage/data")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	summary := tree.Summary()
//	fmt.Printf("Total Coverage: %.1f%%\n", summary.CoverageRate*100)
//	fmt.Printf("Packages: %d\n", summary.TotalPackages)
//	fmt.Printf("Lines: %d/%d\n", summary.CoveredLines, summary.TotalLines)
//
// Loading with metadata extensions:
//
//	tree := covtree.NewCoverageTree()
//	tree.SetMetadata("GoTestName", "TestAPIIntegration")
//	tree.SetMetadata("GoModuleName", "github.com/myorg/service-a")
//	tree.SetMetadata("GoTestPackage", "github.com/myorg/service-a/integration")
//	tree.SetMetadata("TestType", "integration")
//
//	err := tree.LoadFromDirectory("/path/to/coverage/data")
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Cross-module coverage tracking:
//
//	// In service-a's test
//	os.Setenv("COVUTIL_TEST_RUN_ID", "integration-2024-01-15-001")
//	os.Setenv("COVUTIL_MODULE", "github.com/myorg/service-a")
//
//	// In service-b (dependency)
//	tree := covtree.NewCoverageTree()
//	if runID := os.Getenv("COVUTIL_TEST_RUN_ID"); runID != "" {
//		tree.SetMetadata("TestRunID", runID)
//		tree.SetMetadata("GoModuleName", "github.com/myorg/service-b")
//		tree.SetMetadata("TestedBy", os.Getenv("COVUTIL_MODULE"))
//	}
//
// Using fs.FS with depth limiting:
//
//	tree := covtree.NewCoverageTree()
//	fsys := os.DirFS("/path/to/repo")
//	opts := &covtree.LoadOptions{
//		MaxDepth: 3,  // Only scan 3 levels deep
//	}
//	err := tree.LoadFromFS(fsys, ".", opts)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Loading from embedded files:
//
//	//go:embed testdata/coverage/*
//	var coverageFS embed.FS
//
//	tree := covtree.NewCoverageTree()
//	err := tree.LoadFromFS(coverageFS, "testdata/coverage", nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Filtering packages with metadata:
//
//	filter := covtree.Filter{
//		MaxCoverage: 0.8,
//		PackagePattern: "github.com/myproject/*",
//		Metadata: map[string]string{
//			"GoTestName": "TestUnit*",
//		},
//	}
//	lowCoverage := tree.FilterPackages(filter)
//	for _, pkg := range lowCoverage {
//		fmt.Printf("%s: %.1f%% (tested by %s)\n",
//			pkg.ImportPath,
//			pkg.CoverageRate*100,
//			pkg.Metadata["GoTestName"])
//	}
//
// Loading from nested repository structure:
//
//	tree := covtree.NewCoverageTree()
//	err := tree.LoadFromNestedRepository("/path/to/repo")
//	if err != nil {
//		// Handle partial errors
//		if parseErr, ok := err.(*covtree.CoverageParseError); ok {
//			fmt.Printf("Found %d coverage directories but failed to parse\n",
//				parseErr.Count)
//		}
//	}
//
// Accessing the directory tree structure:
//
//	tree := covtree.NewCoverageTree()
//	// ... load coverage data ...
//
//	// Walk the directory tree
//	var walk func(*covtree.DirectoryNode, string)
//	walk = func(dir *covtree.DirectoryNode, indent string) {
//		fmt.Printf("%s%s/ (%.1f%%)\n", indent, dir.Name,
//			float64(dir.CoveredLines)/float64(dir.TotalLines)*100)
//		for _, child := range dir.Children {
//			walk(child, indent+"  ")
//		}
//		for _, pkg := range dir.Packages {
//			fmt.Printf("%s  %s (%.1f%%)\n", indent, pkg.Name,
//				pkg.CoverageRate*100)
//		}
//	}
//	walk(tree.Root, "")
//
// # Integration with covforest
//
// covtree works seamlessly with the covforest package to manage
// coverage data across multiple test runs, machines, and time:
//
//	forest := covforest.NewForest()
//
//	// Add coverage from different sources
//	tree1 := covtree.NewCoverageTree()
//	tree1.SetMetadata("GoTestName", "TestUnit")
//	tree1.LoadFromDirectory("/coverage/unit")
//
//	forest.AddTree(&covforest.Tree{
//		ID:   "unit-tests-2024-01-15",
//		Name: "Unit Tests",
//		Source: covforest.TreeSource{
//			Type:    "ci",
//			Machine: "github-runner-01",
//		},
//		CoverageTree: tree1,
//	})
//
// # Error Handling
//
// The package defines several error types for different failure scenarios:
//
//   - NoCoverageDataError: No coverage data found in the specified location
//   - CoverageParseError: Coverage data found but parsing failed
//
// These errors allow for graceful handling of partial failures when loading
// coverage data from large directory structures.
//
// # Performance Considerations
//
// When working with large repositories:
//
//   - Use LoadOptions.MaxDepth to limit directory traversal
//   - Consider loading specific subdirectories rather than entire repositories
//   - The package loads all coverage data into memory; for very large datasets,
//     consider processing data in chunks
//   - Use metadata filters to reduce the amount of data processed
//
// # Compatibility
//
// This package is compatible with coverage data generated by:
//
//   - go test -cover
//   - go build -cover
//   - go test -coverprofile (through conversion)
//   - Custom coverage collection using Go's runtime/coverage package
//   - covutil's synthetic coverage system for non-Go artifacts
//
// The package requires Go 1.20 or later due to its use of the modern
// coverage format and fs.FS interfaces. The metadata extensions are
// designed to be backward compatible - standard tools will ignore
// the extended metadata while covtree-aware tools can leverage it.
package covtree
