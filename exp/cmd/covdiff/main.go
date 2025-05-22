package covdiff

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// CoverageLine represents a line of coverage data
type CoverageLine struct {
	File        string
	StartLine   int
	StartCol    int
	EndLine     int
	EndCol      int
	NumStmt     int // Number of statements
	Count       int // Execution count (0 = uncovered)
	IsUncovered bool
}

// CoverageDiff represents a difference in coverage between two profiles
type CoverageDiff struct {
	File       string
	StartLine  int
	EndLine    int
	NumStmt    int
	InProfile1 bool
	InProfile2 bool
}

func main() {
	os.Exit(Main())
}

func Main() int {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run coverage-diff.go <profile1> <profile2>")
		fmt.Println("Example: go run coverage-diff.go cover.legacy cover.new")
		return 1
	}

	profile1Path := os.Args[1]
	profile2Path := os.Args[2]

	// Generate the profiles if they don't exist
	if _, err := os.Stat(profile1Path); os.IsNotExist(err) {
		fmt.Println("Generating legacy test coverage profile...")
		cmd := exec.Command("go", "test", "-cover", "-coverprofile="+profile1Path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error generating legacy coverage profile: %v\n", err)
			return 1
		}
	}

	if _, err := os.Stat(profile2Path); os.IsNotExist(err) {
		fmt.Println("Generating scripttest coverage profile...")
		cmd := exec.Command("go", "test", "-tags=scripttest", "-cover", "-run=TestSprigScript", "-coverprofile="+profile2Path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error generating scripttest coverage profile: %v\n", err)
			// Continue anyway as we might have partial coverage
		}
	}

	// Parse the coverage profiles
	lines1, err := parseCoverageProfile(profile1Path)
	if err != nil {
		fmt.Printf("Error parsing profile 1: %v\n", err)
		return 1
	}

	lines2, err := parseCoverageProfile(profile2Path)
	if err != nil {
		fmt.Printf("Error parsing profile 2: %v\n", err)
		return 1
	}

	// Get the coverage stats
	uncoveredLines1 := filterUncovered(lines1)
	uncoveredLines2 := filterUncovered(lines2)

	totalStmts1 := sumStatements(lines1)
	totalStmts2 := sumStatements(lines2)
	uncoveredStmts1 := sumStatements(uncoveredLines1)
	uncoveredStmts2 := sumStatements(uncoveredLines2)
	coveredStmts1 := totalStmts1 - uncoveredStmts1
	coveredStmts2 := totalStmts2 - uncoveredStmts2

	coverage1 := float64(coveredStmts1) / float64(totalStmts1) * 100
	coverage2 := float64(coveredStmts2) / float64(totalStmts2) * 100

	fmt.Printf("\nCoverage Statistics:\n")
	fmt.Printf("  Profile 1 (%s): %.1f%% (%d of %d statements)\n", profile1Path, coverage1, coveredStmts1, totalStmts1)
	fmt.Printf("  Profile 2 (%s): %.1f%% (%d of %d statements)\n", profile2Path, coverage2, coveredStmts2, totalStmts2)

	fmt.Printf("\nCoverage Gap: %.1f%% (%d statements)\n", coverage1-coverage2, coveredStmts1-coveredStmts2)

	// Find differences between the two profiles
	missingCoverage := findMissingCoverage(lines1, lines2)

	// Group by file
	fileGroup := make(map[string][]CoverageDiff)
	for _, diff := range missingCoverage {
		fileGroup[diff.File] = append(fileGroup[diff.File], diff)
	}

	// Find most impactful files
	type fileImpact struct {
		File              string
		MissingStatements int
	}

	var impactList []fileImpact
	for file, diffs := range fileGroup {
		stmtCount := 0
		for _, diff := range diffs {
			stmtCount += diff.NumStmt
		}
		impactList = append(impactList, fileImpact{File: file, MissingStatements: stmtCount})
	}

	sort.Slice(impactList, func(i, j int) bool {
		return impactList[i].MissingStatements > impactList[j].MissingStatements
	})

	// Show top files to focus on
	fmt.Printf("\nTop files to focus on for coverage improvement:\n")
	for i, fi := range impactList {
		if i >= 10 || fi.MissingStatements == 0 {
			break
		}
		filename := filepath.Base(fi.File)
		fmt.Printf("  %d. %s: %d statements (%d%%)\n",
			i+1,
			filename,
			fi.MissingStatements,
			int(float64(fi.MissingStatements)/float64(totalStmts1)*100))
	}

	// Show top specific blocks to focus on
	fmt.Printf("\nTop 20 specific blocks to cover:\n")
	var allMissing []CoverageDiff
	for _, diffs := range fileGroup {
		allMissing = append(allMissing, diffs...)
	}

	sort.Slice(allMissing, func(i, j int) bool {
		return allMissing[i].NumStmt > allMissing[j].NumStmt
	})

	for i, diff := range allMissing {
		if i >= 20 || diff.NumStmt == 0 {
			break
		}
		filename := filepath.Base(diff.File)
		if diff.StartLine == diff.EndLine {
			fmt.Printf("  %s:%d - %d statements\n", filename, diff.StartLine, diff.NumStmt)
		} else {
			fmt.Printf("  %s:%d-%d - %d statements\n", filename, diff.StartLine, diff.EndLine, diff.NumStmt)
		}
	}
	return 0
}

func parseCoverageProfile(filePath string) ([]CoverageLine, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []CoverageLine
	scanner := bufio.NewScanner(file)

	// Skip the first line (mode line)
	if scanner.Scan() {
		// mode line is ignored
	}

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		file := parts[0]
		rest := parts[1]

		// Split by space
		spaceParts := strings.Split(rest, " ")
		if len(spaceParts) < 3 {
			continue
		}

		// Parse position (startline,startcol.endline,endcol)
		positions := strings.Split(spaceParts[0], ",")
		if len(positions) < 4 {
			continue
		}

		startLine, _ := strconv.Atoi(positions[0])
		startCol, _ := strconv.Atoi(positions[1])
		endLine, _ := strconv.Atoi(positions[2])
		endCol, _ := strconv.Atoi(positions[3])

		// Parse statements and count
		numStmt, _ := strconv.Atoi(spaceParts[1])
		count, _ := strconv.Atoi(spaceParts[2])

		covLine := CoverageLine{
			File:        file,
			StartLine:   startLine,
			StartCol:    startCol,
			EndLine:     endLine,
			EndCol:      endCol,
			NumStmt:     numStmt,
			Count:       count,
			IsUncovered: count == 0,
		}

		lines = append(lines, covLine)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func filterUncovered(lines []CoverageLine) []CoverageLine {
	var result []CoverageLine
	for _, line := range lines {
		if line.IsUncovered {
			result = append(result, line)
		}
	}
	return result
}

func sumStatements(lines []CoverageLine) int {
	total := 0
	for _, line := range lines {
		total += line.NumStmt
	}
	return total
}

// Find lines covered in profile1 but not in profile2
func findMissingCoverage(profile1, profile2 []CoverageLine) []CoverageDiff {
	// Create a map of all covered blocks in profile2
	covered2 := make(map[string]bool)
	for _, line := range profile2 {
		if !line.IsUncovered {
			key := fmt.Sprintf("%s:%d,%d.%d,%d", line.File, line.StartLine, line.StartCol, line.EndLine, line.EndCol)
			covered2[key] = true
		}
	}

	// Find blocks that are covered in profile1 but not in profile2
	var missing []CoverageDiff
	for _, line := range profile1 {
		key := fmt.Sprintf("%s:%d,%d.%d,%d", line.File, line.StartLine, line.StartCol, line.EndLine, line.EndCol)

		// If covered in profile1 but not in profile2, add to missing
		if !line.IsUncovered && !covered2[key] {
			missing = append(missing, CoverageDiff{
				File:      line.File,
				StartLine: line.StartLine,
				EndLine:   line.EndLine,
				NumStmt:   line.NumStmt,
			})
		}
	}

	return missing
}
