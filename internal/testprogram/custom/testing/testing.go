// Extended testing package with coverage subdirectory support
// This file overlays the standard testing package to add coverage data organization

package testing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/coverage"
	"strings"
	"sync/atomic"
)

// Run runs f as a subtest of t called name with enhanced coverage data collection.
// If GOCOVERDIR is set, it creates a subdirectory for this test's coverage data.
func (t *T) Run(name string, f func(t *T)) bool {
	if t.cleanupStarted.Load() {
		panic("testing: t.Run called during t.Cleanup")
	}

	t.hasSub.Store(true)
	testName, ok, _ := t.tstate.match.fullName(&t.common, name)
	if !ok || shouldFailFast() {
		return true
	}

	// Record the stack trace at the point of this call so that if the subtest
	// function - which runs in a separate stack - is marked as a helper, we can
	// continue walking the stack into the parent test.
	var pc [maxStackLen]uintptr
	n := runtime.Callers(2, pc[:])

	// Enhanced coverage data setup
	originalGoCoverDir := os.Getenv("GOCOVERDIR")
	var coverageSubdir string
	var coverageCleanup func()

	if originalGoCoverDir != "" {
		// Create coverage subdirectory for this test
		coverageSubdir = createCoverageSubdirectory(originalGoCoverDir, testName)
		if coverageSubdir != "" {
			os.Setenv("GOCOVERDIR", coverageSubdir)
			fmt.Printf("[TESTING OVERLAY] Coverage data for test '%s' will be collected in: %s\n", testName, coverageSubdir)

			// Set up cleanup function
			coverageCleanup = func() {
				// Write coverage data before cleanup
				if err := coverage.WriteMetaDir(coverageSubdir); err != nil {
					fmt.Printf("[TESTING OVERLAY] Failed to write meta data for test '%s': %v\n", testName, err)
				}
				if err := coverage.WriteCountersDir(coverageSubdir); err != nil {
					fmt.Printf("[TESTING OVERLAY] Failed to write counter data for test '%s': %v\n", testName, err)
				}
				fmt.Printf("[TESTING OVERLAY] Coverage data written for test '%s' in: %s\n", testName, coverageSubdir)

				// Restore original GOCOVERDIR
				if originalGoCoverDir != "" {
					os.Setenv("GOCOVERDIR", originalGoCoverDir)
				} else {
					os.Unsetenv("GOCOVERDIR")
				}
			}
		}
	}

	// There's no reason to inherit this context from parent. The user's code can't observe
	// the difference between the background context and the one from the parent test.
	ctx, cancelCtx := context.WithCancel(context.Background())
	t = &T{
		common: common{
			barrier:    make(chan bool),
			signal:     make(chan bool, 1),
			name:       testName,
			parent:     &t.common,
			level:      t.level + 1,
			creator:    pc[:n],
			chatty:     t.chatty,
			finished:   false,
			hasSub:     atomic.Bool{},
			raceErrors: int32(len(race.Errors())),
			runner:     t.runner,
			ctx:        ctx,
			cancelCtx:  cancelCtx,
		},
		context: ctx,
		tstate:  t.tstate,
	}
	t.w = indenter{&t.common}

	// Add coverage cleanup to the test cleanup chain
	if coverageCleanup != nil {
		t.Cleanup(coverageCleanup)
	}

	if t.chatty != nil {
		// We rely on the fact that chatty.UpdatedLen is not read after
		// the FuzzTest is done. See the comment below.
		t.chatty.UpdatedLen()
	}

	go tRunner(t, f)
	<-t.signal
	if t.Failed() {
		t.failParent()
	}
	return !t.Failed()
}

// createCoverageSubdirectory creates a subdirectory within the base coverage directory
// for organizing coverage data by test name
func createCoverageSubdirectory(baseCoverDir, testName string) string {
	// Sanitize test name for use as directory name
	sanitizedName := sanitizeTestName(testName)
	if sanitizedName == "" {
		return ""
	}

	subdir := filepath.Join(baseCoverDir, sanitizedName)

	// Create the subdirectory
	if err := os.MkdirAll(subdir, 0755); err != nil {
		fmt.Printf("[TESTING OVERLAY] Failed to create coverage subdirectory %s: %v\n", subdir, err)
		return ""
	}

	return subdir
}

// sanitizeTestName converts a test name to a valid directory name
func sanitizeTestName(testName string) string {
	// Replace invalid characters with underscores
	sanitized := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		if r == ' ' {
			return '_'
		}
		return r
	}, testName)

	// Remove leading/trailing dots and spaces
	sanitized = strings.Trim(sanitized, ". ")

	// Ensure it's not empty and not too long
	if sanitized == "" || len(sanitized) > 200 {
		return fmt.Sprintf("test_%d", len(testName)) // fallback name
	}

	return sanitized
}
