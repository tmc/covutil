package parsers_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/tmc/covutil/synthetic/parsers"
)

// YAMLParser demonstrates how to create a custom parser
type YAMLParser struct{}

func (p *YAMLParser) Name() string {
	return "yaml"
}

func (p *YAMLParser) Extensions() []string {
	return []string{".yaml", ".yml", "yaml"}
}

func (p *YAMLParser) Description() string {
	return "YAML configuration files"
}

func (p *YAMLParser) ParseScript(content string) map[int]string {
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

func (p *YAMLParser) IsExecutable(line string) bool {
	// Skip comments and empty lines
	if line == "" || strings.HasPrefix(line, "#") {
		return false
	}

	// Track key-value pairs and list items
	patterns := []string{
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*:`, // YAML key
		`^\s*-\s+`,                    // YAML list item
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}

	return false
}

func TestCustomParser(t *testing.T) {
	// Create a new registry for this test
	registry := parsers.NewRegistry()

	// Register the custom YAML parser
	yamlParser := &YAMLParser{}
	err := registry.Register(yamlParser)
	if err != nil {
		t.Fatalf("Failed to register YAML parser: %v", err)
	}

	// Test retrieval by name
	parser, exists := registry.Get("yaml")
	if !exists {
		t.Fatal("YAML parser not found by name")
	}
	if parser.Name() != "yaml" {
		t.Errorf("Expected name 'yaml', got '%s'", parser.Name())
	}

	// Test retrieval by extension
	parser, exists = registry.Get("yml")
	if !exists {
		t.Fatal("YAML parser not found by extension")
	}

	// Test parsing
	yamlContent := `# Configuration
name: myapp
version: 1.0.0
features:
  - auth
  - logging`

	commands := parser.ParseScript(yamlContent)
	// Should find: name:, version:, features:, - auth, - logging
	if len(commands) != 5 {
		t.Errorf("Expected 5 commands, got %d", len(commands))
	}

	// Verify specific lines
	expectedCommands := map[int]string{
		2: "name: myapp",
		3: "version: 1.0.0",
		4: "features:",
		5: "- auth",
		6: "- logging",
	}

	for lineNum, expectedCmd := range expectedCommands {
		if commands[lineNum] != expectedCmd {
			t.Errorf("Line %d: expected '%s', got '%s'", lineNum, expectedCmd, commands[lineNum])
		}
	}
}

// TestParser implements the Parser interface for testing
type TestParser struct{}

func (p *TestParser) Name() string {
	return "test"
}

func (p *TestParser) Extensions() []string {
	return []string{".test"}
}

func (p *TestParser) Description() string {
	return "Test parser"
}

func (p *TestParser) ParseScript(content string) map[int]string {
	return map[int]string{1: "test command"}
}

func (p *TestParser) IsExecutable(line string) bool {
	return line == "test"
}

func TestParserRegistry(t *testing.T) {
	// Create a new registry for this test
	registry := parsers.NewRegistry()

	// Create a simple test parser
	testParser := &TestParser{}

	// Test registration
	err := registry.Register(testParser)
	if err != nil {
		t.Fatalf("Failed to register parser: %v", err)
	}

	// Test duplicate registration
	err = registry.Register(testParser)
	if err == nil {
		t.Error("Expected error when registering duplicate parser")
	}

	// Test listing
	parserList := registry.List()
	if len(parserList) != 1 {
		t.Errorf("Expected 1 parser, got %d", len(parserList))
	}

	// Test registered types
	types := registry.RegisteredTypes()
	if len(types) != 1 {
		t.Errorf("Expected 1 type, got %d", len(types))
	}
}
