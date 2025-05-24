// Package defaults registers all built-in parsers with the default registry.
package defaults

import (
	"github.com/tmc/covutil/synthetic/parsers"
	"github.com/tmc/covutil/synthetic/parsers/bash"
	"github.com/tmc/covutil/synthetic/parsers/gotemplate"
	"github.com/tmc/covutil/synthetic/parsers/python"
	"github.com/tmc/covutil/synthetic/parsers/scripttest"
	"github.com/tmc/covutil/synthetic/parsers/shell"
)

// init registers all default parsers
func init() {
	RegisterDefaults()
}

// RegisterDefaults registers all built-in parsers with the default registry
func RegisterDefaults() {
	parsers.Register(&bash.Parser{})
	parsers.Register(&shell.Parser{})
	parsers.Register(&python.Parser{})
	parsers.Register(&gotemplate.Parser{})
	parsers.Register(&scripttest.Parser{})
}

// MustRegisterDefaults registers all built-in parsers and panics on error
func MustRegisterDefaults() {
	if err := parsers.Register(&bash.Parser{}); err != nil {
		panic(err)
	}
	if err := parsers.Register(&shell.Parser{}); err != nil {
		panic(err)
	}
	if err := parsers.Register(&python.Parser{}); err != nil {
		panic(err)
	}
	if err := parsers.Register(&gotemplate.Parser{}); err != nil {
		panic(err)
	}
	if err := parsers.Register(&scripttest.Parser{}); err != nil {
		panic(err)
	}
}
