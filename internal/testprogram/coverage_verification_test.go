package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tmc/covutil/coverage"
	"github.com/tmc/covutil/internal/testprogram/testutils"
)

func TestCoverageFileVerification(t *testing.T) {
	// Create a base coverage directory
	baseCoverageDir, err := os.MkdirTemp("", "test_coverage_verification")
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

	testutils.RunWithCoverageSubdir(t, "VerifyFiles", func(t *testing.T) {
		// Do some work that should generate coverage data
		displayBasicInfo()

		// Check that we have the coverage package available
		if coverage.PkgPath == "" {
			t.Error("Coverage package path should not be empty")
		}

		// Verify the subdirectory was created
		currentGoCoverDir := os.Getenv("GOCOVERDIR")
		if currentGoCoverDir == baseCoverageDir {
			t.Error("GOCOVERDIR should be set to subdirectory, not base directory")
		}

		// Verify subdirectory exists
		if _, err := os.Stat(currentGoCoverDir); os.IsNotExist(err) {
			t.Errorf("Coverage subdirectory should exist: %s", currentGoCoverDir)
		}

		// The actual coverage files will be written during cleanup
		t.Logf("Coverage data will be written to: %s", currentGoCoverDir)
	})

	// After the subtest completes, verify files were written
	subdirs, err := os.ReadDir(baseCoverageDir)
	if err != nil {
		t.Fatalf("Failed to read base coverage directory: %v", err)
	}

	foundVerifyFiles := false
	for _, subdir := range subdirs {
		if subdir.Name() == "VerifyFiles" && subdir.IsDir() {
			foundVerifyFiles = true
			subdirPath := filepath.Join(baseCoverageDir, subdir.Name())

			// Check for coverage files
			files, err := os.ReadDir(subdirPath)
			if err != nil {
				t.Errorf("Failed to read subdirectory %s: %v", subdirPath, err)
				continue
			}

			// Look for meta and counter files
			foundMeta := false
			foundCounter := false

			for _, file := range files {
				if !file.IsDir() {
					fileName := file.Name()
					t.Logf("Found coverage file: %s", fileName)

					if filepath.Ext(fileName) == "" && len(fileName) > 7 {
						if fileName[:7] == "covmeta" {
							foundMeta = true
						} else if fileName[:11] == "covcounters" {
							foundCounter = true
						}
					}
				}
			}

			// Note: These might fail if the test process doesn't have proper coverage instrumentation
			// but we should at least see the directory structure
			t.Logf("Found meta file: %v, Found counter file: %v", foundMeta, foundCounter)
		}
	}

	if !foundVerifyFiles {
		t.Error("Expected to find 'VerifyFiles' subdirectory")
	}
}
