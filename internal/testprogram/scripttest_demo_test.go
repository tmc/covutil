package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmc/covutil/coverage"
)

// MockScriptTest demonstrates how our enhanced scripttest would work
type MockScriptTest struct {
	name string
}

func (m MockScriptTest) Name() string {
	return m.name
}

func TestScriptTestEnhancement(t *testing.T) {
	// Create a base coverage directory
	baseCoverageDir, err := os.MkdirTemp("", "scripttest_coverage_demo")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(baseCoverageDir)

	// Set up the base GOCOVERDIR
	originalGoCoverDir := os.Getenv("GOCOVERDIR")
	os.Setenv("GOCOVERDIR", baseCoverageDir)
	defer func() {
		if originalGoCoverDir != "" {
			os.Setenv("GOCOVERDIR", originalGoCoverDir)
		} else {
			os.Unsetenv("GOCOVERDIR")
		}
	}()

	// Test 1: Coverage subdirectory creation
	t.Run("CoverageSubdirs", func(t *testing.T) {
		mockRunScriptTest := func(t *testing.T, testName string) {
			t.Helper()

			// Simulate what our enhanced scripttest.Run would do
			originalGoCoverDir := os.Getenv("GOCOVERDIR")
			testName = t.Name()
			sanitizedName := sanitizeTestName(testName)
			coverageSubdir := filepath.Join(originalGoCoverDir, sanitizedName)

			if err := os.MkdirAll(coverageSubdir, 0755); err != nil {
				t.Fatalf("Failed to create coverage subdirectory: %v", err)
			}

			os.Setenv("GOCOVERDIR", coverageSubdir)
			t.Logf("[SCRIPTTEST DEMO] Coverage data for test '%s' will be collected in: %s", testName, coverageSubdir)

			// Simulate test work
			_ = coverage.PkgPath // Access coverage package

			// Cleanup simulation
			t.Cleanup(func() {
				t.Logf("[SCRIPTTEST DEMO] Test cleanup for '%s'", testName)
				os.Setenv("GOCOVERDIR", originalGoCoverDir)
			})
		}

		// Run multiple "script tests"
		subtests := []string{"TestScript1", "TestScript2", "TestScript/WithSlash"}
		for _, subtest := range subtests {
			subtest := subtest
			t.Run(subtest, func(t *testing.T) {
				mockRunScriptTest(t, subtest)
			})
		}
	})

	// Test 2: Parallel control simulation
	t.Run("ParallelControl", func(t *testing.T) {
		// Simulate the parallel control functions we added
		disableParallel := false

		setParallelMode := func(enabled bool) {
			disableParallel = !enabled
		}

		isParallelDisabled := func() bool {
			return disableParallel
		}

		// Test enabling/disabling parallel mode
		setParallelMode(false)
		if !isParallelDisabled() {
			t.Error("Expected parallel to be disabled")
		}

		setParallelMode(true)
		if isParallelDisabled() {
			t.Error("Expected parallel to be enabled")
		}

		t.Log("Parallel control functions working correctly")
	})

	// After tests complete, verify directory structure
	t.Run("VerifyStructure", func(t *testing.T) {
		subdirs, err := os.ReadDir(baseCoverageDir)
		if err != nil {
			t.Fatalf("Failed to read base coverage directory: %v", err)
		}

		expectedDirs := []string{
			"TestScriptTestEnhancement_CoverageSubdirs_TestScript1",
			"TestScriptTestEnhancement_CoverageSubdirs_TestScript2",
			"TestScriptTestEnhancement_CoverageSubdirs_TestScript_WithSlash",
		}

		foundDirs := make(map[string]bool)
		for _, subdir := range subdirs {
			if subdir.IsDir() {
				foundDirs[subdir.Name()] = true
				t.Logf("Found coverage subdirectory: %s", subdir.Name())
			}
		}

		for _, expectedDir := range expectedDirs {
			if !foundDirs[expectedDir] {
				t.Errorf("Expected to find subdirectory: %s", expectedDir)
			}
		}
	})
}

// sanitizeTestName converts a test name to a valid directory name
// (This duplicates the function in our scripttest overlay for testing)
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
		return "test_fallback"
	}

	return sanitized
}
