package covzero

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Options for the no-coverage detector
type Options struct {
	DeltaDir         string  // Directory containing coverage delta files
	SummaryFile      string  // File containing coverage contribution summary
	OutputFile       string  // Output file for the report
	ThresholdPercent float64 // Threshold percentage for minimal coverage
	Verbose          bool    // Enable verbose output
	SkipSummary      bool    // Skip using the contribution summary file
}

// TestCoverage represents the coverage information for a single test
type TestCoverage struct {
	Name            string  // The name of the test
	DeltaFile       string  // Path to the delta coverage file
	Contribution    float64 // Percentage contribution to overall coverage
	LinesCovered    int     // Number of lines covered
	TotalExecutable int     // Total executable lines in the test
}

func main() {
	os.Exit(Main())
}

func Main() int {
	// Parse command line flags
	deltaDir := flag.String("delta-dir", "coverage/delta", "Directory containing coverage delta files")
	summaryFile := flag.String("summary", "coverage/delta/contribution-summary.txt", "File containing coverage contribution summary")
	outputFile := flag.String("output", "coverage/no-coverage-tests.txt", "Output file for the report")
	thresholdPercent := flag.Float64("threshold", 0.1, "Threshold percentage for minimal coverage (default 0.1%)")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	skipSummary := flag.Bool("skip-summary", false, "Skip using the contribution summary file")

	flag.Parse()

	// Create options
	options := &Options{
		DeltaDir:         *deltaDir,
		SummaryFile:      *summaryFile,
		OutputFile:       *outputFile,
		ThresholdPercent: *thresholdPercent,
		Verbose:          *verbose,
		SkipSummary:      *skipSummary,
	}

	// Run the analysis
	if err := RunAnalysis(options); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

// RunAnalysis runs the analysis to find tests with no or minimal coverage
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

	// Analyze tests with no or minimal coverage
	noCoverageTests, minimalCoverageTests, err := analyzeTestCoverage(testCoverage, options.ThresholdPercent)
	if err != nil {
		return fmt.Errorf("failed to analyze test coverage: %w", err)
	}

	// Generate report
	if err := generateReport(noCoverageTests, minimalCoverageTests, options); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Print summary
	printSummary(noCoverageTests, minimalCoverageTests, options)

	return nil
}

// getTestCoverage retrieves coverage information for all tests
func getTestCoverage(options *Options) ([]TestCoverage, error) {
	var testCoverage []TestCoverage

	// Read contribution summary file if not skipped
	var contributionMap map[string]float64
	var err error
	if !options.SkipSummary {
		contributionMap, err = readContributionSummary(options.SummaryFile)
		if err != nil {
			if options.Verbose {
				fmt.Printf("Warning: Failed to read contribution summary: %v\n", err)
				fmt.Println("Proceeding without contribution data...")
			}
			contributionMap = make(map[string]float64)
		}
	} else {
		contributionMap = make(map[string]float64)
	}

	// Find all delta files
	deltaFiles, err := filepath.Glob(filepath.Join(options.DeltaDir, "*.delta.covtxt"))
	if err != nil {
		return nil, fmt.Errorf("failed to find delta files: %w", err)
	}

	// Process each delta file
	for _, deltaFile := range deltaFiles {
		testName := extractTestName(deltaFile)

		// Get coverage information from delta file
		covered, total, err := countCoveredLines(deltaFile)
		if err != nil {
			if options.Verbose {
				fmt.Printf("Warning: Failed to count lines for %s: %v\n", testName, err)
			}
			covered = 0
			total = 0
		}

		// Get contribution percentage
		contribution := contributionMap[testName]

		// Add to results
		testCoverage = append(testCoverage, TestCoverage{
			Name:            testName,
			DeltaFile:       deltaFile,
			Contribution:    contribution,
			LinesCovered:    covered,
			TotalExecutable: total,
		})
	}

	// Sort by contribution (ascending)
	sort.Slice(testCoverage, func(i, j int) bool {
		return testCoverage[i].Contribution < testCoverage[j].Contribution
	})

	return testCoverage, nil
}

// readContributionSummary reads the contribution summary file
func readContributionSummary(filePath string) (map[string]float64, error) {
	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("contribution summary file not found: %s", filePath)
	}

	// Read the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Parse each line
	contributionMap := make(map[string]float64)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Format is "TestSprigScript/file.txt: 1.23% additional coverage"
		parts := strings.Split(line, ": ")
		if len(parts) < 2 {
			continue
		}

		testName := strings.TrimPrefix(parts[0], "TestSprigScript/")

		// Extract percentage
		percentParts := strings.Split(parts[1], "%")
		if len(percentParts) < 1 {
			continue
		}

		percentage, err := strconv.ParseFloat(percentParts[0], 64)
		if err != nil {
			continue
		}

		contributionMap[testName] = percentage
	}

	return contributionMap, nil
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

// countCoveredLines counts the number of covered lines in a coverage file
func countCoveredLines(coverageFile string) (int, int, error) {
	// Read the file
	content, err := ioutil.ReadFile(coverageFile)
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(string(content), "\n")
	var totalLines, coveredLines int

	// Skip the first line (mode: atomic)
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}

		// Format: file:line.column,line.column numExecutions
		// Split by space to get the execution count
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		totalLines++

		// If execution count > 0, the line is covered
		count, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			continue
		}

		if count > 0 {
			coveredLines++
		}
	}

	return coveredLines, totalLines, nil
}

// analyzeTestCoverage analyzes the test coverage to find tests with no or minimal coverage
func analyzeTestCoverage(testCoverage []TestCoverage, threshold float64) ([]TestCoverage, []TestCoverage, error) {
	var noCoverageTests []TestCoverage
	var minimalCoverageTests []TestCoverage

	for _, tc := range testCoverage {
		if tc.Contribution == 0.0 {
			noCoverageTests = append(noCoverageTests, tc)
		} else if tc.Contribution > 0.0 && tc.Contribution < threshold {
			minimalCoverageTests = append(minimalCoverageTests, tc)
		}
	}

	return noCoverageTests, minimalCoverageTests, nil
}

// generateReport generates a report file with the analysis results
func generateReport(noCoverageTests, minimalCoverageTests []TestCoverage, options *Options) error {
	// Create output file
	file, err := os.Create(options.OutputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "Coverage Analysis Report\n")
	fmt.Fprintf(file, "======================\n\n")
	fmt.Fprintf(file, "Generated: %s\n\n", time.Now().Format(time.RFC3339))

	// Write tests with no coverage
	fmt.Fprintf(file, "Tests with NO coverage contribution (0%%):\n")
	fmt.Fprintf(file, "----------------------------------------\n")

	if len(noCoverageTests) == 0 {
		fmt.Fprintf(file, "None found\n")
	} else {
		for _, tc := range noCoverageTests {
			fmt.Fprintf(file, "- %s (lines covered: %d/%d)\n",
				tc.Name, tc.LinesCovered, tc.TotalExecutable)
		}
	}
	fmt.Fprintf(file, "\n")

	// Write tests with minimal coverage
	fmt.Fprintf(file, "Tests with MINIMAL coverage contribution (< %.2f%%):\n", options.ThresholdPercent)
	fmt.Fprintf(file, "-------------------------------------------------\n")

	if len(minimalCoverageTests) == 0 {
		fmt.Fprintf(file, "None found\n")
	} else {
		for _, tc := range minimalCoverageTests {
			fmt.Fprintf(file, "- %s: %.2f%% (lines covered: %d/%d)\n",
				tc.Name, tc.Contribution, tc.LinesCovered, tc.TotalExecutable)
		}
	}

	fmt.Printf("Report written to: %s\n", options.OutputFile)
	return nil
}

// printSummary prints a summary of the analysis results
func printSummary(noCoverageTests, minimalCoverageTests []TestCoverage, options *Options) {
	fmt.Println("\n=== Coverage Analysis Summary ===")

	fmt.Printf("Tests with NO coverage contribution: %d\n", len(noCoverageTests))
	fmt.Printf("Tests with MINIMAL coverage contribution (< %.2f%%): %d\n",
		options.ThresholdPercent, len(minimalCoverageTests))

	if len(noCoverageTests) > 0 {
		fmt.Println("\nTop 5 tests with NO coverage contribution that could potentially be removed:")
		count := 0
		for _, tc := range noCoverageTests {
			fmt.Printf("- %s\n", tc.Name)
			count++
			if count >= 5 {
				break
			}
		}
	}
}
