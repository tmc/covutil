package main

import (
	"fmt"
	"hash/fnv"
	"os"
	"time"

	"github.com/tmc/covutil"
)

func main() {
	fmt.Println("=== Testing High-Level Coverage API ===")

	// Test reading existing coverage data
	fmt.Println("1. Reading existing coverage data...")
	var coverageSet *covutil.CoverageSet
	if _, err := os.Stat("covdata_simple"); err == nil {
		fsys := os.DirFS("covdata_simple")
		var err error
		coverageSet, err = covutil.LoadCoverageSetFromFS(fsys, ".")
		if err != nil {
			fmt.Printf("Error reading coverage data: %v\n", err)
		} else {
			fmt.Printf("✓ Successfully read coverage data with %d pods\n", len(coverageSet.Pods))

			// Display pod info
			for i, pod := range coverageSet.Pods {
				fmt.Printf("  Pod %d: %s (%d packages)\n", i, pod.ID, len(pod.Profile.Meta.Packages))
				for _, pkg := range pod.Profile.Meta.Packages {
					fmt.Printf("    Package: %s (%d functions)\n", pkg.Path, len(pkg.Functions))
				}
			}
		}
	} else {
		fmt.Println("No existing coverage data found (covdata_simple directory)")
	}

	// Test creating synthetic coverage data using existing data as template
	fmt.Println("\n2. Creating synthetic coverage data based on existing data...")

	var templatePod *covutil.Pod
	if coverageSet != nil && len(coverageSet.Pods) > 0 {
		templatePod = coverageSet.Pods[0]
		fmt.Printf("Using template from pod: %s\n", templatePod.ID)
	}

	// Create a synthetic pod based on existing real data
	var pod *covutil.Pod
	if templatePod != nil {
		// Use the real meta file structure and hash
		pod = &covutil.Pod{
			ID: "synthetic-test-pod",
			Profile: &covutil.Profile{
				Meta: templatePod.Profile.Meta, // Use real meta structure
				Counters: make(map[covutil.PkgFuncKey][]uint32),
				Args:     make(map[string]string),
			},
			Labels: map[string]string{
				"type": "synthetic",
				"source": "high-level-api-test",
			},
			Timestamp: time.Now(),
		}

		// Add synthetic counter data based on real functions
		for _, pkg := range templatePod.Profile.Meta.Packages {
			for _, fn := range pkg.Functions {
				key := covutil.PkgFuncKey{PkgPath: pkg.Path, FuncName: fn.FuncName}
				// Create synthetic counters with same length as units
				counters := make([]uint32, len(fn.Units))
				for i := range counters {
					counters[i] = uint32(i + 1) // Some synthetic data
				}
				pod.Profile.Counters[key] = counters
			}
		}

		// Add synthetic args
		pod.Profile.Args["SYNTHETIC"] = "true"
		pod.Profile.Args["GENERATOR"] = "high-level-api-test"

	} else {
		// Fallback to original synthetic approach if no template available
		h := fnv.New128()
		h.Write([]byte("TEST_SYNTHETIC_COVERAGE"))
		h.Write([]byte(time.Now().Format(time.RFC3339Nano)))
		sum := h.Sum(nil)
		var fileHash [16]byte
		copy(fileHash[:], sum)

		pod = &covutil.Pod{
			ID: "synthetic-test-pod",
			Profile: &covutil.Profile{
				Meta: covutil.MetaFile{
					FilePath: "test.meta",
					FileHash: fileHash,
					Mode:     covutil.ModeSet,
					Granularity: covutil.GranularityBlock,
					Packages: []covutil.PackageMeta{
						{
							Path: "github.com/example/testpackage",
							Functions: []covutil.FuncDesc{
								{
									FuncName: "TestFunction1",
									SrcFile:  "test.go",
									Units: []covutil.CoverableUnit{
										{StartLine: 10, StartCol: 1, EndLine: 20, EndCol: 2, NumStmt: 3},
									},
								},
							},
						},
					},
				},
				Counters: make(map[covutil.PkgFuncKey][]uint32),
				Args:     make(map[string]string),
			},
		}

		// Add fallback counter data
		pod.Profile.Counters[covutil.PkgFuncKey{PkgPath: "github.com/example/testpackage", FuncName: "TestFunction1"}] = []uint32{1, 2, 0}
	}

	fmt.Printf("✓ Created synthetic coverage pod with %d packages\n", len(pod.Profile.Meta.Packages))

	// Test writing synthetic coverage data
	fmt.Println("\n3. Writing synthetic coverage data...")
	if err := covutil.WritePodToDirectory("test_synthetic_output", pod); err != nil {
		fmt.Printf("Error writing synthetic coverage: %v\n", err)
	} else {
		fmt.Println("✓ Successfully wrote synthetic coverage data")

		// Test reading it back
		fmt.Println("\n4. Reading synthetic coverage data back...")
		fsys := os.DirFS("test_synthetic_output")
		readBack, err := covutil.LoadCoverageSetFromFS(fsys, ".")
		if err != nil {
			fmt.Printf("Error reading back synthetic coverage: %v\n", err)
		} else {
			fmt.Printf("✓ Successfully read back coverage data with %d pods\n", len(readBack.Pods))
			if len(readBack.Pods) > 0 {
				syntheticPod := readBack.Pods[0]
				fmt.Printf("  Synthetic pod ID: %s\n", syntheticPod.ID)
				fmt.Printf("  Meta file hash: %x\n", syntheticPod.Profile.Meta.FileHash)
				fmt.Printf("  Packages: %d\n", len(syntheticPod.Profile.Meta.Packages))
				fmt.Printf("  Counters: %d\n", len(syntheticPod.Profile.Counters))
			}
		}

		// Clean up
		// Don't clean up for debugging
		// os.RemoveAll("test_synthetic_output")
	}

	fmt.Println("\n✓ High-level API test completed!")
}