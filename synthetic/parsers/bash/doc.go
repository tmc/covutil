// Package bash provides a comprehensive parser for Bash scripts with advanced syntax support.
//
// The bash parser recognizes advanced bash-specific features beyond standard POSIX shell,
// including bash builtins, test constructs, process substitution, arrays, and here documents.
//
// # Features
//
// - Function definitions and calls
// - Here documents (<<EOF)
// - Advanced test constructs ([[ ]] vs [ ])
// - Bash-specific builtins (declare, local, shopt)
// - Process substitution and command substitution
// - Arrays and associative arrays
// - Comprehensive command recognition
//
// # Usage
//
//	import "github.com/tmc/covutil/synthetic/parsers/bash"
//
//	parser := &bash.Parser{}
//	commands := parser.ParseScript(bashScript)
//
// The parser automatically registers itself with the global registry when the
// defaults package is imported.
//
// # Recognized Patterns
//
// The parser identifies the following as executable:
//
//   - Variable assignments (VAR=value)
//   - Command executions with arguments
//   - Control structures (if, for, while, case)
//   - Function definitions and calls
//   - Bash test constructs ([[ ]] and [ ])
//   - Builtins (export, declare, local, readonly, etc.)
//   - I/O redirection and pipes
//   - Process control (kill, jobs, bg, fg)
//   - Directory stack operations (pushd, popd, dirs)
//
// # Examples
//
//	bashScript := `#!/bin/bash
//	# Advanced bash features
//	declare -a ARRAY=("one" "two" "three")
//
//	function process_items() {
//	    local item
//	    for item in "${ARRAY[@]}"; do
//	        if [[ "$item" =~ ^[a-z]+$ ]]; then
//	            echo "Processing: $item"
//	        fi
//	    done
//	}
//
//	# Here document
//	cat <<EOF > config.txt
//	Setting: value
//	EOF
//
//	process_items`
//
//	parser := &bash.Parser{}
//	commands := parser.ParseScript(bashScript)
//	// Returns map of line numbers to executable commands
package bash
