// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/tmc/covutil/covtree"
)

var cmdPkglist = &Command{
	UsageLine: "covtree pkglist -i=<directory>",
	Short:     "report list of packages with coverage data",
	Long: `
Pkglist reports the import paths of packages for which coverage data
is available in the specified input directory tree.

The -i flag specifies a directory to scan recursively for coverage data.
The directory can contain nested subdirectories with coverage data files
produced by running "go build -cover" or similar. All found coverage
directories will be processed.

The -o flag specifies an output file. If not specified, output is written
to stdout.

Example:

	covtree pkglist -i=./coverage-repo
	covtree pkglist -i=/path/to/nested/coverage -o=packages.out
`,
}

var (
	pkglistInputDir = cmdPkglist.Flag.String("i", "", "input directory to scan recursively for coverage data")
	pkglistOutput   = cmdPkglist.Flag.String("o", "", "output file (default stdout)")
)

func init() {
	cmdPkglist.Run = runPkglist
}

func runPkglist(ctx context.Context, args []string) error {
	if *pkglistInputDir == "" {
		return fmt.Errorf("must specify input directory with -i flag")
	}

	// Validate directory exists
	if _, err := os.Stat(*pkglistInputDir); os.IsNotExist(err) {
		return fmt.Errorf("input directory does not exist: %s", *pkglistInputDir)
	}

	// Load coverage data from nested repository
	tree := covtree.NewCoverageTree()
	if err := tree.LoadFromNestedRepository(*pkglistInputDir); err != nil {
		return fmt.Errorf("failed to load coverage data from %s: %v", *pkglistInputDir, err)
	}

	// Get sorted package names
	packageNames := tree.GetPackageNames()
	sort.Strings(packageNames)

	// Prepare output
	var output *os.File
	if *pkglistOutput != "" {
		f, err := os.Create(*pkglistOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer f.Close()
		output = f
	} else {
		output = os.Stdout
	}

	// Print package names
	for _, pkgName := range packageNames {
		fmt.Fprintln(output, pkgName)
	}

	return nil
}
