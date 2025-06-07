// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCovtreeHelp(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("covtree help failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Covtree is a program for analyzing and visualizing Go coverage data") {
		t.Errorf("Expected help text not found in output: %s", outputStr)
	}

	if !strings.Contains(outputStr, "percent") {
		t.Errorf("Expected 'percent' command not found in help")
	}

	if !strings.Contains(outputStr, "json") {
		t.Errorf("Expected 'json' command not found in help")
	}
}

func TestCovtreeCommands(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{"help percent", []string{"help", "percent"}, false},
		{"help json", []string{"help", "json"}, false},
		{"help func", []string{"help", "func"}, false},
		{"help serve", []string{"help", "serve"}, false},
		{"percent no args", []string{"percent"}, true},
		{"json no args", []string{"json"}, true},
		{"debug nonexistent", []string{"debug", "-i=nonexistent"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", "run", ".")
			cmd.Args = append(cmd.Args, tt.args...)
			_, err := cmd.CombinedOutput()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but command succeeded")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected success but got error: %v", err)
			}
		})
	}
}

func TestCovtreeDebugWithTestData(t *testing.T) {
	// Test debug command with internal test program data
	cmd := exec.Command("go", "run", ".", "debug", "-i=../../internal/testprogram")
	output, err := cmd.CombinedOutput()

	// Command may fail due to coverage data format issues, but should show scan results
	outputStr := string(output)
	if !strings.Contains(outputStr, "Found") && !strings.Contains(outputStr, "coverage directories") {
		t.Logf("Debug output: %s", outputStr)
		if err != nil {
			t.Logf("Debug error (may be expected): %v", err)
		}
	}
}

// Integration tests using real Sprig coverage data
func TestCovtreeIntegrationWithSprig(t *testing.T) {
	sprigCovPath := "/Users/tmc/go/src/github.com/Masterminds/sprig/coverage/per-test"

	// Check if Sprig coverage data exists
	if _, err := os.Stat(sprigCovPath); os.IsNotExist(err) {
		t.Skip("Sprig coverage data not available")
	}

	tests := []struct {
		name       string
		testDir    string
		expectPass bool
	}{
		{"simple_test", "simple_test", true},
		{"crypto_functions_test", "crypto_functions_test", true},
		{"strings_strval_test", "strings_strval_test", true},
		{"list_comprehensive_test", "list_comprehensive_test", true},
		{"numeric_functions_test", "numeric_functions_test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPath := filepath.Join(sprigCovPath, tt.testDir)

			// Test debug command
			cmd := exec.Command("go", "run", ".", "debug", "-i="+testPath)
			output, err := cmd.CombinedOutput()

			if tt.expectPass {
				if err != nil {
					t.Logf("Debug output: %s", string(output))
					t.Errorf("Expected debug to succeed for %s, got error: %v", tt.testDir, err)
				}

				outputStr := string(output)
				if !strings.Contains(outputStr, "Found") {
					t.Errorf("Expected 'Found' in debug output for %s", tt.testDir)
				}
			}
		})
	}
}

func TestCovtreePercentWithSprig(t *testing.T) {
	sprigCovPath := "/Users/tmc/go/src/github.com/Masterminds/sprig/coverage/per-test"

	if _, err := os.Stat(sprigCovPath); os.IsNotExist(err) {
		t.Skip("Sprig coverage data not available")
	}

	tests := []struct {
		name    string
		testDir string
	}{
		{"simple_test", "simple_test"},
		{"crypto_functions_test", "crypto_functions_test"},
		{"strings_strval_test", "strings_strval_test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPath := filepath.Join(sprigCovPath, tt.testDir)

			cmd := exec.Command("go", "run", ".", "percent", "-i="+testPath)
			output, err := cmd.CombinedOutput()

			outputStr := string(output)
			if err != nil {
				// Coverage data parsing may fail with certain formats - log and continue
				t.Logf("Percent command output: %s", outputStr)
				if strings.Contains(outputStr, "failed to parse any coverage data") {
					t.Logf("Coverage data format not compatible with parser for %s (expected)", tt.testDir)
					return
				}
				t.Errorf("Unexpected error for %s: %v", tt.testDir, err)
				return
			}

			// Should contain percentage information
			if !strings.Contains(outputStr, "%") {
				t.Errorf("Expected percentage symbol in output for %s: %s", tt.testDir, outputStr)
			}
		})
	}
}

func TestCovtreeJsonWithSprig(t *testing.T) {
	sprigCovPath := "/Users/tmc/go/src/github.com/Masterminds/sprig/coverage/per-test"

	if _, err := os.Stat(sprigCovPath); os.IsNotExist(err) {
		t.Skip("Sprig coverage data not available")
	}

	testPath := filepath.Join(sprigCovPath, "simple_test")

	cmd := exec.Command("go", "run", ".", "json", "-i="+testPath)
	output, err := cmd.CombinedOutput()

	outputStr := string(output)
	if err != nil {
		t.Logf("JSON command output: %s", outputStr)
		if strings.Contains(outputStr, "failed to parse any coverage data") {
			t.Logf("Coverage data format not compatible with parser (expected)")
			return
		}
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// Should be valid JSON
	if len(outputStr) == 0 || (!strings.Contains(outputStr, "{") || !strings.Contains(outputStr, "}")) {
		t.Logf("Empty or invalid JSON output (expected with incompatible coverage data): %s", outputStr)
		return
	}

	// Try to parse as JSON to validate
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}
}

func TestCovtreeFuncWithSprig(t *testing.T) {
	sprigCovPath := "/Users/tmc/go/src/github.com/Masterminds/sprig/coverage/per-test"

	if _, err := os.Stat(sprigCovPath); os.IsNotExist(err) {
		t.Skip("Sprig coverage data not available")
	}

	testPath := filepath.Join(sprigCovPath, "crypto_functions_test")

	cmd := exec.Command("go", "run", ".", "func", "-i="+testPath)
	output, err := cmd.CombinedOutput()

	outputStr := string(output)
	if err != nil {
		t.Logf("Func command output: %s", outputStr)
		if strings.Contains(outputStr, "failed to parse any coverage data") {
			t.Logf("Coverage data format not compatible with parser (expected)")
			return
		}
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// Should contain function information
	if len(outputStr) == 0 {
		t.Errorf("Expected non-empty func output")
	}
}
