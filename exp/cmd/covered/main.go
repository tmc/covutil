// Command covered analyzes Go coverage profiles and outputs colorized source code
// with coverage information, similar to 'go tool cover -html' but for terminals.
package covered

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/term"
	"golang.org/x/tools/cover"
)

var (
	coverFile    = flag.String("c", "cover.out", "coverage profile to read")
	pathRegexp   = flag.String("path", "", "regular expression to filter file paths")
	contextLines = flag.Int("C", 2, "number of context lines to show around covered/uncovered blocks")
	sortByLeast  = flag.Bool("t", false, "sort files by least covered (ascending coverage percentage)")
	colorMode    = flag.String("color", "auto", "colorize output: auto, always, never")
	showAll      = flag.Bool("a", false, "show all uncovered lines (including those marked as unreachable/untested)")
	barMode      = flag.Bool("bar", false, "colorize as background bars instead of text")

	// Global color state
	useColor bool
)

// ANSI color codes
const (
	ColorReset    = "\033[0m"
	ColorRed      = "\033[31m"
	ColorGreen    = "\033[32m"
	ColorYellow   = "\033[33m"
	ColorBlue     = "\033[34m"
	ColorCyan     = "\033[36m"
	ColorGray     = "\033[90m"
	ColorDarkGray = "\033[38;5;8m"
	
	// Subtle green gradient for atomic mode (better contrast)
	ColorGreen1   = "\033[38;5;28m"  // Dark green (low coverage)
	ColorGreen2   = "\033[38;5;34m"  // Medium-dark green
	ColorGreen3   = "\033[38;5;40m"  // Medium green  
	ColorGreen4   = "\033[38;5;46m"  // Medium-bright green
	ColorGreen5   = "\033[38;5;82m"  // Bright green (high coverage)
	
	// Background colors for bar mode with proper text contrast
	ColorRedBg    = "\033[41;37m"       // Red background with white text
	ColorGreenBg1 = "\033[48;5;28;37m"  // Dark green background with white text
	ColorGreenBg2 = "\033[48;5;34;37m"  // Medium-dark green background with white text
	ColorGreenBg3 = "\033[48;5;40;30m"  // Medium green background with black text
	ColorGreenBg4 = "\033[48;5;46;30m"  // Medium-bright green background with black text  
	ColorGreenBg5 = "\033[48;5;82;30m"  // Bright green background with black text
)

// Non-color indicators for when colors are disabled
const (
	IndicatorCovered   = "✓"
	IndicatorUncovered = "✗"
	IndicatorEllipsis  = "⋯"
)

// Skip patterns for lines that are acceptable to be uncovered
var skipPatterns = []string{
	"// unreachable",
	"// untested",
	"// Unreachable",
	"// Untested",
}

// Pkg describes a single package, compatible with the JSON output from 'go list'; see 'go help list'.
type Pkg struct {
	ImportPath string
	Dir        string
	Error      *struct {
		Err string
	}
}

// osExit allows tests to replace the normal exit function
var osExit = os.Exit

func usage() {
	fmt.Fprintf(os.Stderr, "usage: covered [flags] [coverage.out]\n")
	fmt.Fprintf(os.Stderr, "\nFlags:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nOutputs program source code with coverage colorization.\n")
	fmt.Fprintf(os.Stderr, "Covered lines are shown in green, uncovered lines in red.\n")
	fmt.Fprintf(os.Stderr, "\nIf a coverage file is provided as an argument, it takes precedence over the -c flag.\n")
	osExit(2)
}

func main() {
	osExit(Main())
}

func Main() int {
	flag.Usage = usage
	flag.Parse()

	// Initialize color mode
	initColorMode()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "covered: %v\n", err)
		return 1
	}
	return 0
}

// initColorMode sets the global useColor variable based on the color flag and TTY detection
func initColorMode() {
	switch *colorMode {
	case "always":
		useColor = true
	case "never":
		useColor = false
	case "auto":
		// Check if stdout is a TTY
		useColor = term.IsTerminal(int(os.Stdout.Fd()))
	default:
		fmt.Fprintf(os.Stderr, "covered: invalid color mode %q, must be auto, always, or never\n", *colorMode)
		osExit(2)
	}
}

func run() error {
	// Determine coverage file: use positional argument if provided, otherwise use flag
	coverageFile := *coverFile
	if flag.NArg() > 0 {
		coverageFile = flag.Arg(0)
	}

	// Read coverage profile
	profiles, err := cover.ParseProfiles(coverageFile)
	if err != nil {
		return fmt.Errorf("parsing coverage profile: %v", err)
	}

	// Find package directories for file resolution
	dirs, err := findPkgs(profiles)
	if err != nil {
		return fmt.Errorf("finding packages: %v", err)
	}

	// Filter profiles by path regexp if provided
	if *pathRegexp != "" {
		re, err := regexp.Compile(*pathRegexp)
		if err != nil {
			return fmt.Errorf("invalid path regexp: %v", err)
		}
		filtered := make([]*cover.Profile, 0)
		for _, profile := range profiles {
			if re.MatchString(profile.FileName) {
				filtered = append(filtered, profile)
			}
		}
		profiles = filtered
	}

	if len(profiles) == 0 {
		return fmt.Errorf("no matching profiles found")
	}

	// Calculate file coverage percentages for sorting
	var fileInfos []FileInfo
	for _, profile := range profiles {
		coverage := calculateFileCoverage(profile)
		fileInfos = append(fileInfos, FileInfo{
			Profile:  profile,
			Coverage: coverage,
		})
	}

	// Sort by coverage (least covered first by default, can be overridden with -t)
	if *sortByLeast {
		// Explicit request for least covered first
		sort.Slice(fileInfos, func(i, j int) bool {
			return fileInfos[i].Coverage < fileInfos[j].Coverage
		})
	} else {
		// Default: still sort by least covered, but secondary sort by filename for stability
		sort.Slice(fileInfos, func(i, j int) bool {
			if fileInfos[i].Coverage == fileInfos[j].Coverage {
				return fileInfos[i].Profile.FileName < fileInfos[j].Profile.FileName
			}
			return fileInfos[i].Coverage < fileInfos[j].Coverage
		})
	}

	// Process and output each file
	for i, info := range fileInfos {
		if i > 0 {
			fmt.Println()
		}
		if err := processFile(info.Profile, info.Coverage, dirs); err != nil {
			return fmt.Errorf("processing file %s: %v", info.Profile.FileName, err)
		}
	}

	return nil
}

// FileInfo holds profile and coverage information for a file
type FileInfo struct {
	Profile  *cover.Profile
	Coverage float64
}

// calculateFileCoverage calculates the overall coverage percentage for a file
func calculateFileCoverage(profile *cover.Profile) float64 {
	if len(profile.Blocks) == 0 {
		return 0.0
	}

	totalStatements := 0
	coveredStatements := 0

	for _, block := range profile.Blocks {
		totalStatements += block.NumStmt
		if block.Count > 0 {
			coveredStatements += block.NumStmt
		}
	}

	if totalStatements == 0 {
		return 0.0
	}

	return float64(coveredStatements) / float64(totalStatements) * 100.0
}

// processFile reads and outputs a file with coverage colorization
func processFile(profile *cover.Profile, coverage float64, dirs map[string]*Pkg) error {
	// Find the actual file path with fallback
	file, err := findFileWithFallback(dirs, profile.FileName)
	if err != nil {
		return fmt.Errorf("finding file: %v", err)
	}

	// Read the source file
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading source file: %v", err)
	}

	// Output txtar-like header
	outputHeader(profile.FileName, coverage)

	// Create coverage map
	coverageMap := createCoverageMap(profile, content)

	// Get import ranges for colorization
	importRanges := getImportRanges(content)

	// Output the file with colorization
	return outputFileWithCoverage(content, coverageMap, importRanges)
}

// outputHeader outputs a txtar-like header for the file
func outputHeader(filename string, coverage float64) {
	// Make filename relative if possible
	relPath, err := filepath.Rel(".", filename)
	if err == nil && !strings.HasPrefix(relPath, "..") {
		filename = relPath
	}

	header := fmt.Sprintf("-- %s", filename)
	if coverage >= 0 {
		header += fmt.Sprintf(" (%.1f%% covered)", coverage)
	}
	header += " --"

	if useColor {
		fmt.Printf("%s%s%s\n", ColorCyan, header, ColorReset)
	} else {
		fmt.Printf("%s\n", header)
	}
}

// LineCoverage represents the coverage status of a line
type LineCoverage struct {
	Covered  bool
	Count    int
	Partial  bool // true if only part of the line is covered
	StartCol int  // start column of coverage (0-based)
	EndCol   int  // end column of coverage (0-based)
}

// createCoverageMap creates a map of line numbers to coverage information
func createCoverageMap(profile *cover.Profile, content []byte) map[int]*LineCoverage {
	coverageMap := make(map[int]*LineCoverage)

	// Get import ranges using Go parser for accuracy
	importRanges := getImportRanges(content)

	// Initialize lines - but only initialize "important" lines for coverage consideration
	lines := bytes.Split(content, []byte("\n"))
	for i := range lines {
		lineNum := i + 1
		lineContent := string(lines[i])
		
		// Check if this line is in an import block
		inImportBlock := false
		for _, importRange := range importRanges {
			if lineNum >= importRange.Start && lineNum <= importRange.End {
				inImportBlock = true
				break
			}
		}
		
		// Only track lines that are actually relevant for coverage
		// Unimportant lines (imports, comments, etc.) shouldn't be marked as uncovered
		if !isUnimportantLine(lineContent) && !inImportBlock {
			coverageMap[lineNum] = &LineCoverage{
				Covered:  false,
				Count:    0,
				Partial:  false,
				StartCol: 0,
				EndCol:   len(lines[i]),
			}
		} else {
			// Mark unimportant lines as "covered" so they don't show as uncovered
			coverageMap[lineNum] = &LineCoverage{
				Covered:  true,
				Count:    1,
				Partial:  false,
				StartCol: 0,
				EndCol:   len(lines[i]),
			}
		}
	}

	// Apply coverage blocks
	for _, block := range profile.Blocks {
		isCovered := block.Count > 0

		// Handle single-line blocks
		if block.StartLine == block.EndLine {
			lineNum := block.StartLine
			if lineCov, exists := coverageMap[lineNum]; exists {
				if isCovered {
					lineCov.Covered = true
					lineCov.Count = block.Count
					// For single-line blocks, check if it's partial coverage
					if block.StartCol > 1 || block.EndCol < len(lines[lineNum-1]) {
						lineCov.Partial = true
						lineCov.StartCol = block.StartCol - 1
						lineCov.EndCol = block.EndCol - 1
					}
				}
			}
		} else {
			// Handle multi-line blocks
			for line := block.StartLine; line <= block.EndLine; line++ {
				if lineCov, exists := coverageMap[line]; exists {
					if isCovered {
						lineCov.Covered = true
						lineCov.Count = block.Count
						// First and last lines might be partial
						if line == block.StartLine && block.StartCol > 1 {
							lineCov.Partial = true
							lineCov.StartCol = block.StartCol - 1
						}
						if line == block.EndLine && block.EndCol < len(lines[line-1]) {
							lineCov.Partial = true
							lineCov.EndCol = block.EndCol - 1
						}
					}
				}
			}
		}
	}

	return coverageMap
}

// outputFileWithCoverage outputs the file content with coverage colorization
func outputFileWithCoverage(content []byte, coverageMap map[int]*LineCoverage, importRanges []ImportRange) error {
	lines := bytes.Split(content, []byte("\n"))

	// Determine which lines to show based on context
	linesToShow := determineLinestoShow(coverageMap, content)

	var lastShownLine int
	for i, line := range lines {
		lineNum := i + 1
		lineCov := coverageMap[lineNum]

		// Check if we should output this line
		if !linesToShow[lineNum] {
			continue
		}

		// Show ellipsis if there's a gap
		if lastShownLine > 0 && lineNum > lastShownLine+1 {
			if useColor {
				fmt.Printf("%s...%s\n", ColorGray, ColorReset)
			} else {
				fmt.Printf("%s\n", IndicatorEllipsis)
			}
		}
		lastShownLine = lineNum

		// Format line number
		lineNumStr := fmt.Sprintf("%4d", lineNum)

		// Colorize the line based on coverage
		colorizedLine := colorizeLineContent(string(line), lineCov, lineNum, importRanges)

		// Output the line with line number and coverage indicator
		if useColor {
			var lineNumColor, indicator string
			if lineCov.Covered {
				lineNumColor = ColorGreen
				indicator = "+"
			} else {
				lineNumColor = ColorRed
				indicator = "-"
			}
			fmt.Printf("%s%s%s %s %s\n", lineNumColor, lineNumStr, ColorReset, indicator, colorizedLine)
		} else {
			var indicator string
			if lineCov.Covered {
				indicator = IndicatorCovered
			} else {
				indicator = IndicatorUncovered
			}
			fmt.Printf("%s %s %s\n", lineNumStr, indicator, string(line))
		}
	}

	return nil
}

// colorizeLineContent applies color to line content based on coverage
func colorizeLineContent(line string, lineCov *LineCoverage, lineNum int, importRanges []ImportRange) string {
	if !useColor {
		return line
	}

	// Check if this line is in an import block (AST-detected)
	inImportBlock := false
	for _, importRange := range importRanges {
		if lineNum >= importRange.Start && lineNum <= importRange.End {
			inImportBlock = true
			break
		}
	}

	// Structural elements (imports, comments, etc.) should be neutral colored
	if isUnimportantLine(line) || inImportBlock {
		return fmt.Sprintf("%s%s%s", ColorDarkGray, line, ColorReset)
	}

	// Apply coverage-based coloring to actual code
	if lineCov.Covered {
		// Use different shades of green based on coverage count (atomic mode support)
		greenColor := getGreenShade(lineCov.Count)
		return fmt.Sprintf("%s%s%s", greenColor, line, ColorReset)
	} else {
		// Uncovered line - use red background for bar mode, red text for normal mode
		if *barMode {
			return fmt.Sprintf("%s%s%s", ColorRedBg, line, ColorReset)
		} else {
			return fmt.Sprintf("%s%s%s", ColorRed, line, ColorReset)
		}
	}
}

// getGreenShade returns different shades of green based on coverage count and mode
func getGreenShade(count int) string {
	if count == 0 {
		if *barMode {
			return ColorRedBg
		}
		return ColorRed // Should not happen for covered lines, but safety
	}
	
	// Use background colors for bar mode, text colors for normal mode
	if *barMode {
		if count == 1 {
			return ColorGreenBg1 // Dark green background for barely covered
		} else if count <= 3 {
			return ColorGreenBg2 // Medium-dark green background
		} else if count <= 8 {
			return ColorGreenBg3 // Medium green background
		} else if count <= 20 {
			return ColorGreenBg4 // Medium-bright green background
		} else {
			return ColorGreenBg5 // Bright green background for heavily covered
		}
	} else {
		// Text color mode with subtle gradient
		if count == 1 {
			return ColorGreen1 // Dark green for barely covered
		} else if count <= 3 {
			return ColorGreen2 // Medium-dark green
		} else if count <= 8 {
			return ColorGreen3 // Medium green
		} else if count <= 20 {
			return ColorGreen4 // Medium-bright green
		} else {
			return ColorGreen5 // Bright green for heavily covered
		}
	}
}

// determineLinestoShow determines which lines should be displayed based on context rules
func determineLinestoShow(coverageMap map[int]*LineCoverage, content []byte) map[int]bool {
	lines := bytes.Split(content, []byte("\n"))
	totalLines := len(lines)
	linesToShow := make(map[int]bool)

	// If context is 0 or negative, show all lines
	if *contextLines <= 0 {
		for line := 1; line <= totalLines; line++ {
			linesToShow[line] = true
		}
		return linesToShow
	}

	// Find coverage transitions and boundaries
	for line := 1; line <= totalLines; line++ {
		lineCov := coverageMap[line]

		// Always show lines with coverage changes from previous line
		if line > 1 {
			prevLineCov := coverageMap[line-1]
			if lineCov.Covered != prevLineCov.Covered {
				// Show transition and context around it
				for i := max(1, line-*contextLines); i <= min(totalLines, line+*contextLines); i++ {
					linesToShow[i] = true
				}
			}
		}
		
		// Always show function signatures (like uncover does)
		lineContent := ""
		if line-1 < len(lines) {
			lineContent = string(lines[line-1])
		}
		if isFunctionSignature(lineContent) {
			linesToShow[line] = true
		}

		// Show uncovered lines and context around them (but filter out unimportant lines)
		if !lineCov.Covered {
			lineContent := ""
			if line-1 < len(lines) {
				lineContent = string(lines[line-1])
			}

			// Skip lines marked as unreachable/untested unless -a flag is set
			if !*showAll && shouldSkipLine(strings.TrimSpace(lineContent)) {
				continue
			}

			// NEVER show unimportant lines (imports, package, comments) as uncovered
			if isUnimportantLine(lineContent) {
				continue
			}

			// This is actual uncovered code - show it with context
			for i := max(1, line-*contextLines); i <= min(totalLines, line+*contextLines); i++ {
				// When adding context, also filter out unimportant lines in the context
				contextContent := ""
				if i-1 < len(lines) {
					contextContent = string(lines[i-1])
				}
				
				// Don't add unimportant lines even as context
				if !isUnimportantLine(contextContent) {
					linesToShow[i] = true
				}
			}
		}
	}

	return linesToShow
}

// shouldSkipLine checks if a line contains skip patterns
func shouldSkipLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	for _, pattern := range skipPatterns {
		if strings.HasPrefix(trimmed, pattern) {
			return true
		}
	}
	return false
}

// isCommentOrImport checks if a line is a comment or import statement
func isCommentOrImport(line string) bool {
	trimmed := strings.TrimSpace(line)
	
	// Check for comments
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return true
	}
	
	// Check for import-related lines
	if strings.HasPrefix(trimmed, "import") || 
	   strings.HasPrefix(trimmed, ")") ||
	   (strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"")) ||
	   (strings.Contains(trimmed, "import (") || strings.Contains(line, "\t\"")) {
		return true
	}
	
	return false
}

// isUnimportantLine checks if a line should never be shown as uncovered (structural elements)
func isUnimportantLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	
	// Empty lines
	if trimmed == "" {
		return true
	}
	
	// Package declarations - never show as uncovered
	if strings.HasPrefix(trimmed, "package ") {
		return true
	}
	
	// Comments - never show as uncovered
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return true
	}
	
	// Import statements and blocks - use robust detection
	if isImportRelated(trimmed) {
		return true
	}
	
	// Top-level variable declarations and constants - never show as uncovered
	if isTopLevelDeclaration(trimmed) {
		return true
	}
	
	// Single braces or parentheses on their own lines
	if trimmed == "{" || trimmed == "}" || trimmed == "(" || trimmed == ")" {
		return true
	}
	
	return false
}

// isImportRelated checks if a line is part of an import statement or block
func isImportRelated(trimmed string) bool {
	// Direct import statements
	if strings.HasPrefix(trimmed, "import ") || trimmed == "import (" {
		return true
	}
	
	// Individual import lines (quoted strings, possibly with aliases)
	if strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") {
		return true
	}
	
	// Aliased imports: alias "package" (including io/ioutil type imports)
	parts := strings.Fields(trimmed)
	if len(parts) == 2 && strings.HasPrefix(parts[1], "\"") && strings.HasSuffix(parts[1], "\"") {
		return true
	}
	
	// Underscore imports: _ "package"
	if strings.HasPrefix(trimmed, "_") && strings.Contains(trimmed, "\"") {
		return true
	}
	
	// Dot imports: . "package" 
	if strings.HasPrefix(trimmed, ".") && strings.Contains(trimmed, "\"") {
		return true
	}
	
	// Standard library imports without quotes (rare but possible): io/ioutil, fmt, etc.
	if len(parts) == 1 && !strings.Contains(trimmed, "(") && !strings.Contains(trimmed, ")") && 
	   !strings.Contains(trimmed, "{") && !strings.Contains(trimmed, "}") &&
	   !strings.Contains(trimmed, "=") && !strings.Contains(trimmed, "func") &&
	   strings.Contains(trimmed, "/") {
		return true
	}
	
	// Closing paren of import block (simple heuristic)
	if trimmed == ")" {
		return true // This is imperfect but works for most Go code
	}
	
	return false
}

// isTopLevelDeclaration checks if a line is a top-level variable, constant, or type declaration
func isTopLevelDeclaration(trimmed string) bool {
	// Variable declarations
	if strings.HasPrefix(trimmed, "var ") || strings.HasPrefix(trimmed, "var(") {
		return true
	}
	
	// Constant declarations
	if strings.HasPrefix(trimmed, "const ") || strings.HasPrefix(trimmed, "const(") {
		return true
	}
	
	// Type declarations
	if strings.HasPrefix(trimmed, "type ") {
		return true
	}
	
	// Lines that look like variable assignments at the top level
	// This catches things like: coverFile = flag.String(...)
	if !strings.HasPrefix(trimmed, "func ") && 
	   !strings.HasPrefix(trimmed, "if ") &&
	   !strings.HasPrefix(trimmed, "for ") &&
	   !strings.HasPrefix(trimmed, "switch ") &&
	   !strings.HasPrefix(trimmed, "return ") &&
	   strings.Contains(trimmed, " = ") {
		return true
	}
	
	return false
}

// isFunctionSignature checks if a line is a function signature that should always be shown
func isFunctionSignature(line string) bool {
	trimmed := strings.TrimSpace(line)
	
	// Function declarations
	if strings.HasPrefix(trimmed, "func ") {
		return true
	}
	
	// Method declarations (functions with receivers)
	if strings.HasPrefix(trimmed, "func (") {
		return true
	}
	
	return false
}

// ImportRange represents a range of lines containing imports
type ImportRange struct {
	Start int
	End   int
}

// getImportRanges uses Go's parser to accurately detect import block line ranges
func getImportRanges(content []byte) []ImportRange {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		// If parsing fails, fall back to simple heuristics
		return getImportRangesHeuristic(content)
	}

	var ranges []ImportRange
	for _, imp := range file.Imports {
		startPos := fset.Position(imp.Pos())
		endPos := fset.Position(imp.End())
		ranges = append(ranges, ImportRange{
			Start: startPos.Line,
			End:   endPos.Line,
		})
	}

	// Also handle import declarations (the "import (" and ")" lines)
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			startPos := fset.Position(genDecl.Pos())
			endPos := fset.Position(genDecl.End())
			
			// Merge overlapping ranges or extend existing ones
			merged := false
			for i := range ranges {
				if startPos.Line <= ranges[i].End+1 && endPos.Line >= ranges[i].Start-1 {
					ranges[i].Start = min(ranges[i].Start, startPos.Line)
					ranges[i].End = max(ranges[i].End, endPos.Line)
					merged = true
					break
				}
			}
			
			if !merged {
				ranges = append(ranges, ImportRange{
					Start: startPos.Line,
					End:   endPos.Line,
				})
			}
		}
	}

	return ranges
}

// getImportRangesHeuristic provides a fallback when parsing fails
func getImportRangesHeuristic(content []byte) []ImportRange {
	lines := bytes.Split(content, []byte("\n"))
	var ranges []ImportRange
	var currentRange *ImportRange

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(string(line))

		if strings.HasPrefix(trimmed, "import") {
			if currentRange == nil {
				currentRange = &ImportRange{Start: lineNum, End: lineNum}
			}
		} else if currentRange != nil {
			if trimmed == ")" || (!strings.HasPrefix(trimmed, "\"") && 
				!strings.Contains(trimmed, "\"") && trimmed != "") {
				// End of import block
				currentRange.End = lineNum
				ranges = append(ranges, *currentRange)
				currentRange = nil
			} else if strings.Contains(trimmed, "\"") {
				// Extend current range
				currentRange.End = lineNum
			}
		}
	}

	// Handle unclosed import block
	if currentRange != nil {
		currentRange.End = len(lines)
		ranges = append(ranges, *currentRange)
	}

	return ranges
}

// isInImportBlock is a simple heuristic to detect if we're in an import block
func isInImportBlock(line string) bool {
	// This is a simple heuristic - in real usage, a proper parser would be better
	// But for most Go code, a closing paren with this indentation pattern is likely an import
	return strings.HasPrefix(line, ")") || strings.HasPrefix(line, "\t)")
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// findPkgs finds package directories using 'go list'
func findPkgs(profiles []*cover.Profile) (map[string]*Pkg, error) {
	// Run go list to find the location of every package we care about.
	pkgs := make(map[string]*Pkg)
	var list []string
	for _, profile := range profiles {
		if strings.HasPrefix(profile.FileName, ".") || filepath.IsAbs(profile.FileName) {
			// Relative or absolute path.
			continue
		}
		pkg := path.Dir(profile.FileName)
		if _, ok := pkgs[pkg]; !ok {
			pkgs[pkg] = nil
			list = append(list, pkg)
		}
	}

	if len(list) == 0 {
		return pkgs, nil
	}

	cmd := exec.Command("go", append([]string{"list", "-e", "-json"}, list...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cannot run go list: %v\n%s", err, stderr.Bytes())
	}
	dec := json.NewDecoder(bytes.NewReader(stdout))
	for {
		var pkg Pkg
		err := dec.Decode(&pkg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decoding go list json: %v", err)
		}
		pkgs[pkg.ImportPath] = &pkg
	}
	return pkgs, nil
}

// findFile finds the location of the named file in GOROOT, GOPATH etc.
func findFile(pkgs map[string]*Pkg, file string) (string, error) {
	if strings.HasPrefix(file, ".") || filepath.IsAbs(file) {
		// Relative or absolute path.
		return file, nil
	}
	pkg := pkgs[path.Dir(file)]
	if pkg != nil {
		if pkg.Dir != "" {
			return filepath.Join(pkg.Dir, path.Base(file)), nil
		}
		if pkg.Error != nil {
			return "", errors.New(pkg.Error.Err)
		}
	}
	return "", fmt.Errorf("did not find package for %s in go list output", file)
}

// findFileWithFallback tries multiple approaches to find the source file
func findFileWithFallback(pkgs map[string]*Pkg, file string) (string, error) {
	// First try the normal findFile approach
	if result, err := findFile(pkgs, file); err == nil {
		if _, statErr := os.Stat(result); statErr == nil {
			return result, nil
		}
	}

	// Fallback 1: Try the file path as-is
	if _, err := os.Stat(file); err == nil {
		return file, nil
	}

	// Fallback 2: Try relative to current directory
	if !filepath.IsAbs(file) {
		// Extract just the filename from the package path
		filename := path.Base(file)
		if _, err := os.Stat(filename); err == nil {
			return filename, nil
		}

		// Try looking in subdirectories that match package structure
		parts := strings.Split(file, "/")
		if len(parts) > 1 {
			// Try progressively shorter paths
			for i := len(parts) - 2; i >= 0; i-- {
				candidate := filepath.Join(parts[i:]...)
				if _, err := os.Stat(candidate); err == nil {
					return candidate, nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not locate source file %s", file)
}
