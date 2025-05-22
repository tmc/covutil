// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/tmc/covutil/internal/covtree"
)

var cmdPercent = &Command{
	UsageLine: "covtree percent -i=<directory>",
	Short:     "report coverage percentages by package",
	Long: `
Percent reports the coverage percentage for each package found in the
specified input directory tree.

The -i flag specifies a directory to scan recursively for coverage data.
The directory can contain nested subdirectories with coverage data files
produced by running "go build -cover" or similar. All found coverage
directories will be processed.

The -o flag specifies an output file. If not specified, output is written
to stdout.

Example:

	covtree percent -i=./coverage-repo
	covtree percent -i=/path/to/nested/coverage -o=coverage.out
`,
}

var (
	percentInputDir = cmdPercent.Flag.String("i", "", "input directory to scan recursively for coverage data")
	percentOutput   = cmdPercent.Flag.String("o", "", "output file (default stdout)")
)

func init() {
	cmdPercent.Run = runPercent
}

func runPercent(ctx context.Context, args []string) error {
	if *percentInputDir == "" {
		return fmt.Errorf("must specify input directory with -i flag")
	}

	// Validate directory exists
	if _, err := os.Stat(*percentInputDir); os.IsNotExist(err) {
		return fmt.Errorf("input directory does not exist: %s", *percentInputDir)
	}

	// Load coverage data from nested repository
	tree := covtree.NewCoverageTree()
	if err := tree.LoadFromNestedRepository(*percentInputDir); err != nil {
		return fmt.Errorf("failed to load coverage data from %s: %v", *percentInputDir, err)
	}

	// Collect and sort packages
	packages := make([]*covtree.PackageNode, 0, len(tree.Packages))
	for _, pkg := range tree.Packages {
		packages = append(packages, pkg)
	}
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].ImportPath < packages[j].ImportPath
	})

	// Prepare output
	var output *os.File
	if *percentOutput != "" {
		f, err := os.Create(*percentOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer f.Close()
		output = f
	} else {
		output = os.Stdout
	}

	// Print coverage percentages
	for _, pkg := range packages {
		fmt.Fprintf(output, "%s\tcoverage: %.1f%% of statements\n",
			pkg.ImportPath, pkg.CoverageRate*100)
	}

	return nil
}