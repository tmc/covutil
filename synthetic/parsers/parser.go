// Package parsers provides a common interface and registry for script parsers.
package parsers

import (
	"fmt"
	"strings"
	"sync"
)

// Parser defines the interface for parsing different types of scripts.
type Parser interface {
	// Name returns the human-readable name of this parser
	Name() string

	// Extensions returns file extensions this parser can handle
	Extensions() []string

	// ParseScript analyzes script content and identifies executable lines
	ParseScript(content string) map[int]string

	// IsExecutable determines if a line is executable
	IsExecutable(line string) bool

	// Description returns a description of what this parser handles
	Description() string
}

// Registry manages available script parsers
type Registry struct {
	mu      sync.RWMutex
	parsers map[string]Parser
	aliases map[string]string // extension/type -> parser name mapping
}

// NewRegistry creates a new parser registry
func NewRegistry() *Registry {
	return &Registry{
		parsers: make(map[string]Parser),
		aliases: make(map[string]string),
	}
}

// Register adds a parser to the registry
func (r *Registry) Register(parser Parser) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := strings.ToLower(parser.Name())
	if _, exists := r.parsers[name]; exists {
		return fmt.Errorf("parser %s already registered", name)
	}

	r.parsers[name] = parser

	// Register extensions as aliases
	for _, ext := range parser.Extensions() {
		ext = strings.ToLower(strings.TrimPrefix(ext, "."))
		r.aliases[ext] = name
	}

	return nil
}

// Get retrieves a parser by name or extension
func (r *Registry) Get(nameOrExt string) (Parser, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := strings.ToLower(nameOrExt)

	// Try direct lookup first
	if parser, exists := r.parsers[key]; exists {
		return parser, true
	}

	// Try alias lookup
	if parserName, exists := r.aliases[key]; exists {
		if parser, exists := r.parsers[parserName]; exists {
			return parser, true
		}
	}

	return nil, false
}

// List returns all registered parsers
func (r *Registry) List() map[string]Parser {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]Parser)
	for name, parser := range r.parsers {
		result[name] = parser
	}
	return result
}

// RegisteredTypes returns all registered parser names and their extensions
func (r *Registry) RegisteredTypes() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]string)
	for name, parser := range r.parsers {
		result[name] = parser.Extensions()
	}
	return result
}

// DefaultRegistry is the global parser registry
var DefaultRegistry = NewRegistry()

// Register is a convenience function to register with the default registry
func Register(parser Parser) error {
	return DefaultRegistry.Register(parser)
}

// Get is a convenience function to get from the default registry
func Get(nameOrExt string) (Parser, bool) {
	return DefaultRegistry.Get(nameOrExt)
}

// List is a convenience function to list from the default registry
func List() map[string]Parser {
	return DefaultRegistry.List()
}
