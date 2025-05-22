// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package covforest provides functionality for managing multiple coverage trees
// from different sources (machines, repositories, timelines).
package covforest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/tmc/covutil/internal/covtree"
)

// Forest represents a collection of coverage trees from different sources
type Forest struct {
	Trees    map[string]*Tree `json:"trees"`
	Metadata ForestMetadata   `json:"metadata"`
}

// Tree represents a single coverage tree with metadata about its source
type Tree struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Source       TreeSource             `json:"source"`
	CoverageTree *covtree.CoverageTree  `json:"coverage_tree"`
	Metadata     map[string]interface{} `json:"metadata"`
	LastUpdated  time.Time              `json:"last_updated"`
	CreatedAt    time.Time              `json:"created_at"`
}

// TreeSource contains information about where a coverage tree originated
type TreeSource struct {
	Type       string    `json:"type"`       // "local", "remote", "ci", "git"
	Machine    string    `json:"machine"`    // hostname or CI worker ID
	Repository string    `json:"repository"` // repo URL or path
	Branch     string    `json:"branch"`     // git branch
	Commit     string    `json:"commit"`     // git commit hash
	Timestamp  time.Time `json:"timestamp"`  // when coverage was collected
	Path       string    `json:"path"`       // original path to coverage data
}

// ForestMetadata contains overall forest information
type ForestMetadata struct {
	Version     string    `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	LastUpdated time.Time `json:"last_updated"`
	TreeCount   int       `json:"tree_count"`
}

// NewForest creates a new empty coverage forest
func NewForest() *Forest {
	return &Forest{
		Trees: make(map[string]*Tree),
		Metadata: ForestMetadata{
			Version:   "1.0",
			CreatedAt: time.Now(),
		},
	}
}

// AddTree adds a coverage tree to the forest
func (f *Forest) AddTree(tree *Tree) error {
	if tree.ID == "" {
		return fmt.Errorf("tree ID cannot be empty")
	}

	if tree.CreatedAt.IsZero() {
		tree.CreatedAt = time.Now()
	}
	tree.LastUpdated = time.Now()

	f.Trees[tree.ID] = tree
	f.Metadata.TreeCount = len(f.Trees)
	f.Metadata.LastUpdated = time.Now()

	return nil
}

// RemoveTree removes a coverage tree from the forest
func (f *Forest) RemoveTree(id string) error {
	if _, exists := f.Trees[id]; !exists {
		return fmt.Errorf("tree %s not found", id)
	}

	delete(f.Trees, id)
	f.Metadata.TreeCount = len(f.Trees)
	f.Metadata.LastUpdated = time.Now()

	return nil
}

// GetTree retrieves a coverage tree by ID
func (f *Forest) GetTree(id string) (*Tree, error) {
	tree, exists := f.Trees[id]
	if !exists {
		return nil, fmt.Errorf("tree %s not found", id)
	}
	return tree, nil
}

// ListTrees returns all trees sorted by last updated time
func (f *Forest) ListTrees() []*Tree {
	trees := make([]*Tree, 0, len(f.Trees))
	for _, tree := range f.Trees {
		trees = append(trees, tree)
	}

	sort.Slice(trees, func(i, j int) bool {
		return trees[i].LastUpdated.After(trees[j].LastUpdated)
	})

	return trees
}

// Summary returns aggregate statistics across all trees
func (f *Forest) Summary() ForestSummary {
	summary := ForestSummary{
		TreeCount: len(f.Trees),
		Metadata:  f.Metadata,
	}

	packageMap := make(map[string]*PackageSummary)

	for _, tree := range f.Trees {
		if tree.CoverageTree == nil {
			continue
		}

		treeSummary := tree.CoverageTree.Summary()
		summary.TotalLines += treeSummary.TotalLines
		summary.CoveredLines += treeSummary.CoveredLines

		// Track per-package stats across trees
		for _, pkg := range tree.CoverageTree.Packages {
			if existing, ok := packageMap[pkg.ImportPath]; ok {
				existing.Trees = append(existing.Trees, TreeRef{
					ID:           tree.ID,
					Name:         tree.Name,
					TotalLines:   pkg.TotalLines,
					CoveredLines: pkg.CoveredLines,
					CoverageRate: pkg.CoverageRate,
				})
			} else {
				packageMap[pkg.ImportPath] = &PackageSummary{
					ImportPath: pkg.ImportPath,
					Trees: []TreeRef{{
						ID:           tree.ID,
						Name:         tree.Name,
						TotalLines:   pkg.TotalLines,
						CoveredLines: pkg.CoveredLines,
						CoverageRate: pkg.CoverageRate,
					}},
				}
			}
		}
	}

	// Convert map to slice
	for _, pkgSummary := range packageMap {
		summary.Packages = append(summary.Packages, pkgSummary)
	}

	// Sort packages by import path
	sort.Slice(summary.Packages, func(i, j int) bool {
		return summary.Packages[i].ImportPath < summary.Packages[j].ImportPath
	})

	if summary.TotalLines > 0 {
		summary.CoverageRate = float64(summary.CoveredLines) / float64(summary.TotalLines)
	}

	return summary
}

// ForestSummary provides aggregate statistics across all trees
type ForestSummary struct {
	TreeCount    int               `json:"tree_count"`
	TotalLines   int               `json:"total_lines"`
	CoveredLines int               `json:"covered_lines"`
	CoverageRate float64           `json:"coverage_rate"`
	Packages     []*PackageSummary `json:"packages"`
	Metadata     ForestMetadata    `json:"metadata"`
}

// PackageSummary shows how a package appears across different trees
type PackageSummary struct {
	ImportPath string    `json:"import_path"`
	Trees      []TreeRef `json:"trees"`
}

// TreeRef references a tree with coverage stats for a specific package
type TreeRef struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	TotalLines   int     `json:"total_lines"`
	CoveredLines int     `json:"covered_lines"`
	CoverageRate float64 `json:"coverage_rate"`
}

// SaveToFile saves the forest to a JSON file
func (f *Forest) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal forest: %v", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

// LoadFromFile loads a forest from a JSON file
func LoadFromFile(filename string) (*Forest, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return NewForest(), nil
		}
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var forest Forest
	if err := json.Unmarshal(data, &forest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal forest: %v", err)
	}

	return &forest, nil
}

// DefaultForestPath returns the default path for storing forest data
func DefaultForestPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./covforest.json"
	}
	return filepath.Join(home, ".covforest", "forest.json")
}
