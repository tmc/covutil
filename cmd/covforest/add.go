// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/tmc/covutil/internal/covforest"
	"github.com/tmc/covutil/internal/covtree"
)

var cmdAdd = &Command{
	UsageLine: "covforest add -i=<directory> -name=<name> [-machine=<machine>] [-repo=<repo>] [-branch=<branch>]",
	Short:     "add a coverage tree to the forest",
	Long: `
Add processes a coverage directory and adds it as a tree to the forest.

The -i flag specifies the directory containing coverage data.
The -name flag specifies a human-readable name for this tree.
The -machine flag specifies the machine/host where coverage was collected.
The -repo flag specifies the repository URL or path.
The -branch flag specifies the git branch.
The -forest flag specifies the forest file path (default: ~/.covforest/forest.json).

The command will attempt to automatically detect git information if run
within a git repository.

Example:

	covforest add -i=./coverage -name="main-branch-ci"
	covforest add -i=/tmp/coverage -name="feature-branch" -machine="ci-worker-1" -repo="github.com/example/repo"
`,
}

var (
	addInputDir = cmdAdd.Flag.String("i", "", "input directory containing coverage data")
	addName     = cmdAdd.Flag.String("name", "", "human-readable name for this tree")
	addMachine  = cmdAdd.Flag.String("machine", "", "machine/host where coverage was collected")
	addRepo     = cmdAdd.Flag.String("repo", "", "repository URL or path")
	addBranch   = cmdAdd.Flag.String("branch", "", "git branch")
	addForest   = cmdAdd.Flag.String("forest", "", "forest file path (default: ~/.covforest/forest.json)")
)

func init() {
	cmdAdd.Run = runAdd
}

func runAdd(ctx context.Context, args []string) error {
	if *addInputDir == "" {
		return fmt.Errorf("must specify input directory with -i flag")
	}
	if *addName == "" {
		return fmt.Errorf("must specify tree name with -name flag")
	}

	// Validate directory exists
	if _, err := os.Stat(*addInputDir); os.IsNotExist(err) {
		return fmt.Errorf("input directory does not exist: %s", *addInputDir)
	}

	// Load coverage data
	tree := covtree.NewCoverageTree()
	if err := tree.LoadFromNestedRepository(*addInputDir); err != nil {
		return fmt.Errorf("failed to load coverage data from %s: %v", *addInputDir, err)
	}

	// Detect git information if not provided
	source := covforest.TreeSource{
		Type:      "local",
		Path:      *addInputDir,
		Timestamp: time.Now(),
	}

	if *addMachine != "" {
		source.Machine = *addMachine
	} else {
		if hostname, err := os.Hostname(); err == nil {
			source.Machine = hostname
		}
	}

	if *addRepo != "" {
		source.Repository = *addRepo
	} else {
		if repo := detectGitRepo(); repo != "" {
			source.Repository = repo
		}
	}

	if *addBranch != "" {
		source.Branch = *addBranch
	} else {
		if branch := detectGitBranch(); branch != "" {
			source.Branch = branch
		}
	}

	if commit := detectGitCommit(); commit != "" {
		source.Commit = commit
	}

	// Create tree
	forestTree := &covforest.Tree{
		ID:           generateTreeID(*addName, source),
		Name:         *addName,
		Source:       source,
		CoverageTree: tree,
		Metadata: map[string]interface{}{
			"added_by": "covforest add",
		},
	}

	// Load forest
	forestPath := *addForest
	if forestPath == "" {
		forestPath = covforest.DefaultForestPath()
	}

	forest, err := covforest.LoadFromFile(forestPath)
	if err != nil {
		return fmt.Errorf("failed to load forest: %v", err)
	}

	// Add tree to forest
	if err := forest.AddTree(forestTree); err != nil {
		return fmt.Errorf("failed to add tree to forest: %v", err)
	}

	// Save forest
	if err := forest.SaveToFile(forestPath); err != nil {
		return fmt.Errorf("failed to save forest: %v", err)
	}

	fmt.Printf("Added tree %s (%s) to forest\n", forestTree.ID, forestTree.Name)
	fmt.Printf("Forest saved to: %s\n", forestPath)

	summary := tree.Summary()
	fmt.Printf("Coverage: %.1f%% (%d/%d lines, %d packages)\n",
		summary.CoverageRate*100, summary.CoveredLines, summary.TotalLines, summary.TotalPackages)

	return nil
}

func generateTreeID(name string, source covforest.TreeSource) string {
	// Create a unique ID based on name, machine, and timestamp
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.ReplaceAll(id, "_", "-")

	if source.Machine != "" {
		id += "-" + strings.ToLower(source.Machine)
	}

	timestamp := source.Timestamp.Format("20060102-150405")
	id += "-" + timestamp

	return id
}

func detectGitRepo() string {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func detectGitBranch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func detectGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
