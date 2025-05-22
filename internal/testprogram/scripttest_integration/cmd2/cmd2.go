package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// cmd2 - Second level command that can call cmd3 and create recursive patterns
func main() {
	fmt.Printf("[CMD2] Starting cmd2 at %s\n", time.Now().Format(time.RFC3339))
	logCoverageInfo()

	// Coverage copying is now handled automatically by the overlay system

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "[CMD2] Usage: %s <operation> [args...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "[CMD2] Operations: elaborate, process, test, analyze, recursive\n")
		os.Exit(1)
	}

	operation := os.Args[1]
	args := os.Args[2:]

	switch operation {
	case "elaborate":
		handleElaborate(args)
	case "process":
		handleProcess(args)
	case "test":
		handleTest(args)
	case "analyze":
		handleAnalyze(args)
	case "recursive":
		handleRecursive(args)
	default:
		fmt.Fprintf(os.Stderr, "[CMD2] Unknown operation: %s\n", operation)
		os.Exit(1)
	}

	fmt.Printf("[CMD2] Completed operation '%s' at %s\n", operation, time.Now().Format(time.RFC3339))
}

// handleElaborate provides elaborate greeting processing
func handleElaborate(args []string) {
	name := "Someone"
	if len(args) > 0 {
		name = args[0]
	}

	fmt.Printf("[CMD2] Elaborating greeting for %s\n", name)

	// Generate elaborate greeting
	greetings := []string{
		fmt.Sprintf("Welcome, %s!", name),
		fmt.Sprintf("It's great to see you, %s!", name),
		fmt.Sprintf("Hope you're having a wonderful day, %s!", name),
	}

	for i, greeting := range greetings {
		fmt.Printf("[CMD2] Greeting %d: %s\n", i+1, greeting)
	}

	// Call cmd3 for final flourish
	if err := callCmd("cmd3", "flourish", name); err != nil {
		fmt.Printf("[CMD2] Warning: cmd3 flourish failed: %v\n", err)
	}
}

// handleProcess handles processing workflow
func handleProcess(args []string) {
	step := "unknown"
	if len(args) > 0 {
		step = args[0]
	}

	fmt.Printf("[CMD2] Processing step: %s\n", step)

	switch step {
	case "step2":
		fmt.Printf("[CMD2] Executing step2 logic\n")
		performStep2Work()
		// Chain to cmd3 for step3
		if err := callCmd("cmd3", "process", "step3"); err != nil {
			fmt.Printf("[CMD2] cmd3 step3 failed: %v\n", err)
		}
	case "transform":
		fmt.Printf("[CMD2] Transforming data\n")
		performTransformation()
	default:
		fmt.Printf("[CMD2] Generic processing for: %s\n", step)
	}
}

// handleTest demonstrates testing functionality
func handleTest(args []string) {
	suite := "default"
	if len(args) > 0 {
		suite = args[0]
	}

	fmt.Printf("[CMD2] Running test suite: %s\n", suite)

	// Simulate test execution
	testCases := []string{"unit", "integration", "e2e"}
	for i, testCase := range testCases {
		fmt.Printf("[CMD2] Running %s tests...\n", testCase)

		// Call cmd3 for test reporting
		if testCase == "e2e" {
			if err := callCmd("cmd3", "test-report", testCase); err != nil {
				fmt.Printf("[CMD2] Test reporting failed: %v\n", err)
			}
		}

		fmt.Printf("[CMD2] %s tests completed (%d/%d)\n", testCase, i+1, len(testCases))
	}
}

// handleAnalyze performs data analysis
func handleAnalyze(args []string) {
	dataType := "unknown"
	if len(args) > 0 {
		dataType = args[0]
	}

	fmt.Printf("[CMD2] Analyzing data type: %s\n", dataType)

	// Perform analysis steps
	analysisSteps := []string{"collect", "process", "correlate", "summarize"}
	for _, step := range analysisSteps {
		fmt.Printf("[CMD2] Analysis step: %s\n", step)

		// Call cmd3 for summarization
		if step == "summarize" {
			if err := callCmd("cmd3", "summarize", dataType); err != nil {
				fmt.Printf("[CMD2] Summarization failed: %v\n", err)
			}
		}
	}
}

// handleRecursive handles recursive calls
func handleRecursive(args []string) {
	depth := 1
	if len(args) > 0 {
		if d, err := strconv.Atoi(args[0]); err == nil && d > 0 {
			depth = d
		}
	}

	fmt.Printf("[CMD2] Recursive processing with depth %d\n", depth)

	if depth > 1 {
		// Create recursive pattern by calling cmd3 which might call back to cmd1
		if err := callCmd("cmd3", "recursive", strconv.Itoa(depth-1)); err != nil {
			fmt.Printf("[CMD2] Recursive call failed: %v\n", err)
		}
	} else {
		fmt.Printf("[CMD2] Reached recursion base case in cmd2\n")
		// Final call to cmd3 for cleanup
		callCmd("cmd3", "cleanup", "recursive")
	}
}

// performStep2Work simulates work specific to step2
func performStep2Work() {
	fmt.Printf("[CMD2] Performing step2 specific work...\n")

	// Simulate some processing
	operations := []string{"validate", "transform", "optimize"}
	for _, op := range operations {
		fmt.Printf("[CMD2] Step2 operation: %s\n", op)
		time.Sleep(5 * time.Millisecond) // Simulate work
	}

	fmt.Printf("[CMD2] Step2 work completed\n")
}

// performTransformation simulates data transformation
func performTransformation() {
	fmt.Printf("[CMD2] Transforming data...\n")

	transformations := []string{"normalize", "filter", "aggregate"}
	for i, transform := range transformations {
		fmt.Printf("[CMD2] Transformation %d: %s\n", i+1, transform)
	}

	fmt.Printf("[CMD2] Data transformation completed\n")
}

// callCmd executes another binary in our curated PATH
func callCmd(cmdName string, args ...string) error {
	fmt.Printf("[CMD2] Calling %s with args: %v\n", cmdName, args)

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
		fmt.Printf("[CMD2] Coverage enabled, GOCOVERDIR=%s\n", coverDir)
	}
}
