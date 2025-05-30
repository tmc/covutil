// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package scripttest adapts the script engine for use in tests.
package scripttest

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/coverage"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/tools/txtar"
	"rsc.io/script"
)

// DefaultCmds returns a set of broadly useful script commands.
//
// This set includes all of the commands in script.DefaultCmds,
// as well as a "skip" command that halts the script and causes the
// testing.TB passed to Run to be skipped.
func DefaultCmds() map[string]script.Cmd {
	cmds := script.DefaultCmds()
	cmds["skip"] = Skip()
	return cmds
}

// DefaultConds returns a set of broadly useful script conditions.
//
// This set includes all of the conditions in script.DefaultConds,
// as well as:
//
//   - Conditions of the form "exec:foo" are active when the executable "foo" is
//     found in the test process's PATH, and inactive when the executable is
//     not found.
//
//   - "short" is active when testing.Short() is true.
//
//   - "verbose" is active when testing.Verbose() is true.
func DefaultConds() map[string]script.Cond {
	conds := script.DefaultConds()
	conds["exec"] = CachedExec()
	conds["short"] = script.BoolCondition("testing.Short()", testing.Short())
	conds["verbose"] = script.BoolCondition("testing.Verbose()", testing.Verbose())
	return conds
}

// Run runs the script from the given filename starting at the given initial state.
// When the script completes, Run closes the state.
func run(t testing.TB, e *script.Engine, s *script.State, filename string, testScript io.Reader) {
	t.Helper()
	err := func() (err error) {
		log := new(strings.Builder)
		log.WriteString("\n") // Start output on a new line for consistent indentation.

		// Defer writing to the test log in case the script engine panics during execution,
		// but write the log before we write the final "skip" or "FAIL" line.
		t.Helper()
		defer func() {
			t.Helper()

			if closeErr := s.CloseAndWait(log); err == nil {
				err = closeErr
			}

			if log.Len() > 0 {
				t.Log(strings.TrimSuffix(log.String(), "\n"))
			}
		}()

		if testing.Verbose() {
			// Add the environment to the start of the script log.
			wait, err := script.Env().Run(s)
			if err != nil {
				t.Fatal(err)
			}
			if wait != nil {
				stdout, stderr, err := wait(s)
				if err != nil {
					t.Fatalf("env: %v\n%s", err, stderr)
				}
				if len(stdout) > 0 {
					s.Logf("%s\n", stdout)
				}
			}
		}

		return e.Execute(s, filename, bufio.NewReader(testScript), log)
	}()

	if skip := (skipError{}); errors.As(err, &skip) {
		if skip.msg == "" {
			t.Skip("SKIP")
		} else {
			t.Skipf("SKIP: %v", skip.msg)
		}
	}
	if err != nil {
		t.Errorf("FAIL: %v", err)
	}
}

// Skip returns a sentinel error that causes Run to mark the test as skipped.
func Skip() script.Cmd {
	return script.Command(
		script.CmdUsage{
			Summary: "skip the current test",
			Args:    "[msg]",
		},
		func(_ *script.State, args ...string) (script.WaitFunc, error) {
			if len(args) > 1 {
				return nil, script.ErrUsage
			}
			if len(args) == 0 {
				return nil, skipError{""}
			}
			return nil, skipError{args[0]}
		})
}

type skipError struct {
	msg string
}

func (s skipError) Error() string {
	if s.msg == "" {
		return "skip"
	}
	return s.msg
}

// CachedExec returns a Condition that reports whether the PATH of the test
// binary itself (not the script's current environment) contains the named
// executable.
func CachedExec() script.Cond {
	return script.CachedCondition(
		"<suffix> names an executable in the test binary's PATH",
		func(name string) (bool, error) {
			_, err := exec.LookPath(name)
			return err == nil, nil
		})
}

func Test(t *testing.T, ctx context.Context, engine *script.Engine, env []string, pattern string) {
	gracePeriod := 100 * time.Millisecond
	if deadline, ok := t.Deadline(); ok {
		timeout := time.Until(deadline)

		// If time allows, increase the termination grace period to 5% of the
		// remaining time.
		if gp := timeout / 20; gp > gracePeriod {
			gracePeriod = gp
		}

		// When we run commands that execute subprocesses, we want to reserve two
		// grace periods to clean up. We will send the first termination signal when
		// the context expires, then wait one grace period for the process to
		// produce whatever useful output it can (such as a stack trace). After the
		// first grace period expires, we'll escalate to os.Kill, leaving the second
		// grace period for the test function to record its output before the test
		// process itself terminates.
		timeout -= 2 * gracePeriod

		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		t.Cleanup(cancel)
	}

	files, _ := filepath.Glob(pattern)
	if len(files) == 0 {
		t.Fatal("no testdata")
	}
	for _, file := range files {
		file := file
		name := strings.TrimSuffix(filepath.Base(file), ".txt")
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			workdir := t.TempDir()
			s, err := script.NewState(ctx, workdir, env)
			if err != nil {
				t.Fatal(err)
			}

			// Unpack archive.
			a, err := txtar.ParseFile(file)
			if err != nil {
				t.Fatal(err)
			}
			initScriptDirs(t, s)
			if err := s.ExtractFiles(a); err != nil {
				t.Fatal(err)
			}

			t.Log(time.Now().UTC().Format(time.RFC3339))
			work, _ := s.LookupEnv("WORK")
			t.Logf("$WORK=%s", work)

			// Note: Do not use filepath.Base(file) here:
			// editors that can jump to file:line references in the output
			// will work better seeing the full path relative to cmd/go
			// (where the "go test" command is usually run).
			Run(t, engine, s, file, bytes.NewReader(a.Comment))
		})
	}
}

func initScriptDirs(t testing.TB, s *script.State) {
	must := func(err error) {
		if err != nil {
			t.Helper()
			t.Fatal(err)
		}
	}

	work := s.Getwd()
	must(s.Setenv("WORK", work))
	must(os.MkdirAll(filepath.Join(work, "tmp"), 0777))
	must(s.Setenv(tempEnvName(), filepath.Join(work, "tmp")))
}

func tempEnvName() string {
	switch runtime.GOOS {
	case "windows":
		return "TMP"
	case "plan9":
		return "TMPDIR" // actually plan 9 doesn't have one at all but this is fine
	default:
		return "TMPDIR"
	}
}

// Enhanced scripttest additions with coverage support and parallel control

var (
	// Global flag to disable t.Parallel calls
	disableParallel bool
	parallelMutex   sync.RWMutex
)

// SetParallelMode controls whether tests run in parallel
func SetParallelMode(enabled bool) {
	parallelMutex.Lock()
	defer parallelMutex.Unlock()
	disableParallel = !enabled
}

// IsParallelDisabled returns whether parallel execution is disabled
func IsParallelDisabled() bool {
	parallelMutex.RLock()
	defer parallelMutex.RUnlock()
	return disableParallel
}

// Run runs a script test with enhanced coverage data collection and parallel control.
// If GOCOVERDIR is set, it creates a subdirectory for this test's coverage data.
// If parallel mode is disabled, it skips calling t.Parallel().
func Run(t *testing.T, ts TestScript, cmd string) {
	t.Helper()

	// Check if we should skip t.Parallel()
	if !IsParallelDisabled() {
		t.Parallel()
	}

	// Enhanced coverage data setup - create subdirectory if GOCOVERDIR is set
	originalGoCoverDir := os.Getenv("GOCOVERDIR")
	var coverageSubdir string

	if originalGoCoverDir != "" {
		// Create coverage subdirectory for this test
		testName := t.Name()
		sanitizedName := sanitizeTestName(testName)
		coverageSubdir = filepath.Join(originalGoCoverDir, sanitizedName)

		if err := os.MkdirAll(coverageSubdir, 0755); err != nil {
			t.Logf("[SCRIPTTEST OVERLAY] Failed to create coverage subdirectory %s: %v", coverageSubdir, err)
		} else {
			os.Setenv("GOCOVERDIR", coverageSubdir)
			t.Logf("[SCRIPTTEST OVERLAY] Coverage data for test '%s' will be collected in: %s", testName, coverageSubdir)
		}
	}

	// Add coverage cleanup if we created a subdirectory
	if coverageSubdir != "" {
		t.Cleanup(func() {
			// Write coverage data for this test
			if err := coverage.WriteMetaDir(coverageSubdir); err != nil {
				t.Logf("[SCRIPTTEST OVERLAY] Failed to write meta data for test '%s': %v", testName, err)
			}
			if err := coverage.WriteCountersDir(coverageSubdir); err != nil {
				t.Logf("[SCRIPTTEST OVERLAY] Failed to write counter data for test '%s': %v", testName, err)
			}
			t.Logf("[SCRIPTTEST OVERLAY] Coverage data written for test '%s' in: %s", testName, coverageSubdir)

			// Restore original GOCOVERDIR
			if originalGoCoverDir != "" {
				os.Setenv("GOCOVERDIR", originalGoCoverDir)
			} else {
				os.Unsetenv("GOCOVERDIR")
			}
		})
	}

	// Call the original renamed run function
	run(t, ts, cmd)
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
