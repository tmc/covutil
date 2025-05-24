// Package scripttest provides a parser for Go scripttest format files.
package scripttest

import (
	"regexp"
	"strings"
)

// Parser handles Go scripttest format files
type Parser struct{}

// Name returns the parser name
func (p *Parser) Name() string {
	return "scripttest"
}

// Extensions returns supported file extensions
func (p *Parser) Extensions() []string {
	return []string{".txt", ".txtar", "scripttest"}
}

// Description returns parser description
func (p *Parser) Description() string {
	return "Go's scripttest format for integration tests"
}

// ParseScript analyzes scripttest content and identifies executable lines
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

// IsExecutable determines if a line is an executable scripttest command
func (p *Parser) IsExecutable(line string) bool {
	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return false
	}

	// Skip section headers (-- name --)
	if strings.HasPrefix(line, "--") && strings.HasSuffix(line, "--") {
		return false
	}

	// Scripttest command patterns (comprehensive list)
	patterns := []string{
		// Core scripttest commands
		`^exec\s+`,     // exec command
		`^!\s*exec\s+`, // negated exec command
		`^go\s+`,       // go command
		`^!\s*go\s+`,   // negated go command

		// File system operations
		`^cd\s+`,      // cd command
		`^mkdir\s+`,   // mkdir command
		`^rm\s+`,      // rm command
		`^rmdir\s+`,   // rmdir command
		`^cp\s+`,      // cp command
		`^mv\s+`,      // mv command
		`^chmod\s+`,   // chmod command
		`^touch\s+`,   // touch command
		`^ln\s+`,      // ln command
		`^symlink\s+`, // symlink command

		// File content operations
		`^echo\s+`,     // echo command
		`^cat\s+`,      // cat command
		`^grep\s+`,     // grep command
		`^!\s*grep\s+`, // negated grep command
		`^head\s+`,     // head command
		`^tail\s+`,     // tail command
		`^sort\s+`,     // sort command
		`^uniq\s+`,     // uniq command
		`^wc\s+`,       // wc command
		`^sed\s+`,      // sed command
		`^awk\s+`,      // awk command
		`^cut\s+`,      // cut command
		`^tr\s+`,       // tr command
		`^tee\s+`,      // tee command

		// File testing commands
		`^exists\s+`,     // exists command
		`^!\s*exists\s+`, // negated exists command
		`^cmp\s+`,        // cmp command
		`^!\s*cmp\s+`,    // negated cmp command
		`^diff\s+`,       // diff command
		`^!\s*diff\s+`,   // negated diff command
		`^cmpenv\s+`,     // cmpenv command
		`^!\s*cmpenv\s+`, // negated cmpenv command

		// Control flow
		`^skip\s+`,  // skip command
		`^stop\s+`,  // stop command
		`^wait\s+`,  // wait command
		`^sleep\s+`, // sleep command

		// Environment operations
		`^env\s+`,    // env command
		`^unenv\s+`,  // unenv command
		`^setenv\s+`, // setenv command

		// Archive operations
		`^tar\s+`,    // tar command
		`^gzip\s+`,   // gzip command
		`^gunzip\s+`, // gunzip command
		`^zip\s+`,    // zip command
		`^unzip\s+`,  // unzip command

		// Network operations
		`^curl\s+`, // curl command
		`^wget\s+`, // wget command
		`^ping\s+`, // ping command

		// Process operations
		`^ps\s+`,      // ps command
		`^kill\s+`,    // kill command
		`^killall\s+`, // killall command
		`^jobs\s+`,    // jobs command
		`^bg\s+`,      // bg command
		`^fg\s+`,      // fg command
		`^nohup\s+`,   // nohup command

		// System information
		`^date\s*`,   // date command
		`^pwd\s*`,    // pwd command
		`^whoami\s*`, // whoami command
		`^id\s*`,     // id command
		`^uptime\s*`, // uptime command
		`^df\s+`,     // df command
		`^du\s+`,     // du command
		`^free\s*`,   // free command
		`^uname\s+`,  // uname command

		// Text processing with redirection
		`^.*\s*>\s*`,    // output redirection
		`^.*\s*>>\s*`,   // append redirection
		`^.*\s*<\s*`,    // input redirection
		`^.*\s*\|\s*`,   // pipe
		`^.*\s*\|\|\s*`, // logical OR
		`^.*\s*&&\s*`,   // logical AND

		// Special scripttest patterns
		`^stdin:\s*$`,  // stdin marker (for here documents)
		`^stdout:\s*$`, // stdout marker
		`^stderr:\s*$`, // stderr marker

		// Command substitution and variables
		`^\$\w+`,                        // variable expansion
		`^.*\$\(.*\).*`,                 // command substitution
		`^.*` + "`" + `.*` + "`" + `.*`, // backtick command substitution

		// Conditional execution
		`^if\s+`,    // if statement
		`^then\s*$`, // then statement
		`^else\s*$`, // else statement
		`^elif\s+`,  // elif statement
		`^fi\s*$`,   // fi statement
		`^case\s+`,  // case statement
		`^esac\s*$`, // esac statement

		// Loops
		`^for\s+`,   // for loop
		`^while\s+`, // while loop
		`^until\s+`, // until loop
		`^do\s*$`,   // do statement
		`^done\s*$`, // done statement

		// Functions
		`^function\s+`,                        // function definition
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*\(\)\s*\{`, // function definition (alternative syntax)
		`^return\s*`,                          // return statement
		`^exit\s*`,                            // exit statement

		// Test constructs
		`^\[\s+`,   // test construct
		`^\[\[\s+`, // bash test construct
		`^test\s+`, // test command

		// Assignment and export
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*=`, // variable assignment
		`^export\s+`,                  // export command
		`^declare\s+`,                 // declare command
		`^local\s+`,                   // local command
		`^readonly\s+`,                // readonly command
		`^unset\s+`,                   // unset command

		// Debugging and tracing
		`^set\s+`,    // set command
		`^trap\s+`,   // trap command
		`^source\s+`, // source command
		`^\.\s+`,     // dot command (source)

		// Build tools and common utilities
		`^make\s+`,  // make command
		`^cmake\s+`, // cmake command
		`^ninja\s+`, // ninja command
		`^git\s+`,   // git command
		`^svn\s+`,   // svn command
		`^hg\s+`,    // mercurial command

		// Container and orchestration tools
		`^docker\s+`,  // docker command
		`^kubectl\s+`, // kubectl command
		`^helm\s+`,    // helm command
		`^podman\s+`,  // podman command

		// Testing and analysis tools
		`^timeout\s+`,  // timeout command
		`^time\s+`,     // time command
		`^strace\s+`,   // strace command
		`^ltrace\s+`,   // ltrace command
		`^valgrind\s+`, // valgrind command

		// Any line that starts with a known executable pattern
		`^[a-zA-Z_][a-zA-Z0-9_./\-]*\s+`, // command with arguments
		`^[a-zA-Z_][a-zA-Z0-9_./\-]*$`,   // simple command
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}

	return false
}
