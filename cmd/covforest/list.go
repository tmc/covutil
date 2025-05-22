// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/tmc/covutil/internal/covforest"
)

var cmdList = &Command{
	UsageLine: "covforest list [-format=<format>] [-forest=<path>]",
	Short:     "list all coverage trees in the forest",
	Long: `
List displays all coverage trees in the forest with their metadata.

The -format flag specifies the output format: "table" (default), "json", or "csv".
The -forest flag specifies the forest file path (default: ~/.covforest/forest.json).

Example:

	covforest list
	covforest list -format=json
	covforest list -forest=/path/to/forest.json
`,
}

var (
	listFormat = cmdList.Flag.String("format", "table", "output format: table, json, csv")
	listForest = cmdList.Flag.String("forest", "", "forest file path (default: ~/.covforest/forest.json)")
)

func init() {
	cmdList.Run = runList
}

func runList(ctx context.Context, args []string) error {
	forestPath := *listForest
	if forestPath == "" {
		forestPath = covforest.DefaultForestPath()
	}

	forest, err := covforest.LoadFromFile(forestPath)
	if err != nil {
		return fmt.Errorf("failed to load forest: %v", err)
	}

	trees := forest.ListTrees()

	switch *listFormat {
	case "json":
		return outputListJSON(trees)
	case "csv":
		return outputListCSV(trees)
	default:
		return outputListTable(trees)
	}
}

func outputListTable(trees []*covforest.Tree) error {
	if len(trees) == 0 {
		fmt.Println("No coverage trees found in forest.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tMACHINE\tREPO\tBRANCH\tCOVERAGE\tUPDATED")

	for _, tree := range trees {
		coverage := "N/A"
		if tree.CoverageTree != nil {
			summary := tree.CoverageTree.Summary()
			coverage = fmt.Sprintf("%.1f%%", summary.CoverageRate*100)
		}

		repo := tree.Source.Repository
		if len(repo) > 30 {
			repo = "..." + repo[len(repo)-27:]
		}

		updated := tree.LastUpdated.Format("2006-01-02 15:04")

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			tree.ID, tree.Name, tree.Source.Machine, repo, tree.Source.Branch, coverage, updated)
	}

	return w.Flush()
}

func outputListJSON(trees []*covforest.Tree) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	output := struct {
		Trees []*covforest.Tree `json:"trees"`
		Count int               `json:"count"`
	}{
		Trees: trees,
		Count: len(trees),
	}

	return encoder.Encode(output)
}

func outputListCSV(trees []*covforest.Tree) error {
	fmt.Println("id,name,machine,repository,branch,commit,coverage_rate,total_lines,covered_lines,last_updated,created_at")

	for _, tree := range trees {
		coverageRate := "0"
		totalLines := "0"
		coveredLines := "0"

		if tree.CoverageTree != nil {
			summary := tree.CoverageTree.Summary()
			coverageRate = fmt.Sprintf("%.4f", summary.CoverageRate)
			totalLines = fmt.Sprintf("%d", summary.TotalLines)
			coveredLines = fmt.Sprintf("%d", summary.CoveredLines)
		}

		fmt.Printf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
			csvEscape(tree.ID),
			csvEscape(tree.Name),
			csvEscape(tree.Source.Machine),
			csvEscape(tree.Source.Repository),
			csvEscape(tree.Source.Branch),
			csvEscape(tree.Source.Commit),
			coverageRate,
			totalLines,
			coveredLines,
			tree.LastUpdated.Format(time.RFC3339),
			tree.CreatedAt.Format(time.RFC3339))
	}

	return nil
}

func csvEscape(s string) string {
	if strings.Contains(s, ",") || strings.Contains(s, "\"") || strings.Contains(s, "\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
