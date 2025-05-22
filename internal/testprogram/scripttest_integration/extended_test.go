package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"rsc.io/script"
	"rsc.io/script/scripttest"
)

// TestExtendedCoverageScenarios tests additional coverage collection scenarios
func TestExtendedCoverageScenarios(t *testing.T) {
	if os.Getenv("GO_INTEGRATION_COVERAGE") == "" {
		t.Skip("Skipping extended coverage test - set GO_INTEGRATION_COVERAGE=1 to enable")
	}

	displayCoverageConfiguration(t)

	// Setup coverage directory
	coverageDir := os.Getenv("GOCOVERDIR")
	if coverageDir == "" {
		coverageDir = t.TempDir()
		os.Setenv("GOCOVERDIR", coverageDir)
	}

	// Setup curated PATH
	binDir := setupCuratedPath(t)
	engine := &script.Engine{}

	t.Run("recursive_binary_calls", func(subT *testing.T) {
		testCoverDir := filepath.Join(coverageDir, "recursive")
		workDir := subT.TempDir()

		env := []string{
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GOCOVERDIR=" + testCoverDir,
			"GO_INTEGRATION_COVERAGE=1",
		}

		state, err := script.NewState(context.Background(), workDir, env)
		if err != nil {
			subT.Fatalf("Failed to create script state: %v", err)
		}

		scriptContent := `
# Test recursive binary calls with varying depths
exec main recursive 5
exec cmd1 chain
exec cmd2 recursive 3
exec cmd3 recursive 2

# Test error handling paths
! exec main invalid-command
! exec cmd1 unknown-operation
! exec cmd2 bad-args
`

		subT.Logf("Testing recursive binary execution patterns...")
		scripttest.Run(subT, engine, state, "recursive.txt", strings.NewReader(scriptContent))

		collectIntegrationCoverageData(subT, testCoverDir, coverageDir)
		verifyCoverageFiles(subT, testCoverDir, "recursive_binary_calls")
	})

	t.Run("error_handling_coverage", func(subT *testing.T) {
		testCoverDir := filepath.Join(coverageDir, "error_handling")
		workDir := subT.TempDir()

		env := []string{
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GOCOVERDIR=" + testCoverDir,
			"GO_INTEGRATION_COVERAGE=1",
		}

		state, err := script.NewState(context.Background(), workDir, env)
		if err != nil {
			subT.Fatalf("Failed to create script state: %v", err)
		}

		scriptContent := `
# Test error conditions to get coverage of error paths
! exec main
! exec main unknown-command arg1 arg2
! exec cmd1
! exec cmd1 invalid-operation
! exec cmd2 bad-input
! exec cmd3 error-condition

# Test with various invalid arguments
! exec main tool
! exec cmd1 process invalid-file
! exec cmd2 analyze
! exec cmd3 deploy invalid-env
`

		subT.Logf("Testing error handling code paths...")
		scripttest.Run(subT, engine, state, "errors.txt", strings.NewReader(scriptContent))

		collectIntegrationCoverageData(subT, testCoverDir, coverageDir)
		verifyCoverageFiles(subT, testCoverDir, "error_handling_coverage")
	})

	t.Run("parallel_execution", func(subT *testing.T) {
		testCoverDir := filepath.Join(coverageDir, "parallel")
		workDir := subT.TempDir()

		env := []string{
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GOCOVERDIR=" + testCoverDir,
			"GO_INTEGRATION_COVERAGE=1",
		}

		state, err := script.NewState(context.Background(), workDir, env)
		if err != nil {
			subT.Fatalf("Failed to create script state: %v", err)
		}

		scriptContent := `
# Test concurrent execution patterns
# Note: scripttest runs sequentially, but we test rapid succession
exec main hello World &
exec cmd1 greet Universe &
exec cmd2 elaborate Cosmos &
exec cmd3 flourish Galaxy &

# Wait for background processes and test more commands
exec main tool build
exec cmd1 build project
exec cmd2 test suite
exec cmd3 package release

# Test timing-sensitive scenarios
exec main recursive 1
exec cmd1 chain
exec cmd2 analyze quick
exec cmd3 report brief
`

		subT.Logf("Testing parallel/rapid execution patterns...")
		scripttest.Run(subT, engine, state, "parallel.txt", strings.NewReader(scriptContent))

		collectIntegrationCoverageData(subT, testCoverDir, coverageDir)
		verifyCoverageFiles(subT, testCoverDir, "parallel_execution")
	})

	t.Run("stress_test_chains", func(subT *testing.T) {
		testCoverDir := filepath.Join(coverageDir, "stress")
		workDir := subT.TempDir()

		env := []string{
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GOCOVERDIR=" + testCoverDir,
			"GO_INTEGRATION_COVERAGE=1",
		}

		state, err := script.NewState(context.Background(), workDir, env)
		if err != nil {
			subT.Fatalf("Failed to create script state: %v", err)
		}

		scriptContent := `
# Stress test with many binary calls
exec main hello StressTest1
exec main hello StressTest2
exec main hello StressTest3
exec main recursive 2
exec main recursive 1
exec main tool build test deploy

exec cmd1 greet Load1
exec cmd1 greet Load2 
exec cmd1 process Load3
exec cmd1 build Load4
exec cmd1 chain

exec cmd2 elaborate Stress1
exec cmd2 elaborate Stress2
exec cmd2 process Stress3
exec cmd2 test Stress4
exec cmd2 analyze Stress5
exec cmd2 recursive 1

exec cmd3 flourish Final1
exec cmd3 flourish Final2
exec cmd3 process Final3
exec cmd3 package Final4
exec cmd3 deploy Final5
exec cmd3 report Final6
exec cmd3 test-report Final7
exec cmd3 summarize Final8
exec cmd3 cleanup Final9

# Final chain test
exec main tool build deploy test analyze report
`

		subT.Logf("Running stress test with many binary executions...")
		scripttest.Run(subT, engine, state, "stress.txt", strings.NewReader(scriptContent))

		collectIntegrationCoverageData(subT, testCoverDir, coverageDir)
		verifyCoverageFiles(subT, testCoverDir, "stress_test_chains")
	})

	// Final comprehensive verification
	verifyComprehensiveCoverage(t, coverageDir)
}

// TestCoverageCollectionModes tests different collection modes explicitly
func TestCoverageCollectionModes(t *testing.T) {
	if os.Getenv("GO_INTEGRATION_COVERAGE") == "" {
		t.Skip("Skipping mode test - set GO_INTEGRATION_COVERAGE=1 to enable")
	}

	originalMode := os.Getenv("COVERAGE_COLLECTION_MODE")
	defer func() {
		if originalMode != "" {
			os.Setenv("COVERAGE_COLLECTION_MODE", originalMode)
		} else {
			os.Unsetenv("COVERAGE_COLLECTION_MODE")
		}
	}()

	modes := []string{"harness", "overlay", "both", "auto"}

	for _, mode := range modes {
		mode := mode // capture for closure
		t.Run("mode_"+mode, func(subT *testing.T) {
			os.Setenv("COVERAGE_COLLECTION_MODE", mode)

			coverageDir := subT.TempDir()
			os.Setenv("GOCOVERDIR", coverageDir)

			displayCoverageConfiguration(subT)

			binDir := setupCuratedPath(subT)
			engine := &script.Engine{}
			testCoverDir := filepath.Join(coverageDir, "mode_test")
			workDir := subT.TempDir()

			env := []string{
				"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
				"GOCOVERDIR=" + testCoverDir,
				"GO_INTEGRATION_COVERAGE=1",
				"COVERAGE_COLLECTION_MODE=" + mode,
			}

			state, err := script.NewState(context.Background(), workDir, env)
			if err != nil {
				subT.Fatalf("Failed to create script state: %v", err)
			}

			scriptContent := `
# Simple test for each mode
exec main hello ModeTest
exec cmd1 greet ModeTest
exec cmd2 elaborate ModeTest
exec cmd3 flourish ModeTest
`

			subT.Logf("Testing collection mode: %s", mode)
			scripttest.Run(subT, engine, state, "mode_test.txt", strings.NewReader(scriptContent))

			collectIntegrationCoverageData(subT, testCoverDir, coverageDir)

			// Verify mode-specific behavior
			files, err := filepath.Glob(filepath.Join(coverageDir, "cov*"))
			if err != nil {
				subT.Errorf("Failed to find coverage files: %v", err)
			}

			subT.Logf("Mode %s collected %d coverage files", mode, len(files))

			// Check for mode-specific file patterns
			if mode == "harness" || mode == "both" || mode == "auto" {
				harnessFiles, _ := filepath.Glob(filepath.Join(coverageDir, "harness_*"))
				subT.Logf("Found %d harness-pattern files", len(harnessFiles))
			}
		})
	}
}

// TestCoverageDataIntegrity tests the integrity and completeness of coverage data
func TestCoverageDataIntegrity(t *testing.T) {
	if os.Getenv("GO_INTEGRATION_COVERAGE") == "" {
		t.Skip("Skipping integrity test - set GO_INTEGRATION_COVERAGE=1 to enable")
	}

	displayCoverageConfiguration(t)

	coverageDir := t.TempDir()
	os.Setenv("GOCOVERDIR", coverageDir)

	binDir := setupCuratedPath(t)
	engine := &script.Engine{}

	t.Run("data_consistency", func(subT *testing.T) {
		testCoverDir := filepath.Join(coverageDir, "consistency")
		workDir := subT.TempDir()

		env := []string{
			"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
			"GOCOVERDIR=" + testCoverDir,
			"GO_INTEGRATION_COVERAGE=1",
		}

		state, err := script.NewState(context.Background(), workDir, env)
		if err != nil {
			subT.Fatalf("Failed to create script state: %v", err)
		}

		// Run the same commands multiple times to test consistency
		for i := 0; i < 3; i++ {
			scriptContent := `
exec main hello ConsistencyTest
exec cmd1 greet ConsistencyTest
exec cmd2 elaborate ConsistencyTest
exec cmd3 flourish ConsistencyTest
`
			subT.Logf("Running consistency test iteration %d", i+1)
			scripttest.Run(subT, engine, state, "consistency.txt", strings.NewReader(scriptContent))

			collectIntegrationCoverageData(subT, testCoverDir, coverageDir)

			// Brief pause between iterations
			time.Sleep(100 * time.Millisecond)
		}

		// Verify coverage files exist and are readable
		files, err := filepath.Glob(filepath.Join(coverageDir, "cov*"))
		if err != nil {
			subT.Errorf("Failed to find coverage files: %v", err)
		}

		subT.Logf("Found %d coverage files after consistency test", len(files))

		// Check file sizes are reasonable
		for _, file := range files {
			info, err := os.Stat(file)
			if err != nil {
				subT.Errorf("Failed to stat coverage file %s: %v", file, err)
				continue
			}
			if info.Size() == 0 {
				subT.Errorf("Coverage file %s is empty", file)
			}
			if info.Size() > 1024*1024 { // 1MB seems reasonable for test coverage
				subT.Logf("Large coverage file %s: %d bytes", file, info.Size())
			}
		}
	})
}
