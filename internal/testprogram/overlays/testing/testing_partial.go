// This file contains only the replacement T.Run method for testing package overlay
// It will be inserted into the original testing.go by replacing the original Run method

package testing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/coverage"
	"strings"
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

	// Enhanced coverage data setup - create subdirectory if GOCOVERDIR is set
	originalGoCoverDir := os.Getenv("GOCOVERDIR")
	var coverageSubdir string

	if originalGoCoverDir != "" {
		// Create coverage subdirectory for this test
		sanitizedName := sanitizeTestName(testName)
		coverageSubdir = filepath.Join(originalGoCoverDir, sanitizedName)

		if err := os.MkdirAll(coverageSubdir, 0755); err != nil {
			fmt.Printf("[TESTING OVERLAY] Failed to create coverage subdirectory %s: %v\n", coverageSubdir, err)
		} else {
			os.Setenv("GOCOVERDIR", coverageSubdir)
			fmt.Printf("[TESTING OVERLAY] Coverage data for test '%s' will be collected in: %s\n", testName, coverageSubdir)
		}
	}

	// Record the stack trace at the point of this call so that if the subtest
	// function - which runs in a separate stack - is marked as a helper, we can
	// continue walking the stack into the parent test.
	var pc [maxStackLen]uintptr
	n := runtime.Callers(2, pc[:])

	// There's no reason to inherit this context from parent. The user's code can't observe
	// the difference between the background context and the one from the parent test.
	ctx, cancelCtx := context.WithCancel(context.Background())
	t = &T{
		common: common{
			barrier:   make(chan bool),
			signal:    make(chan bool, 1),
			name:      testName,
			parent:    &t.common,
			level:     t.level + 1,
			creator:   pc[:n],
			chatty:    t.chatty,
			ctx:       ctx,
			cancelCtx: cancelCtx,
		},
		tstate: t.tstate,
	}
	t.w = indenter{&t.common}

	// Add coverage cleanup if we created a subdirectory
	if coverageSubdir != "" {
		t.Cleanup(func() {
			// Write coverage data for this test
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
		})
	}

	if t.chatty != nil {
		t.chatty.Updatef(t.name, "=== RUN   %s\n", t.name)
	}
	running.Store(t.name, highPrecisionTimeNow())

	// Instead of reducing the running count of this test before calling the
	// tRunner and increasing it afterwards, we rely on tRunner keeping the
	// count correct. This ensures that a sequence of sequential tests runs
	// without being preempted, even when their parent is a parallel test. This
	// may especially reduce surprises if *parallel == 1.
	go tRunner(t, f)

	// The parent goroutine will block until the subtest either finishes or calls
	// Parallel, but in general we don't know whether the parent goroutine is the
	// top-level test function or some other goroutine it has spawned.
	// To avoid confusing false-negatives, we leave the parent in the running map
	// even though in the typical case it is blocked.

	if !<-t.signal {
		// At this point, it is likely that FailNow was called on one of the
		// parent's ancestors or descendants. Don't bother completing the rest.
		return false
	}
	return !t.Failed()
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
