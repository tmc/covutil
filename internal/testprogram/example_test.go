package main

import (
	"testing"

	"github.com/tmc/covutil/coverage"
)

func TestExample(t *testing.T) {
	t.Run("SubTest1", func(t *testing.T) {
		// This subtest should get its own coverage directory
		displayBasicInfo()
	})

	t.Run("SubTest2", func(t *testing.T) {
		// This subtest should get its own coverage directory
		if pkgPath := coverage.PkgPath; pkgPath == "" {
			t.Error("Package path is empty")
		}
	})
}
