package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tmc/covutil"
)

// CoverageData represents the complete coverage data in JSON-friendly format
type CoverageData struct {
	Pod *covutil.Pod `json:"pod"`
}

// Metadata-related JSON structures for compatibility
type MetadataHeader struct {
	Magic        string `json:"magic"`
	Version      uint32 `json:"version"`
	TotalLength  uint64 `json:"total_length"`
	NumPackages  uint64 `json:"num_packages"`
	MetaFileHash string `json:"meta_file_hash"` // hex string
	StrTabOffset uint32 `json:"str_tab_offset"`
	StrTabLength uint32 `json:"str_tab_length"`
	StrTabCount  uint32 `json:"str_tab_count"`
	CounterMode  string `json:"counter_mode"`
	Granularity  string `json:"granularity"`
}

// Counter-related JSON structures for compatibility
type CounterHeader struct {
	Magic        string `json:"magic"`
	Version      uint32 `json:"version"`
	MetaFileHash string `json:"meta_file_hash"` // hex string
	NumSegments  uint64 `json:"num_segments"`
	Args         []Arg  `json:"args"`
	Flavor       string `json:"flavor"`
	BigEndian    bool   `json:"big_endian"`
}

// Arg represents a command-line argument
type Arg struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ToJSON converts CoverageData to pretty-printed JSON
func (cd *CoverageData) ToJSON() (string, error) {
	data, err := json.MarshalIndent(cd, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %w", err)
	}
	return string(data), nil
}

// FromJSON parses JSON into CoverageData
func FromJSON(jsonData string) (*CoverageData, error) {
	var cd CoverageData
	if err := json.Unmarshal([]byte(jsonData), &cd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return &cd, nil
}

// CoverageDataCmpOptions returns cmp.Options for comparing CoverageData
func CoverageDataCmpOptions() cmp.Options {
	return cmp.Options{
		// Sort slices to make comparison order-independent
		cmpopts.SortSlices(func(a, b covutil.PackageMeta) bool {
			return a.Path < b.Path
		}),
		cmpopts.SortSlices(func(a, b covutil.FuncDesc) bool {
			if a.SrcFile != b.SrcFile {
				return a.SrcFile < b.SrcFile
			}
			if len(a.Units) > 0 && len(b.Units) > 0 {
				if a.Units[0].StartLine != b.Units[0].StartLine {
					return a.Units[0].StartLine < b.Units[0].StartLine
				}
			}
			return a.FuncName < b.FuncName
		}),
		cmpopts.SortSlices(func(a, b Arg) bool {
			return a.Key < b.Key
		}),

		// Equate empty slices with nil slices
		cmpopts.EquateEmpty(),
	}
}

// Diff compares two CoverageData structures and returns a human-readable diff
func (cd *CoverageData) Diff(other *CoverageData) string {
	return cmp.Diff(cd, other, CoverageDataCmpOptions()...)
}

// Equal checks if two CoverageData structures are equal
func (cd *CoverageData) Equal(other *CoverageData) bool {
	return cmp.Equal(cd, other, CoverageDataCmpOptions()...)
}

// Summary returns a summary of the coverage data
func (cd *CoverageData) Summary() string {
	if cd.Pod == nil || cd.Pod.Profile == nil {
		return "No coverage data"
	}

	totalFuncs := 0
	totalStmts := 0
	for _, pkg := range cd.Pod.Profile.Meta.Packages {
		totalFuncs += len(pkg.Functions)
		for _, fn := range pkg.Functions {
			totalStmts += len(fn.Units)
		}
	}

	totalCounters := len(cd.Pod.Profile.Counters)

	return fmt.Sprintf("Packages: %d, Functions: %d, Units: %d, Counters: %d",
		len(cd.Pod.Profile.Meta.Packages), totalFuncs, totalStmts, totalCounters)
}

// CovdataOutput represents the output from go tool covdata commands
type CovdataOutput struct {
	Percent   string `json:"percent"`
	DebugDump string `json:"debug_dump"`
	Directory string `json:"directory"`
	Error     string `json:"error,omitempty"`
}

// runCovdataOnCoverageData generates synthetic coverage files and runs go tool covdata on them
func runCovdataOnCoverageData(cd *CoverageData) CovdataOutput {
	// Create a temporary directory for this comparison
	tempDir := fmt.Sprintf("temp_covdata_%d", time.Now().UnixNano())

	// Create the directory
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return CovdataOutput{
			Directory: tempDir,
			Error:     fmt.Sprintf("failed to create temp directory: %v", err),
		}
	}

	// Clean up on return
	defer func() {
		os.RemoveAll(tempDir)
	}()

	// Generate synthetic coverage files from the pod data
	if cd.Pod == nil {
		return CovdataOutput{
			Directory: tempDir,
			Error:     "no pod data available",
		}
	}

	if err := covutil.WritePodToDirectory(tempDir, cd.Pod); err != nil {
		return CovdataOutput{
			Directory: tempDir,
			Error:     fmt.Sprintf("failed to write pod data: %v", err),
		}
	}

	// Prepare environment for covdata commands
	baseEnv := os.Environ()
	var cmdEnv []string
	if os.Getenv("GOCOVERDEBUG") != "" {
		cmdEnv = append(baseEnv, "GOCOVERDEBUG=1")
	} else {
		cmdEnv = baseEnv
	}

	// Run go tool covdata percent
	percentCmd := exec.Command("go", "tool", "covdata", "percent", fmt.Sprintf("-i=%s", tempDir))
	percentCmd.Env = cmdEnv
	percentOutput, percentErr := percentCmd.CombinedOutput()

	// Run go tool covdata debugdump
	debugCmd := exec.Command("go", "tool", "covdata", "debugdump", fmt.Sprintf("-i=%s", tempDir))
	debugCmd.Env = cmdEnv
	debugOutput, debugErr := debugCmd.CombinedOutput()

	result := CovdataOutput{
		Directory: tempDir,
		Percent:   strings.TrimSpace(string(percentOutput)),
		DebugDump: strings.TrimSpace(string(debugOutput)),
	}

	// Collect any errors
	var errors []string
	if percentErr != nil {
		errors = append(errors, fmt.Sprintf("percent error: %v", percentErr))
	}
	if debugErr != nil {
		errors = append(errors, fmt.Sprintf("debugdump error: %v", debugErr))
	}
	if len(errors) > 0 {
		result.Error = strings.Join(errors, "; ")
	}

	return result
}

// CovdataCompare compares two CoverageData using go tool covdata output
func CovdataCompare(cd1, cd2 *CoverageData) (CovdataOutput, CovdataOutput, string) {
	out1 := runCovdataOnCoverageData(cd1)
	out2 := runCovdataOnCoverageData(cd2)

	// Basic comparison of outputs
	var differences []string

	if out1.Percent != out2.Percent {
		differences = append(differences, fmt.Sprintf("Coverage percent differs: %s vs %s", out1.Percent, out2.Percent))
	}

	if out1.Error != out2.Error {
		differences = append(differences, fmt.Sprintf("Errors differ: %s vs %s", out1.Error, out2.Error))
	}

	// Compare debug dump outputs (simplified)
	if out1.DebugDump != out2.DebugDump {
		differences = append(differences, "Debug dump outputs differ")
	}

	var summary string
	if len(differences) == 0 {
		summary = "✅ Go tool covdata outputs are identical"
	} else {
		summary = fmt.Sprintf("📋 Go tool covdata differences:\n%s", strings.Join(differences, "\n"))
	}

	return out1, out2, summary
}
