package covcompare

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func main() {
	os.Exit(Main())
}

func Main() int {
	var (
		legacyCov     = flag.String("legacy", "coverage/baselines/coverage-legacy.covtxt", "Legacy coverage file")
		scripttestCov = flag.String("scripttest", "coverage/scripttest/scriptest-merged.covtxt", "Scripttest coverage file")
		diffDir       = flag.String("diff-dir", "coverage/diff", "Directory to store diff results")
		help          = flag.Bool("help", false, "Show help")
	)

	flag.Parse()

	if *help {
		printUsage()
		return 0
	}

	return compareCoverage(*legacyCov, *scripttestCov, *diffDir)
}

func printUsage() {
	fmt.Println("Compare coverage between legacy and scripttest tests")
	fmt.Println("")
	fmt.Println("Usage: covcompare [options]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -legacy <file>       Legacy coverage file (default: coverage/baselines/coverage-legacy.covtxt)")
	fmt.Println("  -scripttest <file>   Scripttest coverage file (default: coverage/scripttest/scriptest-merged.covtxt)")
	fmt.Println("  -diff-dir <dir>      Directory to store diff results (default: coverage/diff)")
	fmt.Println("  -help               Show this help")
}

func compareCoverage(legacyCov, scripttestCov, diffDir string) int {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(diffDir, 0755); err != nil {
		fmt.Printf("Error creating diff directory: %v\n", err)
		return 1
	}

	// Check if coverage files exist, generate them if needed
	if _, err := os.Stat(legacyCov); os.IsNotExist(err) {
		fmt.Println("Legacy coverage file not found. Running coverage generation...")
		cmd := exec.Command("make", "-f", "Makefile.coverage.unified", "coverage-legacy")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error generating legacy coverage: %v\n", err)
			return 1
		}
	}

	if _, err := os.Stat(scripttestCov); os.IsNotExist(err) {
		fmt.Println("Scripttest coverage file not found. Running coverage generation...")
		cmd := exec.Command("make", "-f", "Makefile.coverage.unified", "coverage-scripttest")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error generating scripttest coverage: %v\n", err)
			return 1
		}
	}

	fmt.Println("Extracting function coverage information...")

	// Extract functions with coverage info from both files
	legacyFuncs := filepath.Join(diffDir, "legacy_funcs.txt")
	scripttestFuncs := filepath.Join(diffDir, "scripttest_funcs.txt")

	if err := extractFunctionCoverage(legacyCov, legacyFuncs); err != nil {
		fmt.Printf("Error extracting legacy coverage: %v\n", err)
		return 1
	}

	if err := extractFunctionCoverage(scripttestCov, scripttestFuncs); err != nil {
		fmt.Printf("Error extracting scripttest coverage: %v\n", err)
		return 1
	}

	// Create files with just fully covered functions from legacy tests (100%)
	legacyCovered := filepath.Join(diffDir, "legacy_covered.txt")
	if err := extractFullyCovered(legacyFuncs, legacyCovered); err != nil {
		fmt.Printf("Error extracting fully covered functions: %v\n", err)
		return 1
	}

	// Find functions covered in legacy but missing or not fully covered in scripttest
	fmt.Println("=== Functions covered in legacy but missing or partially covered in scripttest ===")

	// Method 1: Using diff to find lines in legacy_funcs.txt that aren't in scripttest_funcs.txt
	fmt.Println("1. Functions in legacy tests that don't appear in scripttest tests:")
	if err := findMissingFunctions(legacyFuncs, scripttestFuncs); err != nil {
		fmt.Printf("Error finding missing functions: %v\n", err)
		return 1
	}

	// Method 2: Compare coverage percentages
	fmt.Println("")
	fmt.Println("2. Functions with better coverage in legacy tests than scripttest tests:")
	if err := compareCoveragePercentages(legacyFuncs, scripttestFuncs); err != nil {
		fmt.Printf("Error comparing coverage percentages: %v\n", err)
		return 1
	}

	fmt.Printf("\nFull comparison results saved to %s\n", diffDir)
	return 0
}

func extractFunctionCoverage(coverageFile, outputFile string) error {
	cmd := exec.Command("go", "tool", "cover", "-func="+coverageFile)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("go tool cover failed: %v", err)
	}

	// Filter out "total:" line and sort
	lines := strings.Split(string(output), "\n")
	var filteredLines []string
	for _, line := range lines {
		if !strings.Contains(line, "total:") && strings.TrimSpace(line) != "" {
			filteredLines = append(filteredLines, line)
		}
	}

	sort.Strings(filteredLines)

	return writeLines(outputFile, filteredLines)
}

func extractFullyCovered(inputFile, outputFile string) error {
	file, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var fullyKoveredLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "100.0%") {
			fullyKoveredLines = append(fullyKoveredLines, line)
		}
	}

	return writeLines(outputFile, fullyKoveredLines)
}

func findMissingFunctions(legacyFile, scripttestFile string) error {
	// Read scripttest functions into a map for fast lookup
	scripttestFuncs, err := readLines(scripttestFile)
	if err != nil {
		return err
	}

	scripttestMap := make(map[string]bool)
	for _, line := range scripttestFuncs {
		scripttestMap[line] = true
	}

	// Read legacy functions and check which ones are missing
	legacyFuncs, err := readLines(legacyFile)
	if err != nil {
		return err
	}

	var missingFuncs []string
	for _, line := range legacyFuncs {
		if !scripttestMap[line] {
			missingFuncs = append(missingFuncs, line)
		}
	}

	sort.Strings(missingFuncs)
	for _, line := range missingFuncs {
		fmt.Println(line)
	}

	return nil
}

type FunctionCoverage struct {
	Function   string
	Percentage float64
	Line       string
}

func compareCoveragePercentages(legacyFile, scripttestFile string) error {
	// Parse legacy coverage
	legacyCov, err := parseFunctionCoverage(legacyFile)
	if err != nil {
		return err
	}

	// Parse scripttest coverage
	scripttestCov, err := parseFunctionCoverage(scripttestFile)
	if err != nil {
		return err
	}

	// Compare and find functions with better coverage in legacy
	var betterInLegacy []FunctionCoverage
	for key, legacyFunc := range legacyCov {
		if scripttestFunc, exists := scripttestCov[key]; exists {
			if legacyFunc.Percentage > scripttestFunc.Percentage {
				betterInLegacy = append(betterInLegacy, FunctionCoverage{
					Function:   key,
					Percentage: legacyFunc.Percentage - scripttestFunc.Percentage,
					Line:       fmt.Sprintf("%s %.1f%% %.1f%% -%.1f%%", key, legacyFunc.Percentage, scripttestFunc.Percentage, legacyFunc.Percentage-scripttestFunc.Percentage),
				})
			}
		}
	}

	// Sort by coverage difference (descending)
	sort.Slice(betterInLegacy, func(i, j int) bool {
		return betterInLegacy[i].Percentage > betterInLegacy[j].Percentage
	})

	// Show top 20
	for i, func_ := range betterInLegacy {
		if i >= 20 {
			break
		}
		fmt.Println(func_.Line)
	}

	return nil
}

func parseFunctionCoverage(filename string) (map[string]FunctionCoverage, error) {
	lines, err := readLines(filename)
	if err != nil {
		return nil, err
	}

	result := make(map[string]FunctionCoverage)
	re := regexp.MustCompile(`^(.+:\d+):\s+(\S+)\s+(\d+(?:\.\d+)?)%`)

	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 4 {
			key := matches[1] + ":" + matches[2]
			percentage, _ := strconv.ParseFloat(matches[3], 64)
			result[key] = FunctionCoverage{
				Function:   matches[2],
				Percentage: percentage,
				Line:       line,
			}
		}
	}

	return result, nil
}

func readLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

func writeLines(filename string, lines []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, line := range lines {
		fmt.Fprintln(file, line)
	}

	return nil
}
