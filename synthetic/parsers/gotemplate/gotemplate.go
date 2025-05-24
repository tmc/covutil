// Package gotemplate provides a parser for Go template files.
package gotemplate

import (
	"regexp"
	"strings"
)

// Parser handles Go template files
type Parser struct{}

// Name returns the parser name
func (p *Parser) Name() string {
	return "gotemplate"
}

// Extensions returns supported file extensions
func (p *Parser) Extensions() []string {
	return []string{".tmpl", "gotemplate", "tmpl"}
}

// Description returns parser description
func (p *Parser) Description() string {
	return "Go template directives and functions"
}

// ParseScript analyzes Go template content and identifies executable lines
func (p *Parser) ParseScript(content string) map[int]string {
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

// IsExecutable determines if a line contains executable Go template directives
func (p *Parser) IsExecutable(line string) bool {
	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "{{/*") {
		return false
	}

	// Skip HTML/text content without template directives
	if !strings.Contains(line, "{{") && !strings.Contains(line, "}}") {
		return false
	}

	// Go template patterns
	patterns := []string{
		`\{\{.*\}\}`,          // any template directive
		`\{\{\s*if\s+`,        // if directive
		`\{\{\s*else\s*\}\}`,  // else directive
		`\{\{\s*else\s+if\s+`, // else if directive
		`\{\{\s*end\s*\}\}`,   // end directive
		`\{\{\s*range\s+`,     // range directive
		`\{\{\s*with\s+`,      // with directive
		`\{\{\s*template\s+`,  // template directive
		`\{\{\s*block\s+`,     // block directive
		`\{\{\s*define\s+`,    // define directive
		`\{\{\s*\.\w+`,        // field access
		`\{\{\s*\$\w+`,        // variable access
		`\{\{\s*\w+\s+`,       // function call
		`\{\{\s*printf\s+`,    // printf function
		`\{\{\s*print\s+`,     // print function
		`\{\{\s*println\s+`,   // println function
		`\{\{\s*len\s+`,       // len function
		`\{\{\s*index\s+`,     // index function
		`\{\{\s*slice\s+`,     // slice function
		`\{\{\s*not\s+`,       // not function
		`\{\{\s*and\s+`,       // and function
		`\{\{\s*or\s+`,        // or function
		`\{\{\s*eq\s+`,        // eq function
		`\{\{\s*ne\s+`,        // ne function
		`\{\{\s*lt\s+`,        // lt function
		`\{\{\s*le\s+`,        // le function
		`\{\{\s*gt\s+`,        // gt function
		`\{\{\s*ge\s+`,        // ge function
		`\{\{\s*call\s+`,      // call function
		`\{\{\s*html\s+`,      // html function
		`\{\{\s*js\s+`,        // js function
		`\{\{\s*json\s+`,      // json function
		`\{\{\s*urlquery\s+`,  // urlquery function
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}

	return false
}
