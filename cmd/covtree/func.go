// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/tmc/covutil/covtree"
)

var cmdFunc = &Command{
	UsageLine: "covtree func -i=<directory>",
	Short:     "report coverage percentages by function",
	Long: `
Func reports the coverage percentage for each function found in the
specified input directory tree.

The -i flag specifies a directory to scan recursively for coverage data.
The directory can contain nested subdirectories with coverage data files
produced by running "go build -cover" or similar. All found coverage
directories will be processed.

The -o flag specifies an output file. If not specified, output is written
to stdout.

Example:

	covtree func -i=./coverage-repo
	covtree func -i=/path/to/nested/coverage -o=functions.out
`,
}

var (
	funcInputDir = cmdFunc.Flag.String("i", "", "input directory to scan recursively for coverage data")
	funcOutput   = cmdFunc.Flag.String("o", "", "output file (default stdout)")
)

func init() {
	cmdFunc.Run = runFunc
}

type funcInfo struct {
	pkg      *covtree.PackageNode
	function *covtree.FunctionNode
}

func runFunc(ctx context.Context, args []string) error {
	if *funcInputDir == "" {
		return fmt.Errorf("must specify input directory with -i flag")
	}

	// Validate directory exists
	if _, err := os.Stat(*funcInputDir); os.IsNotExist(err) {
		return fmt.Errorf("input directory does not exist: %s", *funcInputDir)
	}

	// Load coverage data from nested repository
	tree := covtree.NewCoverageTree()
	if err := tree.LoadFromNestedRepository(*funcInputDir); err != nil {
		return fmt.Errorf("failed to load coverage data from %s: %v", *funcInputDir, err)
	}

	// Collect all functions
	var functions []funcInfo
	for _, pkg := range tree.Packages {
		for _, fn := range pkg.Functions {
			functions = append(functions, funcInfo{pkg: pkg, function: fn})
		}
	}

	// Sort by file:line:function
	sort.Slice(functions, func(i, j int) bool {
		a, b := functions[i], functions[j]
		if a.function.File != b.function.File {
			return a.function.File < b.function.File
		}
		if len(a.function.Units) > 0 && len(b.function.Units) > 0 {
			if a.function.Units[0].StartLine != b.function.Units[0].StartLine {
				return a.function.Units[0].StartLine < b.function.Units[0].StartLine
			}
		}
		return a.function.Name < b.function.Name
	})

	// Prepare output
	var output *os.File
	if *funcOutput != "" {
		f, err := os.Create(*funcOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer f.Close()
		output = f
	} else {
		output = os.Stdout
	}

	// Print function coverage
	for _, fi := range functions {
		fn := fi.function
		var line uint32
		if len(fn.Units) > 0 {
			line = fn.Units[0].StartLine
		}

		// Format similar to go tool covdata func
		funcName := fn.Name
		if strings.Contains(funcName, "$") {
			// Handle anonymous functions and closures
			funcName = strings.Replace(funcName, "$", ".", -1)
		}

		fmt.Fprintf(output, "%s:%d:\t\t%s\t\t%.1f%%\n",
			fn.File, line, funcName, fn.CoverageRate*100)
	}

	return nil
}
