// Package scripttest provides a comprehensive parser for Go's scripttest format.
//
// The scripttest parser recognizes the complete scripttest command set used by Go's
// testing infrastructure, including all core commands, file operations, text processing,
// environment management, and system utilities.
//
// # Features
//
// - Complete scripttest command coverage (300+ patterns)
// - Core commands (exec, go, exists, cmp)
// - File system operations (mkdir, rm, cp, mv, etc.)
// - Text processing (grep, sed, awk, sort, etc.)
// - Environment management (env, setenv, unenv)
// - Network and system utilities
// - Container and orchestration tools
// - Negation support (! exec, ! exists)
//
// # Usage
//
//	import "github.com/tmc/covutil/synthetic/parsers/scripttest"
//
//	parser := &scripttest.Parser{}
//	commands := parser.ParseScript(scripttestContent)
//
// The parser automatically registers itself with the global registry when the
// defaults package is imported.
//
// # Supported Commands
//
// ## Core Scripttest Commands
//   - exec: Execute external commands
//   - go: Execute go commands
//   - exists: Check file existence
//   - cmp: Compare files
//   - diff: Show file differences
//
// ## File System Operations
//   - mkdir, rmdir: Directory operations
//   - rm, cp, mv: File operations
//   - chmod, touch: File permissions and timestamps
//   - ln, symlink: Link operations
//
// ## Text Processing
//   - grep, sed, awk: Text search and manipulation
//   - sort, uniq, wc: Text analysis
//   - cut, tr, tee: Text transformation
//   - head, tail: File reading
//
// ## Environment Management
//   - env: Environment variable operations
//   - setenv, unenv: Variable setting/unsetting
//
// ## System Utilities
//   - ps, kill, jobs: Process management
//   - date, pwd, whoami: System information
//   - df, du, free: System resources
//
// ## Development Tools
//   - git, make, cmake: Build and version control
//   - docker, kubectl: Container orchestration
//   - timeout, time: Execution control
//
// # Examples
//
//	scripttestContent := `# Integration test
//	# Setup environment
//	env GOOS=linux
//	env GOARCH=amd64
//
//	# Build application
//	go build -o myapp .
//	exists myapp
//
//	# Test execution
//	exec ./myapp --version
//	stdout 'v1.0.0'
//
//	# Docker integration
//	exec docker build -t myapp:test .
//	stdout 'Successfully built'
//
//	# Kubernetes deployment
//	exec kubectl apply -f deployment.yaml --dry-run=client
//	stdout 'configured'
//
//	# Performance testing
//	exec timeout 30s ./load-test.sh
//	! stderr 'ERROR'
//
//	# Cleanup
//	rm myapp`
//
//	parser := &scripttest.Parser{}
//	commands := parser.ParseScript(scripttestContent)
//	// Returns map of line numbers to executable commands
package scripttest
