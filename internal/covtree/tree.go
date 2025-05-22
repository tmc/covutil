package covtree

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tmc/covutil/internal/coverage"
	"github.com/tmc/covutil/internal/coverage/decodecounter"
	"github.com/tmc/covutil/internal/coverage/decodemeta"
	"github.com/tmc/covutil/internal/coverage/pods"
)

type CoverageTree struct {
	Packages map[string]*PackageNode
	Root     *DirectoryNode
}

type DirectoryNode struct {
	Name         string
	Path         string
	Children     map[string]*DirectoryNode
	Packages     []*PackageNode
	TotalLines   int
	CoveredLines int
}

type PackageNode struct {
	ImportPath   string
	Name         string
	ModulePath   string
	Functions    []*FunctionNode
	TotalLines   int
	CoveredLines int
	CoverageRate float64
	MetaFile     string
}

type FunctionNode struct {
	Name         string
	File         string
	Units        []CoverableUnitNode
	TotalLines   int
	CoveredLines int
	CoverageRate float64
	IsLiteral    bool
}

type CoverableUnitNode struct {
	StartLine uint32
	StartCol  uint32
	EndLine   uint32
	EndCol    uint32
	Count     uint32
	Covered   bool
}

func NewCoverageTree() *CoverageTree {
	return &CoverageTree{
		Packages: make(map[string]*PackageNode),
		Root: &DirectoryNode{
			Name:     "",
			Path:     "",
			Children: make(map[string]*DirectoryNode),
		},
	}
}

func (ct *CoverageTree) LoadFromDirectory(dir string) error {
	pods, err := pods.CollectPods([]string{dir}, true)
	if err != nil {
		return err
	}

	for _, pod := range pods {
		if err := ct.loadPod(pod); err != nil {
			return fmt.Errorf("loading pod %s: %w", pod.MetaFile, err)
		}
	}

	ct.calculateCoverage()
	return nil
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
		}

		counters := make(map[uint32]map[uint32][]uint32)
		for _, counterFile := range pod.CounterDataFiles {
			if err := ct.loadCounterFile(counterFile, counters); err != nil {
				return err
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

type Filter struct {
	PackagePattern  string
	FunctionPattern string
	MinCoverage     float64
	MaxCoverage     float64
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
