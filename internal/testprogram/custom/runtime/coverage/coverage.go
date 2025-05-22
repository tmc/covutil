package coverage

import (
	"fmt"
	"github.com/tmc/covutil"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

// Force our custom setup to run at package initialization
func init() {
	fmt.Println("CUSTOM: Package init called - setting up coverage hooks")
	customSetup()
}

// Add a compile-time verification that this file is being used
const CUSTOM_OVERLAY_ACTIVE = true

// Custom behavior - log when initHook is called
func initHook(istest bool) {
	fmt.Printf("CUSTOM: initHook called with istest=%v\n", istest)

	// Your custom logic here
	customSetup()

	// Note: Original InitHook behavior is handled by the runtime itself
	// No need to call internal APIs directly
}

func customSetup() {
	fmt.Println("CUSTOM: Setting up custom coverage behavior")

	// Set up signal handlers for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("CUSTOM: Signal received, writing coverage data...")
		flushCoverageData()
		os.Exit(0)
	}()

	// Set up runtime finalizer for process exit
	sentinel := &exitHookSentinel{}
	runtime.SetFinalizer(sentinel, (*exitHookSentinel).finalize)
}

type exitHookSentinel struct{}

func (e *exitHookSentinel) finalize() {
	fmt.Println("CUSTOM: Process finalizing, writing coverage data...")
	flushCoverageData()
}

// flushCoverageData writes coverage data to the default coverage directory
func flushCoverageData() {
	covDir := os.Getenv("GOCOVERDIR")
	if covDir == "" {
		covDir = "covdata_custom"
	}

	if err := os.MkdirAll(covDir, 0755); err != nil {
		fmt.Printf("CUSTOM: Failed to create coverage directory %s: %v\n", covDir, err)
		return
	}

	if err := WriteMetaDir(covDir); err != nil {
		fmt.Printf("CUSTOM: Failed to write meta data: %v\n", err)
	} else {
		fmt.Printf("CUSTOM: Meta data written to %s\n", covDir)
	}

	if err := WriteCountersDir(covDir); err != nil {
		fmt.Printf("CUSTOM: Failed to write counter data: %v\n", err)
	} else {
		fmt.Printf("CUSTOM: Counter data written to %s\n", covDir)
	}
}

// WriteMetaDir writes a coverage meta-data file for the currently
// running program to the directory specified in 'dir'. An error will
// be returned if the operation can't be completed successfully (for
// example, if the currently running program was not built with
// "-cover", or if the directory does not exist).
func WriteMetaDir(dir string) error {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// Use root covutil API - this is a placeholder as the exact API may vary
	// For runtime emission, we typically use WriteMetaFileContent
	f, err := os.Create(dir + "/covmeta.runtime")
	if err != nil {
		return err
	}
	defer f.Close()
	return covutil.WriteMetaFileContent(f)
}

// WriteMeta writes the meta-data content (the payload that would
// normally be emitted to a meta-data file) for the currently running
// program to the writer 'w'. An error will be returned if the
// operation can't be completed successfully (for example, if the
// currently running program was not built with "-cover", or if a
// write fails).
func WriteMeta(w io.Writer) error {
	return covutil.WriteMetaFileContent(w)
}

// WriteCountersDir writes a coverage counter-data file for the
// currently running program to the directory specified in 'dir'. An
// error will be returned if the operation can't be completed
// successfully (for example, if the currently running program was not
// built with "-cover", or if the directory does not exist). The
// counter data written will be a snapshot taken at the point of the
// call.
func WriteCountersDir(dir string) error {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// Use root covutil API
	f, err := os.Create(dir + "/covcounters.runtime")
	if err != nil {
		return err
	}
	defer f.Close()
	return covutil.WriteCounterFileContent(f)
}

// WriteCounters writes coverage counter-data content for the
// currently running program to the writer 'w'. An error will be
// returned if the operation can't be completed successfully (for
// example, if the currently running program was not built with
// "-cover", or if a write fails). The counter data written will be a
// snapshot taken at the point of the invocation.
func WriteCounters(w io.Writer) error {
	return covutil.WriteCounterFileContent(w)
}

// ClearCounters clears/resets all coverage counter variables in the
// currently running program. It returns an error if the program in
// question was not built with the "-cover" flag. Clearing of coverage
// counters is also not supported for programs not using atomic
// counter mode (see more detailed comments below for the rationale
// here).
func ClearCounters() error {
	return covutil.ClearCoverageCounters()
}
