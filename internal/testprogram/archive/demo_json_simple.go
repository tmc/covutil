package main

import (
	"fmt"
	"os"
)

// demonstrateJsonFunctionalitySimple shows off JSON functionality without covdata calls
func demonstrateJsonFunctionalitySimple() {
	fmt.Println("\n=== JSON FUNCTIONALITY DEMONSTRATION (SIMPLE) ===")

	// Load first JSON file
	fmt.Println("\nLoading sample_coverage.json...")
	data1, err := loadJsonFile("sample_coverage.json")
	if err != nil {
		fmt.Printf("Error loading JSON: %v\n", err)
		return
	}
	fmt.Printf("Data 1 Summary: %s\n", data1.Summary())

	// Load second JSON file
	fmt.Println("\nLoading sample_coverage2.json...")
	data2, err := loadJsonFile("sample_coverage2.json")
	if err != nil {
		fmt.Printf("Error loading JSON: %v\n", err)
		return
	}
	fmt.Printf("Data 2 Summary: %s\n", data2.Summary())

	// Compare the two datasets (basic comparison)
	fmt.Println("\n--- BASIC COMPARISON ---")

	// Compare basic metrics
	meta1Hash := data1.Metadata.Header.MetaFileHash
	meta2Hash := data2.Metadata.Header.MetaFileHash
	fmt.Printf("Metadata hashes match: %t\n", meta1Hash == meta2Hash)

	// Compare counter values
	if len(data1.Counters.Segments) > 0 && len(data2.Counters.Segments) > 0 {
		if len(data1.Counters.Segments[0].Functions) > 0 && len(data2.Counters.Segments[0].Functions) > 0 {
			counters1 := data1.Counters.Segments[0].Functions[0].Counters
			counters2 := data2.Counters.Segments[0].Functions[0].Counters
			fmt.Printf("First function counters: %v vs %v\n", counters1, counters2)
			fmt.Printf("Counter values match: %t\n", fmt.Sprintf("%v", counters1) == fmt.Sprintf("%v", counters2))
		}
	}

	// Show package information
	fmt.Println("\n--- PACKAGE DETAILS ---")
	if len(data1.Metadata.Packages) > 0 {
		pkg := data1.Metadata.Packages[0]
		fmt.Printf("Package: %s (path: %s)\n", pkg.Name, pkg.Path)
		fmt.Printf("Files: %d, Functions: %d\n", len(pkg.Files), len(pkg.Functions))

		if len(pkg.Functions) > 0 {
			fmt.Printf("First function: %s at %s:%d-%d\n",
				pkg.Functions[0].Name,
				pkg.Functions[0].File,
				pkg.Functions[0].StartLine,
				pkg.Functions[0].EndLine)
		}
	}

	// Show synthetic file generation
	fmt.Println("\n--- SYNTHETIC FILE GENERATION ---")
	fmt.Println("Generating synthetic coverage files from JSON data...")

	// Generate synthetic files
	tempDir := "demo_synthetic_output"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		fmt.Printf("Error creating temp directory: %v\n", err)
		return
	}
	defer os.RemoveAll(tempDir)

	if err := generateSyntheticFromCoverageData(data1, tempDir); err != nil {
		fmt.Printf("Error generating synthetic data: %v\n", err)
		return
	}

	// List generated files
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		fmt.Printf("Error reading temp directory: %v\n", err)
		return
	}

	fmt.Printf("Generated files in %s:\n", tempDir)
	for _, entry := range entries {
		info, _ := entry.Info()
		fmt.Printf("  %s (%d bytes)\n", entry.Name(), info.Size())
	}

	// Show JSON output sample
	fmt.Println("\n--- JSON OUTPUT (first 400 chars) ---")
	jsonStr, err := data1.ToJSON()
	if err != nil {
		fmt.Printf("Error converting to JSON: %v\n", err)
		return
	}

	if len(jsonStr) > 400 {
		fmt.Printf("%s...\n", jsonStr[:400])
	} else {
		fmt.Println(jsonStr)
	}

	fmt.Println("\n✅ JSON functionality demonstration complete!")
	fmt.Println("\nKey features demonstrated:")
	fmt.Println("  • JSON parsing and serialization")
	fmt.Println("  • Coverage data comparison")
	fmt.Println("  • Synthetic file generation from JSON")
	fmt.Println("  • Coverage data analysis and summaries")
	fmt.Println("  • Structured representation of Go coverage data")
}
