package synthetic

import (
	"strings"
	"testing"
)

func TestBasicTracker(t *testing.T) {
	tracker := NewBasicTracker(
		WithLabels(map[string]string{"test": "basic"}),
	)

	// Test tracking
	tracker.Track("test-script.sh", "1", true)
	tracker.Track("test-script.sh", "2", false)
	tracker.Track("test-script.sh", "3", true)

	// Test report generation
	report := tracker.GetReport()
	if !strings.Contains(report, "test-script.sh") {
		t.Error("Report should contain artifact name")
	}
	if !strings.Contains(report, "2/3") {
		t.Error("Report should show 2/3 executed")
	}

	// Test pod generation
	pod, err := tracker.GeneratePod()
	if err != nil {
		t.Fatalf("Failed to generate pod: %v", err)
	}

	if pod.Profile == nil {
		t.Error("Pod should have a profile")
	}

	if len(pod.Profile.Counters) == 0 {
		t.Error("Profile should have counters")
	}

	if pod.Labels["test"] != "basic" {
		t.Error("Pod should have custom labels")
	}

	// Test text profile generation
	profile, err := tracker.GenerateProfile("text")
	if err != nil {
		t.Fatalf("Failed to generate text profile: %v", err)
	}

	if !strings.Contains(string(profile), "mode: set") {
		t.Error("Text profile should start with mode declaration")
	}
}

func TestScriptTracker(t *testing.T) {
	tracker := NewScriptTracker(
		WithTestName("script-test"),
	)

	// Test shell script parsing
	shellScript := `#!/bin/bash
# This is a comment
echo "Hello World"
ls -la
# Another comment
mkdir test-dir
cd test-dir`

	err := tracker.ParseAndTrack(shellScript, "test.sh", "shell", "my-test")
	if err != nil {
		t.Fatalf("Failed to parse shell script: %v", err)
	}

	// Test execution tracking
	tracker.TrackExecution("test.sh", "my-test", 3) // echo command
	tracker.TrackExecution("test.sh", "my-test", 4) // ls command
	// Don't track mkdir and cd

	report := tracker.GetReport()
	if !strings.Contains(report, "test.sh") {
		t.Error("Report should contain script name")
	}

	// Test scripttest parsing
	scripttestContent := `# Test scripttest commands
exec go version
exec echo "test"
# Comment
go build .
mkdir testdir
cat > test.txt << EOF
test content
EOF`

	err = tracker.ParseAndTrack(scripttestContent, "test.txt", "scripttest", "scripttest-test")
	if err != nil {
		t.Fatalf("Failed to parse scripttest: %v", err)
	}

	// Track some executions
	tracker.TrackExecution("test.txt", "scripttest-test", 2) // exec go version
	tracker.TrackExecution("test.txt", "scripttest-test", 5) // go build

	finalReport := tracker.GetReport()
	if !strings.Contains(finalReport, "test.txt") {
		t.Error("Final report should contain scripttest name")
	}
}

func TestShellScriptParser(t *testing.T) {
	parser := &ShellScriptParser{}

	testScript := `#!/bin/bash
# Comment
echo "test"
VAR=value
ls -la
# Another comment

if [ -f file ]; then
    echo "exists"
fi`

	commands := parser.ParseScript(testScript)

	// Should parse: echo, VAR=value, ls, if statement
	if len(commands) < 4 {
		t.Errorf("Expected at least 4 commands, got %d", len(commands))
	}

	// Test individual line parsing
	testCases := []struct {
		line       string
		executable bool
	}{
		{"echo 'hello'", true},
		{"# comment", false},
		{"", false},
		{"VAR=value", true},
		{"ls -la", true},
		{"if [ -f file ]; then", true},
		{"function test() {", true},
	}

	for _, tc := range testCases {
		result := parser.IsExecutable(tc.line)
		if result != tc.executable {
			t.Errorf("Line '%s': expected %v, got %v", tc.line, tc.executable, result)
		}
	}
}

func TestScriptTestParser(t *testing.T) {
	parser := &ScriptTestParser{}

	testScript := `# Test script
exec go version
! exec go build nonexistent
go test ./...
mkdir testdir
# Comment
exists go.mod
! exists nonexistent.txt
cd testdir
echo "test" > file.txt
cat file.txt`

	commands := parser.ParseScript(testScript)

	// Should parse scripttest commands but not regular shell commands
	expectedCommands := []string{
		"exec go version",
		"! exec go build nonexistent",
		"go test ./...",
		"mkdir testdir",
		"exists go.mod",
		"! exists nonexistent.txt",
		"cd testdir",
		"echo \"test\" > file.txt",
		"cat file.txt",
	}

	if len(commands) != len(expectedCommands) {
		t.Errorf("Expected %d commands, got %d", len(expectedCommands), len(commands))
	}

	// Test individual line parsing
	testCases := []struct {
		line       string
		executable bool
	}{
		{"exec go version", true},
		{"! exec go build", true},
		{"go test", true},
		{"# comment", false},
		{"", false},
		{"mkdir dir", true},
		{"exists file", true},
		{"! exists file", true},
		{"cd dir", true},
		{"echo test", true},
	}

	for _, tc := range testCases {
		result := parser.IsExecutable(tc.line)
		if result != tc.executable {
			t.Errorf("Line '%s': expected %v, got %v", tc.line, tc.executable, result)
		}
	}
}

func TestTrackerReset(t *testing.T) {
	tracker := NewBasicTracker()

	// Add some data
	tracker.Track("test.sh", "1", true)
	tracker.Track("test.sh", "2", false)

	// Verify data exists
	report := tracker.GetReport()
	if strings.Contains(report, "No coverage data") {
		t.Error("Should have coverage data before reset")
	}

	// Reset
	tracker.Reset()

	// Verify data is cleared
	report = tracker.GetReport()
	if !strings.Contains(report, "No coverage data") {
		t.Error("Should have no coverage data after reset")
	}
}

func TestTrackerDisabled(t *testing.T) {
	tracker := NewBasicTracker(WithEnabled(false))

	// Try to track something
	tracker.Track("test.sh", "1", true)

	// Should have no data
	report := tracker.GetReport()
	if !strings.Contains(report, "No coverage data") {
		t.Error("Disabled tracker should not collect data")
	}
}
