package testenv

import (
	"testing"
)

func TestHasGoBuild(t *testing.T) {
	if !HasGoBuild() {
		t.Error("Expected HasGoBuild to return true")
	}
}

func TestGoToolPath(t *testing.T) {
	path := GoToolPath(t)
	if path == "" {
		t.Error("Expected GoToolPath to return a non-empty path")
	}
}

func TestMustHaveGoBuild(t *testing.T) {
	// Should not skip if go is available
	MustHaveGoBuild(t)
}

func TestMustHaveGoRun(t *testing.T) {
	// Should not skip if go is available
	MustHaveGoRun(t)
}
