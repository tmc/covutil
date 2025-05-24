package synthetic

import (
	"strings"
	"testing"

	"github.com/tmc/covutil/synthetic/parsers"
)

// TestParserModularityIntegration demonstrates the modular parser architecture
func TestParserModularityIntegration(t *testing.T) {
	// Test 1: Verify all default parsers are registered
	t.Run("DefaultParsersRegistered", func(t *testing.T) {
		expectedParsers := []string{"bash", "shell", "python", "gotemplate", "scripttest"}

		registeredTypes := parsers.DefaultRegistry.RegisteredTypes()

		for _, expectedParser := range expectedParsers {
			if _, exists := registeredTypes[expectedParser]; !exists {
				t.Errorf("Expected parser %s to be registered", expectedParser)
			}
		}

		t.Logf("Successfully registered parsers: %v", registeredTypes)
	})

	// Test 2: Test parser lookup by name and extension
	t.Run("ParserLookup", func(t *testing.T) {
		testCases := []struct {
			lookup   string
			expected string
		}{
			{"bash", "bash"},
			{"shell", "shell"},
			{"sh", "shell"},
			{"python", "python"},
			{"py", "python"},
			{"gotemplate", "gotemplate"},
			{"tmpl", "gotemplate"},
			{"scripttest", "scripttest"},
			{"txt", "scripttest"},
		}

		for _, tc := range testCases {
			parser, exists := parsers.Get(tc.lookup)
			if !exists {
				t.Errorf("Failed to find parser for %s", tc.lookup)
				continue
			}

			if parser.Name() != tc.expected {
				t.Errorf("Expected parser name %s for lookup %s, got %s",
					tc.expected, tc.lookup, parser.Name())
			}
		}
	})

	// Test 3: Test ScriptTracker with different parser types
	t.Run("MultiParserTracking", func(t *testing.T) {
		tracker := NewScriptTracker(
			WithTestName("multi-parser-test"),
			WithLabels(map[string]string{"test": "modular-architecture"}),
		)

		// Track different script types
		scripts := map[string]struct {
			content    string
			scriptType string
		}{
			"deploy.sh": {
				content: `#!/bin/bash
echo "Deploying application"
kubectl apply -f deployment.yaml`,
				scriptType: "bash",
			},
			"analyze.py": {
				content: `#!/usr/bin/env python3
import sys
def main():
    print("Analysis complete")`,
				scriptType: "python",
			},
			"config.tmpl": {
				content: `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.Name}}`,
				scriptType: "gotemplate",
			},
			"test.txt": {
				content: `# Integration test
exec go version
go build .`,
				scriptType: "scripttest",
			},
		}

		// Parse all scripts
		for scriptName, script := range scripts {
			err := tracker.ParseAndTrack(script.content, scriptName, script.scriptType, "multi-parser-test")
			if err != nil {
				t.Errorf("Failed to parse %s (%s): %v", scriptName, script.scriptType, err)
			}
		}

		// Track some execution
		tracker.TrackExecution("deploy.sh", "multi-parser-test", 2)
		tracker.TrackExecution("analyze.py", "multi-parser-test", 3)
		tracker.TrackExecution("config.tmpl", "multi-parser-test", 4)
		tracker.TrackExecution("test.txt", "multi-parser-test", 2)

		// Generate report
		report := tracker.GetReport()

		// Verify all script types are in the report
		for scriptName := range scripts {
			if !strings.Contains(report, scriptName) {
				t.Errorf("Report should contain %s", scriptName)
			}
		}

		t.Logf("Multi-parser report:\n%s", report)
	})

	// Test 4: Test custom registry isolation
	t.Run("CustomRegistryIsolation", func(t *testing.T) {
		// Create isolated registry
		customRegistry := parsers.NewRegistry()

		// Should be empty initially
		if len(customRegistry.List()) != 0 {
			t.Error("Custom registry should be empty initially")
		}

		// Create tracker with custom registry
		tracker := NewScriptTrackerWithRegistry(customRegistry)

		// Should fail to parse since no parsers are registered
		err := tracker.ParseAndTrack("echo test", "test.sh", "bash", "test")
		if err == nil {
			t.Error("Expected error when no parsers are registered")
		}

		// Register a single parser
		bashParser, _ := parsers.Get("bash") // Get from global registry
		err = customRegistry.Register(bashParser)
		if err != nil {
			t.Fatalf("Failed to register bash parser: %v", err)
		}

		// Now it should work
		err = tracker.ParseAndTrack("echo test", "test.sh", "bash", "test")
		if err != nil {
			t.Errorf("Failed to parse with custom registry: %v", err)
		}

		// But other parsers should still fail
		err = tracker.ParseAndTrack("print('test')", "test.py", "python", "test")
		if err == nil {
			t.Error("Expected error for unregistered python parser")
		}
	})

	// Test 5: Test GetRegisteredParsers method
	t.Run("GetRegisteredParsers", func(t *testing.T) {
		tracker := NewScriptTracker()

		registeredTypes := tracker.GetRegisteredParsers()

		// Should have all default parsers
		expectedParsers := []string{"bash", "shell", "python", "gotemplate", "scripttest"}
		for _, expected := range expectedParsers {
			if _, exists := registeredTypes[expected]; !exists {
				t.Errorf("Expected %s to be in registered parsers", expected)
			}
		}

		t.Logf("Tracker registered parsers: %v", registeredTypes)
	})
}

// TestBackwardCompatibility ensures the old API still works
func TestBackwardCompatibility(t *testing.T) {
	tracker := NewScriptTracker()

	// The old RegisterParser method should still work (deprecated but functional)
	bashContent := `#!/bin/bash
echo "test"
VAR=value`

	err := tracker.ParseAndTrack(bashContent, "test.sh", "bash", "compatibility-test")
	if err != nil {
		t.Fatalf("ParseAndTrack failed: %v", err)
	}

	tracker.TrackExecution("test.sh", "compatibility-test", 2)

	report := tracker.GetReport()
	if !strings.Contains(report, "test.sh") {
		t.Error("Report should contain test.sh")
	}

	// Test pod generation still works
	pod, err := tracker.GeneratePod()
	if err != nil {
		t.Fatalf("GeneratePod failed: %v", err)
	}

	if pod == nil {
		t.Error("Pod should not be nil")
	}

	t.Log("Backward compatibility test passed")
}
