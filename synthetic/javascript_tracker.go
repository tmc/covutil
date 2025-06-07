package synthetic

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/tmc/covutil/synthetic/parsers"
	"github.com/tmc/covutil/synthetic/parsers/plugin"
)

// JavaScriptCoverageProvider is an interface for JavaScript coverage collection
// This allows us to decouple from the chromedp implementation
type JavaScriptCoverageProvider interface {
	CollectCoverage(url string) ([]JavaScriptCoverageResult, error)
	CollectCoverageFromHTML(htmlContent string) ([]JavaScriptCoverageResult, error)
}

// JavaScriptCoverageResult represents coverage data for a JavaScript file
type JavaScriptCoverageResult struct {
	URL        string
	SourceFile string
	Coverage   map[int]bool   // line number -> executed
	Source     map[int]string // line number -> source code
	Functions  []FunctionCoverage
	Type       string // "javascript" or "css"
	Rules      []RuleCoverage
}

// GetCoveragePercentage calculates the coverage percentage
func (r *JavaScriptCoverageResult) GetCoveragePercentage() float64 {
	if len(r.Source) == 0 {
		return 0.0
	}

	executed := 0
	for _, isExecuted := range r.Coverage {
		if isExecuted {
			executed++
		}
	}

	// Count non-empty lines as executable
	executable := 0
	for _, line := range r.Source {
		if strings.TrimSpace(line) != "" {
			executable++
		}
	}

	if executable == 0 {
		return 100.0
	}

	return float64(executed) / float64(executable) * 100.0
}

// FunctionCoverage represents coverage for a JavaScript function
type FunctionCoverage struct {
	Name      string
	StartLine int
	EndLine   int
	StartCol  int
	EndCol    int
	Count     int64
	Executed  bool
}

// RuleCoverage represents coverage for a CSS rule
type RuleCoverage struct {
	Selector  string
	StartLine int
	EndLine   int
	StartCol  int
	EndCol    int
	Used      bool
}

// JavaScriptTracker provides coverage tracking for JavaScript files
type JavaScriptTracker struct {
	*BasicTracker
	provider JavaScriptCoverageProvider
	mu       sync.RWMutex
	results  map[string]*JavaScriptCoverageResult
	testName string
}

// NewJavaScriptTracker creates a new JavaScript coverage tracker
// It will attempt to use the chromedp parser if available
func NewJavaScriptTracker(options ...Option) *JavaScriptTracker {
	basicTracker := NewBasicTracker(options...)

	tracker := &JavaScriptTracker{
		BasicTracker: basicTracker,
		results:      make(map[string]*JavaScriptCoverageResult),
		testName:     "javascript",
	}

	// Extract test name from labels if provided
	if testName, ok := basicTracker.labels["test"]; ok {
		tracker.testName = testName
	}

	// Try to find a JavaScript coverage provider
	// First check if chromedp parser is available via plugin
	if parser, ok := parsers.Get("chromedp"); ok {
		// If the parser implements JavaScriptCoverageProvider, use it
		if provider, ok := parser.(JavaScriptCoverageProvider); ok {
			tracker.provider = provider
		}
	}

	return tracker
}

// SetProvider allows setting a custom JavaScript coverage provider
func (t *JavaScriptTracker) SetProvider(provider JavaScriptCoverageProvider) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.provider = provider
}

// CollectFromURL navigates to a URL and collects JavaScript coverage
func (t *JavaScriptTracker) CollectFromURL(url string) error {
	if t.provider == nil {
		// Try to load chromedp plugin if not already loaded
		_ = plugin.Load("chromedp-parser.so")

		// Check again
		if parser, ok := parsers.Get("chromedp"); ok {
			if provider, ok := parser.(JavaScriptCoverageProvider); ok {
				t.provider = provider
			}
		}

		if t.provider == nil {
			return errors.New("no JavaScript coverage provider available - install chromedp-parser plugin")
		}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	results, err := t.provider.CollectCoverage(url)
	if err != nil {
		return fmt.Errorf("failed to collect coverage from %s: %w", url, err)
	}

	// Store results and update tracker
	for _, result := range results {
		jsResult := &result
		t.results[result.SourceFile] = jsResult

		// Update the basic tracker with coverage data
		coverage := &Coverage{
			ArtifactName:  result.SourceFile,
			TotalLines:    len(result.Source),
			ExecutedLines: result.Coverage,
			Commands:      make(map[int]string),
			TestName:      t.testName,
			Timestamp:     time.Now(),
		}

		// Populate commands with source lines
		for lineNum, sourceCode := range result.Source {
			if strings.TrimSpace(sourceCode) != "" {
				coverage.Commands[lineNum] = sourceCode
			}
		}

		key := fmt.Sprintf("%s:%s", t.testName, result.SourceFile)
		t.coverages[key] = coverage
	}

	return nil
}

// CollectFromHTML collects coverage from inline JavaScript in HTML content
func (t *JavaScriptTracker) CollectFromHTML(htmlContent string) error {
	if t.provider == nil {
		return errors.New("no JavaScript coverage provider available")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	results, err := t.provider.CollectCoverageFromHTML(htmlContent)
	if err != nil {
		return fmt.Errorf("failed to collect coverage from HTML: %w", err)
	}

	// Store results and update tracker
	for _, result := range results {
		jsResult := &result
		t.results[result.SourceFile] = jsResult

		// Update the basic tracker with coverage data
		coverage := &Coverage{
			ArtifactName:  result.SourceFile,
			TotalLines:    len(result.Source),
			ExecutedLines: result.Coverage,
			Commands:      make(map[int]string),
			TestName:      t.testName,
			Timestamp:     time.Now(),
		}

		// Populate commands with source lines
		for lineNum, sourceCode := range result.Source {
			if strings.TrimSpace(sourceCode) != "" {
				coverage.Commands[lineNum] = sourceCode
			}
		}

		key := fmt.Sprintf("%s:%s", t.testName, result.SourceFile)
		t.coverages[key] = coverage
	}

	return nil
}

// GetJavaScriptResults returns the collected JavaScript coverage results
func (t *JavaScriptTracker) GetJavaScriptResults() map[string]*JavaScriptCoverageResult {
	t.mu.RLock()
	defer t.mu.RUnlock()

	results := make(map[string]*JavaScriptCoverageResult)
	for k, v := range t.results {
		results[k] = v
	}
	return results
}

// GetDetailedReport returns a detailed coverage report including function-level coverage
func (t *JavaScriptTracker) GetDetailedReport() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var report strings.Builder
	report.WriteString("JavaScript Coverage Report\n")
	report.WriteString("==========================\n\n")

	totalFiles := len(t.results)
	if totalFiles == 0 {
		report.WriteString("No JavaScript coverage data collected.\n")
		return report.String()
	}

	var totalLines, executedLines int

	for filename, result := range t.results {
		report.WriteString(fmt.Sprintf("File: %s\n", filename))
		report.WriteString(fmt.Sprintf("URL: %s\n", result.URL))

		fileLines := len(result.Source)
		fileExecuted := 0
		for _, executed := range result.Coverage {
			if executed {
				fileExecuted++
			}
		}

		coverage := GetCoveragePercentage(result)
		report.WriteString(fmt.Sprintf("Coverage: %.2f%% (%d/%d lines)\n", coverage, fileExecuted, fileLines))

		// Function coverage
		if len(result.Functions) > 0 {
			report.WriteString("\nFunctions:\n")
			for _, fn := range result.Functions {
				status := "not executed"
				if fn.Executed {
					status = fmt.Sprintf("executed %d times", fn.Count)
				}
				report.WriteString(fmt.Sprintf("  - %s: %s\n", fn.Name, status))
			}
		}

		report.WriteString("\n")

		totalLines += fileLines
		executedLines += fileExecuted
	}

	// Overall summary
	overallCoverage := float64(0)
	if totalLines > 0 {
		overallCoverage = float64(executedLines) / float64(totalLines) * 100
	}

	report.WriteString(fmt.Sprintf("\nOverall Coverage: %.2f%% (%d/%d lines across %d files)\n",
		overallCoverage, executedLines, totalLines, totalFiles))

	return report.String()
}

// ExportToCoverageProfile exports JavaScript coverage in Go coverage profile format
func (t *JavaScriptTracker) ExportToCoverageProfile() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var profiles []string

	for _, result := range t.results {
		profile := ToCoverageProfile(result)
		if profile != "" {
			profiles = append(profiles, profile)
		}
	}

	return strings.Join(profiles, "\n")
}

// GetCoveragePercentage calculates the coverage percentage for a result
func GetCoveragePercentage(r *JavaScriptCoverageResult) float64 {
	if len(r.Source) == 0 {
		return 0.0
	}

	executed := 0
	for _, isExecuted := range r.Coverage {
		if isExecuted {
			executed++
		}
	}

	// Count non-empty lines as executable
	executable := 0
	for _, line := range r.Source {
		if strings.TrimSpace(line) != "" {
			executable++
		}
	}

	if executable == 0 {
		return 100.0
	}

	return float64(executed) / float64(executable) * 100.0
}

// ToCoverageProfile converts JavaScriptCoverageResult to coverage profile format
func ToCoverageProfile(r *JavaScriptCoverageResult) string {
	var lines []string

	// Sort line numbers
	lineNums := make([]int, 0, len(r.Coverage))
	for line := range r.Coverage {
		lineNums = append(lineNums, line)
	}

	// Simple sort implementation
	for i := 0; i < len(lineNums); i++ {
		for j := i + 1; j < len(lineNums); j++ {
			if lineNums[i] > lineNums[j] {
				lineNums[i], lineNums[j] = lineNums[j], lineNums[i]
			}
		}
	}

	// Generate coverage profile entries
	for _, line := range lineNums {
		if r.Coverage[line] {
			// Format: filename:line.column,line.column count
			lines = append(lines, fmt.Sprintf("%s:%d.1,%d.1000 1 1", r.SourceFile, line, line))
		}
	}

	return strings.Join(lines, "\n")
}

// TryLoadChromedpPlugin attempts to load the chromedp parser plugin
// This is a convenience function that can be called at package initialization
func TryLoadChromedpPlugin() {
	pluginPaths := []string{
		"chromedp-parser.so",
		"./chromedp-parser.so",
		"/usr/local/lib/covutil/chromedp-parser.so",
		"/usr/lib/covutil/chromedp-parser.so",
	}

	for _, path := range pluginPaths {
		if err := plugin.Load(path); err == nil {
			log.Printf("Successfully loaded chromedp parser plugin from %s", path)
			return
		}
	}
}
