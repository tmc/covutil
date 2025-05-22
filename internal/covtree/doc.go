// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package covtree provides functionality for analyzing and visualizing
// Go coverage data in a hierarchical tree structure.
//
// The package can load coverage data from directories containing
// coverage metadata and counter files produced by Go's coverage
// instrumentation, and organize this data into a tree structure
// that mirrors the package hierarchy.
//
// Example usage:
//
//	tree := covtree.NewCoverageTree()
//	err := tree.LoadFromDirectory("/path/to/coverage/data")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	summary := tree.Summary()
//	fmt.Printf("Coverage: %.1f%%\n", summary.CoverageRate*100)
//
//	// Filter packages with low coverage
//	filter := covtree.Filter{MaxCoverage: 0.8}
//	lowCoverage := tree.FilterPackages(filter)
package covtree
