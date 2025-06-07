package covtree

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tmc/covutil/internal/coverage"
	"github.com/tmc/covutil/internal/coverage/decodecounter"
	"github.com/tmc/covutil/internal/coverage/decodemeta"
	"github.com/tmc/covutil/internal/coverage/pods"
)

// CoverageTree is the main data structure that holds all coverage information
// organized in a hierarchical tree structure. It provides methods for loading,
// analyzing, and filtering coverage data with support for metadata extensions.
type CoverageTree struct {
	// Packages maps package import paths to their coverage data
	Packages map[string]*PackageNode
	// Root is the root directory node of the coverage tree
	Root *DirectoryNode
	// Metadata contains extended metadata beyond standard coverage format
	Metadata map[string]string
}

// DirectoryNode represents a directory in the coverage tree hierarchy.
// It can contain both subdirectories and packages, allowing for
// hierarchical organization of coverage data.
type DirectoryNode struct {
	// Name is the directory name (base name, not full path)
	Name string
	// Path is the full path to this directory
	Path string
	// Children maps child directory names to their nodes
	Children map[string]*DirectoryNode
	// Packages contains all packages directly in this directory
	Packages []*PackageNode
	// TotalLines is the sum of all lines in this directory and its children
	TotalLines int
	// CoveredLines is the sum of covered lines in this directory and its children
	CoveredLines int
}

// PackageNode represents coverage data for a single Go package.
// It contains aggregate coverage statistics and detailed function-level data,
// plus metadata extensions for advanced coverage tracking.
type PackageNode struct {
	// ImportPath is the full import path of the package
	ImportPath string
	// Name is the package name (typically the last segment of ImportPath)
	Name string
	// ModulePath is the module path this package belongs to
	ModulePath string
	// Functions contains coverage data for all functions in the package
	Functions []*FunctionNode
	// TotalLines is the total number of executable lines in the package
	TotalLines int
	// CoveredLines is the number of lines that were executed during testing
	CoveredLines int
	// CoverageRate is the percentage of lines covered (0.0 to 1.0)
	CoverageRate float64
	// MetaFile is the path to the coverage metadata file
	MetaFile string
	// Metadata contains extended metadata for this package
	// Common keys: GoTestName, GoTestPackage, TestType, TestRunID
	Metadata map[string]string
}

// FunctionNode represents coverage data for a single function.
// It contains detailed coverage information at the statement/block level.
type FunctionNode struct {
	// Name is the function name (including receiver for methods)
	Name string
	// File is the source file containing this function
	File string
	// Units are the individual coverage units (blocks) within the function
	Units []CoverableUnitNode
	// TotalLines is the total number of executable lines in the function
	TotalLines int
	// CoveredLines is the number of lines that were executed
	CoveredLines int
	// CoverageRate is the percentage of lines covered (0.0 to 1.0)
	CoverageRate float64
	// IsLiteral indicates if this is a function literal (anonymous function)
	IsLiteral bool
}

// CoverableUnitNode represents a single coverage unit (basic block)
// within a function. This is the most granular level of coverage data.
type CoverableUnitNode struct {
	// StartLine is the starting line number of this unit
	StartLine uint32
	// StartCol is the starting column number
	StartCol uint32
	// EndLine is the ending line number of this unit
	EndLine uint32
	// EndCol is the ending column number
	EndCol uint32
	// Count is the number of times this unit was executed
	Count uint32
	// Covered indicates whether this unit was executed at least once
	Covered bool
}

// LoadOptions configures coverage loading behavior
type LoadOptions struct {
	// MaxDepth limits how deep to scan for coverage files (0 = unlimited)
	MaxDepth int
	// FollowSymlinks determines whether to follow symbolic links
	FollowSymlinks bool
}

// NewCoverageTree creates a new empty CoverageTree ready to be populated
// with coverage data.
func NewCoverageTree() *CoverageTree {
	return &CoverageTree{
		Packages: make(map[string]*PackageNode),
		Root: &DirectoryNode{
			Name:     "",
			Path:     "",
			Children: make(map[string]*DirectoryNode),
		},
		Metadata: make(map[string]string),
	}
}

// SetMetadata sets a metadata value for the coverage tree.
// Common metadata keys include:
//   - GoTestName: The test that generated this coverage
//   - GoModuleName: The module being tested
//   - GoTestPackage: The package containing the test
//   - TestType: unit, integration, e2e, cross-module
//   - TestRunID: Unique identifier for cross-module correlation
func (ct *CoverageTree) SetMetadata(key, value string) {
	if ct.Metadata == nil {
		ct.Metadata = make(map[string]string)
	}
	ct.Metadata[key] = value
}

// GetMetadata retrieves a metadata value by key
func (ct *CoverageTree) GetMetadata(key string) string {
	if ct.Metadata == nil {
		return ""
	}
	return ct.Metadata[key]
}

// LoadMetadataFromEnv loads metadata from environment variables.
// It looks for common covutil environment variables:
//   - COVUTIL_TEST_RUN_ID -> TestRunID
//   - COVUTIL_MODULE -> GoModuleName
//   - COVUTIL_TEST_NAME -> GoTestName
//   - COVUTIL_TEST_PACKAGE -> GoTestPackage
//   - COVUTIL_TEST_TYPE -> TestType
func (ct *CoverageTree) LoadMetadataFromEnv() {
	envMappings := map[string]string{
		"COVUTIL_TEST_RUN_ID":  "TestRunID",
		"COVUTIL_MODULE":       "GoModuleName",
		"COVUTIL_TEST_NAME":    "GoTestName",
		"COVUTIL_TEST_PACKAGE": "GoTestPackage",
		"COVUTIL_TEST_TYPE":    "TestType",
		"GITHUB_RUN_ID":        "CIRunID",
		"GITHUB_REPOSITORY":    "Repository",
		"GITHUB_REF":           "GitRef",
		"JENKINS_BUILD_ID":     "CIRunID",
		"CI":                   "CI",
	}

	for envVar, metaKey := range envMappings {
		if value := os.Getenv(envVar); value != "" {
			ct.SetMetadata(metaKey, value)
		}
	}
}

// LoadFromDirectory loads coverage data from a directory containing
// coverage metadata and counter files. This is typically a directory
// created by running a Go program built with -cover.
func (ct *CoverageTree) LoadFromDirectory(dir string) error {
	// Convert to absolute path for better handling
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	return ct.LoadFromFSWithBase(os.DirFS(dir), ".", absDir, nil)
}

// LoadFromFS loads coverage data from a filesystem interface.
// This allows loading coverage data from embedded filesystems or
// other fs.FS implementations.
func (ct *CoverageTree) LoadFromFS(fsys fs.FS, root string, opts *LoadOptions) error {
	return ct.LoadFromFSWithBase(fsys, root, "", opts)
}

// LoadFromFSWithBase loads coverage data from a filesystem interface with a known base path.
// The basePath parameter is used to construct absolute paths when the fs.FS is relative.
// Currently supports depth-limited scanning but delegates to OS filesystem for loading.
func (ct *CoverageTree) LoadFromFSWithBase(fsys fs.FS, root string, basePath string, opts *LoadOptions) error {
	// Scan for coverage directories using fs.FS
	coverageDirs, err := ct.scanForCoverageDirectoriesFS(fsys, root, opts)
	if err != nil {
		return err
	}

	if len(coverageDirs) == 0 {
		return &NoCoverageDataError{Dir: root}
	}

	// Currently, we need to delegate to the OS filesystem for actual loading
	// because the underlying coverage packages expect *os.File
	// This is a limitation that could be improved in the future
	loadedCount := 0
	for _, dir := range coverageDirs {
		// Resolve the OS path using the base path if provided
		var osPath string
		if basePath != "" {
			if dir == "." {
				osPath = basePath
			} else {
				osPath = filepath.Join(basePath, dir)
			}
		} else {
			osPath = dir
		}

		pods, err := pods.CollectPods([]string{osPath}, true)
		if err != nil {
			continue // Continue with other directories if one fails
		}

		for _, pod := range pods {
			if err := ct.loadPod(pod); err != nil {
				continue // Continue with other pods if one fails
			}
			loadedCount++
		}
	}

	if loadedCount == 0 && len(coverageDirs) > 0 {
		return &CoverageParseError{
			Dir:   root,
			Count: len(coverageDirs),
		}
	}

	ct.calculateCoverage()
	return nil
}

// scanForCoverageDirectoriesFS recursively scans the given filesystem for coverage data directories
func (ct *CoverageTree) scanForCoverageDirectoriesFS(fsys fs.FS, root string, opts *LoadOptions) ([]string, error) {
	var coverageDirs []string

	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Check depth limit if options provided
		if opts != nil && opts.MaxDepth > 0 {
			relPath, _ := filepath.Rel(root, path)
			depth := strings.Count(relPath, string(filepath.Separator))
			if depth > opts.MaxDepth {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}

		if !d.IsDir() {
			return nil
		}

		// Check if this directory contains coverage files
		if ct.containsCoverageFilesFS(fsys, path) {
			coverageDirs = append(coverageDirs, path)
		}

		return nil
	})

	return coverageDirs, err
}

// containsCoverageFilesFS checks if a directory contains Go coverage files via fs.FS
func (ct *CoverageTree) containsCoverageFilesFS(fsys fs.FS, dir string) bool {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return false
	}

	hasMetaFile := false
	hasCounterFile := false

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, "covmeta.") {
			hasMetaFile = true
		}
		if strings.HasPrefix(name, "covcounters.") {
			hasCounterFile = true
		}

		// Need both meta and counter files to be a valid coverage directory
		if hasMetaFile && hasCounterFile {
			return true
		}
	}

	return false
}

func (ct *CoverageTree) loadPod(pod pods.Pod) error {
	file, err := os.Open(pod.MetaFile)
	if err != nil {
		return err
	}
	defer file.Close()

	metaFileReader, err := decodemeta.NewCoverageMetaFileReader(file, nil)
	if err != nil {
		return err
	}

	for pkgIdx := uint32(0); pkgIdx < uint32(metaFileReader.NumPackages()); pkgIdx++ {
		metaData, _, err := metaFileReader.GetPackageDecoder(pkgIdx, nil)
		if err != nil {
			return err
		}

		pkg := &PackageNode{
			ImportPath: metaData.PackagePath(),
			Name:       metaData.PackageName(),
			ModulePath: metaData.ModulePath(),
			Functions:  make([]*FunctionNode, 0, metaData.NumFuncs()),
			MetaFile:   pod.MetaFile,
			Metadata:   make(map[string]string),
		}

		// Copy tree-level metadata to package
		for k, v := range ct.Metadata {
			pkg.Metadata[k] = v
		}

		// Add module-specific metadata if not already set
		if pkg.Metadata["GoModuleName"] == "" && pkg.ModulePath != "" {
			pkg.Metadata["GoModuleName"] = pkg.ModulePath
		}

		counters := make(map[uint32]map[uint32][]uint32)
		for _, counterFile := range pod.CounterDataFiles {
			if err := ct.loadCounterFile(counterFile, counters); err != nil {
				// Skip counter loading if it fails - we can still provide function info
				// This is a workaround for "short read on string table" issue
				break
			}
		}

		for i := uint32(0); i < metaData.NumFuncs(); i++ {
			var funcDesc coverage.FuncDesc
			if err := metaData.ReadFunc(i, &funcDesc); err != nil {
				return err
			}

			fn := &FunctionNode{
				Name:      funcDesc.Funcname,
				File:      funcDesc.Srcfile,
				Units:     make([]CoverableUnitNode, len(funcDesc.Units)),
				IsLiteral: funcDesc.Lit,
			}

			funcCounters := counters[pkgIdx][i]
			for j, unit := range funcDesc.Units {
				count := uint32(0)
				if funcCounters != nil && j < len(funcCounters) {
					count = funcCounters[j]
				}
				fn.Units[j] = CoverableUnitNode{
					StartLine: unit.StLine,
					StartCol:  unit.StCol,
					EndLine:   unit.EnLine,
					EndCol:    unit.EnCol,
					Count:     count,
					Covered:   count > 0,
				}
			}

			pkg.Functions = append(pkg.Functions, fn)
		}

		ct.Packages[pkg.ImportPath] = pkg
		ct.addToDirectoryTree(pkg)
	}
	return nil
}

func (ct *CoverageTree) loadCounterFile(filename string, counters map[uint32]map[uint32][]uint32) error {
	file := openFile(filename)
	if closer, ok := file.(io.Closer); ok {
		defer closer.Close()
	}

	reader, err := decodecounter.NewCounterDataReader(filename, file)
	if err != nil {
		return err
	}

	for {
		hasNext, err := reader.BeginNextSegment()
		if err != nil {
			return err
		}
		if !hasNext {
			break
		}

		for {
			var payload decodecounter.FuncPayload
			hasFunc, err := reader.NextFunc(&payload)
			if err != nil {
				return err
			}
			if !hasFunc {
				break
			}

			if counters[payload.PkgIdx] == nil {
				counters[payload.PkgIdx] = make(map[uint32][]uint32)
			}
			counters[payload.PkgIdx][payload.FuncIdx] = payload.Counters
		}
	}

	return nil
}

func (ct *CoverageTree) addToDirectoryTree(pkg *PackageNode) {
	parts := strings.Split(pkg.ImportPath, "/")
	current := ct.Root

	for _, part := range parts {
		if current.Children[part] == nil {
			current.Children[part] = &DirectoryNode{
				Name:     part,
				Path:     filepath.Join(current.Path, part),
				Children: make(map[string]*DirectoryNode),
			}
		}
		current = current.Children[part]
	}

	current.Packages = append(current.Packages, pkg)
}

func (ct *CoverageTree) calculateCoverage() {
	for _, pkg := range ct.Packages {
		for _, fn := range pkg.Functions {
			for _, unit := range fn.Units {
				fn.TotalLines += int(unit.EndLine - unit.StartLine + 1)
				if unit.Covered {
					fn.CoveredLines += int(unit.EndLine - unit.StartLine + 1)
				}
			}
			if fn.TotalLines > 0 {
				fn.CoverageRate = float64(fn.CoveredLines) / float64(fn.TotalLines)
			}
			pkg.TotalLines += fn.TotalLines
			pkg.CoveredLines += fn.CoveredLines
		}
		if pkg.TotalLines > 0 {
			pkg.CoverageRate = float64(pkg.CoveredLines) / float64(pkg.TotalLines)
		}
	}

	ct.calculateDirectoryCoverage(ct.Root)
}

func (ct *CoverageTree) calculateDirectoryCoverage(dir *DirectoryNode) {
	for _, child := range dir.Children {
		ct.calculateDirectoryCoverage(child)
		dir.TotalLines += child.TotalLines
		dir.CoveredLines += child.CoveredLines
	}

	for _, pkg := range dir.Packages {
		dir.TotalLines += pkg.TotalLines
		dir.CoveredLines += pkg.CoveredLines
	}
}

// Filter defines criteria for filtering packages and functions.
// It supports pattern matching and metadata filtering for advanced queries.
type Filter struct {
	PackagePattern  string
	FunctionPattern string
	MinCoverage     float64
	MaxCoverage     float64
	// Metadata filters - all specified metadata must match
	Metadata map[string]string
}

func (ct *CoverageTree) FilterPackages(filter Filter) []*PackageNode {
	var result []*PackageNode
	for _, pkg := range ct.Packages {
		if ct.matchesFilter(pkg, filter) {
			result = append(result, pkg)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ImportPath < result[j].ImportPath
	})
	return result
}

func (ct *CoverageTree) matchesFilter(pkg *PackageNode, filter Filter) bool {
	if filter.PackagePattern != "" {
		matched, _ := filepath.Match(filter.PackagePattern, pkg.ImportPath)
		if !matched {
			return false
		}
	}

	// Check metadata filters
	for key, value := range filter.Metadata {
		pkgValue, exists := pkg.Metadata[key]
		if !exists {
			return false
		}
		// Support wildcard patterns in metadata values
		if strings.Contains(value, "*") {
			matched, _ := filepath.Match(value, pkgValue)
			if !matched {
				return false
			}
		} else if pkgValue != value {
			return false
		}
	}

	if filter.MinCoverage > 0 && pkg.CoverageRate < filter.MinCoverage {
		return false
	}

	if filter.MaxCoverage > 0 && pkg.CoverageRate > filter.MaxCoverage {
		return false
	}

	return true
}

func (ct *CoverageTree) GetPackage(importPath string) *PackageNode {
	return ct.Packages[importPath]
}

func (ct *CoverageTree) GetPackageNames() []string {
	var names []string
	for name := range ct.Packages {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (ct *CoverageTree) Summary() CoverageSummary {
	var total, covered int
	for _, pkg := range ct.Packages {
		total += pkg.TotalLines
		covered += pkg.CoveredLines
	}

	rate := 0.0
	if total > 0 {
		rate = float64(covered) / float64(total)
	}

	return CoverageSummary{
		TotalPackages: len(ct.Packages),
		TotalLines:    total,
		CoveredLines:  covered,
		CoverageRate:  rate,
	}
}

type CoverageSummary struct {
	TotalPackages int
	TotalLines    int
	CoveredLines  int
	CoverageRate  float64
}
