package covdup

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Options for the redundant test detector
type Options struct {
	DeltaDir   string // Directory containing coverage delta files
	OutputFile string // Output file for the report
	Verbose    bool   // Enable verbose output
}

// TestCoverage represents the coverage information for a single test
type TestCoverage struct {
	Name         string           // The name of the test
	DeltaFile    string           // Path to the delta coverage file
	CoveredLines map[string]bool  // Set of covered lines (file:line format)
	FileCoverage map[string][]int // Map of file to covered lines
}

// RedundancyResult represents the result of redundancy analysis for a test
type RedundancyResult struct {
	Test          string   // Test name
	RedundantWith []string // Tests that cover all lines this test covers
	UniqueLines   int      // Number of lines uniquely covered by this test
	TotalLines    int      // Total lines covered by this test
}

func main() {
	os.Exit(Main())
}

func Main() int {
	// Parse command line flags
	deltaDir := flag.String("delta-dir", "coverage/delta", "Directory containing coverage delta files")
	outputFile := flag.String("output", "coverage/redundant-tests.txt", "Output file for the report")
	verbose := flag.Bool("verbose", false, "Enable verbose output")

	flag.Parse()

	// Create options
	options := &Options{
		DeltaDir:   *deltaDir,
		OutputFile: *outputFile,
		Verbose:    *verbose,
	}

	// Run the analysis
	if err := RunAnalysis(options); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

// RunAnalysis runs the analysis to find redundant tests
func RunAnalysis(options *Options) error {
	// Ensure output directory exists
	outputDir := filepath.Dir(options.OutputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get coverage information for all tests
	testCoverage, err := getTestCoverage(options)
	if err != nil {
		return fmt.Errorf("failed to get test coverage: %w", err)
	}

	// Find redundant tests
	redundancyResults, err := findRedundantTests(testCoverage, options)
	if err != nil {
		return fmt.Errorf("failed to find redundant tests: %w", err)
	}

	// Generate report
	if err := generateReport(redundancyResults, options); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Print summary
	printSummary(redundancyResults, options)

	return nil
}

// getTestCoverage retrieves coverage information for all tests
func getTestCoverage(options *Options) (map[string]TestCoverage, error) {
	testCoverage := make(map[string]TestCoverage)

	// Find all delta files
	deltaFiles, err := filepath.Glob(filepath.Join(options.DeltaDir, "*.delta.covtxt"))
	if err != nil {
		return nil, fmt.Errorf("failed to find delta files: %w", err)
	}

	fmt.Printf("Processing %d test coverage files...\n", len(deltaFiles))

	// Process each delta file
	for _, deltaFile := range deltaFiles {
		testName := extractTestName(deltaFile)

		// Parse the coverage file to get covered lines
		coveredLines, fileCoverage, err := parseCoverageFile(deltaFile)
		if err != nil {
			if options.Verbose {
				fmt.Printf("Warning: Failed to parse coverage for %s: %v\n", testName, err)
			}
			continue
		}

		// Add to results
		testCoverage[testName] = TestCoverage{
			Name:         testName,
			DeltaFile:    deltaFile,
			CoveredLines: coveredLines,
			FileCoverage: fileCoverage,
		}
	}

	return testCoverage, nil
}

// extractTestName extracts the test name from a delta file path
func extractTestName(filePath string) string {
	// Format: "coverage/delta/TestSprigScript-file.txt.delta.covtxt"
	baseName := filepath.Base(filePath)
	// Remove "TestSprigScript-" prefix and ".delta.covtxt" suffix
	name := strings.TrimPrefix(baseName, "TestSprigScript-")
	name = strings.TrimSuffix(name, ".delta.covtxt")
	return name
}

// parseCoverageFile parses a coverage file to get covered lines
func parseCoverageFile(coverageFile string) (map[string]bool, map[string][]int, error) {
	// Read the file
	content, err := ioutil.ReadFile(coverageFile)
	if err != nil {
		return nil, nil, err
	}

	lines := strings.Split(string(content), "\n")
	coveredLines := make(map[string]bool)
	fileCoverage := make(map[string][]int)

	// Skip the first line (mode: atomic)
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}

		// Format: github.com/Masterminds/sprig/v3/file.go:12.34,56.78 count
		parts := strings.Split(line, " ")
		if len(parts) < 2 {
			continue
		}

		// Get file and line information
		fileLineInfo := parts[0]
		count := parts[1]

		// Skip if not covered
		if count == "0" {
			continue
		}

		// Extract file path and line ranges
		colonIdx := strings.LastIndex(fileLineInfo, ":")
		if colonIdx == -1 {
			continue
		}

		filePath := fileLineInfo[:colonIdx]
		lineRangeInfo := fileLineInfo[colonIdx+1:]

		// Parse line ranges (format: startLine.startCol,endLine.endCol)
		rangeParts := strings.Split(lineRangeInfo, ",")
		if len(rangeParts) < 2 {
			continue
		}

		// Get start and end line numbers
		startLineParts := strings.Split(rangeParts[0], ".")
		endLineParts := strings.Split(rangeParts[1], ".")

		if len(startLineParts) < 1 || len(endLineParts) < 1 {
			continue
		}

		startLine := startLineParts[0]
		endLine := endLineParts[0]

		// Create the file:line key
		key := fmt.Sprintf("%s:%s-%s", filePath, startLine, endLine)
		coveredLines[key] = true

		// Add to file coverage map
		if _, ok := fileCoverage[filePath]; !ok {
			fileCoverage[filePath] = []int{}
		}

		// Convert line numbers to integers
		start, err := parseLineNumber(startLine)
		if err != nil {
			continue
		}

		end, err := parseLineNumber(endLine)
		if err != nil {
			continue
		}

		// Add all lines in the range
		for i := start; i <= end; i++ {
			fileCoverage[filePath] = append(fileCoverage[filePath], i)
		}
	}

	return coveredLines, fileCoverage, nil
}

// parseLineNumber converts a string line number to an integer
func parseLineNumber(lineStr string) (int, error) {
	var lineNum int
	_, err := fmt.Sscanf(lineStr, "%d", &lineNum)
	return lineNum, err
}

// findRedundantTests finds tests that are redundant (all their coverage is contained in other tests)
func findRedundantTests(testCoverage map[string]TestCoverage, options *Options) ([]RedundancyResult, error) {
	var results []RedundancyResult

	// For each test, check if its covered lines are a subset of other tests' covered lines
	for testName, testInfo := range testCoverage {
		result := RedundancyResult{
			Test:       testName,
			TotalLines: len(testInfo.CoveredLines),
		}

		// Skip tests with no coverage
		if len(testInfo.CoveredLines) == 0 {
			continue
		}

		// Check against all other tests
		uniqueLines := make(map[string]bool)
		for line := range testInfo.CoveredLines {
			isUnique := true

			// Check if any other test covers this line
			for otherTestName, otherTestInfo := range testCoverage {
				if otherTestName == testName {
					continue
				}

				if otherTestInfo.CoveredLines[line] {
					isUnique = false
					break
				}
			}

			if isUnique {
				uniqueLines[line] = true
			}
		}

		result.UniqueLines = len(uniqueLines)

		// Find tests that contain all of this test's coverage
		for otherTestName, otherTestInfo := range testCoverage {
			if otherTestName == testName {
				continue
			}

			// Check if this test's coverage is a subset of the other test
			isSubset := true
			for line := range testInfo.CoveredLines {
				if !otherTestInfo.CoveredLines[line] {
					isSubset = false
					break
				}
			}

			if isSubset {
				result.RedundantWith = append(result.RedundantWith, otherTestName)
			}
		}

		// Add to results
		results = append(results, result)
	}

	// Sort by number of redundant tests (descending)
	sort.Slice(results, func(i, j int) bool {
		return len(results[i].RedundantWith) > len(results[j].RedundantWith)
	})

	return results, nil
}

// generateReport generates a report file with the analysis results
func generateReport(results []RedundancyResult, options *Options) error {
	// Create output file
	file, err := os.Create(options.OutputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "Redundant Test Analysis Report\n")
	fmt.Fprintf(file, "============================\n\n")
	fmt.Fprintf(file, "Generated: %s\n\n", time.Now().Format(time.RFC3339))

	// Write fully redundant tests
	fmt.Fprintf(file, "FULLY REDUNDANT TESTS (covered completely by at least one other test):\n")
	fmt.Fprintf(file, "----------------------------------------------------------------\n")

	fullyRedundantFound := false
	for _, result := range results {
		if len(result.RedundantWith) > 0 {
			fmt.Fprintf(file, "- %s: covered by %d other tests (%d covered lines)\n",
				result.Test, len(result.RedundantWith), result.TotalLines)
			fmt.Fprintf(file, "  Redundant with: %s\n\n", strings.Join(result.RedundantWith, ", "))
			fullyRedundantFound = true
		}
	}

	if !fullyRedundantFound {
		fmt.Fprintf(file, "None found\n\n")
	}

	// Write tests with unique coverage
	fmt.Fprintf(file, "TESTS WITH UNIQUE COVERAGE (provide coverage that no other test provides):\n")
	fmt.Fprintf(file, "----------------------------------------------------------------\n")

	// Sort by number of unique lines (descending)
	uniqueSorted := make([]RedundancyResult, len(results))
	copy(uniqueSorted, results)
	sort.Slice(uniqueSorted, func(i, j int) bool {
		return uniqueSorted[i].UniqueLines > uniqueSorted[j].UniqueLines
	})

	uniqueFound := false
	for _, result := range uniqueSorted {
		if result.UniqueLines > 0 {
			fmt.Fprintf(file, "- %s: %d unique lines (%.1f%% of its total coverage)\n",
				result.Test, result.UniqueLines, float64(result.UniqueLines)/float64(result.TotalLines)*100)
			uniqueFound = true
		}
	}

	if !uniqueFound {
		fmt.Fprintf(file, "None found\n")
	}

	fmt.Printf("Report written to: %s\n", options.OutputFile)
	return nil
}

// printSummary prints a summary of the analysis results
func printSummary(results []RedundancyResult, options *Options) {
	fmt.Println("\n=== Redundant Test Analysis Summary ===")

	// Count fully redundant tests
	fullyRedundantCount := 0
	for _, result := range results {
		if len(result.RedundantWith) > 0 {
			fullyRedundantCount++
		}
	}

	// Count tests with unique coverage
	uniqueCount := 0
	for _, result := range results {
		if result.UniqueLines > 0 {
			uniqueCount++
		}
	}

	fmt.Printf("Total tests analyzed: %d\n", len(results))
	fmt.Printf("Tests that are fully redundant: %d\n", fullyRedundantCount)
	fmt.Printf("Tests with unique coverage: %d\n", uniqueCount)

	if fullyRedundantCount > 0 {
		fmt.Println("\nTop 5 redundant tests that could potentially be removed:")
		count := 0
		for _, result := range results {
			if len(result.RedundantWith) > 0 {
				fmt.Printf("- %s (covered by %d other tests)\n",
					result.Test, len(result.RedundantWith))
				count++
				if count >= 5 {
					break
				}
			}
		}
	}
}
