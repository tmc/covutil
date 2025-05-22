package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

// cmd1 - First level command that can call cmd2 and cmd3
func main() {
	fmt.Printf("[CMD1] Starting cmd1 at %s\n", time.Now().Format(time.RFC3339))
	logCoverageInfo()

	// Coverage copying is now handled automatically by the overlay system

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "[CMD1] Usage: %s <operation> [args...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "[CMD1] Operations: greet, process, build, chain\n")
		os.Exit(1)
	}

	operation := os.Args[1]
	args := os.Args[2:]

	switch operation {
	case "greet":
		handleGreet(args)
	case "process":
		handleProcess(args)
	case "build":
		handleBuild(args)
	case "chain":
		handleChain(args)
	default:
		fmt.Fprintf(os.Stderr, "[CMD1] Unknown operation: %s\n", operation)
		os.Exit(1)
	}

	fmt.Printf("[CMD1] Completed operation '%s' at %s\n", operation, time.Now().Format(time.RFC3339))
}

// handleGreet provides greeting functionality
func handleGreet(args []string) {
	name := "Anonymous"
	if len(args) > 0 {
		name = args[0]
	}

	fmt.Printf("[CMD1] Greetings from cmd1 to %s!\n", name)

	// Call cmd2 for additional greeting processing
	if err := callCmd("cmd2", "elaborate", name); err != nil {
		fmt.Printf("[CMD1] Warning: cmd2 elaborate failed: %v\n", err)
	}
}

// handleProcess demonstrates processing workflow
func handleProcess(args []string) {
	step := "unknown"
	if len(args) > 0 {
		step = args[0]
	}

	fmt.Printf("[CMD1] Processing step: %s\n", step)

	// Perform step-specific work
	switch step {
	case "step1":
		fmt.Printf("[CMD1] Executing step1 logic\n")
		performStep1Work()
		// Chain to cmd2 for step2
		callCmd("cmd2", "process", "step2")
	case "validate":
		fmt.Printf("[CMD1] Validating input\n")
		performValidation()
	default:
		fmt.Printf("[CMD1] Generic processing for: %s\n", step)
	}
}

// handleBuild demonstrates build functionality
func handleBuild(args []string) {
	project := "default"
	if len(args) > 0 {
		project = args[0]
	}

	fmt.Printf("[CMD1] Building project: %s\n", project)

	// Simulate build steps
	steps := []string{"compile", "link", "package"}
	for i, buildStep := range steps {
		fmt.Printf("[CMD1] Build step %d: %s\n", i+1, buildStep)

		// Call cmd3 for packaging if we reach that step
		if buildStep == "package" {
			if err := callCmd("cmd3", "package", project); err != nil {
				fmt.Printf("[CMD1] Package step failed: %v\n", err)
			}
		}
	}
}

// handleChain demonstrates chaining to multiple commands
func handleChain(args []string) {
	fmt.Printf("[CMD1] Starting command chain from cmd1\n")

	// Call both cmd2 and cmd3 in parallel-ish fashion
	fmt.Printf("[CMD1] Calling cmd2 for analysis\n")
	if err := callCmd("cmd2", "analyze", "data"); err != nil {
		fmt.Printf("[CMD1] cmd2 analysis failed: %v\n", err)
	}

	fmt.Printf("[CMD1] Calling cmd3 for reporting\n")
	if err := callCmd("cmd3", "report", "results"); err != nil {
		fmt.Printf("[CMD1] cmd3 reporting failed: %v\n", err)
	}
}

// performStep1Work simulates work specific to step1
func performStep1Work() {
	fmt.Printf("[CMD1] Performing step1 specific work...\n")
	time.Sleep(10 * time.Millisecond) // Simulate some work
	fmt.Printf("[CMD1] Step1 work completed\n")
}

// performValidation simulates validation logic
func performValidation() {
	fmt.Printf("[CMD1] Running validation checks...\n")
	checks := []string{"syntax", "dependencies", "security"}

	for _, check := range checks {
		fmt.Printf("[CMD1] Validation check: %s - PASSED\n", check)
	}
}

// callCmd executes another binary in our curated PATH
func callCmd(cmdName string, args ...string) error {
	fmt.Printf("[CMD1] Calling %s with args: %v\n", cmdName, args)

	cmdPath, err := exec.LookPath(cmdName)
	if err != nil {
		return fmt.Errorf("command %s not found: %w", cmdName, err)
	}

	cmd := exec.Command(cmdPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ() // Pass through GOCOVERDIR and other env vars

	return cmd.Run()
}

// logCoverageInfo logs coverage-related information
func logCoverageInfo() {
	if coverDir := os.Getenv("GOCOVERDIR"); coverDir != "" {
		fmt.Printf("[CMD1] Coverage enabled, GOCOVERDIR=%s\n", coverDir)
	}
}
