package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSyntheticCoverage tests the synthetic coverage tracking system
func TestSyntheticCoverage(t *testing.T) {
	if os.Getenv("GO_INTEGRATION_COVERAGE") == "" {
		t.Skip("Skipping synthetic coverage test - set GO_INTEGRATION_COVERAGE=1 to enable")
	}

	// Enable synthetic coverage
	os.Setenv("SYNTHETIC_COVERAGE", "1")
	defer os.Unsetenv("SYNTHETIC_COVERAGE")

	// Initialize synthetic coverage system
	InitSyntheticCoverage()

	// Test script content with various commands
	scriptContent := `
# This is a test script for synthetic coverage
exec main hello World
exec cmd1 greet Universe
exec cmd2 elaborate Testing
exec cmd3 flourish Coverage

# Test with different command patterns
go mod init testproject
go mod tidy
mkdir testdir
echo "Hello" > test.txt
cat test.txt

# Test negated commands
! exec main invalid-command
! exec cmd1 unknown-operation

# Test with Go tools
go version
go env
`

	// Parse and track the script
	t.Logf("Parsing script for synthetic coverage tracking...")
	ParseAndTrackScript(scriptContent, "synthetic_test.txt", t.Name())

	// Simulate execution of commands (in real implementation, this would be done by scripttest)
	lines := strings.Split(scriptContent, "\n")
	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		// Skip comments and empty lines
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Mark executable commands as executed for demonstration
		if isExecutableCommand(trimmed) {
			// Skip some commands (lines 5-10) to demonstrate uncovered code
			if lineNum >= 5 && lineNum <= 10 {
				t.Logf("  Skipped execution: Line %d: %s", lineNum, trimmed)
			} else {
				TrackScriptExecution("synthetic_test.txt", t.Name(), trimmed, lineNum)
				t.Logf("  Tracked execution: Line %d: %s", lineNum, trimmed)
			}
		}
	}

	// Generate coverage report
	t.Logf("Generating synthetic coverage report...")
	report := GetSyntheticCoverageReport()
	t.Logf("Synthetic Coverage Report:\n%s", report)

	// Write synthetic coverage profile
	coverageDir := t.TempDir()
	syntheticProfile := filepath.Join(coverageDir, "synthetic_test.cov")
	if err := WriteSyntheticCoverageProfile(syntheticProfile); err != nil {
		t.Errorf("Failed to write synthetic coverage profile: %v", err)
	} else {
		t.Logf("Synthetic coverage profile written to: %s", syntheticProfile)

		// Read and display the profile
		content, err := os.ReadFile(syntheticProfile)
		if err != nil {
			t.Errorf("Failed to read synthetic coverage profile: %v", err)
		} else {
			t.Logf("Synthetic Coverage Profile Content:\n%s", string(content))
		}
	}

	// Integrate synthetic coverage
	if err := IntegrateSyntheticCoverage(coverageDir); err != nil {
		t.Errorf("Failed to integrate synthetic coverage: %v", err)
	} else {
		t.Logf("Synthetic coverage integrated successfully")

		// Check for generated files
		files, err := filepath.Glob(filepath.Join(coverageDir, "*"))
		if err != nil {
			t.Errorf("Failed to list coverage files: %v", err)
		} else {
			t.Logf("Generated synthetic coverage files:")
			for _, file := range files {
				info, _ := os.Stat(file)
				t.Logf("  %s (%d bytes)", filepath.Base(file), info.Size())
			}
		}
	}
}

// TestCombineCoverageProfiles tests combining multiple coverage profiles
func TestCombineCoverageProfiles(t *testing.T) {
	// Create test coverage profiles
	tempDir := t.TempDir()

	// Profile 1: Go code coverage
	profile1 := filepath.Join(tempDir, "profile1.cov")
	profile1Content := `mode: set
github.com/example/pkg1/file1.go:10.1,15.2 1 1
github.com/example/pkg1/file1.go:17.1,20.2 1 0
`
	if err := os.WriteFile(profile1, []byte(profile1Content), 0644); err != nil {
		t.Fatalf("Failed to create profile1: %v", err)
	}

	// Profile 2: More Go code coverage
	profile2 := filepath.Join(tempDir, "profile2.cov")
	profile2Content := `mode: set
github.com/example/pkg2/file2.go:5.1,8.2 1 1
github.com/example/pkg2/file2.go:10.1,12.2 1 1
`
	if err := os.WriteFile(profile2, []byte(profile2Content), 0644); err != nil {
		t.Fatalf("Failed to create profile2: %v", err)
	}

	// Profile 3: Synthetic scripttest coverage
	profile3 := filepath.Join(tempDir, "synthetic.cov")
	profile3Content := `mode: set
scripttest://TestExample/test.txt:1.1,1.20 1 1
scripttest://TestExample/test.txt:3.1,3.15 1 1
scripttest://TestExample/test.txt:5.1,5.25 1 0
`
	if err := os.WriteFile(profile3, []byte(profile3Content), 0644); err != nil {
		t.Fatalf("Failed to create synthetic profile: %v", err)
	}

	// Combine all profiles
	combinedProfile := filepath.Join(tempDir, "combined.cov")
	profiles := []string{profile1, profile2, profile3}

	if err := CombineCoverageProfiles(profiles, combinedProfile); err != nil {
		t.Fatalf("Failed to combine coverage profiles: %v", err)
	}

	// Read and verify combined profile
	combinedContent, err := os.ReadFile(combinedProfile)
	if err != nil {
		t.Fatalf("Failed to read combined profile: %v", err)
	}

	combinedStr := string(combinedContent)
	t.Logf("Combined Coverage Profile:\n%s", combinedStr)

	// Verify it contains content from all profiles
	expectedContent := []string{
		"mode: set",
		"github.com/example/pkg1/file1.go",
		"github.com/example/pkg2/file2.go",
		"scripttest://TestExample/test.txt",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(combinedStr, expected) {
			t.Errorf("Combined profile missing expected content: %s", expected)
		}
	}

	t.Logf("Successfully combined %d coverage profiles into single profile", len(profiles))
}
