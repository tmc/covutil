// Package testprogram provides a unified entry point for various coverage testing utilities.
// It consolidates different testing and demonstration functionalities into a single program.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) > 1 {
		// Command line mode
		handleCommand(os.Args[1])
		return
	}

	// Interactive mode
	showMenu()
	
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\nEnter your choice (1-9, or 'q' to quit): ")
		if !scanner.Scan() {
			break
		}
		
		input := strings.TrimSpace(scanner.Text())
		if input == "q" || input == "quit" {
			fmt.Println("Goodbye!")
			break
		}
		
		choice, err := strconv.Atoi(input)
		if err != nil {
			fmt.Printf("Invalid input: %s. Please enter a number 1-9 or 'q' to quit.\n", input)
			continue
		}
		
		handleChoice(choice)
	}
}

func showMenu() {
	fmt.Println("=== Coverage Testing Program ===")
	fmt.Println("1. Run high-level API tests")
	fmt.Println("2. Run simple API tests")
	fmt.Println("3. Run synthetic coverage demo")
	fmt.Println("4. Run JSON functionality demo")
	fmt.Println("5. Run coverage verification tests")
	fmt.Println("6. Parse coverage from directory")
	fmt.Println("7. Generate format analysis")
	fmt.Println("8. Run simple test main")
	fmt.Println("9. Show this menu again")
	fmt.Println("q. Quit")
}

func handleCommand(command string) {
	switch command {
	case "high-level", "highlevel", "1":
		runHighLevelAPITests()
	case "simple", "simple-api", "2":
		runSimpleAPITests()
	case "synthetic", "synthetic-demo", "3":
		runSyntheticDemo()
	case "json", "json-demo", "4":
		runJSONDemo()
	case "verify", "verification", "5":
		runCoverageVerification()
	case "parse", "parse-dir", "6":
		runDirectoryParsing()
	case "format", "format-analysis", "7":
		runFormatAnalysis()
	case "test-main", "8":
		runSimpleTestMain()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available commands: high-level, simple, synthetic, json, verify, parse, format, test-main")
	}
}

func handleChoice(choice int) {
	switch choice {
	case 1:
		runHighLevelAPITests()
	case 2:
		runSimpleAPITests()
	case 3:
		runSyntheticDemo()
	case 4:
		runJSONDemo()
	case 5:
		runCoverageVerification()
	case 6:
		runDirectoryParsing()
	case 7:
		runFormatAnalysis()
	case 8:
		runSimpleTestMain()
	case 9:
		showMenu()
	default:
		fmt.Printf("Invalid choice: %d. Please enter a number 1-9.\n", choice)
	}
}

// Placeholder functions that would call into the appropriate functionality
func runHighLevelAPITests() {
	fmt.Println("\n=== Running High-Level API Tests ===")
	fmt.Println("This would run the high-level API tests.")
	fmt.Println("To run this functionality, use: go run cmd/test-apis/test_highlevel_api.go")
}

func runSimpleAPITests() {
	fmt.Println("\n=== Running Simple API Tests ===")
	fmt.Println("This would run the simple API tests.")
	fmt.Println("To run this functionality, use: go run cmd/test-apis/test_simple_api.go")
}

func runSyntheticDemo() {
	fmt.Println("\n=== Running Synthetic Coverage Demo ===")
	fmt.Println("This would run the synthetic coverage generation demo.")
	fmt.Println("To run this functionality, use: go run cmd/synthetic-demo/synthetic_proper.go")
}

func runJSONDemo() {
	fmt.Println("\n=== Running JSON Functionality Demo ===")
	fmt.Println("This would demonstrate JSON conversion and comparison features.")
	demonstrateJsonFunctionality()
}

func runCoverageVerification() {
	fmt.Println("\n=== Running Coverage Verification Tests ===")
	fmt.Println("This would run coverage verification tests.")
	fmt.Println("Note: This functionality is available in the test files (*_test.go)")
}

func runDirectoryParsing() {
	fmt.Println("\n=== Running Directory Parsing ===")
	fmt.Println("This would parse coverage data from a directory.")
	fmt.Println("Please specify a directory path:")
	
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		dir := strings.TrimSpace(scanner.Text())
		if dir != "" {
			parseCoverageDirectory(dir)
		}
	}
}

func runFormatAnalysis() {
	fmt.Println("\n=== Running Format Analysis ===")
	fmt.Println("This would analyze coverage file formats.")
	analyzeFormats()
}

func runSimpleTestMain() {
	fmt.Println("\n=== Running Simple Test Main ===")
	fmt.Println("This would run the simple test main functionality.")
	simpleTestMain()
}