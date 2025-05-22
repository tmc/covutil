// Package testutils provides utilities for enhanced testing with coverage data organization
package testutils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/coverage"
	"strings"
	"testing"
)

// RunWithCoverageSubdir runs a subtest with coverage data organized in a subdirectory
// This demonstrates how we can organize coverage data per test without overlaying the testing package
func RunWithCoverageSubdir(t *testing.T, name string, f func(t *testing.T)) bool {
	// Get the original GOCOVERDIR
	originalGoCoverDir := os.Getenv("GOCOVERDIR")

	// If GOCOVERDIR is not set, run the test normally
	if originalGoCoverDir == "" {
		return t.Run(name, f)
	}

	// Create a subdirectory for this test's coverage data
	sanitizedName := sanitizeTestName(name)
	coverageSubdir := filepath.Join(originalGoCoverDir, sanitizedName)

	// Create the subdirectory
	if err := os.MkdirAll(coverageSubdir, 0755); err != nil {
		t.Logf("Failed to create coverage subdirectory %s: %v", coverageSubdir, err)
		return t.Run(name, f)
	}

	fmt.Printf("[TESTUTILS] Coverage data for test '%s' will be collected in: %s\n", name, coverageSubdir)

	// Set the new GOCOVERDIR for this test
	os.Setenv("GOCOVERDIR", coverageSubdir)

	// Run the test with cleanup
	result := t.Run(name, func(t *testing.T) {
		// Add cleanup to restore original GOCOVERDIR and write coverage data
		t.Cleanup(func() {
			// Write coverage data for this test
			if err := coverage.WriteMetaDir(coverageSubdir); err != nil {
				t.Logf("Failed to write meta data for test '%s': %v", name, err)
			}
			if err := coverage.WriteCountersDir(coverageSubdir); err != nil {
				t.Logf("Failed to write counter data for test '%s': %v", name, err)
			}
			fmt.Printf("[TESTUTILS] Coverage data written for test '%s' in: %s\n", name, coverageSubdir)

			// Restore original GOCOVERDIR
			if originalGoCoverDir != "" {
				os.Setenv("GOCOVERDIR", originalGoCoverDir)
			} else {
				os.Unsetenv("GOCOVERDIR")
			}
		})

		// Run the actual test
		f(t)
	})

	return result
}

// sanitizeTestName converts a test name to a valid directory name
func sanitizeTestName(testName string) string {
	// Replace invalid characters with underscores
	sanitized := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		if r == ' ' {
			return '_'
		}
		return r
	}, testName)

	// Remove leading/trailing dots and spaces
	sanitized = strings.Trim(sanitized, ". ")

	// Ensure it's not empty and not too long
	if sanitized == "" || len(sanitized) > 200 {
		return fmt.Sprintf("test_%d", len(testName)) // fallback name
	}

	return sanitized
}
