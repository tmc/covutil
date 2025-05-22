// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package covtree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/covutil/internal/coverage/pods"
)

// ScanForCoverageDirectories recursively scans the given root directory
// for coverage data directories and returns all found pod directories.
func ScanForCoverageDirectories(root string) ([]string, error) {
	var coverageDirs []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		// Check if this directory contains coverage files
		if containsCoverageFiles(path) {
			coverageDirs = append(coverageDirs, path)
		}

		return nil
	})

	return coverageDirs, err
}

// containsCoverageFiles checks if a directory contains Go coverage files
// (covmeta.* or covcounters.* files)
func containsCoverageFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	hasMetaFile := false
	hasCounterFile := false

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, "covmeta.") {
			hasMetaFile = true
		}
		if strings.HasPrefix(name, "covcounters.") {
			hasCounterFile = true
		}

		// Need both meta and counter files to be a valid coverage directory
		if hasMetaFile && hasCounterFile {
			return true
		}
	}

	return false
}

// LoadFromNestedRepository loads coverage data from a nested repository structure.
// It recursively scans the root directory for coverage data directories and loads
// data from all found directories.
func (ct *CoverageTree) LoadFromNestedRepository(root string) error {
	coverageDirs, err := ScanForCoverageDirectories(root)
	if err != nil {
		return err
	}

	if len(coverageDirs) == 0 {
		return &NoCoverageDataError{Dir: root}
	}

	// Load coverage data from all discovered directories
	loadedCount := 0
	for _, dir := range coverageDirs {
		pods, err := pods.CollectPods([]string{dir}, true)
		if err != nil {
			// Continue with other directories if one fails
			continue
		}

		for _, pod := range pods {
			if err := ct.loadPod(pod); err != nil {
				// Continue with other pods if one fails
				continue
			}
			loadedCount++
		}
	}

	if loadedCount == 0 && len(coverageDirs) > 0 {
		return &CoverageParseError{
			Dir:   root,
			Count: len(coverageDirs),
		}
	}

	ct.calculateCoverage()
	return nil
}

// NoCoverageDataError is returned when no coverage data is found in the specified directory
type NoCoverageDataError struct {
	Dir string
}

func (e *NoCoverageDataError) Error() string {
	return "no coverage data found in directory: " + e.Dir
}

// CoverageParseError is returned when coverage directories are found but cannot be parsed
type CoverageParseError struct {
	Dir   string
	Count int
}

func (e *CoverageParseError) Error() string {
	return fmt.Sprintf("found %d coverage directories in %s but failed to parse any coverage data", e.Count, e.Dir)
}
