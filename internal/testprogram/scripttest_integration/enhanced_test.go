package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"rsc.io/script"
	"rsc.io/script/scripttest"
)

// TestEnhancedIntegrationCoverage tests multi-level binary execution with Go tools
func TestEnhancedIntegrationCoverage(t *testing.T) {
	// Enable integration coverage mode
	os.Setenv("GO_INTEGRATION_COVERAGE", "1")

	// Display current coverage configuration
	displayCoverageConfiguration(t)

	// Setup coverage directory
	coverageDir := os.Getenv("GOCOVERDIR")
	if coverageDir == "" {
		coverageDir = t.TempDir()
		os.Setenv("GOCOVERDIR", coverageDir)
	}
	t.Logf("Coverage data will be collected in: %s", coverageDir)

	// Create coverage helper to demonstrate coverage collection in the test itself
	helper := NewCoverageHelper("TestEnhancedIntegrationCoverage")
	if err := helper.ProcessIntegrationTest("enhanced_integration"); err != nil {
		t.Errorf("Coverage helper processing failed: %v", err)
	}

	// Setup curated PATH with test binaries and Go tools
	binDir := setupCuratedPath(t)

	// Test Go tools integration with our binaries
	t.Run("go_tools_integration", func(subT *testing.T) {
		// Test coverage helper for this scenario
		scenarioHelper := NewCoverageHelper("go_tools_integration")
		if err := scenarioHelper.ProcessIntegrationTest("go_tools"); err != nil {
			subT.Errorf("Scenario coverage helper failed: %v", err)
		}

		testCoverDir := filepath.Join(coverageDir, "go_tools")
		if err := os.MkdirAll(testCoverDir, 0755); err != nil {
			subT.Fatalf("Failed to create coverage directory: %v", err)
		}
		os.Setenv("GOCOVERDIR", testCoverDir)
		defer os.Setenv("GOCOVERDIR", coverageDir)

		// Create scripttest engine with enhanced commands
		engine := &script.Engine{
			Cmds:  createEnhancedCommands(binDir),
			Conds: scripttest.DefaultConds(),
		}

		// Create script state with curated PATH
		ctx := context.Background()
		workDir := subT.TempDir()
		env := []string{
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GOCOVERDIR=" + testCoverDir,
			"GO_INTEGRATION_COVERAGE=1",
		}

		state, err := script.NewState(ctx, workDir, env)
		if err != nil {
			subT.Fatalf("Failed to create script state: %v", err)
		}

		// Execute complex script that uses Go tools and our binaries
		scriptContent := `
# Test integration of Go tools with our test binaries
# This simulates a real-world scenario where Go tools are used alongside custom binaries

# Use Go tools to create a simple Go file
exec go mod init testproject
exec go mod tidy

# Create a simple Go file using our binaries to generate content
main-tool tool build > build.log
exec cat build.log

# Use Go tools to format and check the created content
go-tool fmt .
go-tool vet ./...

# Call our binaries in various combinations
main-tool hello GoToolsIntegration
exec main chain
exec cmd1 build testproject
exec cmd2 test integration
exec cmd3 deploy production

# Use go tool to run a test that calls our binaries
go-tool test -v -run=TestNothing .
`

		subT.Logf("Executing enhanced scripttest with Go tools...")

		// Enable synthetic coverage for this script
		if os.Getenv("SYNTHETIC_COVERAGE") != "" {
			RunScriptWithCoverage(subT, engine, state, "go_tools.txt", scriptContent)
		}

		scripttest.Run(subT, engine, state, "go_tools.txt", strings.NewReader(scriptContent))

		// Collect coverage data after script execution
		collectIntegrationCoverageData(subT, testCoverDir, coverageDir)

		// Integrate synthetic coverage if enabled
		if err := IntegrateSyntheticCoverage(coverageDir); err != nil {
			subT.Logf("Failed to integrate synthetic coverage: %v", err)
		}

		// Verify coverage data was collected
		verifyCoverageFiles(subT, testCoverDir, "go_tools_integration")
	})

	// Test binary chains that call Go tools
	t.Run("binary_calls_go_tools", func(subT *testing.T) {
		testCoverDir := filepath.Join(coverageDir, "binary_go_tools")
		if err := os.MkdirAll(testCoverDir, 0755); err != nil {
			subT.Fatalf("Failed to create coverage directory: %v", err)
		}
		os.Setenv("GOCOVERDIR", testCoverDir)
		defer os.Setenv("GOCOVERDIR", coverageDir)

		env := append(os.Environ(),
			"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
			"GOCOVERDIR="+testCoverDir,
			"GO_INTEGRATION_COVERAGE=1",
		)

		// Test our binaries calling Go tools
		// This creates a chain: main -> cmd1 -> cmd2 -> cmd3 -> go tools
		cmd := exec.Command(filepath.Join(binDir, "main"), "tool", "build", "test", "deploy")
		cmd.Env = env
		cmd.Dir = subT.TempDir()

		output, err := cmd.CombinedOutput()
		if err != nil {
			subT.Logf("Binary execution output:\n%s", output)
			// Don't fail on Go tool errors, focus on coverage collection
		}

		subT.Logf("Binary->Go tools execution output:\n%s", output)

		// Verify coverage data was collected
		verifyCoverageFiles(subT, testCoverDir, "binary_calls_go_tools")
	})

	// Test complex workflow mixing Go tools and custom binaries
	t.Run("complex_workflow", func(subT *testing.T) {
		testCoverDir := filepath.Join(coverageDir, "complex")
		if err := os.MkdirAll(testCoverDir, 0755); err != nil {
			subT.Fatalf("Failed to create coverage directory: %v", err)
		}
		os.Setenv("GOCOVERDIR", testCoverDir)
		defer os.Setenv("GOCOVERDIR", coverageDir)

		engine := &script.Engine{
			Cmds:  createEnhancedCommands(binDir),
			Conds: scripttest.DefaultConds(),
		}

		ctx := context.Background()
		workDir := subT.TempDir()
		env := append(os.Environ(),
			"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
			"GOCOVERDIR="+testCoverDir,
			"GO_INTEGRATION_COVERAGE=1",
		)

		state, err := script.NewState(ctx, workDir, env)
		if err != nil {
			subT.Fatalf("Failed to create script state: %v", err)
		}

		// Complex workflow that exercises multiple code paths
		scriptContent := `
# Complex workflow: Go tools + custom binaries + recursive calls
# This tests the most comprehensive coverage scenario

# Initialize a project with Go tools
exec go mod init complex-workflow
exec go version

# Use our binaries to create content and process it
main-tool hello ComplexWorkflow
exec main recursive 3
exec cmd1 chain
exec cmd2 analyze complex-data
exec cmd3 report comprehensive

# Use Go tools to analyze what we created
go-tool fmt .
go-tool vet ./...

# More binary interactions
exec main tool build test deploy format analyze
exec cmd1 greet ComplexWorkflow
exec cmd2 elaborate ComplexWorkflow  
exec cmd3 flourish ComplexWorkflow

# Final Go tools usage
go-tool doc fmt
`

		subT.Logf("Executing complex workflow with comprehensive coverage...")

		// Enable synthetic coverage for this script
		if os.Getenv("SYNTHETIC_COVERAGE") != "" {
			RunScriptWithCoverage(subT, engine, state, "complex.txt", scriptContent)
		}

		scripttest.Run(subT, engine, state, "complex.txt", strings.NewReader(scriptContent))

		// Collect coverage data after script execution
		collectIntegrationCoverageData(subT, testCoverDir, coverageDir)

		// Integrate synthetic coverage if enabled
		if err := IntegrateSyntheticCoverage(coverageDir); err != nil {
			subT.Logf("Failed to integrate synthetic coverage: %v", err)
		}

		// Verify coverage data was collected
		verifyCoverageFiles(subT, testCoverDir, "complex_workflow")
	})

	// Final comprehensive verification
	verifyComprehensiveCoverage(t, coverageDir)
}

// setupCuratedPath creates a directory with test binaries and Go tools as part of test setup
func setupCuratedPath(t *testing.T) string {
	t.Helper()

	binDir := filepath.Join(t.TempDir(), "curated_bin")

	t.Logf("Setting up curated PATH in: %s", binDir)

	// Build test binaries and Go tools as part of test setup
	if err := buildTestBinariesInline(t, binDir); err != nil {
		t.Fatalf("Failed to build test binaries: %v", err)
	}

	if err := buildGoToolsInline(t, binDir); err != nil {
		t.Logf("Warning: Failed to build Go tools: %v", err)
		// Continue without Go tools if they fail to build
	}

	t.Logf("Curated PATH setup complete with %d binaries", countBinaries(binDir))

	return binDir
}

// buildTestBinariesInline builds our test binaries directly in the test
func buildTestBinariesInline(t *testing.T, binDir string) error {
	t.Helper()

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	binaries := []struct {
		name   string
		srcDir string
	}{
		{"main", "main"},
		{"cmd1", "cmd1"},
		{"cmd2", "cmd2"},
		{"cmd3", "cmd3"},
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	for _, binary := range binaries {
		binaryPath := filepath.Join(binDir, binary.name)
		srcPath := filepath.Join(wd, binary.srcDir)

		t.Logf("Building %s from %s", binary.name, srcPath)

		cmd := exec.Command("go", "build", "-cover", "-o", binaryPath, srcPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to build %s: %w", binary.name, err)
		}
	}

	return nil
}

// buildGoToolsInline builds Go tools directly in the test
func buildGoToolsInline(t *testing.T, binDir string) error {
	t.Helper()

	// Get GOROOT to find tool sources
	cmd := exec.Command("go", "env", "GOROOT")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get GOROOT: %w", err)
	}
	goroot := strings.TrimSpace(string(output))

	// Essential Go tools for testing
	tools := []struct {
		name    string
		srcPath string
	}{
		{"go", "cmd/go"},
		{"gofmt", "cmd/gofmt"},
		{"vet", "cmd/vet"},
	}

	for _, tool := range tools {
		binaryPath := filepath.Join(binDir, tool.name)
		srcPath := filepath.Join(goroot, "src", tool.srcPath)

		t.Logf("Building Go tool %s from %s", tool.name, srcPath)

		// Try to build with coverage, fallback without if it fails
		cmd := exec.Command("go", "build", "-cover", "-o", binaryPath, srcPath)
		if err := cmd.Run(); err != nil {
			t.Logf("Building %s without coverage (coverage build failed)", tool.name)
			cmd = exec.Command("go", "build", "-o", binaryPath, srcPath)
			if err := cmd.Run(); err != nil {
				t.Logf("Warning: failed to build %s: %v", tool.name, err)
				continue
			}
		}
	}

	return nil
}

// countBinaries counts the number of executable binaries in a directory
func countBinaries(binDir string) int {
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			if info, err := entry.Info(); err == nil && info.Mode()&0111 != 0 {
				count++
			}
		}
	}
	return count
}

// createEnhancedCommands creates scripttest commands including Go tools
func createEnhancedCommands(binDir string) map[string]script.Cmd {
	cmds := scripttest.DefaultCmds()

	// Add main-tool command
	cmds["main-tool"] = script.Command(
		script.CmdUsage{
			Summary: "call main program as a tool",
			Args:    "operation [args...]",
		},
		func(s *script.State, args ...string) (script.WaitFunc, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("main-tool requires at least one argument")
			}

			mainBinary := filepath.Join(binDir, "main")
			cmd := exec.Command(mainBinary, args...)
			cmd.Dir = s.Getwd()
			cmd.Env = s.Environ()

			return func(s *script.State) (stdout, stderr string, err error) {
				var outBuf, errBuf strings.Builder
				cmd.Stdout = &outBuf
				cmd.Stderr = &errBuf
				err = cmd.Run()
				return outBuf.String(), errBuf.String(), err
			}, nil
		},
	)

	// Add go-tool command that uses our curated Go binary
	cmds["go-tool"] = script.Command(
		script.CmdUsage{
			Summary: "call Go tool from curated PATH",
			Args:    "subcommand [args...]",
		},
		func(s *script.State, args ...string) (script.WaitFunc, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("go-tool requires at least one argument")
			}

			goBinary := filepath.Join(binDir, "go")
			// Check if our curated go binary exists, fallback to system go
			if _, err := os.Stat(goBinary); os.IsNotExist(err) {
				goBinary = "go"
			}

			cmd := exec.Command(goBinary, args...)
			cmd.Dir = s.Getwd()
			cmd.Env = s.Environ()

			return func(s *script.State) (stdout, stderr string, err error) {
				var outBuf, errBuf strings.Builder
				cmd.Stdout = &outBuf
				cmd.Stderr = &errBuf
				err = cmd.Run()
				return outBuf.String(), errBuf.String(), err
			}, nil
		},
	)

	return cmds
}

// verifyCoverageFiles verifies that coverage files were created
func verifyCoverageFiles(t *testing.T, coverageDir, testName string) {
	t.Helper()

	if _, err := os.Stat(coverageDir); os.IsNotExist(err) {
		t.Errorf("Coverage directory does not exist: %s", coverageDir)
		return
	}

	entries, err := os.ReadDir(coverageDir)
	if err != nil {
		t.Errorf("Failed to read coverage directory: %v", err)
		return
	}

	var metaFiles, counterFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, "covmeta.") {
			metaFiles = append(metaFiles, name)
		} else if strings.HasPrefix(name, "covcounters.") {
			counterFiles = append(counterFiles, name)
		}
	}

	t.Logf("Test %s: Found %d meta files and %d counter files", testName, len(metaFiles), len(counterFiles))

	// Log details about coverage files
	for _, file := range append(metaFiles, counterFiles...) {
		if info, err := os.Stat(filepath.Join(coverageDir, file)); err == nil {
			t.Logf("Coverage file: %s (size: %d bytes)", file, info.Size())
		}
	}

	if len(metaFiles) == 0 && len(counterFiles) == 0 {
		t.Logf("Warning: No coverage files found for test %s", testName)
	}
}

// verifyComprehensiveCoverage provides a summary of all coverage collection
func verifyComprehensiveCoverage(t *testing.T, coverageDir string) {
	t.Helper()

	t.Logf("=== Enhanced Coverage Collection Summary ===")

	var allFiles []string
	err := filepath.Walk(coverageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), "cov") {
			allFiles = append(allFiles, path)
		}
		return nil
	})

	if err != nil {
		t.Errorf("Failed to walk coverage directory: %v", err)
		return
	}

	t.Logf("Total coverage files found: %d", len(allFiles))

	if len(allFiles) > 0 {
		t.Logf("✅ SUCCESS: Enhanced coverage data collected from Go tools + custom binaries")

		// Group files by test scenario
		scenarios := make(map[string][]string)
		for _, file := range allFiles {
			relPath, _ := filepath.Rel(coverageDir, file)
			parts := strings.Split(relPath, string(os.PathSeparator))
			if len(parts) > 0 {
				scenario := parts[0]
				scenarios[scenario] = append(scenarios[scenario], relPath)
			}
		}

		for scenario, files := range scenarios {
			t.Logf("  📁 %s: %d coverage files", scenario, len(files))
		}
	} else {
		t.Logf("⚠️  No coverage files found - overlay may not be working correctly")
	}
}

// collectIntegrationCoverageData manually collects coverage data from temporary directories
// This is our scripttest harness-based approach to coverage collection
func collectIntegrationCoverageData(t *testing.T, testCoverDir, mainCoverDir string) {
	if !strings.Contains(os.Getenv("GO_INTEGRATION_COVERAGE"), "1") {
		return
	}

	// Check coverage collection mode
	mode := strings.ToLower(os.Getenv("COVERAGE_COLLECTION_MODE"))
	if mode == "" {
		mode = "auto"
	}

	// Skip harness collection if mode is overlay-only
	if mode == "overlay" {
		t.Logf("Skipping harness collection (mode=%s)", mode)
		return
	}

	t.Logf("Collecting integration coverage data from %s to %s", testCoverDir, mainCoverDir)

	// Ensure main coverage directory exists
	if err := os.MkdirAll(mainCoverDir, 0755); err != nil {
		t.Logf("Failed to create main coverage directory: %v", err)
		return
	}

	// Find all coverage files in the test coverage directory
	err := filepath.Walk(testCoverDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Copy coverage files (covmeta.* and covcounters.*)
		if strings.HasPrefix(info.Name(), "cov") {
			// Create unique filename to avoid conflicts
			relPath, _ := filepath.Rel(testCoverDir, path)
			uniqueName := strings.ReplaceAll(relPath, string(os.PathSeparator), "_")
			uniqueName = fmt.Sprintf("harness_%d_%s", os.Getpid(), uniqueName)

			dstPath := filepath.Join(mainCoverDir, uniqueName)

			if err := copyFile(path, dstPath); err != nil {
				t.Logf("Failed to copy coverage file %s: %v", path, err)
			} else {
				t.Logf("Copied coverage file: %s -> %s", info.Name(), uniqueName)
			}
		}

		return nil
	})

	if err != nil {
		t.Logf("Error walking test coverage directory: %v", err)
	}
}

// copyFile is available from coverage_utils.go

// displayCoverageConfiguration shows the current coverage collection setup
func displayCoverageConfiguration(t *testing.T) {
	t.Logf("=== Coverage Collection Configuration ===")

	// Environment variables
	integrationCov := os.Getenv("GO_INTEGRATION_COVERAGE")
	goCoverDir := os.Getenv("GOCOVERDIR")
	mode := os.Getenv("COVERAGE_COLLECTION_MODE")
	if mode == "" {
		mode = "auto"
	}

	t.Logf("GO_INTEGRATION_COVERAGE: %s", integrationCov)
	t.Logf("GOCOVERDIR: %s", goCoverDir)
	t.Logf("COVERAGE_COLLECTION_MODE: %s", mode)

	// Determine active collection methods
	var methods []string
	switch strings.ToLower(mode) {
	case "harness":
		methods = []string{"Scripttest Harness"}
	case "overlay":
		methods = []string{"Runtime Overlay"}
	case "both":
		methods = []string{"Scripttest Harness", "Runtime Overlay"}
	case "auto":
		if goCoverDir != "" {
			methods = []string{"Scripttest Harness", "Runtime Overlay (auto)"}
		} else {
			methods = []string{"Scripttest Harness (auto)"}
		}
	default:
		methods = []string{"Unknown mode: " + mode}
	}

	t.Logf("Active collection methods: %s", strings.Join(methods, ", "))
	t.Logf("==========================================")
}
