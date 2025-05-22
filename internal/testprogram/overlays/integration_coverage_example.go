//go:build ignore

// Example of using the integration coverage overlay to address Go issue #60182
// This demonstrates collecting coverage from binaries executed during integration tests

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// This example shows how the overlay solves the coverage collection problem
func ExampleIntegrationCoverage(t *testing.T) {
	// Setup: Build a test binary that we'll execute during our integration test
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "testbinary")

	// Build a simple test binary (this would be your actual binary)
	buildCmd := exec.Command("go", "build", "-cover", "-o", binaryPath, "./testprogram.go")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build test binary: %v", err)
	}

	// The overlay automatically handles coverage collection when we execute the binary
	t.RunWithIntegrationCoverage("execute_binary_test", func(subT *testing.T) {
		// This method is provided by the testing overlay
		// It ensures coverage data is collected from executed binaries
		if err := subT.ExecuteBinaryWithCoverage(binaryPath, "arg1", "arg2"); err != nil {
			subT.Fatalf("Binary execution failed: %v", err)
		}

		// The overlay automatically:
		// 1. Creates a coverage subdirectory for this test
		// 2. Sets GOCOVERDIR for the executed binary
		// 3. Collects coverage files after execution
		// 4. Merges them back to the main coverage directory
	})
}

// Example of manual coverage data collection using the overlay functions
func ExampleManualCoverageCollection() {
	// Set integration coverage mode
	os.Setenv("GO_INTEGRATION_COVERAGE", "1")
	os.Setenv("GOCOVERDIR", "./coverage_data")

	// The coverage overlay automatically snapshots existing files on init

	// Execute your integration tests here...
	fmt.Println("Running integration tests...")

	// After tests complete, get the new coverage files
	if newFiles, err := coverage.GetNewCoverageFiles(); err == nil {
		fmt.Printf("Integration tests generated %d new coverage files:\n", len(newFiles))
		for _, file := range newFiles {
			fmt.Printf("  - %s\n", file)
		}
	}

	// Force any remaining coverage data to be written
	if err := coverage.ForceWriteCoverageData(); err != nil {
		fmt.Printf("Warning: failed to write coverage data: %v\n", err)
	}
}

func main() {
	fmt.Println("Integration Coverage Overlay Example")
	fmt.Println("====================================")
	fmt.Println()
	fmt.Println("This overlay addresses Go issue #60182 by:")
	fmt.Println("1. Snapshotting existing coverage files before test execution")
	fmt.Println("2. Forcing coverage data writing from executed binaries")
	fmt.Println("3. Collecting only new coverage files after execution")
	fmt.Println("4. Providing enhanced testing methods for integration tests")
	fmt.Println()
	fmt.Println("Key features:")
	fmt.Println("- RunWithIntegrationCoverage() for enhanced test execution")
	fmt.Println("- ExecuteBinaryWithCoverage() for binary execution with coverage")
	fmt.Println("- Automatic coverage file snapshots and aggregation")
	fmt.Println("- Integration with existing GOCOVERDIR workflows")
}
