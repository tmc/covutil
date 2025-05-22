// package coverage defines the public types for covutil,
// acting as an adapter/wrapper around internal representations if needed.
package coverage

import (
	icoverage "github.com/tmc/covutil/internal/coverage" // The existing internal definitions
	idecodecounter "github.com/tmc/covutil/internal/coverage/decodecounter"
	idecodemeta "github.com/tmc/covutil/internal/coverage/decodemeta"

	"bytes"
	"fmt"
	"io"
	"os"
)

// Package path constant
const PkgPath = "github.com/tmc/covutil/coverage"

// Expose needed internal types as public types
type InternalCounterMode = icoverage.CounterMode
type InternalCounterGranularity = icoverage.CounterGranularity

// Constants for internal types
const (
	InternalCtrModeSet     = icoverage.CtrModeSet
	InternalCtrModeCount   = icoverage.CtrModeCount
	InternalCtrModeAtomic  = icoverage.CtrModeAtomic
	InternalCtrModeInvalid = icoverage.CtrModeInvalid

	InternalCtrGranularityPerBlock = icoverage.CtrGranularityPerBlock
	InternalCtrGranularityPerFunc  = icoverage.CtrGranularityPerFunc
	InternalCtrGranularityInvalid  = icoverage.CtrGranularityInvalid
)

// CounterMode defines how coverage counters are interpreted.
type CounterMode icoverage.CounterMode

const (
	ModeSet     = CounterMode(icoverage.CtrModeSet)
	ModeCount   = CounterMode(icoverage.CtrModeCount)
	ModeAtomic  = CounterMode(icoverage.CtrModeAtomic)
	ModeInvalid = CounterMode(icoverage.CtrModeInvalid)
	ModeDefault = ModeCount
)

// String returns the string representation of the CounterMode.
func (cm CounterMode) String() string {
	return icoverage.CounterMode(cm).String()
}

// CounterGranularity defines the scope of each counter.
type CounterGranularity icoverage.CounterGranularity

const (
	GranularityBlock   = CounterGranularity(icoverage.CtrGranularityPerBlock)
	GranularityFunc    = CounterGranularity(icoverage.CtrGranularityPerFunc)
	GranularityInvalid = CounterGranularity(icoverage.CtrGranularityInvalid)
	GranularityDefault = GranularityBlock
)

// String returns the string representation of the CounterGranularity.
func (cg CounterGranularity) String() string {
	return icoverage.CounterGranularity(cg).String()
}

// CoverableUnit describes a single coverable region of code.
type CoverableUnit struct {
	StartLine, StartCol uint32
	EndLine, EndCol     uint32
	NumStmt             uint32
}

// FuncDesc describes a function's coverage metadata.
// It includes PackagePath for context when this struct is used outside PackageMeta.
type FuncDesc struct {
	PackagePath string // Added for context
	FuncName    string
	SrcFile     string
	Units       []CoverableUnit
	IsLiteral   bool
}

// PackageMeta represents parsed meta-data for a single package.
type PackageMeta struct {
	Path         string
	Name         string
	ModulePath   string
	Functions    []FuncDesc // Note: FuncDesc here will have its PackagePath field populated
	internalHash [16]byte   // Hash of this package's original meta-data blob
}

// MetaFile represents parsed coverage meta-data from a meta-data file.
type MetaFile struct {
	FilePath    string   // Path where meta-file was found (FS-relative or absolute)
	FileHash    [16]byte // Overall hash from the meta-file header
	Mode        CounterMode
	Granularity CounterGranularity
	Packages    []PackageMeta
}

// CounterFile represents parsed coverage counter data from a counter data file.
type CounterFile struct {
	FilePath     string   // Path where counter-file was found
	MetaFileHash [16]byte // Hash from counter-file header, linking to a MetaFile
	Segments     []CounterDataSegment
	Goos         string
	Goarch       string
}

// CounterDataSegment holds data for one execution segment.
type CounterDataSegment struct {
	Args      map[string]string
	Functions []FunctionCounters
}

// FunctionCounters holds raw counter values for a function.
type FunctionCounters struct {
	PackageIndex  uint32 // Index into the MetaFile.Packages array
	FunctionIndex uint32 // Index into MetaFile.Packages[PackageIndex].Functions
	Counts        []uint32
}

// --- Parsing wrapper functions ---

// ParseMetaFile wraps the internal meta file parsing.
func ParseMetaFile(r io.Reader, filePath string) (*MetaFile, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading meta-data from %s: %w", filePath, err)
	}
	// Create a temporary file from the data
	tmpFile, err := os.CreateTemp("", "covmeta")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		return nil, fmt.Errorf("writing temp file: %w", err)
	}
	if _, err := tmpFile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("seeking temp file: %w", err)
	}

	mfr, err := idecodemeta.NewCoverageMetaFileReader(tmpFile, data)
	if err != nil {
		return nil, fmt.Errorf("initializing internal meta-data reader for %s: %w", filePath, err)
	}

	mf := &MetaFile{
		FilePath:    filePath,
		FileHash:    mfr.FileHash(),
		Mode:        CounterMode(mfr.CounterMode()),
		Granularity: CounterGranularity(mfr.CounterGranularity()),
		Packages:    make([]PackageMeta, mfr.NumPackages()),
	}

	var payloadBuf []byte
	for i := uint64(0); i < mfr.NumPackages(); i++ {
		pkgDecoder, newPayloadBuf, err := mfr.GetPackageDecoder(uint32(i), payloadBuf)
		if err != nil {
			return nil, fmt.Errorf("decoding package %d from %s: %w", i, filePath, err)
		}
		payloadBuf = newPayloadBuf

		var internalPkgHash [16]byte
		// For now, leave internalPkgHash as zero since we don't have direct access

		pm := PackageMeta{
			Path:         pkgDecoder.PackagePath(),
			Name:         pkgDecoder.PackageName(),
			ModulePath:   pkgDecoder.ModulePath(),
			Functions:    make([]FuncDesc, pkgDecoder.NumFuncs()),
			internalHash: internalPkgHash,
		}
		for j := uint32(0); j < pkgDecoder.NumFuncs(); j++ {
			var internalFD icoverage.FuncDesc
			if err := pkgDecoder.ReadFunc(j, &internalFD); err != nil {
				return nil, fmt.Errorf("reading function %d for package %s from %s: %w", j, pm.Path, filePath, err)
			}
			publicFD := FuncDesc{
				PackagePath: pm.Path, // Populate PackagePath here
				FuncName:    internalFD.Funcname,
				SrcFile:     internalFD.Srcfile,
				IsLiteral:   internalFD.Lit,
				Units:       make([]CoverableUnit, len(internalFD.Units)),
			}
			for k, u := range internalFD.Units {
				publicFD.Units[k] = CoverableUnit{
					StartLine: u.StLine, StartCol: u.StCol,
					EndLine: u.EnLine, EndCol: u.EnCol,
					NumStmt: u.NxStmts,
				}
			}
			pm.Functions[j] = publicFD
		}
		mf.Packages[i] = pm
	}
	return mf, nil
}

// ParseCounterFile wraps the internal counter file parsing.
func ParseCounterFile(r io.Reader, filePath string) (*CounterFile, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading counter data from %s: %w", filePath, err)
	}
	byteReader := bytes.NewReader(data)
	cdr, err := idecodecounter.NewCounterDataReader(filePath, byteReader)
	if err != nil {
		return nil, fmt.Errorf("initializing internal counter data reader for %s: %w", filePath, err)
	}

	// We need to extract the meta hash from the counter data reader
	// For now, we'll use a zero hash since the field is not exported
	var metaHash [16]byte

	cf := &CounterFile{
		FilePath:     filePath,
		MetaFileHash: metaHash,
		Segments:     make([]CounterDataSegment, 0, cdr.NumSegments()),
		Goos:         cdr.Goos(),
		Goarch:       cdr.Goarch(),
	}

	for s := uint32(0); s < cdr.NumSegments(); s++ {
		if s > 0 {
			ok, err := cdr.BeginNextSegment()
			if err != nil {
				return nil, fmt.Errorf("beginning segment %d in %s: %w", s, filePath, err)
			}
			if !ok {
				return nil, fmt.Errorf("expected segment %d in %s but found no more", s, filePath)
			}
		}
		segment := CounterDataSegment{
			Args:      make(map[string]string),
			Functions: make([]FunctionCounters, 0, cdr.NumFunctionsInSegment()),
		}
		// Get args from the counter data reader
		segment.Args["GOOS"] = cdr.Goos()
		segment.Args["GOARCH"] = cdr.Goarch()
		// Add OS args if available
		osArgs := cdr.OsArgs()
		if len(osArgs) > 0 {
			segment.Args["argc"] = fmt.Sprintf("%d", len(osArgs))
			for i, arg := range osArgs {
				segment.Args[fmt.Sprintf("argv%d", i)] = arg
			}
		}

		var fp idecodecounter.FuncPayload
		for {
			ok, err := cdr.NextFunc(&fp)
			if err != nil {
				return nil, fmt.Errorf("reading function in segment %d of %s: %w", s, filePath, err)
			}
			if !ok {
				break
			}
			countsCopy := make([]uint32, len(fp.Counters))
			copy(countsCopy, fp.Counters)
			segment.Functions = append(segment.Functions, FunctionCounters{
				PackageIndex:  fp.PkgIdx,
				FunctionIndex: fp.FuncIdx,
				Counts:        countsCopy,
			})
		}
		cf.Segments = append(cf.Segments, segment)
	}
	return cf, nil
}

// FormatterAPI defines the interface the public Formatter expects from an internal formatter.
// This allows the public Formatter to wrap an internal one without exposing internal types.
type FormatterAPI interface {
	SetPackage(importpath string)
	AddUnit(file string, fname string, isfnlit bool, unit icoverage.CoverableUnit, count uint32)
	EmitTextual(pkgs []string, w io.Writer) error
	EmitPercent(w io.Writer, pkgs []string, inpkgs string, noteEmpty bool, aggregate bool) error
	EmitFuncs(w io.Writer) error
	Mode() icoverage.CounterMode
}

// FormatterWrapper wraps the internal formatter to implement FormatterAPI
type FormatterWrapper struct {
	formatter *InternalFormatter
	mode      icoverage.CounterMode
}

// InternalFormatter represents the internal cformat.Formatter (we import it as interface to avoid coupling)
type InternalFormatter interface {
	SetPackage(importpath string)
	AddUnit(file string, fname string, isfnlit bool, unit icoverage.CoverableUnit, count uint32)
	EmitTextual(pkgs []string, w io.Writer) error
	EmitPercent(w io.Writer, pkgs []string, inpkgs string, noteEmpty bool, aggregate bool) error
	EmitFuncs(w io.Writer) error
}

func NewFormatterWrapper(formatter InternalFormatter, mode icoverage.CounterMode) *FormatterWrapper {
	return &FormatterWrapper{
		formatter: &formatter,
		mode:      mode,
	}
}

func (fw *FormatterWrapper) SetPackage(importpath string) {
	(*fw.formatter).SetPackage(importpath)
}

func (fw *FormatterWrapper) AddUnit(file string, fname string, isfnlit bool, unit icoverage.CoverableUnit, count uint32) {
	(*fw.formatter).AddUnit(file, fname, isfnlit, unit, count)
}

func (fw *FormatterWrapper) EmitTextual(pkgs []string, w io.Writer) error {
	return (*fw.formatter).EmitTextual(pkgs, w)
}

func (fw *FormatterWrapper) EmitPercent(w io.Writer, pkgs []string, inpkgs string, noteEmpty bool, aggregate bool) error {
	return (*fw.formatter).EmitPercent(w, pkgs, inpkgs, noteEmpty, aggregate)
}

func (fw *FormatterWrapper) EmitFuncs(w io.Writer) error {
	return (*fw.formatter).EmitFuncs(w)
}

func (fw *FormatterWrapper) Mode() icoverage.CounterMode {
	return fw.mode
}

// PkgFuncKey identifies a function within a package for counter lookup.
type PkgFuncKey struct {
	PkgPath  string
	FuncName string
}

// AddProfileToFormatter is a helper function to populate a FormatterAPI
// from public Profile types (MetaFile and Counters map).
// This belongs in the covutil/coverage package as it bridges public types to the internal formatter API.
func AddProfileToFormatter(f FormatterAPI, meta *MetaFile, counters map[PkgFuncKey][]uint32) error {
	if meta == nil {
		return fmt.Errorf("cannot add nil meta to formatter")
	}
	// Check formatter mode compatibility with profile mode
	if icoverage.CounterMode(meta.Mode) != f.Mode() && meta.Mode != ModeInvalid {
		return fmt.Errorf("profile mode %s mismatches formatter mode %s", meta.Mode, CounterMode(f.Mode()))
	}

	for _, pkgMeta := range meta.Packages {
		f.SetPackage(pkgMeta.Path)
		for _, fnDesc := range pkgMeta.Functions {
			key := PkgFuncKey{PkgPath: pkgMeta.Path, FuncName: fnDesc.FuncName}
			fnCounts, hasCounts := counters[key]

			for unitIdx, unit := range fnDesc.Units {
				var countVal uint32
				if hasCounts && unitIdx < len(fnCounts) {
					countVal = fnCounts[unitIdx]
				}
				internalUnit := icoverage.CoverableUnit{
					StLine: unit.StartLine, StCol: unit.StartCol,
					EnLine: unit.EndLine, EnCol: unit.EndCol,
					NxStmts: unit.NumStmt,
					Parent:  0,
				}
				f.AddUnit(fnDesc.SrcFile, fnDesc.FuncName, fnDesc.IsLiteral, internalUnit, countVal)
			}
		}
	}
	return nil
}
