// Package parsers provides a modular, extensible architecture for script parsing
// in the synthetic coverage system.
//
// The parsers package defines the common Parser interface and Registry system that
// enables easy extension of script coverage tracking to new file types and languages.
//
// # Architecture
//
// The package uses a registry-based system where parsers implement a common interface
// and register themselves for discovery and use by ScriptTrackers.
//
//	interface Parser {
//	    Name() string
//	    Extensions() []string
//	    Description() string
//	    ParseScript(content string) map[int]string
//	    IsExecutable(line string) bool
//	}
//
// # Registry System
//
// The Registry manages parser registration and lookup:
//
//	// Global registry used by default
//	parsers.Register(myParser)
//	parser, exists := parsers.Get("mytype")
//	allParsers := parsers.List()
//
//	// Custom registry for isolated environments
//	registry := parsers.NewRegistry()
//	tracker := synthetic.NewScriptTrackerWithRegistry(registry)
//
// # Built-in Parsers
//
// The package includes several built-in parser subpackages:
//
//   - bash: Advanced bash syntax with functions, arrays, here docs
//   - shell: POSIX shell compatibility
//   - python: Python 3 syntax including classes, decorators, imports
//   - gotemplate: Go template directives and functions
//   - scripttest: Go's scripttest format (300+ commands)
//
// # Usage Examples
//
// ## Using Built-in Parsers
//
//	import _ "github.com/tmc/covutil/synthetic/parsers/defaults"
//
//	tracker := synthetic.NewScriptTracker()
//	err := tracker.ParseAndTrack(bashScript, "script.sh", "bash", "test")
//
// ## Creating Custom Parsers
//
//	type DockerfileParser struct{}
//
//	func (p *DockerfileParser) Name() string {
//	    return "dockerfile"
//	}
//
//	func (p *DockerfileParser) Extensions() []string {
//	    return []string{".dockerfile", "Dockerfile"}
//	}
//
//	func (p *DockerfileParser) Description() string {
//	    return "Docker build files"
//	}
//
//	func (p *DockerfileParser) ParseScript(content string) map[int]string {
//	    // Implementation...
//	}
//
//	func (p *DockerfileParser) IsExecutable(line string) bool {
//	    // Implementation...
//	}
//
//	// Register globally
//	parsers.Register(&DockerfileParser{})
//
// ## Parser Discovery
//
//	// List all registered parsers
//	allParsers := parsers.List()
//	for name, parser := range allParsers {
//	    fmt.Printf("%s: %s\n", name, parser.Description())
//	}
//
//	// Get parser by name or extension
//	parser, exists := parsers.Get("dockerfile")
//	parser, exists = parsers.Get("Dockerfile")  // Same result
//
//	// Get all registered types and their extensions
//	types := parsers.RegisteredTypes()
//	for name, exts := range types {
//	    fmt.Printf("%s supports: %v\n", name, exts)
//	}
//
// # Extensibility Patterns
//
// ## Package-based Parsers
//
// For complex parsers, create dedicated packages:
//
//	// parsers/myformat/myformat.go
//	package myformat
//
//	type Parser struct{}
//	// ... implement interface
//
//	func init() {
//	    parsers.Register(&Parser{})
//	}
//
//	// main.go
//	import _ "myproject/parsers/myformat"
//
// ## Plugin-style Registration
//
//	// Register parsers conditionally
//	if featureEnabled {
//	    parsers.Register(&AdvancedParser{})
//	}
//
// ## Testing Custom Parsers
//
//	func TestMyParser(t *testing.T) {
//	    parser := &MyParser{}
//
//	    content := `my script content`
//	    commands := parser.ParseScript(content)
//
//	    if len(commands) != expectedCount {
//	        t.Errorf("Expected %d commands, got %d", expectedCount, len(commands))
//	    }
//	}
//
// The parsers package provides a clean, extensible foundation for adding coverage
// tracking support to any script or configuration file format.
package parsers
