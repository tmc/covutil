// This tool takes a function name and outputs the source code with
// annotations for lines not covered by scripttest tests.
package covshow

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	os.Exit(Main())
}

func Main() int {
	// Parse command line flags
	functionName := flag.String("func", "", "Function name to analyze")
	flag.Parse()

	if *functionName == "" {
		fmt.Println("Usage: go run show-uncovered-lines.go -func=functionName")
		return 1
	}

	// Get project root directory
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error finding project root: %v\n", err)
		return 1
	}
	projectRoot := strings.TrimSpace(string(output))

	// Find the file containing the function
	fileInfo, err := findFunctionFile(projectRoot, *functionName)
	if err != nil {
		fmt.Printf("Error finding function: %v\n", err)
		return 1
	}

	fmt.Printf("Found function %s in file %s starting at line %d\n\n",
		*functionName, fileInfo.filePath, fileInfo.startLine)

	// Get uncovered lines from scripttest coverage
	uncoveredLines, err := getUncoveredLines(projectRoot, fileInfo.filePath)
	if err != nil {
		fmt.Printf("Error getting uncovered lines: %v\n", err)
		return 1
	}

	// Output the function with annotations for uncovered lines
	err = outputFunctionWithAnnotations(fileInfo, uncoveredLines)
	if err != nil {
		fmt.Printf("Error outputting function: %v\n", err)
		return 1
	}
	return 0
}

type functionInfo struct {
	filePath  string
	startLine int
	endLine   int
}

// findFunctionFile finds the file containing the specified function
func findFunctionFile(projectRoot, functionName string) (functionInfo, error) {
	info := functionInfo{}

	// Use grep to find the function definition
	cmd := exec.Command("grep", "-r", "--include=*.go",
		fmt.Sprintf("func.*%s(", functionName), projectRoot)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return info, fmt.Errorf("grep failed: %v\n%s", err, string(output))
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return info, fmt.Errorf("function %s not found", functionName)
	}

	// Parse the grep output to get the file path
	parts := strings.SplitN(lines[0], ":", 2)
	if len(parts) < 2 {
		return info, fmt.Errorf("unexpected grep output format: %s", lines[0])
	}

	info.filePath = parts[0]

	// Find the exact line number
	lineNumberCmd := exec.Command("grep", "-n",
		fmt.Sprintf("func.*%s(", functionName), info.filePath)
	lineOutput, err := lineNumberCmd.CombinedOutput()
	if err != nil {
		return info, fmt.Errorf("error finding line number: %v\n%s", err, string(lineOutput))
	}

	lineMatch := strings.SplitN(string(lineOutput), ":", 2)
	if len(lineMatch) < 2 {
		return info, fmt.Errorf("unexpected line number output: %s", string(lineOutput))
	}

	fmt.Sscanf(lineMatch[0], "%d", &info.startLine)

	// Find the end of the function by counting braces
	file, err := os.Open(info.filePath)
	if err != nil {
		return info, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	braceCount := 0
	foundStart := false

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		if lineCount == info.startLine {
			foundStart = true
		}

		if foundStart {
			// Count opening and closing braces
			for _, char := range line {
				if char == '{' {
					braceCount++
				} else if char == '}' {
					braceCount--
					if braceCount == 0 {
						info.endLine = lineCount
						return info, nil
					}
				}
			}
		}
	}

	// If we get here, we didn't find the end of the function
	return info, fmt.Errorf("couldn't determine function end")
}

// getUncoveredLines returns a map of uncovered line numbers in the file
func getUncoveredLines(projectRoot, filePath string) (map[int]bool, error) {
	// Run go tool cover to get uncovered lines
	coverageFile := filepath.Join(projectRoot, "coverage/scripttest/scriptest-merged.covtxt")

	// Check if the coverage file exists
	_, err := os.Stat(coverageFile)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("coverage file not found: %s", coverageFile)
	}

	// Use go tool cover to get uncovered blocks
	cmd := exec.Command("go", "tool", "cover", "-html", coverageFile)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error running go tool cover: %v", err)
	}

	uncoveredLines := make(map[int]bool)

	// Parse the HTML output to find uncovered lines for our file
	// This is a bit hacky since we're parsing HTML, but it works for simple cases
	relFilePath := strings.TrimPrefix(filePath, projectRoot)
	relFilePath = strings.TrimPrefix(relFilePath, "/")

	// Look for uncovered lines in the HTML
	htmlContent := string(output)
	filePattern := regexp.MustCompile(fmt.Sprintf(`<option value="file\d+">%s</option>`, regexp.QuoteMeta(relFilePath)))
	match := filePattern.FindStringSubmatch(htmlContent)

	if len(match) == 0 {
		return uncoveredLines, nil // File not found in coverage, all lines uncovered
	}

	// Extract the file ID
	fileIDPattern := regexp.MustCompile(`value="(file\d+)"`)
	fileIDMatch := fileIDPattern.FindStringSubmatch(match[0])
	if len(fileIDMatch) < 2 {
		return nil, fmt.Errorf("couldn't extract file ID from HTML")
	}
	fileID := fileIDMatch[1]

	// Find uncovered sections for this file
	uncoveredPattern := regexp.MustCompile(fmt.Sprintf(`<pre class="%s">(.*?)</pre>`, regexp.QuoteMeta(fileID)))
	uncoveredMatch := uncoveredPattern.FindStringSubmatch(htmlContent)

	if len(uncoveredMatch) < 2 {
		return uncoveredLines, nil
	}

	// Parse the line spans to extract uncovered line numbers
	lineSpanPattern := regexp.MustCompile(`<span class="cov0" title="0">(.*?)</span>`)
	lineSpans := lineSpanPattern.FindAllStringSubmatch(uncoveredMatch[1], -1)

	for _, lineSpan := range lineSpans {
		// Count how many newlines are in this span to determine line numbers
		lines := strings.Split(lineSpan[1], "\n")
		for i := range lines {
			// Note: This is a simplification and might not be perfectly accurate
			// for multi-line spans
			uncoveredLines[i+1] = true
		}
	}

	return uncoveredLines, nil
}

// outputFunctionWithAnnotations prints the function with annotations for uncovered lines
func outputFunctionWithAnnotations(info functionInfo, uncoveredLines map[int]bool) error {
	file, err := os.Open(info.filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	inFunction := false

	fmt.Println("--- Function with annotations for uncovered lines ---")
	for scanner.Scan() {
		lineCount++
		line := scanner.Text()

		if lineCount == info.startLine {
			inFunction = true
		}

		if inFunction {
			// Check if this line is uncovered
			if uncoveredLines[lineCount] {
				fmt.Printf("%4d: %s  // NOT COVERED IN SCRIPTTEST\n", lineCount, line)
			} else {
				fmt.Printf("%4d: %s\n", lineCount, line)
			}

			if lineCount == info.endLine {
				break
			}
		}
	}
	fmt.Println("--- End of function ---")

	return nil
}
