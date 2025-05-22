// Package testenv provides a mock of Go's internal/testenv for testing purposes
package testenv

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// MustHaveGoBuild checks that the current system can build Go programs.
func MustHaveGoBuild(t testing.TB) {
	t.Helper()
	if !HasGoBuild() {
		t.Skip("skipping test: 'go build' not available")
	}
}

// MustHaveGoRun checks that the current system can run Go programs.
func MustHaveGoRun(t testing.TB) {
	t.Helper()
	MustHaveGoBuild(t)
}

// HasGoBuild reports whether the current system can build Go programs.
func HasGoBuild() bool {
	_, err := exec.LookPath("go")
	return err == nil
}

// HasCGO reports whether the current system can use cgo.
func HasCGO() bool {
	// Simple heuristic - check if CGO_ENABLED is not explicitly disabled
	if cgo := os.Getenv("CGO_ENABLED"); cgo == "0" {
		return false
	}
	// Check if we have a C compiler
	if runtime.GOOS == "windows" {
		_, err := exec.LookPath("gcc")
		if err != nil {
			_, err = exec.LookPath("clang")
		}
		return err == nil
	}
	// On Unix-like systems, check for common C compilers
	for _, cc := range []string{"gcc", "clang", "cc"} {
		if _, err := exec.LookPath(cc); err == nil {
			return true
		}
	}
	return false
}

// GoToolPath returns the path to the Go tool.
func GoToolPath(t testing.TB) string {
	t.Helper()

	// Try to find go in PATH
	goPath, err := exec.LookPath("go")
	if err != nil {
		// Try GOROOT if available
		if goroot := os.Getenv("GOROOT"); goroot != "" {
			goPath = filepath.Join(goroot, "bin", "go")
			if runtime.GOOS == "windows" {
				goPath += ".exe"
			}
			if _, err := os.Stat(goPath); err == nil {
				return goPath
			}
		}
		t.Fatalf("cannot find go tool: %v", err)
	}

	return goPath
}
