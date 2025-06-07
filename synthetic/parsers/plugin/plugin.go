// Package plugin provides a plugin system for optional parsers
package plugin

import (
	"fmt"
	"log"
	"plugin"
	"sync"

	"github.com/tmc/covutil/synthetic/parsers"
)

// Loader manages dynamic loading of parser plugins
type Loader struct {
	mu      sync.RWMutex
	loaded  map[string]*plugin.Plugin
	parsers map[string]parsers.Parser
}

// NewLoader creates a new plugin loader
func NewLoader() *Loader {
	return &Loader{
		loaded:  make(map[string]*plugin.Plugin),
		parsers: make(map[string]parsers.Parser),
	}
}

// Load attempts to load a parser plugin from the given path
func (l *Loader) Load(path string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if already loaded
	if _, exists := l.loaded[path]; exists {
		return fmt.Errorf("plugin %s already loaded", path)
	}

	// Load the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to load plugin %s: %w", path, err)
	}

	// Look for the required symbols
	newParserSym, err := p.Lookup("NewParser")
	if err != nil {
		return fmt.Errorf("plugin %s missing NewParser function: %w", path, err)
	}

	// Try to cast to expected function signature
	newParserFunc, ok := newParserSym.(func() parsers.Parser)
	if !ok {
		return fmt.Errorf("plugin %s: NewParser has wrong signature", path)
	}

	// Create parser instance
	parser := newParserFunc()
	if parser == nil {
		return fmt.Errorf("plugin %s: NewParser returned nil", path)
	}

	// Store the plugin and parser
	l.loaded[path] = p
	l.parsers[parser.Name()] = parser

	// Register with the default registry
	if err := parsers.Register(parser); err != nil {
		// Don't fail completely, just log the error
		log.Printf("Warning: failed to register parser %s: %v", parser.Name(), err)
	}

	return nil
}

// LoadDir loads all plugins from a directory
func (l *Loader) LoadDir(dir string) error {
	// This would scan the directory for .so files and load them
	// Implementation omitted for brevity
	return fmt.Errorf("LoadDir not implemented")
}

// GetParser retrieves a loaded parser by name
func (l *Loader) GetParser(name string) (parsers.Parser, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	parser, exists := l.parsers[name]
	return parser, exists
}

// ListLoaded returns all loaded parsers
func (l *Loader) ListLoaded() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	names := make([]string, 0, len(l.parsers))
	for name := range l.parsers {
		names = append(names, name)
	}
	return names
}

// PluginInterface defines the interface that parser plugins must implement
type PluginInterface interface {
	// NewParser creates a new parser instance
	NewParser() parsers.Parser
}

// DefaultLoader is the global plugin loader
var DefaultLoader = NewLoader()

// Load is a convenience function to load a plugin with the default loader
func Load(path string) error {
	return DefaultLoader.Load(path)
}
