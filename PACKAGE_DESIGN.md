# CovUtil Package Architecture Design

## Overview
This document outlines the refactoring of the covutil project into reusable packages for better organization, maintainability, and community adoption.

## Package Structure

```
github.com/tmc/covutil/
├── covutil.go                    # Main API entry point
├── coverage/                     # Public coverage types and interfaces
├── synthetic/                    # Synthetic coverage generation
├── scripttest/                   # Enhanced scripttest integration  
├── integration/                  # Integration testing framework
├── covmeta/                      # Coverage metadata system (existing)
├── covtree/                      # Coverage tree visualization (promote)
├── covforest/                    # Multi-source coverage management (promote)
├── overlays/                     # Build overlay system (promote)
├── internal/                     # Internal implementation details
│   ├── coverage/                 # Low-level coverage handling
│   └── testprogram/              # Test programs and examples
├── cmd/                          # Command-line tools
│   ├── covtree/
│   ├── covforest/
│   └── covutil/
└── examples/                     # Usage examples and demos
```

## Priority Packages for Creation

### 1. `synthetic/` - Synthetic Coverage Generation
**Purpose**: Generate coverage data for non-Go artifacts (scripts, configs, etc.)
**Key Features**:
- Execution tracking for any artifact type
- Coverage pod generation using covutil API
- Multiple output formats (binary, text, JSON)
- Template-based synthesis from real coverage data

**Public API**:
```go
package synthetic

type Tracker interface {
    Track(artifact, location string, executed bool)
    GeneratePod() (*covutil.Pod, error)
    GenerateProfile(format string) ([]byte, error)
}

type ScriptTracker struct { ... }
type ConfigTracker struct { ... }
```

### 2. `scripttest/` - Enhanced Scripttest Integration
**Purpose**: Scripttest framework with built-in coverage support
**Key Features**:
- Parallel execution control
- Per-test coverage isolation
- Automatic coverage collection
- Integration with Go's scripttest

**Public API**:
```go
package scripttest

type CoverageEngine struct { ... }

func NewCoverageEngine(opts ...Option) *CoverageEngine
func (e *CoverageEngine) RunScript(name, content string) (*Result, error)
func (e *CoverageEngine) CollectCoverage() (*covutil.CoverageSet, error)
```

### 3. `integration/` - Integration Testing Framework
**Purpose**: Multi-level coverage collection for integration tests
**Key Features**:
- Multiple collection strategies (harness, overlay, both)
- Binary instrumentation and execution
- Coverage aggregation across test levels
- Environment variable control

**Public API**:
```go
package integration

type Framework struct { ... }
type CollectionMode string

const (
    ModeHarness CollectionMode = "harness"
    ModeOverlay CollectionMode = "overlay"
    ModeBoth    CollectionMode = "both"
    ModeAuto    CollectionMode = "auto"
)

func NewFramework(mode CollectionMode) *Framework
func (f *Framework) RunTest(test TestSpec) (*Result, error)
```

## Implementation Strategy

### Phase 1: Core Packages (High Priority)
1. Create `synthetic/` package with core synthetic coverage functionality
2. Create `scripttest/` package with enhanced scripttest integration
3. Create `integration/` package with multi-level coverage framework

### Phase 2: Promote Internal Packages (Medium Priority)
1. Promote `covtree/` from internal to public package
2. Promote `covforest/` from internal to public package
3. Promote `overlays/` from internal to public package (if broadly useful)

### Phase 3: Refactoring and Polish (Lower Priority)
1. Refactor existing code to use new package structure
2. Update all tests to use new packages
3. Create comprehensive examples and documentation
4. Update command-line tools to use new packages

## Design Principles

### 1. **Minimal Dependencies**
Each package should have minimal external dependencies and clear boundaries.

### 2. **Clear Interfaces**
Public APIs should be simple, intuitive, and well-documented.

### 3. **Composability**
Packages should work well together while remaining independent.

### 4. **Backward Compatibility**
Existing functionality should remain available during the transition.

### 5. **Test Coverage**
Each package should have comprehensive test coverage.

## Benefits

### For Users
- **Focused Functionality**: Import only what you need
- **Clear Documentation**: Each package has specific purpose
- **Easier Integration**: Well-defined APIs for specific use cases

### For Maintainers
- **Better Organization**: Clear separation of concerns
- **Easier Testing**: Isolated functionality with clear boundaries
- **Simpler Dependencies**: Reduced coupling between components

### For Community
- **Reusability**: Packages can be used independently
- **Contribution**: Clear areas for community contributions
- **Adoption**: Lower barrier to adoption for specific use cases

## Migration Path

### 1. Create New Packages
Start with empty packages that expose the desired APIs.

### 2. Gradual Migration
Move functionality piece by piece while maintaining backward compatibility.

### 3. Update Imports
Update existing code to use new package structure.

### 4. Deprecation
Mark old APIs as deprecated with clear migration paths.

### 5. Cleanup
Remove deprecated code after appropriate deprecation period.

## Success Metrics

1. **Package Independence**: Each package can be used standalone
2. **API Clarity**: Public APIs are intuitive and well-documented
3. **Test Coverage**: >90% test coverage for all public packages
4. **Community Adoption**: Evidence of external usage
5. **Maintenance Ease**: Simpler development and debugging

This architecture will make the covutil project more accessible, maintainable, and valuable to the broader Go community.