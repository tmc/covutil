package synthetic

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tmc/covutil/synthetic/parsers"
	_ "github.com/tmc/covutil/synthetic/parsers/defaults" // Auto-register default parsers
)

// ScriptTracker provides specialized tracking for script execution
type ScriptTracker struct {
	*BasicTracker
	registry *parsers.Registry
}

// ScriptParser defines how to parse different types of scripts
type ScriptParser interface {
	// ParseScript analyzes script content and identifies executable lines
	ParseScript(content string) map[int]string

	// IsExecutable determines if a line is executable
	IsExecutable(line string) bool
}

// NewScriptTracker creates a new ScriptTracker with default parsers
func NewScriptTracker(options ...Option) *ScriptTracker {
	st := &ScriptTracker{
		BasicTracker: NewBasicTracker(options...),
		registry:     parsers.DefaultRegistry,
	}

	// Register default parsers
	st.RegisterParser("shell", &ShellScriptParser{})
	st.RegisterParser("bash", &ShellScriptParser{})
	st.RegisterParser("scripttest", &ScriptTestParser{})

	return st
}

// NewScriptTrackerWithRegistry creates a ScriptTracker with a custom parser registry
func NewScriptTrackerWithRegistry(registry *parsers.Registry, options ...Option) *ScriptTracker {
	return &ScriptTracker{
		BasicTracker: NewBasicTracker(options...),
		registry:     registry,
	}
}

// RegisterParser registers a parser for a specific script type
// Deprecated: Use parsers.Register() instead for global registration
func (st *ScriptTracker) RegisterParser(scriptType string, parser parsers.Parser) {
	// For backward compatibility, we'll create a local registry if needed
	if st.registry == parsers.DefaultRegistry {
		st.registry = parsers.NewRegistry()
		// Copy existing parsers
		for _, p := range parsers.DefaultRegistry.List() {
			st.registry.Register(p)
		}
	}
	st.registry.Register(parser)
}

// ParseAndTrack parses a script and sets up tracking for its executable lines
func (st *ScriptTracker) ParseAndTrack(scriptContent, scriptName, scriptType, testName string) error {
	parser, exists := st.registry.Get(scriptType)
	if !exists {
		return fmt.Errorf("no parser registered for script type: %s", scriptType)
	}

	// Parse the script to identify executable lines
	commands := parser.ParseScript(scriptContent)

	// Set up tracking for each executable line
	st.mu.Lock()
	defer st.mu.Unlock()

	key := fmt.Sprintf("%s:%s", testName, scriptName)
	if st.coverages[key] == nil {
		st.coverages[key] = &Coverage{
			ArtifactName:  scriptName,
			ExecutedLines: make(map[int]bool),
			Commands:      make(map[int]string),
			TestName:      testName,
		}
	}

	coverage := st.coverages[key]
	coverage.Commands = commands
	coverage.TotalLines = len(commands)

	return nil
}

// GetRegisteredParsers returns information about all registered parsers
func (st *ScriptTracker) GetRegisteredParsers() map[string][]string {
	return st.registry.RegisteredTypes()
}

// TrackExecution records that a specific line in a script was executed
func (st *ScriptTracker) TrackExecution(scriptName, testName string, lineNumber int) {
	key := fmt.Sprintf("%s:%s", testName, scriptName)

	st.mu.Lock()
	defer st.mu.Unlock()

	if coverage, exists := st.coverages[key]; exists {
		coverage.ExecutedLines[lineNumber] = true
	}
}

// ShellScriptParser handles shell/bash scripts
type ShellScriptParser struct{}

func (p *ShellScriptParser) Name() string {
	return "shell"
}

func (p *ShellScriptParser) Extensions() []string {
	return []string{".sh", ".bash"}
}

func (p *ShellScriptParser) Description() string {
	return "Parser for shell and bash scripts"
}

func (p *ShellScriptParser) ParseScript(content string) map[int]string {
	lines := strings.Split(content, "\n")
	commands := make(map[int]string)

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		if p.IsExecutable(trimmed) {
			commands[lineNum] = trimmed
		}
	}

	return commands
}

func (p *ShellScriptParser) IsExecutable(line string) bool {
	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return false
	}

	// Basic patterns for shell commands
	patterns := []string{
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*=`,    // Variable assignment
		`^[a-zA-Z_][a-zA-Z0-9_./\-]*\s+`, // Command with arguments
		`^[a-zA-Z_][a-zA-Z0-9_./\-]*$`,   // Simple command
		`^if\s+`,                         // if statement
		`^for\s+`,                        // for loop
		`^while\s+`,                      // while loop
		`^case\s+`,                       // case statement
		`^function\s+`,                   // function definition
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}

	return false
}

// ScriptTestParser handles Go scripttest format
type ScriptTestParser struct{}

func (p *ScriptTestParser) Name() string {
	return "scripttest"
}

func (p *ScriptTestParser) Extensions() []string {
	return []string{".txtar", ".txt"}
}

func (p *ScriptTestParser) Description() string {
	return "Parser for Go scripttest format files"
}

func (p *ScriptTestParser) ParseScript(content string) map[int]string {
	lines := strings.Split(content, "\n")
	commands := make(map[int]string)

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		if p.IsExecutable(trimmed) {
			commands[lineNum] = trimmed
		}
	}

	return commands
}

func (p *ScriptTestParser) IsExecutable(line string) bool {
	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return false
	}

	// Scripttest command patterns
	patterns := []string{
		`^exec\s+`,       // exec command
		`^!\s*exec\s+`,   // negated exec command
		`^go\s+`,         // go command
		`^cd\s+`,         // cd command
		`^mkdir\s+`,      // mkdir command
		`^cp\s+`,         // cp command
		`^rm\s+`,         // rm command
		`^echo\s+`,       // echo command
		`^cat\s+`,        // cat command
		`^grep\s+`,       // grep command
		`^exists\s+`,     // exists command
		`^!\s*exists\s+`, // negated exists command
		`^cmp\s+`,        // cmp command
		`^!\s*cmp\s+`,    // negated cmp command
		`^skip\s+`,       // skip command
		`^stop\s+`,       // stop command
		`^env\s+`,        // env command
		`^unenv\s+`,      // unenv command
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}

	return false
}

// Enhanced options for script tracking

// WithTestName sets a default test name for the tracker
func WithTestName(testName string) Option {
	return func(t *BasicTracker) {
		// Store test name in labels for easy access
		if t.labels == nil {
			t.labels = make(map[string]string)
		}
		t.labels["test_name"] = testName
	}
}
