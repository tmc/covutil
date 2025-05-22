// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/tmc/covutil/internal/covforest"
)

var cmdSummary = &Command{
	UsageLine: "covforest summary [-format=<format>] [-forest=<path>]",
	Short:     "show summary statistics across all trees",
	Long: `
Summary displays aggregate statistics across all coverage trees in the forest.

The -format flag specifies the output format: "table" (default) or "json".
The -forest flag specifies the forest file path (default: ~/.covforest/forest.json).

Example:

	covforest summary
	covforest summary -format=json
`,
}

var (
	summaryFormat = cmdSummary.Flag.String("format", "table", "output format: table, json")
	summaryForest = cmdSummary.Flag.String("forest", "", "forest file path (default: ~/.covforest/forest.json)")
)

func init() {
	cmdSummary.Run = runSummary
}

func runSummary(ctx context.Context, args []string) error {
	forestPath := *summaryForest
	if forestPath == "" {
		forestPath = covforest.DefaultForestPath()
	}

	forest, err := covforest.LoadFromFile(forestPath)
	if err != nil {
		return fmt.Errorf("failed to load forest: %v", err)
	}

	summary := forest.Summary()

	switch *summaryFormat {
	case "json":
		return outputSummaryJSON(summary)
	default:
		return outputSummaryTable(summary)
	}
}

func outputSummaryTable(summary covforest.ForestSummary) error {
	fmt.Printf("Forest Summary\n")
	fmt.Printf("=============\n")
	fmt.Printf("Trees: %d\n", summary.TreeCount)
	fmt.Printf("Total Lines: %s\n", formatNumber(summary.TotalLines))
	fmt.Printf("Covered Lines: %s\n", formatNumber(summary.CoveredLines))
	fmt.Printf("Coverage Rate: %.2f%%\n\n", summary.CoverageRate*100)

	if len(summary.Packages) > 0 {
		fmt.Printf("Package Coverage Across Trees\n")
		fmt.Printf("============================\n")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PACKAGE\tTREES\tAVG COVERAGE\tRANGE")

		for _, pkg := range summary.Packages {
			if len(pkg.Trees) == 0 {
				continue
			}

			// Calculate average and range
			var total float64
			minCov := pkg.Trees[0].CoverageRate
			maxCov := pkg.Trees[0].CoverageRate

			for _, tree := range pkg.Trees {
				total += tree.CoverageRate
				if tree.CoverageRate < minCov {
					minCov = tree.CoverageRate
				}
				if tree.CoverageRate > maxCov {
					maxCov = tree.CoverageRate
				}
			}

			avg := total / float64(len(pkg.Trees))
			rangeStr := fmt.Sprintf("%.1f%%-%.1f%%", minCov*100, maxCov*100)

			fmt.Fprintf(w, "%s\t%d\t%.1f%%\t%s\n",
				pkg.ImportPath, len(pkg.Trees), avg*100, rangeStr)
		}

		w.Flush()
	}

	return nil
}

func outputSummaryJSON(summary covforest.ForestSummary) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(summary)
}

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}
