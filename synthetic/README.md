# Synthetic Coverage Tracking

The `synthetic` package provides comprehensive functionality for generating synthetic coverage data for non-Go artifacts such as scripts, configuration files, templates, and other resources.

## Features

- **Modular Architecture**: Parsers organized in subpackages for better extensibility and maintainability
- **Multi-language Support**: Built-in parsers for Bash, Shell, Python, Go templates, and Scripttest
- **Parser Registry**: Central registry system for managing and discovering parsers
- **Easy Extension**: Simple interface for adding custom parsers for any script type
- **Go Coverage Integration**: Seamlessly integrates with Go's coverage ecosystem
- **Multiple Output Formats**: Binary pods, text profiles, and JSON (planned)
- **Real-world Testing**: Designed for integration testing and deployment validation

## Architecture

```
synthetic/
├── tracker.go              # Basic and script trackers
├── parsers/                # Parser subpackages
│   ├── parser.go           # Common interface and registry
│   ├── defaults/           # Auto-registers built-in parsers
│   ├── bash/              # Bash script parser
│   ├── shell/             # POSIX shell parser
│   ├── python/            # Python script parser
│   ├── gotemplate/        # Go template parser
│   └── scripttest/        # Scripttest format parser
└── examples_test.go       # Comprehensive usage examples
```

## Supported Script Types

| Script Type | Extensions | Description |
|-------------|------------|-------------|
| Bash | `.bash`, `bash` | Advanced bash syntax with functions, arrays, here docs |
| Shell | `.sh`, `shell` | POSIX shell compatibility |
| Python | `.py`, `python` | Python 3 syntax including classes, decorators, imports |
| Go Templates | `.tmpl`, `gotemplate` | Go template directives and functions |
| Scripttest | `.txt`, `.txtar`, `scripttest` | Go's scripttest format for integration tests |

## Quick Start

### Basic Tracking

```go
package main

import (
    "fmt"
    "log"
    "github.com/tmc/covutil/synthetic"
)

func main() {
    // Create a basic tracker
    tracker := synthetic.NewBasicTracker(
        synthetic.WithLabels(map[string]string{"test": "my-test"}),
    )

    // Track execution of artifact locations
    tracker.Track("my-script.sh", "5", true)  // Line 5 executed
    tracker.Track("my-script.sh", "10", false) // Line 10 not executed

    // Generate coverage data
    pod, err := tracker.GeneratePod()
    if err != nil {
        log.Fatal(err)
    }

    // Generate report
    report := tracker.GetReport()
    fmt.Println(report)
}
```

### Script Tracking

```go
// Create a script tracker
tracker := synthetic.NewScriptTracker(
    synthetic.WithTestName("integration-test"),
)

// Parse and set up tracking for a script
bashScript := `#!/bin/bash
echo "Starting deployment"
kubectl apply -f config.yaml
echo "Deployment complete"`

err := tracker.ParseAndTrack(bashScript, "deploy.sh", "bash", "deployment")
if err != nil {
    log.Fatal(err)
}

// Track execution of specific lines
tracker.TrackExecution("deploy.sh", "deployment", 2) // echo command
tracker.TrackExecution("deploy.sh", "deployment", 3) // kubectl command

// Generate report
report := tracker.GetReport()
fmt.Println(report)
```

## Language-Specific Examples

### Bash Scripts

```go
bashScript := `#!/bin/bash
# Setup script
export PATH=/usr/local/bin:$PATH
LOGFILE=/tmp/test.log

function cleanup() {
    rm -f "$LOGFILE"
}
trap cleanup EXIT

if [[ ! -f "$LOGFILE" ]]; then
    touch "$LOGFILE"
fi

echo "Starting test" >> "$LOGFILE"
./run-tests.sh 2>&1 | tee -a "$LOGFILE"`

tracker := synthetic.NewScriptTracker()
err := tracker.ParseAndTrack(bashScript, "setup.bash", "bash", "integration-test")

// Track execution
tracker.TrackExecution("setup.bash", "integration-test", 3) // export PATH
tracker.TrackExecution("setup.bash", "integration-test", 9) // if statement
```

Bash Parser Features:
- Function definitions and calls
- Here documents (`<<EOF`)
- Advanced test constructs (`[[ ]]` vs `[ ]`)
- Bash-specific builtins (`declare`, `local`, `shopt`)
- Process substitution and command substitution
- Arrays and associative arrays

### Python Scripts

```go
pythonScript := `#!/usr/bin/env python3
import sys
import json
from pathlib import Path

class TestRunner:
    def __init__(self, config_path):
        self.config = self.load_config(config_path)
    
    def load_config(self, path):
        with open(path) as f:
            return json.load(f)
    
    def run_tests(self):
        for test in self.config["tests"]:
            print(f"Running {test}")
        return True

if __name__ == "__main__":
    runner = TestRunner(sys.argv[1])
    runner.run_tests()`

tracker := synthetic.NewScriptTracker()
err := tracker.ParseAndTrack(pythonScript, "test_runner.py", "python", "python-test")

// Track execution
tracker.TrackExecution("test_runner.py", "python-test", 2) // import sys
tracker.TrackExecution("test_runner.py", "python-test", 6) // class definition
```

Python Parser Features:
- Import statements and from-imports
- Class and function definitions
- Decorators (`@property`, `@staticmethod`)
- Context managers (`with` statements)
- Exception handling (`try`/`except`/`finally`)
- Comprehensions and generators
- Multiline string handling

### Go Templates

```go
goTemplate := `{{/* Configuration template */}}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.Name}}
  namespace: {{.Namespace | default "default"}}
data:
  config.yaml: |
    {{if .Debug}}
    log_level: debug
    {{else}}
    log_level: info
    {{end}}
    
    servers:
    {{range .Servers}}
    - name: {{.Name}}
      url: {{.URL}}
    {{end}}`

tracker := synthetic.NewScriptTracker()
err := tracker.ParseAndTrack(goTemplate, "config.tmpl", "gotemplate", "template-test")

// Track template execution
tracker.TrackExecution("config.tmpl", "template-test", 5) // .Name
tracker.TrackExecution("config.tmpl", "template-test", 8) // if .Debug
```

Go Template Parser Features:
- Template directives (`if`, `range`, `with`)
- Template functions (`printf`, `len`, `index`)
- Variable assignments (`{{$var := .Value}}`)
- Template inclusion (`template`, `block`, `define`)
- Pipeline operations (`{{.Value | function}}`)
- Comparison functions (`eq`, `ne`, `lt`, `gt`)

### Scripttest Files

```go
scripttestContent := `# Integration test for go build
exec go version
stdout 'go version'

# Test building the project
go build -o test-binary .
exists test-binary

# Run the binary
exec ./test-binary --help
stdout 'Usage:'

# Cleanup
rm test-binary`

tracker := synthetic.NewScriptTracker()
err := tracker.ParseAndTrack(scripttestContent, "build_test.txt", "scripttest", "build-test")

// Track scripttest execution
tracker.TrackExecution("build_test.txt", "build-test", 2) // exec go version
tracker.TrackExecution("build_test.txt", "build-test", 5) // go build
```

Scripttest Parser Features:
- All Go scripttest commands (`exec`, `go`, `exists`, `cmp`)
- Negation support (`!exec`, `!exists`)
- Environment manipulation (`env`, `unenv`, `setenv`)
- File operations (`mkdir`, `rm`, `cp`, `mv`)
- Text processing (`grep`, `sed`, `awk`)
- Process control (`skip`, `stop`, `wait`)

## Multi-Language Support

Track multiple script types in a single test:

```go
tracker := synthetic.NewScriptTracker()

// Parse different script types
tracker.ParseAndTrack(bashContent, "deploy.sh", "bash", "deploy-test")
tracker.ParseAndTrack(pythonContent, "health.py", "python", "deploy-test")
tracker.ParseAndTrack(templateContent, "config.tmpl", "gotemplate", "deploy-test")
tracker.ParseAndTrack(scripttestContent, "test.txt", "scripttest", "deploy-test")

// Generate unified coverage report
report := tracker.GetReport()
pod, _ := tracker.GeneratePod()
```

## Custom Parsers

The synthetic package uses a modular parser architecture with subpackages for different script types. You can easily extend support to any script type by implementing the `parsers.Parser` interface:

```go
package main

import (
    "regexp"
    "strings"
    
    "github.com/tmc/covutil/synthetic"
    "github.com/tmc/covutil/synthetic/parsers"
)

// YAMLParser demonstrates creating a custom parser
type YAMLParser struct{}

func (p *YAMLParser) Name() string {
    return "yaml"
}

func (p *YAMLParser) Extensions() []string {
    return []string{".yaml", ".yml", "yaml"}
}

func (p *YAMLParser) Description() string {
    return "YAML configuration files"
}

func (p *YAMLParser) ParseScript(content string) map[int]string {
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

func (p *YAMLParser) IsExecutable(line string) bool {
    // Skip comments and empty lines
    if line == "" || strings.HasPrefix(line, "#") {
        return false
    }
    
    // Track key-value pairs and list items
    patterns := []string{
        `^[a-zA-Z_][a-zA-Z0-9_]*\s*:`,  // YAML key
        `^\s*-\s+`,                      // YAML list item
    }
    
    for _, pattern := range patterns {
        if matched, _ := regexp.MatchString(pattern, line); matched {
            return true
        }
    }
    
    return false
}

func main() {
    // Register the custom parser globally
    err := parsers.Register(&YAMLParser{})
    if err != nil {
        panic(err)
    }

    // Now it can be used by any ScriptTracker
    tracker := synthetic.NewScriptTracker()
    err = tracker.ParseAndTrack(yamlContent, "config.yaml", "yaml", "config-test")
    if err != nil {
        panic(err)
    }
}
```

### Parser Subpackages

The parsers are now organized in subpackages for better modularity:

```
synthetic/parsers/
├── parser.go          // Common interface and registry
├── defaults/          // Auto-registers all built-in parsers  
│   └── defaults.go
├── bash/             // Bash script parser
│   └── bash.go
├── shell/            // POSIX shell parser
│   └── shell.go
├── python/           // Python script parser
│   └── python.go
├── gotemplate/       // Go template parser
│   └── gotemplate.go
├── scripttest/       // Scripttest format parser
│   └── scripttest.go
└── example_test.go   // Examples and tests
```

### Creating Custom Parser Packages

For complex parsers, create a separate package:

```go
// parsers/dockerfile/dockerfile.go
package dockerfile

import (
    "regexp"
    "strings"
)

type Parser struct{}

func (p *Parser) Name() string {
    return "dockerfile"
}

func (p *Parser) Extensions() []string {
    return []string{".dockerfile", "Dockerfile"}
}

func (p *Parser) Description() string {
    return "Docker build files"
}

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

func (p *Parser) IsExecutable(line string) bool {
    // Skip comments and empty lines
    if line == "" || strings.HasPrefix(line, "#") {
        return false
    }
    
    // Dockerfile instruction patterns
    patterns := []string{
        `^FROM\s+`,      // FROM instruction
        `^RUN\s+`,       // RUN instruction
        `^COPY\s+`,      // COPY instruction
        `^ADD\s+`,       // ADD instruction
        `^ENV\s+`,       // ENV instruction
        `^EXPOSE\s+`,    // EXPOSE instruction
        `^CMD\s+`,       // CMD instruction
        `^ENTRYPOINT\s+`, // ENTRYPOINT instruction
        `^WORKDIR\s+`,   // WORKDIR instruction
        `^USER\s+`,      // USER instruction
        `^VOLUME\s+`,    // VOLUME instruction
        `^LABEL\s+`,     // LABEL instruction
    }
    
    for _, pattern := range patterns {
        if matched, _ := regexp.MatchString(`(?i)`+pattern, line); matched {
            return true
        }
    }
    
    return false
}

// Register automatically when imported
func init() {
    parsers.Register(&Parser{})
}
```

Then use it in your application:

```go
import (
    "github.com/tmc/covutil/synthetic"
    "github.com/tmc/covutil/synthetic/parsers"
    _ "your-project/parsers/dockerfile" // Auto-register
)

func main() {
    tracker := synthetic.NewScriptTracker()
    
    dockerfileContent := `FROM alpine:latest
RUN apk add --no-cache curl
COPY app /usr/local/bin/
EXPOSE 8080
CMD ["app"]`
    
    err := tracker.ParseAndTrack(dockerfileContent, "Dockerfile", "dockerfile", "build-test")
    if err != nil {
        panic(err)
    }
    
    // Track execution
    tracker.TrackExecution("Dockerfile", "build-test", 1) // FROM
    tracker.TrackExecution("Dockerfile", "build-test", 2) // RUN
    tracker.TrackExecution("Dockerfile", "build-test", 3) // COPY
    
    report := tracker.GetReport()
    fmt.Println(report)
}
```

## Integration with covutil

Combine synthetic coverage with real Go coverage:

```go
// Load real Go coverage data
realSet, err := covutil.LoadCoverageSetFromDirectory("real-coverage")
if err != nil {
    log.Fatal(err)
}

// Generate synthetic coverage for scripts
scriptTracker := synthetic.NewScriptTracker()
err = scriptTracker.ParseAndTrack(deployScript, "deploy.sh", "bash", "deployment")
if err != nil {
    log.Fatal(err)
}

syntheticPod, err := scriptTracker.GeneratePod()
if err != nil {
    log.Fatal(err)
}

// Combine real and synthetic coverage
combinedSet := &covutil.CoverageSet{
    Pods: append(realSet.Pods, syntheticPod),
}

// Generate unified HTML report
htmlReport, err := covutil.GenerateHTMLReport(combinedSet)
if err != nil {
    log.Fatal(err)
}

// Write to file
err = os.WriteFile("coverage-report.html", htmlReport, 0644)
```

## Output Formats

### Binary Pod Format
```go
pod, err := tracker.GeneratePod()
// Compatible with all covutil tools and Go coverage infrastructure
```

### Text Profile Format
```go
profile, err := tracker.GenerateProfile("text")
// Compatible with 'go tool cover' and other standard tools
// Format: file:startLine.startCol,endLine.endCol numStmt count
```

### JSON Format (Planned)
```go
jsonData, err := tracker.GenerateProfile("json")
// Machine-readable format for custom tooling and dashboards
```

## Scripttest Integration Examples

### Complete CI/CD Pipeline Test

The synthetic package provides comprehensive support for tracking scripttest execution with coverage. Here's a complete example of a CI/CD pipeline test:

```go
func TestScripttestIntegrationWorkflow(t *testing.T) {
    // Define a comprehensive scripttest for CI/CD pipeline
    scripttestContent := `# CI/CD Pipeline Integration Test
# Tests the complete build, test, and deployment workflow

# Environment setup
env GO111MODULE=on
env GOPROXY=direct
env GOSUMDB=off

# Clean workspace
rm -rf ./build
mkdir build
cd build

# Build the application
go build -o app .
exists app

# Test phase with coverage
go test -v ./...
stdout 'PASS'

# Coverage collection
go test -cover ./...
stdout 'coverage:'

# Integration test - start app and test endpoint
exec timeout 5s ./app &
sleep 2
exec curl -f http://localhost:8080/health
stdout 'healthy'

# Build for deployment
env GOOS=linux
env GOARCH=amd64
go build -o app-linux .
exists app-linux

# Create deployment artifacts
cat > Dockerfile <<EOF
FROM alpine:latest
COPY app-linux /app
EXPOSE 8080
CMD ["/app"]
EOF

# Validate deployment config
exec grep -q "testapp" k8s-deployment.yaml
exec grep -q "8080" k8s-deployment.yaml

# Cleanup
cd ..
rm -rf build`

    // Set up synthetic coverage tracking
    tracker := synthetic.NewScriptTracker(
        synthetic.WithTestName("ci-cd-pipeline"),
        synthetic.WithLabels(map[string]string{
            "pipeline":    "ci-cd",
            "environment": "test",
            "stage":       "integration",
        }),
    )

    err := tracker.ParseAndTrack(scripttestContent, "ci_cd_pipeline.txt", "scripttest", "ci-cd-pipeline")
    require.NoError(t, err)

    // In a real implementation, the scripttest runner would be instrumented
    // to call TrackExecution() for each command as it runs
    
    // Simulate execution tracking for successful pipeline run
    tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 4)  // env GO111MODULE
    tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 9)  // rm -rf ./build
    tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 10) // mkdir build
    // ... track other executed lines

    // Generate coverage report
    report := tracker.GetReport()
    t.Logf("Scripttest Coverage Report:\n%s", report)

    // Generate coverage pod for integration with covutil
    pod, err := tracker.GeneratePod()
    require.NoError(t, err)

    // Combine with Go coverage data
    goCoverage, err := covutil.LoadCoverageSetFromDirectory("go-coverage")
    require.NoError(t, err)

    fullCoverage := &covutil.CoverageSet{
        Pods: append(goCoverage.Pods, pod),
    }

    // Generate unified HTML report showing both Go and scripttest coverage
    htmlReport, err := covutil.GenerateHTMLReport(fullCoverage)
    require.NoError(t, err)
    
    os.WriteFile("unified-coverage.html", htmlReport, 0644)
}
```

### Enhanced Scripttest Runner Integration

To integrate synthetic coverage with the actual scripttest runner, you can instrument the execution like this:

```go
// Enhanced scripttest runner with coverage tracking
func RunScripttestWithCoverage(t *testing.T, scriptContent string, coverageTracker *synthetic.ScriptTracker) {
    // Parse the script for coverage tracking
    err := coverageTracker.ParseAndTrack(scriptContent, t.Name()+".txt", "scripttest", t.Name())
    if err != nil {
        t.Fatal(err)
    }

    // Run the scripttest with instrumentation
    engine := script.NewEngine()
    
    // Custom command that tracks execution
    engine.Cmds["exec"] = trackingExecCommand(coverageTracker, t.Name())
    engine.Cmds["go"] = trackingGoCommand(coverageTracker, t.Name())
    
    // Run the script
    scripttest.Run(t, engine, state, scriptContent)
    
    // Generate coverage report after execution
    report := coverageTracker.GetReport()
    t.Logf("Scripttest coverage:\n%s", report)
}

func trackingExecCommand(tracker *synthetic.ScriptTracker, testName string) script.Cmd {
    return script.Command(
        script.CmdUsage{Summary: "execute command with coverage tracking"},
        func(s *script.State, args ...string) (script.WaitFunc, error) {
            // Track that this exec command was executed
            lineNum := getCurrentLineNumber(s) // Implementation detail
            tracker.TrackExecution(testName+".txt", testName, lineNum)
            
            // Execute the actual command
            return script.Exec().Run(s, args...)
        },
    )
}
```

## Real-World Integration Example

Here's how to integrate synthetic coverage into a real project's test suite:

```go
func TestDeploymentWorkflow(t *testing.T) {
    // Set up synthetic coverage for deployment scripts
    deployTracker := synthetic.NewScriptTracker(
        synthetic.WithTestName("deployment"),
        synthetic.WithLabels(map[string]string{
            "environment": "staging",
            "version": "v1.2.3",
            "component": "deployment",
        }),
    )
    
    // Track multiple deployment artifacts
    err := deployTracker.ParseAndTrack(deployScript, "deploy.sh", "bash", "deployment")
    require.NoError(t, err)
    
    err = deployTracker.ParseAndTrack(configTemplate, "app.tmpl", "gotemplate", "deployment")
    require.NoError(t, err)
    
    err = deployTracker.ParseAndTrack(validationScript, "validate.txt", "scripttest", "deployment")
    require.NoError(t, err)
    
    // Execute deployment with coverage tracking
    output := executeWithTracking(t, "./deploy.sh", deployTracker)
    
    // Parse logs to track which template variables were used
    trackTemplateExecution(configTemplate, output, deployTracker)
    
    // Run validation scripttest with coverage
    runValidationWithCoverage(t, validationScript, deployTracker)
    
    // Generate comprehensive coverage
    deploymentPod, err := deployTracker.GeneratePod()
    require.NoError(t, err)
    
    // Load Go test coverage from GOCOVERDIR
    goCoverage, err := covutil.LoadCoverageSetFromDirectory(os.Getenv("GOCOVERDIR"))
    require.NoError(t, err)
    
    // Combine all coverage data
    fullCoverage := &covutil.CoverageSet{
        Pods: append(goCoverage.Pods, deploymentPod),
    }
    
    // Generate unified reports
    htmlReport, err := covutil.GenerateHTMLReport(fullCoverage)
    require.NoError(t, err)
    
    textProfile, err := deployTracker.GenerateProfile("text")
    require.NoError(t, err)
    
    // Save coverage artifacts
    os.WriteFile("deployment-coverage.html", htmlReport, 0644)
    os.WriteFile("deployment.cov", textProfile, 0644)
    
    // Assert coverage thresholds
    coverage := calculateOverallCoverage(fullCoverage)
    assert.Greater(t, coverage, 0.8, "Overall coverage should be >80%")
    
    // Assert specific artifact coverage
    report := deployTracker.GetReport()
    assert.Contains(t, report, "deploy.sh")
    assert.Contains(t, report, "app.tmpl")
    assert.Contains(t, report, "validate.txt")
}
```

## Performance and Best Practices

### Efficient Tracking
- Use `ScriptTracker` for multiple related scripts
- Batch execution tracking calls when possible
- Reset trackers between test runs to avoid memory buildup

### Memory Management
- Call `Reset()` on trackers after generating reports
- Use `WithEnabled(false)` to disable tracking in production
- Consider separate trackers for different test suites

### Error Handling
- Always check errors from `ParseAndTrack()`
- Verify parser registration for custom script types
- Use labels to identify coverage sources in combined reports

## API Reference

### Types

#### `Tracker` Interface
```go
type Tracker interface {
    Track(artifact, location string, executed bool)
    GeneratePod() (*covutil.Pod, error)
    GenerateProfile(format string) ([]byte, error)
    GetReport() string
    Reset()
}
```

#### `parsers.Parser` Interface
```go
type Parser interface {
    Name() string
    Extensions() []string
    Description() string
    ParseScript(content string) map[int]string
    IsExecutable(line string) bool
}
```

#### `parsers.Registry` Type
```go
type Registry struct {
    // Private fields
}

func NewRegistry() *Registry
func (r *Registry) Register(parser Parser) error
func (r *Registry) Get(nameOrExt string) (Parser, bool)
func (r *Registry) List() map[string]Parser
func (r *Registry) RegisteredTypes() map[string][]string
```

### Functions

#### `NewBasicTracker(options ...Option) *BasicTracker`
Creates a new basic tracker with configurable options.

#### `NewScriptTracker(options ...Option) *ScriptTracker`
Creates a new script tracker with access to all registered parsers.

#### `NewScriptTrackerWithRegistry(registry *parsers.Registry, options ...Option) *ScriptTracker`
Creates a new script tracker with a custom parser registry.

#### Parser Registry Functions

#### `parsers.Register(parser Parser) error`
Registers a parser with the global registry.

#### `parsers.Get(nameOrExt string) (Parser, bool)`
Retrieves a parser by name or extension from the global registry.

#### `parsers.List() map[string]Parser`
Returns all registered parsers from the global registry.

### Options

#### `WithLabels(labels map[string]string) Option`
Sets custom labels for the tracker.

#### `WithEnabled(enabled bool) Option`
Controls whether tracking is enabled.

#### `WithTestName(testName string) Option`
Sets a default test name for the tracker.

### Built-in Parsers

The following parsers are automatically registered when importing the package:

- `parsers/bash.Parser`: Advanced bash syntax support
- `parsers/shell.Parser`: POSIX shell compatibility  
- `parsers/python.Parser`: Python 3 language support
- `parsers/gotemplate.Parser`: Go template directives
- `parsers/scripttest.Parser`: Go scripttest format (300+ commands)

## Testing

Run the synthetic package tests:

```bash
go test ./synthetic/...
```

Run with coverage:

```bash
go test -cover ./synthetic/...
```

Run benchmarks:

```bash
go test -bench=. ./synthetic/...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

When adding new parsers:
1. Implement the `ScriptParser` interface
2. Add comprehensive tests
3. Update documentation with examples
4. Register the parser in `NewScriptTracker()`

## License

This package is part of the covutil project and follows the same license terms.