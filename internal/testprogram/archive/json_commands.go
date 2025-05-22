package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// convertToJson converts coverage data from the output directory to JSON format
func convertToJson() {
	fmt.Println("\n=== CONVERTING COVERAGE DATA TO JSON ===")

	// Check if output directory exists
	if _, err := os.Stat(*outputDir); os.IsNotExist(err) {
		fmt.Printf("Coverage data directory %s does not exist\n", *outputDir)
		fmt.Println("Run without flags first to generate coverage data, then use -to-json")
		os.Exit(1)
	}

	// Parse coverage data
	coverageData, err := ParseCoverageFromDirectory(*outputDir)
	if err != nil {
		fmt.Printf("Error parsing coverage data: %v\n", err)
		os.Exit(1)
	}

	// Convert to JSON
	jsonStr, err := coverageData.ToJSON()
	if err != nil {
		fmt.Printf("Error converting to JSON: %v\n", err)
		os.Exit(1)
	}

	// Write JSON to file
	jsonFile := filepath.Join(*outputDir, "coverage.json")
	if err := os.WriteFile(jsonFile, []byte(jsonStr), 0644); err != nil {
		fmt.Printf("Error writing JSON file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Coverage data summary: %s\n", coverageData.Summary())
	fmt.Printf("JSON representation written to: %s\n", jsonFile)
}

// generateFromJson generates synthetic coverage data from a JSON file
func generateFromJson(jsonFile string) {
	fmt.Println("\n=== GENERATING COVERAGE DATA FROM JSON ===")

	// Read JSON file
	jsonData, err := os.ReadFile(jsonFile)
	if err != nil {
		fmt.Printf("Error reading JSON file %s: %v\n", jsonFile, err)
		os.Exit(1)
	}

	// Parse JSON
	coverageData, err := FromJSON(string(jsonData))
	if err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded coverage data: %s\n", coverageData.Summary())

	// Clear existing covdata directory if it exists
	if _, err := os.Stat(*outputDir); !os.IsNotExist(err) {
		fmt.Printf("Removing existing directory: %s\n", *outputDir)
		if err := os.RemoveAll(*outputDir); err != nil {
			fmt.Printf("Error removing directory: %v\n", err)
			os.Exit(1)
		}
	}

	// Create coverage data directory
	fmt.Printf("Creating directory: %s\n", *outputDir)
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Generate synthetic files from JSON
	if err := generateSyntheticFromCoverageData(coverageData, *outputDir); err != nil {
		fmt.Printf("Error generating synthetic data: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Synthetic coverage data generated in: %s\n", *outputDir)

	// Run covdata tool if not skipped
	if !*skipCovdata {
		fmt.Printf("\nRunning command: go tool covdata percent -i=%s\n", *outputDir)
		cmd := exec.Command("go", "tool", "covdata", "percent", fmt.Sprintf("-i=%s", *outputDir))
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Error running covdata command: %v\n", err)
			fmt.Printf("Output: %s\n", string(output))
		} else {
			fmt.Printf("Coverage percentage: %s\n", string(output))
		}
	}
}

// compareCoverageData compares coverage data from two directories
func compareCoverageData(dir1, dir2 string) {
	fmt.Println("\n=== COMPARING COVERAGE DATA ===")

	// Parse first directory
	fmt.Printf("Parsing coverage data from %s...\n", dir1)
	data1, err := ParseCoverageFromDirectory(dir1)
	if err != nil {
		fmt.Printf("Error parsing coverage data from %s: %v\n", dir1, err)
		os.Exit(1)
	}

	// Parse second directory
	fmt.Printf("Parsing coverage data from %s...\n", dir2)
	data2, err := ParseCoverageFromDirectory(dir2)
	if err != nil {
		fmt.Printf("Error parsing coverage data from %s: %v\n", dir2, err)
		os.Exit(1)
	}

	// Compare the data
	fmt.Printf("\nDirectory 1 (%s): %s\n", dir1, data1.Summary())
	fmt.Printf("Directory 2 (%s): %s\n", dir2, data2.Summary())

	// Show diff
	diff := data1.Diff(data2)
	if diff == "" {
		fmt.Println("\n✅ Coverage data is identical!")
	} else {
		fmt.Println("\n📋 Differences found:")
		fmt.Println(diff)
	}

	// Check if they're equal
	if data1.Equal(data2) {
		fmt.Println("✅ Coverage data structures are equivalent")
	} else {
		fmt.Println("❌ Coverage data structures differ")
	}
}
