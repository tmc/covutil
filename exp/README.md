# Coverage Utilities

This directory contains experimental tools for working with Go test coverage profiles.

## Commands

The following commands operate on coverage profiles produced by `go test -coverprofile=`:

**covanalyze** - Go program providing unified interface to coverage analysis commands.
Supports comparing legacy vs scripttest coverage, comprehensive analysis, pattern matching, and more.

**covcompare** - Go program for comparing coverage between legacy and scripttest profiles.
Extracts function-level coverage differences and identifies gaps in scripttest coverage.

**covdiff** - Compare coverage profiles, showing which lines gained or lost coverage.
Analyzes differences between two coverage profiles and highlights coverage gaps.

**covdup** - Find tests that produce identical coverage, indicating potential redundancy.
Analyzes test suite for redundant tests that can potentially be removed.

**covered** - Annotate source files with coverage information as comments.
Can output individual files or txtar archives with colorized coverage display.
Shows which specific tests cover each line when requested.

**covshow** - Display uncovered lines for specified functions.
Takes a function name and shows source code with annotations for lines not covered by scripttest tests.

**covzero** - Identify tests that contribute no new coverage beyond other tests.
Finds tests with minimal or zero coverage contribution that could be candidates for removal.

**uncover** - List uncovered functions and statements from coverage profiles.
Supports multiple output formats and can rank functions by coverage percentage.
(Note: This tool is left unchanged from its original state)

## Testing

All commands (except uncover) are tested using `rsc.io/script/scripttest`. 
Run tests with:

	go test ./cmd/...

## Usage

Each command accepts standard Go coverage profiles.
Generate a profile with:

	go test -coverprofile=coverage.out ./...

Then analyze with any of the tools above.

### Examples

Compare legacy and scripttest coverage:
```bash
covcompare -legacy legacy.out -scripttest scripttest.out
```

Find redundant tests:
```bash
covdup -delta-dir coverage/delta -verbose
```

Show coverage for a specific function:
```bash
covshow -func=MyFunction
```

Analyze overall coverage patterns:
```bash
covanalyze -compare
covanalyze -pattern crypto
```