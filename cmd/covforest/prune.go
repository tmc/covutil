// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/tmc/covutil/internal/covforest"
)

var cmdPrune = &Command{
	UsageLine: "covforest prune [-older-than=<duration>] [-forest=<path>]",
	Short:     "remove old or invalid coverage trees",
	Long: `
Prune removes coverage trees from the forest based on age or validity.

The -older-than flag specifies the age threshold (e.g., "30d", "1w", "24h").
Trees older than this threshold will be removed.

The -forest flag specifies the forest file path (default: ~/.covforest/forest.json).

Example:

	covforest prune -older-than=30d
	covforest prune -older-than=1w
`,
}

var (
	pruneOlderThan = cmdPrune.Flag.String("older-than", "", "remove trees older than this duration (e.g., 30d, 1w, 24h)")
	pruneForest    = cmdPrune.Flag.String("forest", "", "forest file path (default: ~/.covforest/forest.json)")
)

func init() {
	cmdPrune.Run = runPrune
}

func runPrune(ctx context.Context, args []string) error {
	forestPath := *pruneForest
	if forestPath == "" {
		forestPath = covforest.DefaultForestPath()
	}

	forest, err := covforest.LoadFromFile(forestPath)
	if err != nil {
		return fmt.Errorf("failed to load forest: %v", err)
	}

	var threshold time.Time
	if *pruneOlderThan != "" {
		duration, err := time.ParseDuration(*pruneOlderThan)
		if err != nil {
			return fmt.Errorf("invalid duration format: %v", err)
		}
		threshold = time.Now().Add(-duration)
	}

	var removed []string
	for id, tree := range forest.Trees {
		shouldRemove := false

		// Remove if older than threshold
		if !threshold.IsZero() && tree.LastUpdated.Before(threshold) {
			shouldRemove = true
		}

		// Remove if coverage tree is nil (invalid)
		if tree.CoverageTree == nil {
			shouldRemove = true
		}

		if shouldRemove {
			if err := forest.RemoveTree(id); err != nil {
				return fmt.Errorf("failed to remove tree %s: %v", id, err)
			}
			removed = append(removed, id)
		}
	}

	if len(removed) == 0 {
		fmt.Println("No trees to prune.")
		return nil
	}

	// Save updated forest
	if err := forest.SaveToFile(forestPath); err != nil {
		return fmt.Errorf("failed to save forest: %v", err)
	}

	fmt.Printf("Pruned %d trees:\n", len(removed))
	for _, id := range removed {
		fmt.Printf("  - %s\n", id)
	}
	fmt.Printf("Forest saved to: %s\n", forestPath)

	return nil
}
