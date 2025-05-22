package main

import (
	"fmt"
	"os"

	"github.com/tmc/covutil"
)

// parseCoverageDirectory parses coverage from a directory
func parseCoverageDirectory(dir string) {
	fmt.Printf("Parsing coverage data from directory: %s\n", dir)
	
	// Use the updated coverage parser functionality
	data, err := ParseCoverageFromDirectory(dir)
	if err != nil {
		fmt.Printf("Error parsing coverage data: %v\n", err)
		return
	}
	
	fmt.Printf("✓ Successfully parsed coverage data\n")
	if data.Pod != nil {
		fmt.Printf("Pod ID: %s\n", data.Pod.ID)
		if data.Pod.Profile != nil {
			fmt.Printf("Packages: %d\n", len(data.Pod.Profile.Meta.Packages))
			fmt.Printf("Counters: %d\n", len(data.Pod.Profile.Counters))
		}
	}
}

// analyzeFormats analyzes different coverage file formats
func analyzeFormats() {
	fmt.Println("Analyzing coverage file formats...")
	
	// This would analyze different format types
	testDirs := []string{"covdata_simple", "covdata_debug", "covdata"}
	
	for _, dir := range testDirs {
		fmt.Printf("\n--- Analyzing directory: %s ---\n", dir)
		
		fsys := os.DirFS(dir)
		coverageSet, err := covutil.LoadCoverageSetFromFS(fsys, ".")
		if err != nil {
			fmt.Printf("Error loading coverage set: %v\n", err)
			continue
		}
		
		fmt.Printf("Found %d pods\n", len(coverageSet.Pods))
		for i, pod := range coverageSet.Pods {
			fmt.Printf("  Pod %d: %s\n", i, pod.ID)
			fmt.Printf("    Mode: %s\n", pod.Profile.Meta.Mode)
			fmt.Printf("    Granularity: %s\n", pod.Profile.Meta.Granularity)
			fmt.Printf("    Packages: %d\n", len(pod.Profile.Meta.Packages))
		}
	}
}

// simpleTestMain runs simple test functionality
func simpleTestMain() {
	fmt.Println("Running simple test main functionality...")
	
	// Create a simple synthetic pod for testing
	pod := &covutil.Pod{
		ID: "simple-test-pod",
		Profile: &covutil.Profile{
			Meta: covutil.MetaFile{
				FilePath: "simple_test.meta",
				Mode:     covutil.ModeSet,
				Granularity: covutil.GranularityBlock,
				Packages: []covutil.PackageMeta{
					{
						Path: "example.com/simple",
						Functions: []covutil.FuncDesc{
							{
								FuncName: "SimpleFunction",
								SrcFile:  "simple.go",
								Units: []covutil.CoverableUnit{
									{StartLine: 1, StartCol: 1, EndLine: 5, EndCol: 1, NumStmt: 1},
								},
							},
						},
					},
				},
			},
			Counters: make(map[covutil.PkgFuncKey][]uint32),
		},
	}
	
	// Add counter data
	key := covutil.PkgFuncKey{PkgPath: "example.com/simple", FuncName: "SimpleFunction"}
	pod.Profile.Counters[key] = []uint32{1}
	
	fmt.Printf("✓ Created simple test pod with ID: %s\n", pod.ID)
	fmt.Printf("  Package: %s\n", pod.Profile.Meta.Packages[0].Path)
	fmt.Printf("  Functions: %d\n", len(pod.Profile.Meta.Packages[0].Functions))
	fmt.Printf("  Counters: %d\n", len(pod.Profile.Counters))
}

// displayBasicInfo displays basic information about coverage data
// It tries common coverage directories
func displayBasicInfo() {
	fmt.Println("=== Basic Coverage Information ===")

	// Try common coverage directories
	dirs := []string{"covdata_simple", "covdata_debug", "covdata"}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); err == nil {
			fmt.Printf("\n--- Directory: %s ---\n", dir)
			fsys := os.DirFS(dir)
			coverageSet, err := covutil.LoadCoverageSetFromFS(fsys, ".")
			if err != nil {
				fmt.Printf("Error reading coverage data: %v\n", err)
				continue
			}

			fmt.Printf("Found %d pods\n", len(coverageSet.Pods))
			for i, pod := range coverageSet.Pods {
				fmt.Printf("Pod %d:\n", i)
				fmt.Printf("  ID: %s\n", pod.ID)
				fmt.Printf("  Packages: %d\n", len(pod.Profile.Meta.Packages))
				fmt.Printf("  Counters: %d\n", len(pod.Profile.Counters))
				fmt.Printf("  Mode: %s\n", pod.Profile.Meta.Mode)
				fmt.Printf("  Granularity: %s\n", pod.Profile.Meta.Granularity)
			}
			return // Only display info for first available directory
		}
	}

	fmt.Println("No coverage directories found")
}