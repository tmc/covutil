// Package bash provides a parser for Bash scripts with advanced syntax support.
package bash

import (
	"regexp"
	"strings"
)

// Parser handles bash scripts with extended bash-specific features
type Parser struct{}

// Name returns the parser name
func (p *Parser) Name() string {
	return "bash"
}

// Extensions returns supported file extensions
func (p *Parser) Extensions() []string {
	return []string{".bash", "bash"}
}

// Description returns parser description
func (p *Parser) Description() string {
	return "Advanced bash syntax with functions, arrays, here docs"
}

// ParseScript analyzes bash script content and identifies executable lines
func (p *Parser) ParseScript(content string) map[int]string {
	lines := strings.Split(content, "\n")
	commands := make(map[int]string)

	inFunction := false
	inHereDoc := false
	hereDocMarker := ""

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		// Handle here documents
		if inHereDoc {
			if strings.TrimSpace(line) == hereDocMarker {
				inHereDoc = false
				hereDocMarker = ""
			}
			// Count here document content as executed if the here doc itself was tracked
			continue
		}

		// Check for here document start
		if strings.Contains(trimmed, "<<") {
			parts := strings.Split(trimmed, "<<")
			if len(parts) >= 2 {
				marker := strings.TrimSpace(parts[1])
				marker = strings.Trim(marker, "\"'")
				if marker != "" {
					inHereDoc = true
					hereDocMarker = marker
				}
			}
		}

		// Track function definitions
		if strings.Contains(trimmed, "function ") || strings.HasSuffix(trimmed, "() {") {
			inFunction = true
		}
		if trimmed == "}" && inFunction {
			inFunction = false
		}

		if p.IsExecutable(trimmed) {
			commands[lineNum] = trimmed
		}
	}

	return commands
}

// IsExecutable determines if a line is executable bash code
func (p *Parser) IsExecutable(line string) bool {
	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return false
	}

	// Bash-specific patterns (more comprehensive than shell)
	patterns := []string{
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*=`,         // Variable assignment
		`^[a-zA-Z_][a-zA-Z0-9_./\-]*\s+`,      // Command with arguments
		`^[a-zA-Z_][a-zA-Z0-9_./\-]*$`,        // Simple command
		`^if\s+`,                              // if statement
		`^elif\s+`,                            // elif statement
		`^else\s*$`,                           // else statement
		`^fi\s*$`,                             // fi statement
		`^for\s+`,                             // for loop
		`^while\s+`,                           // while loop
		`^until\s+`,                           // until loop
		`^do\s*$`,                             // do statement
		`^done\s*$`,                           // done statement
		`^case\s+`,                            // case statement
		`^esac\s*$`,                           // esac statement
		`^function\s+`,                        // function definition
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*\(\)\s*\{`, // function definition (alternative syntax)
		`^\[\[\s+`,                            // bash test construct
		`^\[\s+`,                              // test construct
		`^test\s+`,                            // test command
		`^source\s+`,                          // source command
		`^\.\s+`,                              // dot command (source)
		`^export\s+`,                          // export command
		`^declare\s+`,                         // declare command
		`^local\s+`,                           // local command
		`^readonly\s+`,                        // readonly command
		`^unset\s+`,                           // unset command
		`^return\s*`,                          // return statement
		`^exit\s*`,                            // exit statement
		`^break\s*`,                           // break statement
		`^continue\s*`,                        // continue statement
		`^shift\s*`,                           // shift command
		`^getopts\s+`,                         // getopts command
		`^read\s+`,                            // read command
		`^printf\s+`,                          // printf command
		`^echo\s+`,                            // echo command
		`^eval\s+`,                            // eval command
		`^exec\s+`,                            // exec command
		`^trap\s+`,                            // trap command
		`^wait\s*`,                            // wait command
		`^kill\s+`,                            // kill command
		`^jobs\s*`,                            // jobs command
		`^bg\s*`,                              // bg command
		`^fg\s*`,                              // fg command
		`^pushd\s+`,                           // pushd command
		`^popd\s*`,                            // popd command
		`^dirs\s*`,                            // dirs command
		`^alias\s+`,                           // alias command
		`^unalias\s+`,                         // unalias command
		`^type\s+`,                            // type command
		`^which\s+`,                           // which command
		`^command\s+`,                         // command command
		`^builtin\s+`,                         // builtin command
		`^enable\s+`,                          // enable command
		`^help\s*`,                            // help command
		`^history\s*`,                         // history command
		`^set\s+`,                             // set command
		`^shopt\s+`,                           // shopt command
		`^ulimit\s+`,                          // ulimit command
		`^umask\s*`,                           // umask command
		`^cd\s+`,                              // cd command
		`^pwd\s*`,                             // pwd command
		`^\}\s*$`,                             // closing brace
		`^.*\|\|.*`,                           // logical OR
		`^.*\&\&.*`,                           // logical AND
		`^.*\|.*`,                             // pipe
		`^.*\>.*`,                             // redirection
		`^.*\<.*`,                             // input redirection
		`^.*\>\>.*`,                           // append redirection
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}

	return false
}
