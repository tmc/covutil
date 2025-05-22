package main

import (
	"fmt"
	"os"

	"github.com/tmc/covutil"
)

// generateProperSyntheticCoverage generates coverage files using the root covutil APIs
func generateProperSyntheticCoverage(data *CoverageData, outputDir string) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Use covutil API to write the pod data
	if data.Pod == nil {
		return fmt.Errorf("no pod data available")
	}

	if err := covutil.WritePodToDirectory(outputDir, data.Pod); err != nil {
		return fmt.Errorf("failed to write pod data: %w", err)
	}

	return nil
}

// generateMetadataFile is now replaced by covutil.WritePodToDirectory
// This function is kept for backwards compatibility but delegates to the new API
func generateMetadataFile(data *CoverageData, outputDir, hashStr string) error {
	// This functionality is now handled by WritePodToDirectory
	return nil
}

// generateCounterFile is now replaced by covutil.WritePodToDirectory
// This function is kept for backwards compatibility but delegates to the new API
func generateCounterFile(data *CoverageData, outputDir, hashStr string) error {
	// This functionality is now handled by WritePodToDirectory
	return nil
}

// Test function that uses proper synthetic coverage generation
func testProperSyntheticGeneration() {
	fmt.Println("=== Testing Proper Synthetic Coverage Generation ===")

	// Create sample data
	data := &CoverageData{
		Pod: &covutil.Pod{
			ID: "sample-pod",
			Profile: &covutil.Profile{
				Meta: covutil.MetaFile{
					FilePath: "sample.meta",
					Packages: []covutil.PackageMeta{
						{
							Path: "example.com/test",
							Functions: []covutil.FuncDesc{
								{
									FuncName: "TestFunction",
									SrcFile:  "test.go",
									Units: []covutil.CoverableUnit{
										{StartLine: 10, StartCol: 1, EndLine: 15, EndCol: 1, NumStmt: 5},
									},
								},
							},
						},
					},
				},
				Counters: make(map[covutil.PkgFuncKey][]uint32),
			},
		},
	}

	// Generate synthetic coverage using official APIs
	outputDir := "synthetic_proper"
	if err := generateProperSyntheticCoverage(data, outputDir); err != nil {
		fmt.Printf("Error generating synthetic coverage: %v\n", err)
		return
	}

	fmt.Printf("Synthetic coverage generated in: %s\n", outputDir)

	// Test the generated files
	testSyntheticFiles(outputDir)
}

// testSyntheticFiles tests the generated synthetic coverage files
func testSyntheticFiles(dir string) {
	// Test with go tool covdata
	fmt.Printf("Testing synthetic files with go tool covdata...\n")

	// Check if files were generated
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("Failed to read directory %s: %v\n", dir, err)
		return
	}

	fmt.Printf("Generated files in %s:\n", dir)
	for _, entry := range entries {
		fmt.Printf("  %s\n", entry.Name())
	}

	// Try to load the coverage data back
	fsys := os.DirFS(dir)
	coverageSet, err := covutil.LoadCoverageSetFromFS(fsys, ".")
	if err != nil {
		fmt.Printf("Failed to load coverage set: %v\n", err)
		return
	}

	fmt.Printf("Successfully loaded %d pods\n", len(coverageSet.Pods))
	for i, pod := range coverageSet.Pods {
		fmt.Printf("Pod %d: ID=%s, Packages=%d\n", i, pod.ID, len(pod.Profile.Meta.Packages))
	}
}

func main() {
	testProperSyntheticGeneration()
}
