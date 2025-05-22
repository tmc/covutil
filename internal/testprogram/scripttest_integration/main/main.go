package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Main program that calls other binaries - this is our primary target for coverage
func main() {
	fmt.Printf("[MAIN] Starting main program at %s\n", time.Now().Format(time.RFC3339))

	// Coverage copying is now handled automatically by the overlay system

	// Parse command line arguments
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <operation> [args...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Operations: hello, chain, recursive, tool\n")
		os.Exit(1)
	}

	operation := os.Args[1]
	args := os.Args[2:]

	switch operation {
	case "hello":
		handleHello(args)
	case "chain":
		handleChain(args)
	case "recursive":
		handleRecursive(args)
	case "tool":
		handleTool(args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown operation: %s\n", operation)
		os.Exit(1)
	}

	fmt.Printf("[MAIN] Completed operation '%s' at %s\n", operation, time.Now().Format(time.RFC3339))
}

// handleHello demonstrates basic functionality with coverage
func handleHello(args []string) {
	name := "World"
	if len(args) > 0 {
		name = args[0]
	}

	fmt.Printf("[MAIN] Hello, %s!\n", name)

	// Call cmd1 for additional processing
	if err := callCmd("cmd1", "greet", name); err != nil {
		fmt.Printf("[MAIN] Warning: cmd1 failed: %v\n", err)
	}
}

// handleChain demonstrates chaining multiple binary calls
func handleChain(args []string) {
	fmt.Printf("[MAIN] Starting command chain\n")

	// Call cmd1 -> cmd2 -> cmd3 in sequence
	commands := []struct {
		cmd  string
		args []string
	}{
		{"cmd1", []string{"process", "step1"}},
		{"cmd2", []string{"process", "step2"}},
		{"cmd3", []string{"process", "step3"}},
	}

	for i, cmdInfo := range commands {
		fmt.Printf("[MAIN] Chain step %d: calling %s\n", i+1, cmdInfo.cmd)
		if err := callCmd(cmdInfo.cmd, cmdInfo.args...); err != nil {
			fmt.Printf("[MAIN] Chain step %d failed: %v\n", i+1, err)
			return
		}
	}

	fmt.Printf("[MAIN] Command chain completed successfully\n")
}

// handleRecursive demonstrates recursive binary calls
func handleRecursive(args []string) {
	depth := 1
	if len(args) > 0 {
		if d, err := strconv.Atoi(args[0]); err == nil && d > 0 {
			depth = d
		}
	}

	fmt.Printf("[MAIN] Recursive call with depth %d\n", depth)

	if depth > 1 {
		// Call cmd2 which will call cmd3, creating a recursive pattern
		if err := callCmd("cmd2", "recursive", strconv.Itoa(depth-1)); err != nil {
			fmt.Printf("[MAIN] Recursive call failed: %v\n", err)
		}
	} else {
		fmt.Printf("[MAIN] Reached recursion base case\n")
	}
}

// handleTool demonstrates calling this program as a go tool
func handleTool(args []string) {
	fmt.Printf("[MAIN] Tool mode activated with args: %v\n", args)

	// Perform some tool-specific work
	for i, arg := range args {
		fmt.Printf("[MAIN] Processing tool arg %d: %s\n", i+1, arg)

		// Each arg triggers a different command
		switch arg {
		case "build":
			callCmd("cmd1", "build", "project")
		case "test":
			callCmd("cmd2", "test", "suite")
		case "deploy":
			callCmd("cmd3", "deploy", "production")
		default:
			fmt.Printf("[MAIN] Unknown tool arg: %s\n", arg)
		}
	}
}

// callCmd executes another binary in our curated PATH
func callCmd(cmdName string, args ...string) error {
	fmt.Printf("[MAIN] Calling %s with args: %v\n", cmdName, args)

	// Find the command in PATH
	cmdPath, err := exec.LookPath(cmdName)
	if err != nil {
		return fmt.Errorf("command %s not found in PATH: %w", cmdName, err)
	}

	fmt.Printf("[MAIN] Found %s at %s\n", cmdName, cmdPath)

	// Execute the command
	cmd := exec.Command(cmdPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ() // Pass through environment including GOCOVERDIR

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command %s failed: %w", cmdName, err)
	}

	fmt.Printf("[MAIN] Successfully executed %s\n", cmdName)
	return nil
}

// getExecutablePath returns the path to the current executable
func getExecutablePath() string {
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return os.Args[0]
}

// logCoverageInfo logs information about coverage collection
func logCoverageInfo() {
	if coverDir := os.Getenv("GOCOVERDIR"); coverDir != "" {
		fmt.Printf("[MAIN] Coverage collection enabled, GOCOVERDIR=%s\n", coverDir)

		// List existing coverage files
		if entries, err := os.ReadDir(coverDir); err == nil {
			fmt.Printf("[MAIN] Found %d entries in coverage directory\n", len(entries))
			for _, entry := range entries {
				if !entry.IsDir() {
					fmt.Printf("[MAIN] Coverage file: %s\n", entry.Name())
				}
			}
		}
	} else {
		fmt.Printf("[MAIN] Coverage collection disabled\n")
	}
}

// copyCoverageDataUp copies coverage data from current directory to parent directory
func copyCoverageDataUp() {
	coverDir := os.Getenv("GOCOVERDIR")
	if coverDir == "" {
		return
	}

	// Check if we're in integration coverage mode
	if os.Getenv("GO_INTEGRATION_COVERAGE") == "" {
		return
	}

	fmt.Printf("[MAIN] Copying coverage data up from %s\n", coverDir)

	// Find parent directory
	parentDir := filepath.Dir(coverDir)
	if parentDir == coverDir || parentDir == "/" {
		fmt.Printf("[MAIN] No parent directory to copy to\n")
		return
	}

	// Read coverage files from current directory
	entries, err := os.ReadDir(coverDir)
	if err != nil {
		fmt.Printf("[MAIN] Failed to read coverage directory: %v\n", err)
		return
	}

	copiedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, "cov") {
			srcPath := filepath.Join(coverDir, name)
			dstPath := filepath.Join(parentDir, name)

			if err := copyFile(srcPath, dstPath); err != nil {
				fmt.Printf("[MAIN] Failed to copy %s: %v\n", name, err)
			} else {
				copiedCount++
			}
		}
	}

	if copiedCount > 0 {
		fmt.Printf("[MAIN] Copied %d coverage files to %s\n", copiedCount, parentDir)
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = srcFile.WriteTo(dstFile)
	return err
}
