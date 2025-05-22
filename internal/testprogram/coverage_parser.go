package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tmc/covutil"
)

// ParseCoverageFromDirectory reads coverage files from a directory and converts to JSON representation
func ParseCoverageFromDirectory(dir string) (*CoverageData, error) {
	// Use root covutil API to load coverage data
	fsys := os.DirFS(dir)
	coverageSet, err := covutil.LoadCoverageSetFromFS(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to load coverage set from directory %s: %w", dir, err)
	}

	if len(coverageSet.Pods) == 0 {
		return nil, fmt.Errorf("no coverage pods found in directory %s", dir)
	}

	// Return the first pod as CoverageData
	return &CoverageData{
		Pod: coverageSet.Pods[0],
	}, nil
}

// ParseMetadataFile reads and parses a coverage metadata file using covutil APIs
func ParseMetadataFile(filePath string) (*covutil.MetaFile, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open metadata file: %w", err)
	}
	defer f.Close()

	return covutil.LoadMetaFile(f, filePath)
}

// CreatePodFromFiles creates a Pod from metadata and counter files
func CreatePodFromFiles(metaFile, counterFile string) (*covutil.Pod, error) {
	// Use the directory containing the files to load as a coverage set
	dir := filepath.Dir(metaFile)
	fsys := os.DirFS(dir)
	coverageSet, err := covutil.LoadCoverageSetFromFS(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to load coverage set: %w", err)
	}

	if len(coverageSet.Pods) == 0 {
		return nil, fmt.Errorf("no pods found")
	}

	return coverageSet.Pods[0], nil
}

// ParseCounterFile reads and parses a coverage counter file using covutil APIs
func ParseCounterFile(filePath string) (*covutil.CounterFile, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open counter file: %w", err)
	}
	defer f.Close()

	return covutil.LoadCounterFile(f, filePath)
}
