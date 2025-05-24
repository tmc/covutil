// Package synthetic provides functionality for generating synthetic coverage data
// for non-Go artifacts such as scripts, configuration files, and other resources.
package synthetic

import (
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
	"time"

	"github.com/tmc/covutil"
)

// Tracker defines the interface for tracking coverage of non-Go artifacts
type Tracker interface {
	// Track records whether a specific location in an artifact was executed
	Track(artifact, location string, executed bool)

	// GeneratePod creates a covutil.Pod containing the synthetic coverage data
	GeneratePod() (*covutil.Pod, error)

	// GenerateProfile creates a coverage profile in the specified format
	GenerateProfile(format string) ([]byte, error)

	// GetReport returns a human-readable coverage report
	GetReport() string

	// Reset clears all tracked coverage data
	Reset()
}

// Coverage represents coverage information for a single artifact
type Coverage struct {
	ArtifactName  string
	TotalLines    int
	ExecutedLines map[int]bool
	Commands      map[int]string
	TestName      string
	Timestamp     time.Time
}

// BasicTracker provides a simple implementation of the Tracker interface
type BasicTracker struct {
	mu        sync.RWMutex
	coverages map[string]*Coverage
	enabled   bool
	labels    map[string]string
}

// NewBasicTracker creates a new BasicTracker instance
func NewBasicTracker(options ...Option) *BasicTracker {
	t := &BasicTracker{
		coverages: make(map[string]*Coverage),
		enabled:   true,
		labels:    make(map[string]string),
	}

	for _, opt := range options {
		opt(t)
	}

	return t
}

// Option defines configuration options for trackers
type Option func(*BasicTracker)

// WithLabels sets custom labels for the tracker
func WithLabels(labels map[string]string) Option {
	return func(t *BasicTracker) {
		for k, v := range labels {
			t.labels[k] = v
		}
	}
}

// WithEnabled controls whether tracking is enabled
func WithEnabled(enabled bool) Option {
	return func(t *BasicTracker) {
		t.enabled = enabled
	}
}

// Track records coverage information for a specific artifact location
func (t *BasicTracker) Track(artifact, location string, executed bool) {
	if !t.enabled {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	key := fmt.Sprintf("%s:%s", t.getTestName(artifact), artifact)
	if t.coverages[key] == nil {
		t.coverages[key] = &Coverage{
			ArtifactName:  artifact,
			ExecutedLines: make(map[int]bool),
			Commands:      make(map[int]string),
			TestName:      t.getTestName(artifact),
			Timestamp:     time.Now(),
		}
	}

	coverage := t.coverages[key]

	// Parse location as line number (simple implementation)
	var lineNum int
	fmt.Sscanf(location, "%d", &lineNum)
	if lineNum > 0 {
		if executed {
			coverage.ExecutedLines[lineNum] = true
		}
		// Store the command/content for this line
		if cmd, exists := coverage.Commands[lineNum]; !exists || cmd == "" {
			coverage.Commands[lineNum] = location
		}
	}
}

// GeneratePod creates a covutil.Pod with synthetic coverage data
func (t *BasicTracker) GeneratePod() (*covutil.Pod, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.coverages) == 0 {
		return nil, fmt.Errorf("no coverage data to generate")
	}

	// Create synthetic package metadata
	packages := make([]covutil.PackageMeta, 0)
	counters := make(map[covutil.PkgFuncKey][]uint32)

	for _, coverage := range t.coverages {
		pkgPath := fmt.Sprintf("synthetic/%s/%s", coverage.TestName, coverage.ArtifactName)

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
				SrcFile:  coverage.ArtifactName,
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

	// Generate a unique hash for the meta file
	fileHash := t.generateMetaFileHash()

	// Create the meta file
	metaFile := covutil.MetaFile{
		FilePath:    "synthetic_coverage",
		FileHash:    fileHash,
		Mode:        covutil.ModeSet,
		Granularity: covutil.GranularityBlock,
		Packages:    packages,
	}

	// Create the profile
	profile := &covutil.Profile{
		Meta:     metaFile,
		Counters: counters,
		Args: map[string]string{
			"SYNTHETIC": "true",
			"TYPE":      "generic",
		},
	}

	// Merge custom labels
	labels := map[string]string{
		"type":      "synthetic",
		"generator": "basic-tracker",
	}
	for k, v := range t.labels {
		labels[k] = v
	}

	// Create the pod
	pod := &covutil.Pod{
		ID:        fmt.Sprintf("synthetic-%d", time.Now().UnixNano()),
		Profile:   profile,
		Labels:    labels,
		Timestamp: time.Now(),
	}

	return pod, nil
}

// GenerateProfile creates a coverage profile in the specified format
func (t *BasicTracker) GenerateProfile(format string) ([]byte, error) {
	switch format {
	case "text", "profile":
		return t.generateTextProfile()
	case "json":
		return t.generateJSONProfile()
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// GetReport returns a human-readable coverage report
func (t *BasicTracker) GetReport() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.coverages) == 0 {
		return "No coverage data collected"
	}

	report := "=== Synthetic Coverage Report ===\n\n"

	totalCommands := 0
	totalExecuted := 0

	// Sort coverages by artifact name for deterministic output
	keys := make([]string, 0, len(t.coverages))
	for key := range t.coverages {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		coverage := t.coverages[key]
		commandCount := len(coverage.Commands)
		executedCount := len(coverage.ExecutedLines)

		totalCommands += commandCount
		totalExecuted += executedCount

		percentage := float64(0)
		if commandCount > 0 {
			percentage = float64(executedCount) / float64(commandCount) * 100
		}

		report += fmt.Sprintf("Artifact: %s (Test: %s)\n", coverage.ArtifactName, coverage.TestName)
		report += fmt.Sprintf("  Commands: %d total, %d executed (%.1f%%)\n",
			commandCount, executedCount, percentage)
		report += "\n"
	}

	// Overall summary
	overallPercentage := float64(0)
	if totalCommands > 0 {
		overallPercentage = float64(totalExecuted) / float64(totalCommands) * 100
	}

	report += fmt.Sprintf("Overall: %d/%d commands executed (%.1f%%)\n",
		totalExecuted, totalCommands, overallPercentage)

	return report
}

// Reset clears all tracked coverage data
func (t *BasicTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.coverages = make(map[string]*Coverage)
}

// Helper methods

func (t *BasicTracker) getTestName(artifact string) string {
	// Simple heuristic - could be made configurable
	return "synthetic-test"
}

func (t *BasicTracker) generateMetaFileHash() [16]byte {
	h := fnv.New128()

	// Add synthetic marker
	h.Write([]byte("SYNTHETIC_COVERAGE"))

	// Add current timestamp for uniqueness
	h.Write([]byte(time.Now().Format(time.RFC3339Nano)))

	// Add coverage content for uniqueness
	for _, coverage := range t.coverages {
		h.Write([]byte(coverage.ArtifactName))
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

func (t *BasicTracker) generateTextProfile() ([]byte, error) {
	profile := "mode: set\n"

	for _, coverage := range t.coverages {
		for lineNum, command := range coverage.Commands {
			executed := 0
			if coverage.ExecutedLines[lineNum] {
				executed = 1
			}

			// Create a synthetic file path
			filePath := fmt.Sprintf("synthetic://%s/%s", coverage.TestName, coverage.ArtifactName)

			// Format: file:startLine.startCol,endLine.endCol numStmt count
			profile += fmt.Sprintf("%s:%d.1,%d.%d 1 %d\n",
				filePath, lineNum, lineNum, len(command)+1, executed)
		}
	}

	return []byte(profile), nil
}

func (t *BasicTracker) generateJSONProfile() ([]byte, error) {
	// This would implement JSON format generation
	// For now, return a simple implementation
	return []byte(`{"format": "json", "note": "JSON format not yet implemented"}`), nil
}
