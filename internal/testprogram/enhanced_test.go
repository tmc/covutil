package main

import (
	"testing"

	"github.com/tmc/covutil/coverage"
	"github.com/tmc/covutil/internal/testprogram/testutils"
)

func TestEnhancedCoverage(t *testing.T) {
	// Test our enhanced testing functionality without overlays
	testutils.RunWithCoverageSubdir(t, "SubTest1", func(t *testing.T) {
		// This subtest should get its own coverage directory
		displayBasicInfo()
		if pkgPath := coverage.PkgPath; pkgPath == "" {
			t.Error("Package path is empty")
		}
	})

	testutils.RunWithCoverageSubdir(t, "SubTest2", func(t *testing.T) {
		// This subtest should get its own coverage directory
		// Test that we have coverage functionality available
		if coverage.PkgPath == "" {
			t.Error("Coverage package path should not be empty")
		}
	})
}
