// Package shell provides a parser for POSIX shell scripts.
package shell

import (
	"regexp"
	"strings"
)

// Parser handles POSIX shell scripts
type Parser struct{}

// Name returns the parser name
func (p *Parser) Name() string {
	return "shell"
}

// Extensions returns supported file extensions
func (p *Parser) Extensions() []string {
	return []string{".sh", "shell", "sh"}
}

// Description returns parser description
func (p *Parser) Description() string {
	return "POSIX shell compatibility"
}

// ParseScript analyzes shell script content and identifies executable lines
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

// IsExecutable determines if a line is executable shell code
func (p *Parser) IsExecutable(line string) bool {
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
		`^export\s+`,                     // export command
		`^cd\s+`,                         // cd command
		`^echo\s+`,                       // echo command
		`^test\s+`,                       // test command
		`^\[\s+`,                         // test construct
		`^return\s*`,                     // return statement
		`^exit\s*`,                       // exit statement
		`^shift\s*`,                      // shift command
		`^read\s+`,                       // read command
		`^exec\s+`,                       // exec command
		`^trap\s+`,                       // trap command
		`^wait\s*`,                       // wait command
		`^kill\s+`,                       // kill command
		`^pwd\s*`,                        // pwd command
		`^.*\|\|.*`,                      // logical OR
		`^.*\&\&.*`,                      // logical AND
		`^.*\|.*`,                        // pipe
		`^.*\>.*`,                        // redirection
		`^.*\<.*`,                        // input redirection
		`^.*\>\>.*`,                      // append redirection
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}

	return false
}
