// Package synthetic provides comprehensive functionality for generating synthetic coverage data
// for non-Go artifacts such as scripts, configuration files, templates, and other resources.
//
// The package uses a modular architecture with separate subpackages for different script parsers,
// making it highly extensible and maintainable.
//
// # Architecture Overview
//
// The package offers different types of trackers for various artifact types:
//
//   - BasicTracker: Generic tracker for any type of artifact
//   - ScriptTracker: Specialized tracker with access to registered parsers from subpackages
//
// # Parser Subpackages
//
// Script parsers are organized in subpackages under synthetic/parsers/:
//
//   - parsers/bash: Advanced bash syntax with functions, arrays, here docs
//   - parsers/shell: POSIX shell compatibility
//   - parsers/python: Python 3 syntax including classes, decorators, imports
//   - parsers/gotemplate: Go template directives and functions
//   - parsers/scripttest: Go's scripttest format for integration tests (300+ commands)
//   - parsers/defaults: Auto-registers all built-in parsers
//
// # Supported Script Types
//
// The ScriptTracker automatically detects and uses appropriate parsers for:
//
//   - Bash scripts (.bash, bash): Advanced bash syntax with functions, arrays, here docs
//   - Shell scripts (.sh, shell): POSIX shell compatibility
//   - Python scripts (.py, python): Python 3 syntax including classes, decorators, imports
//   - Go templates (.tmpl, gotemplate): Go template directives and functions
//   - Scripttest files (.txt, .txtar, scripttest): Go's scripttest format for integration tests
//
// # Basic Usage
//
//	// Create a basic tracker
//	tracker := synthetic.NewBasicTracker(
//		synthetic.WithLabels(map[string]string{"test": "my-test"}),
//	)
//
//	// Track execution of artifact locations
//	tracker.Track("my-script.sh", "5", true)  // Line 5 executed
//	tracker.Track("my-script.sh", "10", false) // Line 10 not executed
//
//	// Generate coverage data
//	pod, err := tracker.GeneratePod()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Write coverage data
//	err = covutil.WritePodToDirectory("coverage", pod)
//
// # Script Tracking
//
//	// Create a script tracker
//	tracker := synthetic.NewScriptTracker(
//		synthetic.WithTestName("integration-test"),
//	)
//
//	// Parse and set up tracking for a script
//	err := tracker.ParseAndTrack(scriptContent, "test.sh", "bash", "my-test")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Track execution of specific lines
//	tracker.TrackExecution("test.sh", "my-test", 5)
//	tracker.TrackExecution("test.sh", "my-test", 10)
//
//	// Generate report
//	report := tracker.GetReport()
//	fmt.Println(report)
//
// # Bash Script Example
//
//	bashScript := `#!/bin/bash
//	# Setup script
//	export PATH=/usr/local/bin:$PATH
//	LOGFILE=/tmp/test.log
//
//	function cleanup() {
//	    rm -f "$LOGFILE"
//	}
//	trap cleanup EXIT
//
//	if [[ ! -f "$LOGFILE" ]]; then
//	    touch "$LOGFILE"
//	fi
//
//	echo "Starting test" >> "$LOGFILE"
//	./run-tests.sh 2>&1 | tee -a "$LOGFILE"`
//
//	tracker := synthetic.NewScriptTracker()
//	err := tracker.ParseAndTrack(bashScript, "setup.bash", "bash", "integration-test")
//
//	// Simulate execution tracking
//	tracker.TrackExecution("setup.bash", "integration-test", 3) // export PATH
//	tracker.TrackExecution("setup.bash", "integration-test", 4) // LOGFILE assignment
//	tracker.TrackExecution("setup.bash", "integration-test", 9) // if statement
//	tracker.TrackExecution("setup.bash", "integration-test", 13) // echo command
//
// # Python Script Example
//
//	pythonScript := `#!/usr/bin/env python3
//	import sys
//	import json
//	from pathlib import Path
//
//	class TestRunner:
//	    def __init__(self, config_path):
//	        self.config = self.load_config(config_path)
//
//	    def load_config(self, path):
//	        with open(path) as f:
//	            return json.load(f)
//
//	    def run_tests(self):
//	        for test in self.config["tests"]:
//	            print(f"Running {test}")
//	            if not self.execute_test(test):
//	                return False
//	        return True
//
//	if __name__ == "__main__":
//	    runner = TestRunner(sys.argv[1])
//	    success = runner.run_tests()
//	    sys.exit(0 if success else 1)`
//
//	tracker := synthetic.NewScriptTracker()
//	err := tracker.ParseAndTrack(pythonScript, "test_runner.py", "python", "python-test")
//
//	// Track execution
//	tracker.TrackExecution("test_runner.py", "python-test", 2) // import sys
//	tracker.TrackExecution("test_runner.py", "python-test", 7) // class definition
//	tracker.TrackExecution("test_runner.py", "python-test", 20) // if __name__
//
// # Go Template Example
//
//	goTemplate := `{{/* Configuration template */}}
//	apiVersion: v1
//	kind: ConfigMap
//	metadata:
//	  name: {{.Name}}
//	  namespace: {{.Namespace | default "default"}}
//	data:
//	  config.yaml: |
//	    {{if .Debug}}
//	    log_level: debug
//	    {{else}}
//	    log_level: info
//	    {{end}}
//
//	    servers:
//	    {{range .Servers}}
//	    - name: {{.Name}}
//	      url: {{.URL}}
//	      {{if .TLS}}
//	      tls: true
//	      {{end}}
//	    {{end}}`
//
//	tracker := synthetic.NewScriptTracker()
//	err := tracker.ParseAndTrack(goTemplate, "config.tmpl", "gotemplate", "template-test")
//
//	// Track template execution
//	tracker.TrackExecution("config.tmpl", "template-test", 5) // .Name
//	tracker.TrackExecution("config.tmpl", "template-test", 8) // if .Debug
//	tracker.TrackExecution("config.tmpl", "template-test", 15) // range .Servers
//
// # Scripttest Example
//
//	scripttestContent := `# Integration test for go build
//	exec go version
//	stdout 'go version'
//
//	# Test building the project
//	go build -o test-binary .
//	exists test-binary
//
//	# Run the binary
//	exec ./test-binary --help
//	stdout 'Usage:'
//
//	# Test with invalid flag
//	! exec ./test-binary --invalid-flag
//	stderr 'unknown flag'
//
//	# Cleanup
//	rm test-binary`
//
//	tracker := synthetic.NewScriptTracker()
//	err := tracker.ParseAndTrack(scripttestContent, "build_test.txt", "scripttest", "build-test")
//
//	// Track scripttest execution
//	tracker.TrackExecution("build_test.txt", "build-test", 2) // exec go version
//	tracker.TrackExecution("build_test.txt", "build-test", 5) // go build
//	tracker.TrackExecution("build_test.txt", "build-test", 6) // exists test-binary
//
// # Advanced Features
//
// ## Multi-Language Support
//
// The ScriptTracker automatically detects and parses different script types:
//
//	tracker := synthetic.NewScriptTracker()
//
//	// Each script type is parsed with its specific parser
//	tracker.ParseAndTrack(bashContent, "deploy.bash", "bash", "deploy-test")
//	tracker.ParseAndTrack(pythonContent, "analyze.py", "python", "analysis-test")
//	tracker.ParseAndTrack(templateContent, "config.tmpl", "gotemplate", "config-test")
//	tracker.ParseAndTrack(scripttestContent, "integration.txt", "scripttest", "integration-test")
//
//	// Generate unified coverage report
//	report := tracker.GetReport()
//	pod, _ := tracker.GeneratePod()
//
// ## Parser-Specific Features
//
// ### Bash Parser Features:
//   - Function definitions and calls
//   - Here documents (<<EOF)
//   - Advanced test constructs ([[ ]] vs [ ])
//   - Bash-specific builtins (declare, local, shopt)
//   - Process substitution and command substitution
//   - Arrays and associative arrays
//
// ### Python Parser Features:
//   - Import statements and from-imports
//   - Class and function definitions
//   - Decorators (@property, @staticmethod)
//   - Context managers (with statements)
//   - Exception handling (try/except/finally)
//   - Comprehensions and generators
//   - Multiline string handling
//
// ### Go Template Parser Features:
//   - Template directives (if, range, with)
//   - Template functions (printf, len, index)
//   - Variable assignments ({{$var := .Value}})
//   - Template inclusion (template, block, define)
//   - Pipeline operations ({{.Value | function}})
//   - Comparison functions (eq, ne, lt, gt)
//
// ### Scripttest Parser Features:
//   - All Go scripttest commands (exec, go, exists, cmp)
//   - Negation support (! exec, ! exists)
//   - Environment manipulation (env, unenv, setenv)
//   - File operations (mkdir, rm, cp, mv)
//   - Text processing (grep, sed, awk)
//   - Process control (skip, stop, wait)
//
// # Integration with covutil
//
// The synthetic package seamlessly integrates with the main covutil package:
//
//	// Load real Go coverage data
//	realSet, err := covutil.LoadCoverageSetFromDirectory("real-coverage")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Generate synthetic coverage for scripts
//	scriptTracker := synthetic.NewScriptTracker()
//	err = scriptTracker.ParseAndTrack(deployScript, "deploy.sh", "bash", "deployment")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Simulate script execution tracking
//	scriptTracker.TrackExecution("deploy.sh", "deployment", 5)
//	scriptTracker.TrackExecution("deploy.sh", "deployment", 10)
//
//	syntheticPod, err := scriptTracker.GeneratePod()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Combine real and synthetic coverage
//	combinedSet := &covutil.CoverageSet{
//		Pods: append(realSet.Pods, syntheticPod),
//	}
//
//	// Generate unified HTML report
//	htmlReport, err := covutil.GenerateHTMLReport(combinedSet)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Write to file
//	err = os.WriteFile("coverage-report.html", htmlReport, 0644)
//
// # Custom Parsers and Extensibility
//
// The package uses a modular parser architecture with a central registry system.
// Parsers are organized in subpackages and can be easily extended.
//
// ## Parser Registry System
//
// The parsers.Registry manages parser registration and lookup:
//
//	import "github.com/tmc/covutil/synthetic/parsers"
//
//	// Global registry used by default
//	err := parsers.Register(myParser)
//	parser, exists := parsers.Get("mytype")
//	allParsers := parsers.List()
//
//	// Custom registry for isolated environments
//	registry := parsers.NewRegistry()
//	tracker := synthetic.NewScriptTrackerWithRegistry(registry)
//
// ## Implementing Custom Parsers
//
// Implement custom parsers by implementing the parsers.Parser interface:
//
//	import "github.com/tmc/covutil/synthetic/parsers"
//
//	// Example: YAML parser for tracking configuration changes
//	type YAMLParser struct{}
//
//	func (p *YAMLParser) Name() string {
//		return "yaml"
//	}
//
//	func (p *YAMLParser) Extensions() []string {
//		return []string{".yaml", ".yml", "yaml"}
//	}
//
//	func (p *YAMLParser) Description() string {
//		return "YAML configuration files"
//	}
//
//	func (p *YAMLParser) ParseScript(content string) map[int]string {
//		lines := strings.Split(content, "\n")
//		commands := make(map[int]string)
//
//		for i, line := range lines {
//			lineNum := i + 1
//			trimmed := strings.TrimSpace(line)
//
//			if p.IsExecutable(trimmed) {
//				commands[lineNum] = trimmed
//			}
//		}
//
//		return commands
//	}
//
//	func (p *YAMLParser) IsExecutable(line string) bool {
//		// Skip comments and empty lines
//		if line == "" || strings.HasPrefix(line, "#") {
//			return false
//		}
//
//		// Track key-value pairs and list items
//		patterns := []string{
//			`^[a-zA-Z_][a-zA-Z0-9_]*\s*:`,  // YAML key
//			`^\s*-\s+`,                      // YAML list item
//		}
//
//		for _, pattern := range patterns {
//			if matched, _ := regexp.MatchString(pattern, line); matched {
//				return true
//			}
//		}
//
//		return false
//	}
//
//	// Register globally for use by all ScriptTrackers
//	err := parsers.Register(&YAMLParser{})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Or register with a specific tracker (deprecated, but still supported)
//	tracker := synthetic.NewScriptTracker()
//	tracker.RegisterParser("yaml", &YAMLParser{})
//
//	err = tracker.ParseAndTrack(yamlContent, "config.yaml", "yaml", "config-test")
//	if err != nil {
//		log.Fatal(err)
//	}
//
// # Output Formats and Integration
//
// The package supports multiple output formats for maximum compatibility:
//
// The synthetic coverage data can be consumed by any tool that works with
// Go coverage data, including the go tool cover command and various coverage
// visualization tools.
package synthetic
