// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tmc/covutil/covtree"
)

var cmdDebug = &Command{
	UsageLine: "covtree debug -i=<directory>",
	Short:     "debug coverage directory scanning",
	Long: `
Debug scans a directory for coverage data and reports what it finds
without attempting to load the data.

The -i flag specifies a directory to scan recursively for coverage data.

Example:

	covtree debug -i=./coverage-repo
`,
}

var (
	debugInputDir = cmdDebug.Flag.String("i", "", "input directory to scan recursively for coverage data")
)

func init() {
	cmdDebug.Run = runDebug
}

func runDebug(ctx context.Context, args []string) error {
	if *debugInputDir == "" {
		return fmt.Errorf("must specify input directory with -i flag")
	}

	// Validate directory exists
	if _, err := os.Stat(*debugInputDir); os.IsNotExist(err) {
		return fmt.Errorf("input directory does not exist: %s", *debugInputDir)
	}

	// Scan for coverage directories
	coverageDirs, err := covtree.ScanForCoverageDirectories(*debugInputDir)
	if err != nil {
		return fmt.Errorf("failed to scan for coverage directories: %v", err)
	}

	fmt.Printf("Found %d coverage directories:\n", len(coverageDirs))
	for i, dir := range coverageDirs {
		fmt.Printf("%d. %s\n", i+1, dir)

		// List files in each directory
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Printf("   Error reading directory: %v\n", err)
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				info, _ := entry.Info()
				fmt.Printf("   - %s (%d bytes)\n", entry.Name(), info.Size())
			}
		}
		fmt.Println()
	}

	return nil
}
