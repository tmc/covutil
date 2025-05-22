package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tmc/covutil"
)

func main() {
	fmt.Println("=== Testing Simplified Coverage API ===")

	// Test reading from existing coverage directory
	testDirs := []string{"covdata_simple", "covdata_debug", "covdata"}

	for _, dir := range testDirs {
		if _, err := os.Stat(dir); err == nil {
			fmt.Printf("\n--- Testing directory: %s ---\n", dir)

			fsys := os.DirFS(dir)
			coverageSet, err := covutil.LoadCoverageSetFromFS(fsys, ".")
			if err != nil {
				fmt.Printf("Error reading coverage from %s: %v\n", dir, err)
				continue
			}

			fmt.Printf("✓ Successfully read coverage data with %d pods\n", len(coverageSet.Pods))

			// Print coverage report for each pod
			for i, pod := range coverageSet.Pods {
				fmt.Printf("\n--- Pod %d (%s) ---\n", i, pod.ID)
				fmt.Printf("Packages: %d\n", len(pod.Profile.Meta.Packages))
				fmt.Printf("Counters: %d\n", len(pod.Profile.Counters))
				
				// Show package details
				for _, pkg := range pod.Profile.Meta.Packages {
					fmt.Printf("  Package: %s (%d functions)\n", pkg.Path, len(pkg.Functions))
					
					// Show function coverage stats
					coveredFuncs := 0
					totalFuncs := len(pkg.Functions)
					
					for _, fn := range pkg.Functions {
						key := covutil.PkgFuncKey{PkgPath: pkg.Path, FuncName: fn.FuncName}
						if counters, ok := pod.Profile.Counters[key]; ok {
							// Check if any counter is > 0
							for _, count := range counters {
								if count > 0 {
									coveredFuncs++
									break
								}
							}
						}
					}
					
					if totalFuncs > 0 {
						coverage := float64(coveredFuncs) / float64(totalFuncs) * 100
						fmt.Printf("    Coverage: %d/%d functions (%.1f%%)\n", coveredFuncs, totalFuncs, coverage)
					}
				}
			}

			// Convert first pod to JSON for inspection
			if len(coverageSet.Pods) > 0 {
				jsonData, err := json.MarshalIndent(coverageSet.Pods[0], "", "  ")
				if err != nil {
					fmt.Printf("Error marshaling to JSON: %v\n", err)
				} else {
					fmt.Printf("\nJSON representation (first 500 chars):\n%s...\n",
						string(jsonData[:min(500, len(jsonData))]))
				}
			}

			break // Just test the first available directory
		}
	}

	fmt.Println("\n✓ Simplified API test completed!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}