// Package python provides a parser for Python scripts.
package python

import (
	"regexp"
	"strings"
)

// Parser handles Python scripts
type Parser struct{}

// Name returns the parser name
func (p *Parser) Name() string {
	return "python"
}

// Extensions returns supported file extensions
func (p *Parser) Extensions() []string {
	return []string{".py", "python", "py"}
}

// Description returns parser description
func (p *Parser) Description() string {
	return "Python 3 syntax including classes, decorators, imports"
}

// ParseScript analyzes Python script content and identifies executable lines
func (p *Parser) ParseScript(content string) map[int]string {
	lines := strings.Split(content, "\n")
	commands := make(map[int]string)

	inMultilineString := false
	stringDelimiter := ""

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		// Handle multiline strings
		if inMultilineString {
			if strings.Contains(line, stringDelimiter) {
				inMultilineString = false
				stringDelimiter = ""
			}
			continue
		}

		// Check for multiline string start
		if strings.Contains(trimmed, `"""`) || strings.Contains(trimmed, "'''") {
			if strings.Contains(trimmed, `"""`) {
				stringDelimiter = `"""`
			} else {
				stringDelimiter = "'''"
			}
			// Check if it's a single-line triple quote
			parts := strings.Split(trimmed, stringDelimiter)
			if len(parts) < 3 {
				inMultilineString = true
			}
		}

		if p.IsExecutable(trimmed) {
			commands[lineNum] = trimmed
		}
	}

	return commands
}

// IsExecutable determines if a line is executable Python code
func (p *Parser) IsExecutable(line string) bool {
	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return false
	}

	// Skip docstrings
	if strings.HasPrefix(line, `"""`) || strings.HasPrefix(line, "'''") {
		return false
	}

	// Python patterns
	patterns := []string{
		`^import\s+`,                         // import statement
		`^from\s+.*\s+import\s+`,             // from import statement
		`^def\s+[a-zA-Z_][a-zA-Z0-9_]*\s*\(`, // function definition
		`^class\s+[a-zA-Z_][a-zA-Z0-9_]*`,    // class definition
		`^if\s+`,                             // if statement
		`^elif\s+`,                           // elif statement
		`^else\s*:`,                          // else statement
		`^for\s+`,                            // for loop
		`^while\s+`,                          // while loop
		`^try\s*:`,                           // try statement
		`^except\s*`,                         // except statement
		`^finally\s*:`,                       // finally statement
		`^with\s+`,                           // with statement
		`^assert\s+`,                         // assert statement
		`^return\s*`,                         // return statement
		`^yield\s*`,                          // yield statement
		`^break\s*$`,                         // break statement
		`^continue\s*$`,                      // continue statement
		`^pass\s*$`,                          // pass statement
		`^raise\s*`,                          // raise statement
		`^del\s+`,                            // del statement
		`^global\s+`,                         // global statement
		`^nonlocal\s+`,                       // nonlocal statement
		`^lambda\s+`,                         // lambda expression
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*=`,        // variable assignment
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*\+=`,      // augmented assignment
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*-=`,       // augmented assignment
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*\*=`,      // augmented assignment
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*/=`,       // augmented assignment
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*//=`,      // augmented assignment
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*%=`,       // augmented assignment
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*\*\*=`,    // augmented assignment
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*&=`,       // augmented assignment
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*\|=`,      // augmented assignment
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*\^=`,      // augmented assignment
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*<<=`,      // augmented assignment
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*>>=`,      // augmented assignment
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*\(`,       // function call
		`^print\s*\(`,                        // print function
		`^len\s*\(`,                          // built-in functions
		`^str\s*\(`,                          // built-in functions
		`^int\s*\(`,                          // built-in functions
		`^float\s*\(`,                        // built-in functions
		`^bool\s*\(`,                         // built-in functions
		`^list\s*\(`,                         // built-in functions
		`^dict\s*\(`,                         // built-in functions
		`^tuple\s*\(`,                        // built-in functions
		`^set\s*\(`,                          // built-in functions
		`^open\s*\(`,                         // file operations
		`^range\s*\(`,                        // range function
		`^enumerate\s*\(`,                    // enumerate function
		`^zip\s*\(`,                          // zip function
		`^map\s*\(`,                          // map function
		`^filter\s*\(`,                       // filter function
		`^sorted\s*\(`,                       // sorted function
		`^sum\s*\(`,                          // sum function
		`^min\s*\(`,                          // min function
		`^max\s*\(`,                          // max function
		`^abs\s*\(`,                          // abs function
		`^round\s*\(`,                        // round function
		`^@[a-zA-Z_][a-zA-Z0-9_]*`,           // decorator
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}

	return false
}
