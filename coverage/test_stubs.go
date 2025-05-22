package coverage

import icoverage "github.com/tmc/covutil/internal/coverage"

// Test stubs for mocking - these are only used in tests
type PackageMetaStub struct {
	Path       string
	Name       string
	ModulePath string
	MetaHash   [16]byte
	Functions  []icoverage.FuncDesc
}

// Counter stubs for testing
type FuncPayload struct {
	PkgIdx   uint32
	FuncIdx  uint32
	Counters []uint32
}

var (
	MockMetaPackages = []PackageMetaStub{
		{
			Path:       "test/package",
			Name:       "package",
			ModulePath: "test/module",
			Functions: []icoverage.FuncDesc{
				{
					Funcname: "TestFunction",
					Srcfile:  "test.go",
					Units: []icoverage.CoverableUnit{
						{StLine: 1, StCol: 1, EnLine: 2, EnCol: 1, NxStmts: 1},
					},
				},
			},
		},
	}
	MockMetaFileHash [16]byte
	MockPackage1     = PackageMetaStub{
		Path:       "test/package",
		Name:       "package",
		ModulePath: "test/module",
		Functions: []icoverage.FuncDesc{
			{
				Funcname: "TestFunction",
				Srcfile:  "test.go",
				Units: []icoverage.CoverableUnit{
					{StLine: 1, StCol: 1, EnLine: 2, EnCol: 1, NxStmts: 1},
				},
			},
		},
	}

	// Counter mock data
	MockCounterSegments = [][]FuncPayload{
		{{PkgIdx: 0, FuncIdx: 0, Counters: []uint32{1, 1}}},
	}
	MockCounterArgs = []map[string]string{
		{"GOOS": "linux", "GOARCH": "amd64"},
	}
	MockCounterMetaFileHash [16]byte
)
