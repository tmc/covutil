package main

import (
	"fmt"
	"os"
)

// demonstrateJsonFunctionality shows off the JSON conversion and comparison features
func demonstrateJsonFunctionality() {
	fmt.Println("\n=== JSON FUNCTIONALITY DEMONSTRATION ===")

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
	if data1.Pod != nil && data2.Pod != nil {
		meta1Hash := data1.Pod.Profile.Meta.FileHash
		meta2Hash := data2.Pod.Profile.Meta.FileHash
		fmt.Printf("Metadata hashes match: %t\n", meta1Hash == meta2Hash)
	}

	// Compare counter values
	if data1.Pod != nil && data2.Pod != nil {
		if len(data1.Pod.Profile.Counters) > 0 && len(data2.Pod.Profile.Counters) > 0 {
			// Get first counter from each
			var counters1, counters2 []uint32
			for _, v := range data1.Pod.Profile.Counters {
				counters1 = v
				break
			}
			for _, v := range data2.Pod.Profile.Counters {
				counters2 = v
				break
			}
			if len(counters1) > 0 && len(counters2) > 0 {
				fmt.Printf("First function counters: %v vs %v\n", counters1, counters2)
				fmt.Printf("Counter values match: %t\n", fmt.Sprintf("%v", counters1) == fmt.Sprintf("%v", counters2))
			}
		}
	}

	// Compare using go tool covdata
	fmt.Println("\n--- GO TOOL COVDATA COMPARISON ---")
	fmt.Println("Comparing coverage data using go tool covdata...")

	out1, out2, summary := CovdataCompare(data1, data2)
	fmt.Printf("\nDataset 1 covdata output:\n")
	fmt.Printf("  Percent: %s\n", out1.Percent)
	if out1.Error != "" {
		fmt.Printf("  Error: %s\n", out1.Error)
	}

	fmt.Printf("\nDataset 2 covdata output:\n")
	fmt.Printf("  Percent: %s\n", out2.Percent)
	if out2.Error != "" {
		fmt.Printf("  Error: %s\n", out2.Error)
	}

	fmt.Printf("\n%s\n", summary)

	// Show JSON output for first dataset
	fmt.Println("\n--- JSON OUTPUT (first 300 chars) ---")
	jsonStr, err := data1.ToJSON()
	if err != nil {
		fmt.Printf("Error converting to JSON: %v\n", err)
		return
	}

	if len(jsonStr) > 300 {
		fmt.Printf("%s...\n", jsonStr[:300])
	} else {
		fmt.Println(jsonStr)
	}

	fmt.Println("\n✅ JSON functionality working correctly!")
}

// loadJsonFile is a helper to load and parse JSON files
func loadJsonFile(filename string) (*CoverageData, error) {
	jsonData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	return FromJSON(string(jsonData))
}
