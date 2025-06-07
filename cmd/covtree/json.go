// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tmc/covutil/covtree"
)

var cmdJSON = &Command{
	UsageLine: "covtree json -i=<directory> [-o=<file>]",
	Short:     "convert coverage data to NDJSON format",
	Long: `
JSON converts coverage data to newline-delimited JSON (NDJSON) format,
with each line representing a coverage record.

The -i flag specifies a directory to scan recursively for coverage data.
The directory can contain nested subdirectories with coverage data files
produced by running "go build -cover" or similar.

The -o flag specifies an output file. If not specified,
output is written to stdout.

Example:

	covtree json -i=./coverage-repo
	covtree json -i=/path/to/coverage -o=coverage.ndjson
	covtree json -i=./coverage | jq .

For watch mode with auto-reload, use covtree-web with the -watch flag.
`,
}

var (
	jsonInputDir = cmdJSON.Flag.String("i", "", "input directory to scan recursively for coverage data")
	jsonOutput   = cmdJSON.Flag.String("o", "", "output file (default stdout)")
)

func init() {
	cmdJSON.Run = runJSON
}

// CoverageRecord represents a single coverage record in NDJSON format
type CoverageRecord struct {
	Timestamp    time.Time              `json:"timestamp"`
	Source       string                 `json:"source"`
	Package      string                 `json:"package"`
	Function     string                 `json:"function,omitempty"`
	File         string                 `json:"file,omitempty"`
	StartLine    uint32                 `json:"start_line,omitempty"`
	EndLine      uint32                 `json:"end_line,omitempty"`
	TotalLines   int                    `json:"total_lines"`
	CoveredLines int                    `json:"covered_lines"`
	CoverageRate float64                `json:"coverage_rate"`
	Type         string                 `json:"type"` // "package", "function", or "unit"
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

func runJSON(ctx context.Context, args []string) error {
	if *jsonInputDir == "" {
		return fmt.Errorf("must specify input directory with -i flag")
	}

	// Validate directory exists
	if _, err := os.Stat(*jsonInputDir); os.IsNotExist(err) {
		return fmt.Errorf("input directory does not exist: %s", *jsonInputDir)
	}

	// Prepare output
	var output *os.File
	if *jsonOutput != "" {
		f, err := os.Create(*jsonOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %v", err)
		}
		defer f.Close()
		output = f
	} else {
		output = os.Stdout
	}

	encoder := json.NewEncoder(output)

	// Process directory once
	return processDirectoryToJSON(*jsonInputDir, encoder)
}

func processDirectoryToJSON(dir string, encoder *json.Encoder) error {
	tree := covtree.NewCoverageTree()
	if err := tree.LoadFromNestedRepository(dir); err != nil {
		// Try to get partial data even if some files fail
		if _, ok := err.(*covtree.CoverageParseError); !ok {
			return fmt.Errorf("failed to load coverage data from %s: %v", dir, err)
		}
	}

	timestamp := time.Now()
	source := filepath.Base(dir)

	// Emit package-level records
	for _, pkg := range tree.Packages {
		record := CoverageRecord{
			Timestamp:    timestamp,
			Source:       source,
			Package:      pkg.ImportPath,
			TotalLines:   pkg.TotalLines,
			CoveredLines: pkg.CoveredLines,
			CoverageRate: pkg.CoverageRate,
			Type:         "package",
			Metadata: map[string]interface{}{
				"module_path": pkg.ModulePath,
				"meta_file":   pkg.MetaFile,
			},
		}
		if err := encoder.Encode(record); err != nil {
			return fmt.Errorf("failed to encode JSON: %v", err)
		}

		// Emit function-level records
		for _, fn := range pkg.Functions {
			fnRecord := CoverageRecord{
				Timestamp:    timestamp,
				Source:       source,
				Package:      pkg.ImportPath,
				Function:     fn.Name,
				File:         fn.File,
				TotalLines:   fn.TotalLines,
				CoveredLines: fn.CoveredLines,
				CoverageRate: fn.CoverageRate,
				Type:         "function",
				Metadata: map[string]interface{}{
					"is_literal": fn.IsLiteral,
				},
			}
			if err := encoder.Encode(fnRecord); err != nil {
				return fmt.Errorf("failed to encode JSON: %v", err)
			}

			// Emit unit-level records
			for _, unit := range fn.Units {
				unitRecord := CoverageRecord{
					Timestamp:    timestamp,
					Source:       source,
					Package:      pkg.ImportPath,
					Function:     fn.Name,
					File:         fn.File,
					StartLine:    unit.StartLine,
					EndLine:      unit.EndLine,
					TotalLines:   int(unit.EndLine - unit.StartLine + 1),
					CoveredLines: 0,
					CoverageRate: 0.0,
					Type:         "unit",
					Metadata: map[string]interface{}{
						"start_col": unit.StartCol,
						"end_col":   unit.EndCol,
						"count":     unit.Count,
						"covered":   unit.Covered,
					},
				}
				if unit.Covered {
					unitRecord.CoveredLines = unitRecord.TotalLines
					unitRecord.CoverageRate = 1.0
				}
				if err := encoder.Encode(unitRecord); err != nil {
					return fmt.Errorf("failed to encode JSON: %v", err)
				}
			}
		}
	}

	return nil
}
