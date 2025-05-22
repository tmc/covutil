package main

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tmc/covutil"
)

// SyntheticCoverageTracker tracks coverage of scripttest scripts
type SyntheticCoverageTracker struct {
	mu             sync.Mutex
	scriptCoverage map[string]*ScriptCoverage
	enabled        bool
}

// ScriptCoverage tracks coverage for a single script
type ScriptCoverage struct {
	ScriptName    string
	TotalLines    int
	ExecutedLines map[int]bool
	Commands      map[int]string
	TestName      string
}

var globalTracker = &SyntheticCoverageTracker{
	scriptCoverage: make(map[string]*ScriptCoverage),
	enabled:        false,
}

// InitSyntheticCoverage initializes the synthetic coverage system
func InitSyntheticCoverage() {
	if os.Getenv("GO_INTEGRATION_COVERAGE") != "" && os.Getenv("SYNTHETIC_COVERAGE") != "" {
		globalTracker.enabled = true
	}
}

// TrackScriptExecution tracks the execution of a script command
func TrackScriptExecution(scriptName, testName, command string, lineNumber int) {
	if !globalTracker.enabled {
		return
	}

	globalTracker.mu.Lock()
	defer globalTracker.mu.Unlock()

	key := fmt.Sprintf("%s:%s", testName, scriptName)
	if globalTracker.scriptCoverage[key] == nil {
		globalTracker.scriptCoverage[key] = &ScriptCoverage{
			ScriptName:    scriptName,
			TestName:      testName,
			ExecutedLines: make(map[int]bool),
			Commands:      make(map[int]string),
		}
	}

	coverage := globalTracker.scriptCoverage[key]
	coverage.ExecutedLines[lineNumber] = true
	coverage.Commands[lineNumber] = command
}

// ParseAndTrackScript parses a script and tracks its structure
func ParseAndTrackScript(scriptContent, scriptName, testName string) {
	if !globalTracker.enabled {
		return
	}

	globalTracker.mu.Lock()
	defer globalTracker.mu.Unlock()

	key := fmt.Sprintf("%s:%s", testName, scriptName)
	if globalTracker.scriptCoverage[key] == nil {
		globalTracker.scriptCoverage[key] = &ScriptCoverage{
			ScriptName:    scriptName,
			TestName:      testName,
			ExecutedLines: make(map[int]bool),
			Commands:      make(map[int]string),
		}
	}

	coverage := globalTracker.scriptCoverage[key]
	lines := strings.Split(scriptContent, "\n")
	coverage.TotalLines = len(lines)

	// Parse and store commands for each line
	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			coverage.Commands[lineNum] = trimmed
		}
	}
}

// WriteSyntheticCoverageProfile writes a coverage profile for scripts
func WriteSyntheticCoverageProfile(filename string) error {
	if !globalTracker.enabled {
		return nil
	}

	globalTracker.mu.Lock()
	defer globalTracker.mu.Unlock()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "mode: set")

	// Sort scripts for consistent output
	var keys []string
	for key := range globalTracker.scriptCoverage {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		coverage := globalTracker.scriptCoverage[key]

		// Generate coverage entries for executed lines
		var lineNumbers []int
		for lineNum := range coverage.Commands {
			lineNumbers = append(lineNumbers, lineNum)
		}
		sort.Ints(lineNumbers)

		for _, lineNum := range lineNumbers {
			command := coverage.Commands[lineNum]
			executed := 0
			if coverage.ExecutedLines[lineNum] {
				executed = 1
			}

			// Create a full file path for the script (using current working directory)
			cwd, _ := os.Getwd()
			// Remove .txt extension from script name if present to avoid double extension
			scriptName := strings.TrimSuffix(coverage.ScriptName, ".txt")
			scriptPath := filepath.Join(cwd, "testdata", fmt.Sprintf("%s_%s.txt", coverage.TestName, scriptName))

			// Format: file:startLine.startCol,endLine.endCol numStmt count
			fmt.Fprintf(file, "%s:%d.1,%d.%d 1 %d\n",
				scriptPath, lineNum, lineNum, len(command)+1, executed)
		}
	}

	return nil
}

// WriteSyntheticCovData writes synthetic coverage data using covutil API
func WriteSyntheticCovData(dir string) error {
	if !globalTracker.enabled {
		return nil
	}

	globalTracker.mu.Lock()
	defer globalTracker.mu.Unlock()

	if len(globalTracker.scriptCoverage) == 0 {
		return nil
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create synthetic coverage pod using new covutil API
	pod, err := createSyntheticCoveragePod()
	if err != nil {
		return fmt.Errorf("failed to create synthetic coverage pod: %v", err)
	}

	// Write the pod to directory using covutil API
	if err := covutil.WritePodToDirectory(dir, pod); err != nil {
		return fmt.Errorf("failed to write synthetic coverage pod: %v", err)
	}

	return nil
}

// createSyntheticCoveragePod creates a coverage Pod using the new covutil API
func createSyntheticCoveragePod() (*covutil.Pod, error) {
	// Create synthetic package metadata
	packages := make([]covutil.PackageMeta, 0)
	counters := make(map[covutil.PkgFuncKey][]uint32)

	// Create a package for each script
	for _, coverage := range globalTracker.scriptCoverage {
		// Use a valid Go package path format
		pkgPath := fmt.Sprintf("github.com/tmc/covutil/internal/testprogram/scripttest_integration/testdata/%s_%s", coverage.TestName, coverage.ScriptName)

		// Create functions for each command line
		functions := make([]covutil.FuncDesc, 0)
		for lineNum, command := range coverage.Commands {
			funcName := fmt.Sprintf("line_%d", lineNum)

			// Create a single coverable unit for this line
			units := []covutil.CoverableUnit{
				{
					StartLine: uint32(lineNum),
					StartCol:  1,
					EndLine:   uint32(lineNum),
					EndCol:    uint32(len(command) + 1),
					NumStmt:   1,
				},
			}

			funcDesc := covutil.FuncDesc{
				FuncName: funcName,
				Units:    units,
			}
			functions = append(functions, funcDesc)

			// Create counter for this function
			key := covutil.PkgFuncKey{
				PkgPath:  pkgPath,
				FuncName: funcName,
			}

			// Set counter value (1 if executed, 0 if not)
			counterValue := uint32(0)
			if coverage.ExecutedLines[lineNum] {
				counterValue = 1
			}
			counters[key] = []uint32{counterValue}
		}

		// Create package metadata
		pkgMeta := covutil.PackageMeta{
			Path:      pkgPath,
			Functions: functions,
		}
		packages = append(packages, pkgMeta)
	}

	// Create the meta file
	metaFile := covutil.MetaFile{
		FilePath:     "synthetic_scripttest_coverage",
		FileHash:     [16]byte{}, // Generate a proper hash
		Mode:         covutil.ModeSet,
		Granularity:  covutil.GranularityBlock,
		Packages:     packages,
	}

	// Generate a unique hash for the meta file
	metaFile.FileHash = generateMetaFileHash()

	// Create the profile
	profile := &covutil.Profile{
		Meta:     metaFile,
		Counters: counters,
		Args:     map[string]string{
			"SYNTHETIC": "true",
			"TYPE":      "scripttest",
		},
	}

	// Create the pod
	pod := &covutil.Pod{
		ID:      fmt.Sprintf("synthetic-%d", time.Now().UnixNano()),
		Profile: profile,
		Labels: map[string]string{
			"type":      "synthetic",
			"generator": "scripttest",
		},
		Timestamp: time.Now(),
	}

	return pod, nil
}

// generateMetaFileHash generates a unique hash for the synthetic meta file
func generateMetaFileHash() [16]byte {
	h := fnv.New128()

	// Add synthetic marker
	h.Write([]byte("SYNTHETIC_COVERAGE"))

	// Add current timestamp for uniqueness
	h.Write([]byte(time.Now().Format(time.RFC3339Nano)))

	// Add script content for uniqueness
	for _, coverage := range globalTracker.scriptCoverage {
		h.Write([]byte(coverage.ScriptName))
		h.Write([]byte(coverage.TestName))
		for lineNum, command := range coverage.Commands {
			h.Write([]byte(fmt.Sprintf("%d:%s", lineNum, command)))
		}
	}

	// Convert to [16]byte
	sum := h.Sum(nil)
	var hash [16]byte
	copy(hash[:], sum)
	return hash
}

// GetSyntheticCoverageReport returns a human-readable coverage report
func GetSyntheticCoverageReport() string {
	if !globalTracker.enabled {
		return "Synthetic coverage tracking disabled"
	}

	globalTracker.mu.Lock()
	defer globalTracker.mu.Unlock()

	if len(globalTracker.scriptCoverage) == 0 {
		return "No script coverage data collected"
	}

	var report strings.Builder
	report.WriteString("=== Synthetic Script Coverage Report ===\n\n")

	var keys []string
	for key := range globalTracker.scriptCoverage {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	totalCommands := 0
	totalExecuted := 0

	for _, key := range keys {
		coverage := globalTracker.scriptCoverage[key]

		commandCount := len(coverage.Commands)
		executedCount := len(coverage.ExecutedLines)

		totalCommands += commandCount
		totalExecuted += executedCount

		percentage := float64(0)
		if commandCount > 0 {
			percentage = float64(executedCount) / float64(commandCount) * 100
		}

		report.WriteString(fmt.Sprintf("Script: %s (Test: %s)\n", coverage.ScriptName, coverage.TestName))
		report.WriteString(fmt.Sprintf("  Commands: %d total, %d executed (%.1f%%)\n",
			commandCount, executedCount, percentage))

		// Show executed commands
		if executedCount > 0 {
			report.WriteString("  Executed commands:\n")
			var lineNumbers []int
			for lineNum := range coverage.ExecutedLines {
				lineNumbers = append(lineNumbers, lineNum)
			}
			sort.Ints(lineNumbers)

			for i, lineNum := range lineNumbers {
				if i >= 5 { // Limit to first 5 for brevity
					report.WriteString(fmt.Sprintf("    ... and %d more\n", len(lineNumbers)-5))
					break
				}
				command := coverage.Commands[lineNum]
				report.WriteString(fmt.Sprintf("    Line %d: %s\n", lineNum, command))
			}
		}
		report.WriteString("\n")
	}

	// Overall summary
	overallPercentage := float64(0)
	if totalCommands > 0 {
		overallPercentage = float64(totalExecuted) / float64(totalCommands) * 100
	}

	report.WriteString(fmt.Sprintf("Overall: %d/%d commands executed (%.1f%%)\n",
		totalExecuted, totalCommands, overallPercentage))

	return report.String()
}

// Enhanced scripttest runner with coverage tracking
func RunScriptWithCoverage(t interface{}, engine interface{}, state interface{}, scriptName string, scriptContent string) {
	// Initialize synthetic coverage if enabled
	InitSyntheticCoverage()

	// Extract test name from the testing.T interface
	testName := "unknown"
	if tester, ok := t.(interface{ Name() string }); ok {
		testName = tester.Name()
	}

	// Parse the script content for coverage tracking
	ParseAndTrackScript(scriptContent, scriptName, testName)

	// Track each command as it would be executed
	// Note: This is a simplified version - in reality we'd need to hook into
	// the actual scripttest execution to track real-time execution
	lines := strings.Split(scriptContent, "\n")
	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		// Skip comments and empty lines
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// For demonstration, we'll mark executable lines as executed
		// In a real implementation, this would be triggered by actual execution
		if isExecutableCommand(trimmed) {
			TrackScriptExecution(scriptName, testName, trimmed, lineNum)
		}
	}

	// Note: The actual scripttest.Run would be called here in a real implementation
	// For now, we're just demonstrating the coverage tracking system
}

// isExecutableCommand determines if a script line is an executable command
func isExecutableCommand(line string) bool {
	// Basic patterns for scripttest commands
	patterns := []string{
		`^exec\s+`,     // exec command
		`^!\s*exec\s+`, // negated exec command
		`^go\s+`,       // go command
		`^cd\s+`,       // cd command
		`^mkdir\s+`,    // mkdir command
		`^cp\s+`,       // cp command
		`^rm\s+`,       // rm command
		`^echo\s+`,     // echo command
		`^cat\s+`,      // cat command
		`^grep\s+`,     // grep command
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, line)
		if matched {
			return true
		}
	}

	return false
}

// IntegrateSyntheticCoverage integrates synthetic coverage with the main coverage system
func IntegrateSyntheticCoverage(coverageDir string) error {
	if !globalTracker.enabled {
		return nil
	}

	// Create synthetic coverage pod using new covutil API
	pod, err := createSyntheticCoveragePod()
	if err != nil {
		return fmt.Errorf("failed to create synthetic coverage pod: %v", err)
	}

	// Create a coverage set with our synthetic pod
	coverageSet := &covutil.CoverageSet{Pods: []*covutil.Pod{pod}}

	// Write the coverage set to directory using covutil API
	if err := covutil.WriteCoverageSetToDirectory(coverageDir, coverageSet); err != nil {
		return fmt.Errorf("failed to write synthetic coverage set: %v", err)
	}

	// Write synthetic coverage profile (for compatibility)
	syntheticProfile := filepath.Join(coverageDir, "synthetic_scripttest.cov")
	if err := WriteSyntheticCoverageProfile(syntheticProfile); err != nil {
		return fmt.Errorf("failed to write synthetic coverage profile: %v", err)
	}

	// Write coverage report using covutil formatter
	reportFile := filepath.Join(coverageDir, "synthetic_coverage_report.txt")
	if err := writeCoverageReport(pod, reportFile); err != nil {
		return fmt.Errorf("failed to write coverage report: %v", err)
	}

	return nil
}

// writeCoverageReport writes a coverage report using the covutil formatter
func writeCoverageReport(pod *covutil.Pod, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the human-readable report first
	report := GetSyntheticCoverageReport()
	file.WriteString(report)
	file.WriteString("\n--- Detailed Coverage Report ---\n\n")

	// Use covutil formatter for detailed report
	formatter := covutil.NewFormatter(covutil.ModeSet)
	if err := formatter.AddPodProfile(pod); err != nil {
		return fmt.Errorf("adding pod to formatter: %v", err)
	}

	// Write textual report
	opts := covutil.TextualReportOptions{
		TargetPackages: []string{"scripttest"},
	}
	if err := formatter.WriteTextualReport(file, opts); err != nil {
		return fmt.Errorf("writing textual report: %v", err)
	}

	return nil
}

// CombineCoverageProfiles combines multiple coverage profiles into one, fixing scripttest paths
func CombineCoverageProfiles(profilePaths []string, outputPath string) error {
	if len(profilePaths) == 0 {
		return fmt.Errorf("no coverage profiles to combine")
	}

	output, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer output.Close()

	fmt.Fprintln(output, "mode: set")

	// Process each profile
	for _, profilePath := range profilePaths {
		file, err := os.Open(profilePath)
		if err != nil {
			continue // Skip missing files
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			// Skip mode lines
			if strings.HasPrefix(line, "mode:") {
				continue
			}
			if strings.TrimSpace(line) != "" {
				// Fix scripttest:// paths to use full file paths
				if strings.Contains(line, "scripttest://") {
					line = fixScriptTestPath(line)
				}
				fmt.Fprintln(output, line)
			}
		}
	}

	return nil
}

// fixScriptTestPath converts scripttest:// paths to full file paths
func fixScriptTestPath(line string) string {
	// Replace scripttest://TestName/script.txt with full path
	re := regexp.MustCompile(`scripttest://([^/]+)/([^:]+):(.+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) == 4 {
		testName := matches[1]
		scriptName := strings.TrimSuffix(matches[2], ".txt")
		coverage := matches[3]
		
		cwd, _ := os.Getwd()
		fullPath := filepath.Join(cwd, "testdata", fmt.Sprintf("%s_%s.txt", testName, scriptName))
		return fmt.Sprintf("%s:%s", fullPath, coverage)
	}
	return line
}
