package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestBinaryCovData tests the binary covdata format generation
func TestBinaryCovData(t *testing.T) {
	if os.Getenv("GO_INTEGRATION_COVERAGE") == "" {
		t.Skip("Skipping binary covdata test - set GO_INTEGRATION_COVERAGE=1 to enable")
	}

	// Enable synthetic coverage
	os.Setenv("SYNTHETIC_COVERAGE", "1")
	defer os.Unsetenv("SYNTHETIC_COVERAGE")

	// Initialize synthetic coverage system
	InitSyntheticCoverage()

	// Test script content
	scriptContent := `
# Binary covdata test script
exec main hello BinaryCovData
exec cmd1 greet BinaryCovData
exec cmd2 elaborate BinaryCovData
exec cmd3 flourish BinaryCovData

# Go commands
go mod init binarytest
go mod tidy
go version

# File operations
mkdir testdir
echo "test" > testfile
cat testfile
rm testfile
`

	testName := "TestBinaryCovData"
	scriptName := "binary_test.txt"

	// Parse and track the script
	t.Logf("Parsing script for binary covdata generation...")
	ParseAndTrackScript(scriptContent, scriptName, testName)

	// Simulate execution of commands
	lines := strings.Split(scriptContent, "\n")
	executedCount := 0
	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		// Skip comments and empty lines
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Mark executable commands as executed
		if isExecutableCommand(trimmed) {
			TrackScriptExecution(scriptName, testName, trimmed, lineNum)
			executedCount++
			t.Logf("  Executed: Line %d: %s", lineNum, trimmed)
		}
	}

	t.Logf("Tracked %d executed commands", executedCount)

	// Create coverage directory
	coverageDir := t.TempDir()
	t.Logf("Coverage directory: %s", coverageDir)

	// Write binary covdata format
	t.Logf("Writing binary covdata format...")
	if err := WriteSyntheticCovData(coverageDir); err != nil {
		t.Fatalf("Failed to write binary covdata: %v", err)
	}

	// Check generated files - with new covutil API, files are in pod directories
	files, err := filepath.Glob(filepath.Join(coverageDir, "*"))
	if err != nil {
		t.Fatalf("Failed to list coverage files: %v", err)
	}

	t.Logf("Generated coverage files using new covutil API:")
	podDirs := 0
	for _, file := range files {
		info, _ := os.Stat(file)
		fileName := filepath.Base(file)
		t.Logf("  %s (%d bytes)", fileName, info.Size())

		if info.IsDir() {
			podDirs++
			// Check contents of pod directory
			podFiles, err := filepath.Glob(filepath.Join(file, "*"))
			if err == nil {
				for _, podFile := range podFiles {
					podInfo, _ := os.Stat(podFile)
					t.Logf("    %s (%d bytes)", filepath.Base(podFile), podInfo.Size())
				}
			}
		}
	}

	// Verify we have at least one pod directory
	if podDirs == 0 {
		t.Errorf("Expected at least 1 pod directory, got %d", podDirs)
	}

	// Verify directories contain metadata files
	foundMetadata := false
	for _, file := range files {
		if info, _ := os.Stat(file); info.IsDir() {
			metadataPath := filepath.Join(file, "pod_metadata.json")
			if _, err := os.Stat(metadataPath); err == nil {
				foundMetadata = true
				break
			}
		}
	}

	if !foundMetadata {
		t.Errorf("Expected to find pod_metadata.json in at least one pod directory")
	}

	// Test integration with main coverage system
	t.Logf("Testing integration with main coverage system...")
	if err := IntegrateSyntheticCoverage(coverageDir); err != nil {
		t.Fatalf("Failed to integrate synthetic coverage: %v", err)
	}

	// Check for all expected output files
	expectedFiles := []string{
		"synthetic_coverage_report.txt",
		"synthetic_scripttest.cov",
	}

	for _, expectedFile := range expectedFiles {
		fullPath := filepath.Join(coverageDir, expectedFile)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s not found", expectedFile)
		} else {
			info, _ := os.Stat(fullPath)
			t.Logf("Generated %s (%d bytes)", expectedFile, info.Size())
		}
	}

	// Generate human-readable report
	report := GetSyntheticCoverageReport()
	t.Logf("Binary CovData Coverage Report:\n%s", report)

	t.Logf("✅ Binary covdata format test completed successfully")
}

// TestCovDataCompatibility tests compatibility with Go's covdata tools
func TestCovDataCompatibility(t *testing.T) {
	if os.Getenv("GO_INTEGRATION_COVERAGE") == "" {
		t.Skip("Skipping covdata compatibility test - set GO_INTEGRATION_COVERAGE=1 to enable")
	}

	// Enable synthetic coverage
	os.Setenv("SYNTHETIC_COVERAGE", "1")
	defer os.Unsetenv("SYNTHETIC_COVERAGE")

	// Initialize and generate some coverage data
	InitSyntheticCoverage()

	// Simple script for testing
	scriptContent := `exec main hello CompatibilityTest
exec cmd1 greet CompatibilityTest
go version`

	ParseAndTrackScript(scriptContent, "compat_test.txt", "TestCovDataCompatibility")

	// Track execution
	lines := strings.Split(scriptContent, "\n")
	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && isExecutableCommand(trimmed) {
			TrackScriptExecution("compat_test.txt", "TestCovDataCompatibility", trimmed, lineNum)
		}
	}

	// Generate binary covdata
	coverageDir := t.TempDir()
	if err := WriteSyntheticCovData(coverageDir); err != nil {
		t.Fatalf("Failed to write binary covdata: %v", err)
	}

	// Check if we can find the pod directories created by new covutil API
	files, err := filepath.Glob(filepath.Join(coverageDir, "*"))
	if err != nil {
		t.Fatalf("Failed to find coverage files: %v", err)
	}

	podDirs := 0
	metadataFiles := 0
	for _, file := range files {
		if info, _ := os.Stat(file); info.IsDir() {
			podDirs++

			// Check for metadata file in pod directory
			metadataPath := filepath.Join(file, "pod_metadata.json")
			if _, err := os.Stat(metadataPath); err == nil {
				metadataFiles++
			}
		}
	}

	t.Logf("Compatibility test results:")
	t.Logf("  Pod directories: %d", podDirs)
	t.Logf("  Metadata files: %d", metadataFiles)

	// Verify we have at least one pod directory with metadata
	if podDirs == 0 {
		t.Errorf("Expected at least 1 pod directory, got %d", podDirs)
	}
	if metadataFiles == 0 {
		t.Errorf("Expected at least 1 metadata file, got %d", metadataFiles)
	}

	// Test if metadata files contain expected structure
	for _, file := range files {
		if info, _ := os.Stat(file); info.IsDir() {
			metadataPath := filepath.Join(file, "pod_metadata.json")
			if content, err := os.ReadFile(metadataPath); err == nil {
				if !strings.Contains(string(content), "synthetic") {
					t.Errorf("Metadata file doesn't contain expected 'synthetic' marker")
				}
				if !strings.Contains(string(content), "scripttest") {
					t.Errorf("Metadata file doesn't contain expected 'scripttest' marker")
				}
				t.Logf("  Metadata content preview: %s", string(content)[:min(100, len(content))])
			}
		}
	}

	t.Logf("✅ CovData compatibility test completed successfully")
}
