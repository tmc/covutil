package synthetic_test

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"testing"

	"github.com/tmc/covutil/synthetic"
)

// Example demonstrates basic usage of the synthetic tracker
func ExampleBasicTracker() {
	// Create a basic tracker
	tracker := synthetic.NewBasicTracker(
		synthetic.WithLabels(map[string]string{"test": "my-test"}),
	)

	// Track execution of artifact locations
	tracker.Track("my-script.sh", "5", true)   // Line 5 executed
	tracker.Track("my-script.sh", "10", false) // Line 10 not executed

	// Generate coverage data
	pod, err := tracker.GeneratePod()
	if err != nil {
		log.Fatal(err)
	}

	// Generate report
	report := tracker.GetReport()
	fmt.Println(report)

	// Pod contains synthetic coverage data compatible with covutil
	fmt.Printf("Pod type: %s\n", pod.Labels["type"])

	// Output:
	// === Synthetic Coverage Report ===
	//
	// Artifact: my-script.sh (Test: synthetic-test)
	//   Commands: 2 total, 1 executed (50.0%)
	//
	// Overall: 1/2 commands executed (50.0%)
	//
	// Pod type: synthetic
}

// Example demonstrates bash script tracking
func ExampleScriptTracker_bash() {
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
	if err != nil {
		log.Fatal(err)
	}

	// Simulate execution tracking
	tracker.TrackExecution("setup.bash", "integration-test", 3)  // export PATH
	tracker.TrackExecution("setup.bash", "integration-test", 4)  // LOGFILE assignment
	tracker.TrackExecution("setup.bash", "integration-test", 9)  // if statement
	tracker.TrackExecution("setup.bash", "integration-test", 13) // echo command

	report := tracker.GetReport()
	fmt.Println(report)

	// Output:
	// === Synthetic Coverage Report ===
	//
	// Artifact: setup.bash (Test: integration-test)
	//   Commands: 11 total, 4 executed (36.4%)
	//
	// Overall: 4/11 commands executed (36.4%)
}

// Example demonstrates Python script tracking
func ExampleScriptTracker_python() {
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
            if not self.execute_test(test):
                return False
        return True

if __name__ == "__main__":
    runner = TestRunner(sys.argv[1])
    success = runner.run_tests()
    sys.exit(0 if success else 1)`

	tracker := synthetic.NewScriptTracker()
	err := tracker.ParseAndTrack(pythonScript, "test_runner.py", "python", "python-test")
	if err != nil {
		log.Fatal(err)
	}

	// Track execution
	tracker.TrackExecution("test_runner.py", "python-test", 2)  // import sys
	tracker.TrackExecution("test_runner.py", "python-test", 7)  // class definition
	tracker.TrackExecution("test_runner.py", "python-test", 20) // if __name__

	report := tracker.GetReport()
	fmt.Println(report)

	// Output:
	// === Synthetic Coverage Report ===
	//
	// Artifact: test_runner.py (Test: python-test)
	//   Commands: 17 total, 3 executed (17.6%)
	//
	// Overall: 3/17 commands executed (17.6%)
}

// Example demonstrates Go template tracking
func ExampleScriptTracker_gotemplate() {
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
      {{if .TLS}}
      tls: true
      {{end}}
    {{end}}`

	tracker := synthetic.NewScriptTracker()
	err := tracker.ParseAndTrack(goTemplate, "config.tmpl", "gotemplate", "template-test")
	if err != nil {
		log.Fatal(err)
	}

	// Track template execution
	tracker.TrackExecution("config.tmpl", "template-test", 5)  // .Name
	tracker.TrackExecution("config.tmpl", "template-test", 8)  // if .Debug
	tracker.TrackExecution("config.tmpl", "template-test", 15) // range .Servers

	report := tracker.GetReport()
	fmt.Println(report)

	// Output:
	// === Synthetic Coverage Report ===
	//
	// Artifact: config.tmpl (Test: template-test)
	//   Commands: 11 total, 3 executed (27.3%)
	//
	// Overall: 3/11 commands executed (27.3%)
}

// Example demonstrates scripttest tracking
func ExampleScriptTracker_scripttest() {
	scripttestContent := `# Integration test for go build
exec go version
stdout 'go version'

# Test building the project
go build -o test-binary .
exists test-binary

# Run the binary
exec ./test-binary --help
stdout 'Usage:'

# Test with invalid flag
! exec ./test-binary --invalid-flag
stderr 'unknown flag'

# Cleanup
rm test-binary`

	tracker := synthetic.NewScriptTracker()
	err := tracker.ParseAndTrack(scripttestContent, "build_test.txt", "scripttest", "build-test")
	if err != nil {
		log.Fatal(err)
	}

	// Track scripttest execution
	tracker.TrackExecution("build_test.txt", "build-test", 2) // exec go version
	tracker.TrackExecution("build_test.txt", "build-test", 5) // go build
	tracker.TrackExecution("build_test.txt", "build-test", 6) // exists test-binary

	report := tracker.GetReport()
	fmt.Println(report)

	// Output:
	// === Synthetic Coverage Report ===
	//
	// Artifact: build_test.txt (Test: build-test)
	//   Commands: 9 total, 3 executed (33.3%)
	//
	// Overall: 3/9 commands executed (33.3%)
}

// Test demonstrates multi-language tracking
func TestMultiLanguageTracking(t *testing.T) {
	tracker := synthetic.NewScriptTracker()

	// Parse different script types
	bashContent := `#!/bin/bash
echo "Deploying application"
kubectl apply -f config.yaml`

	pythonContent := `#!/usr/bin/env python3
import requests
def health_check():
    return requests.get("http://localhost:8080/health")`

	templateContent := `{{.AppName}}-{{.Version}}
{{if .Production}}production{{else}}staging{{end}}`

	scripttestContent := `exec ./deploy.sh
exists deployment.yaml`

	// Track all script types
	err := tracker.ParseAndTrack(bashContent, "deploy.sh", "bash", "deploy-test")
	if err != nil {
		t.Fatal(err)
	}

	err = tracker.ParseAndTrack(pythonContent, "health.py", "python", "deploy-test")
	if err != nil {
		t.Fatal(err)
	}

	err = tracker.ParseAndTrack(templateContent, "version.tmpl", "gotemplate", "deploy-test")
	if err != nil {
		t.Fatal(err)
	}

	err = tracker.ParseAndTrack(scripttestContent, "test.txt", "scripttest", "deploy-test")
	if err != nil {
		t.Fatal(err)
	}

	// Track some executions
	tracker.TrackExecution("deploy.sh", "deploy-test", 2)
	tracker.TrackExecution("health.py", "deploy-test", 3)
	tracker.TrackExecution("version.tmpl", "deploy-test", 1)
	tracker.TrackExecution("test.txt", "deploy-test", 1)

	report := tracker.GetReport()
	if !strings.Contains(report, "Overall: 4/9 commands executed (44.4%)") {
		t.Errorf("Expected overall coverage to be 44.4%%, got: %s", report)
	}

	// Verify all script types are tracked
	if !strings.Contains(report, "deploy.sh") {
		t.Error("Report should contain bash script")
	}
	if !strings.Contains(report, "health.py") {
		t.Error("Report should contain python script")
	}
	if !strings.Contains(report, "version.tmpl") {
		t.Error("Report should contain template")
	}
	if !strings.Contains(report, "test.txt") {
		t.Error("Report should contain scripttest")
	}
}

// Custom YAML parser for configuration files
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
		`^[a-zA-Z_][a-zA-Z0-9_]*\s*:`, // YAML key
		`^\s*-\s+`,                    // YAML list item
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}

	return false
}

// Test demonstrates custom parser registration
func TestCustomParserRegistration(t *testing.T) {
	yamlContent := `# Configuration file
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: my-app`

	tracker := synthetic.NewScriptTracker()
	tracker.RegisterParser("yaml", &YAMLParser{})

	err := tracker.ParseAndTrack(yamlContent, "service.yaml", "yaml", "config-test")
	if err != nil {
		t.Fatal(err)
	}

	// Track configuration usage
	tracker.TrackExecution("service.yaml", "config-test", 2) // apiVersion
	tracker.TrackExecution("service.yaml", "config-test", 5) // name
	tracker.TrackExecution("service.yaml", "config-test", 8) // port

	report := tracker.GetReport()

	// Verify the custom parser worked
	if !strings.Contains(report, "service.yaml") {
		t.Error("Report should contain YAML file")
	}
	if !strings.Contains(report, "3/10 commands executed (30.0%)") {
		t.Errorf("Expected 30%% coverage, got: %s", report)
	}
}

// ExampleScriptTracker_integrationWorkflow demonstrates a complete integration test workflow
func ExampleScriptTracker_integrationWorkflow() {
	// This example shows how to track multiple artifacts in a complete integration workflow
	tracker := synthetic.NewScriptTracker(
		synthetic.WithTestName("integration-workflow"),
		synthetic.WithLabels(map[string]string{
			"environment": "ci",
			"version":     "1.2.3",
		}),
	)

	// 1. Track deployment script
	deployScript := `#!/bin/bash
set -euo pipefail
echo "Starting deployment process..."
kubectl apply -f k8s-manifests/
kubectl rollout status deployment/myapp
echo "Deployment completed successfully"`

	err := tracker.ParseAndTrack(deployScript, "deploy.sh", "bash", "integration-workflow")
	if err != nil {
		log.Fatal(err)
	}

	// 2. Track configuration template
	configTemplate := `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.AppName}}-config
  namespace: {{.Namespace | default "default"}}
data:
  config.yaml: |
    {{if .Debug}}
    log_level: debug
    {{else}}
    log_level: info
    {{end}}
    database:
      host: {{.Database.Host}}
      port: {{.Database.Port}}`

	err = tracker.ParseAndTrack(configTemplate, "config.tmpl", "gotemplate", "integration-workflow")
	if err != nil {
		log.Fatal(err)
	}

	// 3. Track validation scripttest
	validationTest := `# Integration test for deployment validation
exec kubectl get pods
stdout 'myapp-'

# Check deployment status
exec kubectl get deployment myapp -o jsonpath='{.status.readyReplicas}'
stdout '^1$'

# Test service endpoint
exec curl -f http://localhost:8080/health
stdout 'healthy'

# Cleanup test resources
exec kubectl delete deployment myapp --ignore-not-found=true`

	err = tracker.ParseAndTrack(validationTest, "validate.txt", "scripttest", "integration-workflow")
	if err != nil {
		log.Fatal(err)
	}

	// 4. Track execution (in real usage, this would be done by instrumented test runner)
	// Deploy script execution
	tracker.TrackExecution("deploy.sh", "integration-workflow", 2) // set command
	tracker.TrackExecution("deploy.sh", "integration-workflow", 3) // echo "Starting..."
	tracker.TrackExecution("deploy.sh", "integration-workflow", 4) // kubectl apply
	tracker.TrackExecution("deploy.sh", "integration-workflow", 5) // kubectl rollout
	tracker.TrackExecution("deploy.sh", "integration-workflow", 6) // echo "Deployment..."

	// Template execution (variables rendered)
	tracker.TrackExecution("config.tmpl", "integration-workflow", 4)  // .AppName
	tracker.TrackExecution("config.tmpl", "integration-workflow", 8)  // if .Debug
	tracker.TrackExecution("config.tmpl", "integration-workflow", 13) // .Database.Host

	// Scripttest execution
	tracker.TrackExecution("validate.txt", "integration-workflow", 2) // exec kubectl get pods
	tracker.TrackExecution("validate.txt", "integration-workflow", 5) // exec kubectl get deployment
	tracker.TrackExecution("validate.txt", "integration-workflow", 8) // exec curl

	// Generate comprehensive report
	report := tracker.GetReport()
	fmt.Println(report)

	// Output:
	// === Synthetic Coverage Report ===
	//
	// Artifact: config.tmpl (Test: integration-workflow)
	//   Commands: 7 total, 3 executed (42.9%)
	//
	// Artifact: deploy.sh (Test: integration-workflow)
	//   Commands: 5 total, 5 executed (100.0%)
	//
	// Artifact: validate.txt (Test: integration-workflow)
	//   Commands: 7 total, 3 executed (42.9%)
	//
	// Overall: 11/19 commands executed (57.9%)
}

// ExampleScriptTracker_fullScripttestIntegration demonstrates complete scripttest integration with coverage
func ExampleScriptTracker_fullScripttestIntegration() {
	// This example shows how to integrate synthetic coverage with actual scripttest execution
	scripttestContent := `# Full integration test for Go project build and deployment
# This scripttest demonstrates comprehensive CI/CD workflow testing

# Environment setup
env GOOS=linux
env GOARCH=amd64
env CGO_ENABLED=0

# Build the application
go build -o myapp .
exists myapp

# Run unit tests with coverage
go test -cover ./...
stdout 'coverage:'

# Build Docker image
exec docker build -t myapp:test .
stdout 'Successfully built'

# Start test environment
exec docker-compose up -d
stdout 'Creating'

# Wait for services to be ready
exec sleep 5
exec curl -f http://localhost:8080/health
stdout 'healthy'

# Run integration tests
go test -tags=integration ./tests/...
stdout 'PASS'

# Test configuration deployment
exec kubectl apply -f k8s-manifests/ --dry-run=client
stdout 'configured'

# Test Helm chart validation
exec helm template myapp charts/myapp --dry-run
stdout 'apiVersion'

# Performance testing
exec ab -n 100 -c 10 http://localhost:8080/api/status
stdout 'Requests per second'

# Security scanning
exec trivy image myapp:test
! stderr 'HIGH'
! stderr 'CRITICAL'

# Load testing with custom script
exec ./scripts/load-test.sh
stdout 'Load test completed'

# Database migration test
exec ./scripts/migrate.sh --dry-run
stdout 'Migration plan'

# Cleanup
exec docker-compose down
exec docker rmi myapp:test
rm myapp`

	tracker := synthetic.NewScriptTracker(
		synthetic.WithTestName("full-scripttest-integration"),
		synthetic.WithLabels(map[string]string{
			"type":        "integration",
			"environment": "ci",
			"stage":       "full-test",
		}),
	)

	err := tracker.ParseAndTrack(scripttestContent, "integration_test.txt", "scripttest", "full-scripttest-integration")
	if err != nil {
		log.Fatal(err)
	}

	// Simulate scripttest execution with various success/failure scenarios
	// In real usage, this would be instrumented in the scripttest runner

	// Environment and build phase - all successful
	tracker.TrackExecution("integration_test.txt", "full-scripttest-integration", 4)  // env GOOS
	tracker.TrackExecution("integration_test.txt", "full-scripttest-integration", 5)  // env GOARCH
	tracker.TrackExecution("integration_test.txt", "full-scripttest-integration", 6)  // env CGO_ENABLED
	tracker.TrackExecution("integration_test.txt", "full-scripttest-integration", 9)  // go build
	tracker.TrackExecution("integration_test.txt", "full-scripttest-integration", 10) // exists myapp

	// Testing phase - partial execution
	tracker.TrackExecution("integration_test.txt", "full-scripttest-integration", 13) // go test -cover
	tracker.TrackExecution("integration_test.txt", "full-scripttest-integration", 16) // docker build

	// Infrastructure phase - some steps skipped in this run
	tracker.TrackExecution("integration_test.txt", "full-scripttest-integration", 19) // docker-compose up
	tracker.TrackExecution("integration_test.txt", "full-scripttest-integration", 22) // sleep 5
	tracker.TrackExecution("integration_test.txt", "full-scripttest-integration", 23) // curl health check

	// Advanced testing - partially executed
	tracker.TrackExecution("integration_test.txt", "full-scripttest-integration", 26) // go test integration
	tracker.TrackExecution("integration_test.txt", "full-scripttest-integration", 29) // kubectl apply

	// Performance and security - skipped in this run
	// (These lines would not be tracked, showing incomplete coverage)

	// Cleanup - executed
	tracker.TrackExecution("integration_test.txt", "full-scripttest-integration", 47) // docker-compose down
	tracker.TrackExecution("integration_test.txt", "full-scripttest-integration", 49) // rm myapp

	report := tracker.GetReport()
	fmt.Println(report)

	// Generate coverage pod for integration with covutil
	pod, err := tracker.GeneratePod()
	if err != nil {
		log.Fatal(err)
	}

	var totalFunctions int
	for _, pkg := range pod.Profile.Meta.Packages {
		totalFunctions += len(pkg.Functions)
	}
	fmt.Printf("Coverage pod generated with %d functions\n", totalFunctions)
	fmt.Printf("Pod labels: %v\n", pod.Labels)

	// Output:
	// === Synthetic Coverage Report ===
	//
	// Artifact: integration_test.txt (Test: full-scripttest-integration)
	//   Commands: 30 total, 14 executed (46.7%)
	//
	// Overall: 14/30 commands executed (46.7%)
	//
	// Coverage pod generated with 30 functions
	// Pod labels: map[environment:ci generator:basic-tracker stage:full-test test_name:full-scripttest-integration type:integration]
}

// ExampleScriptTracker_realWorldScripttest shows a realistic scripttest with synthetic coverage tracking
func ExampleScriptTracker_realWorldScripttest() {
	// This example shows how to implement coverage tracking for an actual scripttest file
	// that would be used in a real Go project's test suite

	scripttestFile := `# Test the covutil CLI tool functionality
# This scripttest tests the main CLI commands and validates outputs

# Setup test environment
mkdir testdata
cd testdata

# Create sample Go coverage data
cat > sample.go <<EOF
package main
import "fmt"
func main() {
    fmt.Println("Hello, World!")
}
func unused() {
    fmt.Println("This is never called")
}
EOF

# Build with coverage
go build -cover -o sample sample.go

# Generate coverage data
env GOCOVERDIR=./coverage
mkdir coverage
exec ./sample

# Test covutil commands
exec covutil summary ./coverage
stdout 'Coverage Summary'
stdout 'sample.go'

# Test tree view
exec covutil tree ./coverage
stdout 'Coverage Tree'

# Test JSON output
exec covutil json ./coverage
stdout '{'
stdout '"packages"'

# Test serve command (quick start/stop)
exec timeout 2s covutil serve ./coverage --port 8081 &
sleep 1
exec curl -f http://localhost:8081/
stdout 'Coverage Report'

# Test coverage percentage calculation
exec covutil percent ./coverage
stdout '50.0%'

# Test with multiple coverage directories
mkdir coverage2
cp coverage/* coverage2/
exec covutil summary ./coverage ./coverage2
stdout 'Combined Coverage'

# Test error conditions
! exec covutil summary ./nonexistent
stderr 'no coverage data found'

# Test help output
exec covutil help
stdout 'Usage:'
stdout 'Commands:'

# Cleanup
cd ..
rm -rf testdata`

	tracker := synthetic.NewScriptTracker(
		synthetic.WithTestName("covutil-cli-test"),
		synthetic.WithLabels(map[string]string{
			"component": "cli",
			"category":  "integration",
			"tool":      "covutil",
		}),
	)

	err := tracker.ParseAndTrack(scripttestFile, "covutil_cli_test.txt", "scripttest", "covutil-cli-test")
	if err != nil {
		log.Fatal(err)
	}

	// Simulate test execution - successful path through most commands
	executedLines := []int{
		4,  // mkdir testdata
		5,  // cd testdata
		8,  // cat > sample.go
		18, // go build -cover
		21, // env GOCOVERDIR
		22, // mkdir coverage
		23, // exec ./sample
		26, // exec covutil summary
		30, // exec covutil tree
		33, // exec covutil json
		38, // exec timeout
		39, // sleep 1
		40, // exec curl
		43, // exec covutil percent
		46, // mkdir coverage2
		47, // cp coverage/*
		48, // exec covutil summary (combined)
		55, // exec covutil help
		59, // cd ..
		60, // rm -rf testdata
	}

	for _, line := range executedLines {
		tracker.TrackExecution("covutil_cli_test.txt", "covutil-cli-test", line)
	}

	// Generate detailed coverage report
	report := tracker.GetReport()
	fmt.Println(report)

	// Output:
	// === Synthetic Coverage Report ===
	//
	// Artifact: covutil_cli_test.txt (Test: covutil-cli-test)
	//   Commands: 37 total, 20 executed (54.1%)
	//
	// Overall: 20/37 commands executed (54.1%)
}

// Test demonstrates generating different output formats
func TestOutputFormats(t *testing.T) {
	tracker := synthetic.NewBasicTracker(
		synthetic.WithLabels(map[string]string{
			"environment": "test",
			"version":     "1.0.0",
		}),
	)

	// Track some executions
	tracker.Track("script.sh", "1", true)
	tracker.Track("script.sh", "2", true)
	tracker.Track("script.sh", "3", false)

	// Test pod generation
	pod, err := tracker.GeneratePod()
	if err != nil {
		t.Fatalf("Failed to generate pod: %v", err)
	}

	if pod.Labels["environment"] != "test" {
		t.Errorf("Expected environment=test, got %s", pod.Labels["environment"])
	}

	// Test text profile generation
	profile, err := tracker.GenerateProfile("text")
	if err != nil {
		t.Fatalf("Failed to generate text profile: %v", err)
	}

	if len(profile) == 0 {
		t.Error("Text profile should not be empty")
	}

	// Test report generation
	report := tracker.GetReport()
	if report == "" {
		t.Error("Report should not be empty")
	}

	t.Logf("Generated report:\n%s", report)
	t.Logf("Generated profile:\n%s", string(profile))
}

// Benchmark demonstrates performance characteristics
func BenchmarkBasicTracker(b *testing.B) {
	tracker := synthetic.NewBasicTracker()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.Track("script.sh", fmt.Sprintf("%d", i%100), i%2 == 0)
	}
}

// TestScripttestIntegrationWorkflow demonstrates a complete workflow integrating scripttest with synthetic coverage
func TestScripttestIntegrationWorkflow(t *testing.T) {
	// This test shows how to add synthetic coverage tracking to an existing scripttest
	// It simulates the workflow of a real CI/CD pipeline test

	// Step 1: Define a realistic scripttest
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

# Clone project (simulated with file creation)
cat > main.go <<EOF
package main

import (
    "fmt"
    "net/http"
)

func main() {
    http.HandleFunc("/health", healthHandler)
    http.ListenAndServe(":8080", nil)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "healthy")
}

func unused() {
    fmt.Println("This function is never called")
}
EOF

cat > main_test.go <<EOF
package main

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestHealthHandler(t *testing.T) {
    req := httptest.NewRequest("GET", "/health", nil)
    w := httptest.NewRecorder()
    healthHandler(w, req)
    
    if w.Body.String() != "healthy" {
        t.Errorf("Expected 'healthy', got %s", w.Body.String())
    }
}
EOF

cat > go.mod <<EOF
module testapp
go 1.21
EOF

# Build phase
go mod tidy
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

cat > k8s-deployment.yaml <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: testapp
spec:
  replicas: 1
  selector:
    matchLabels:
      app: testapp
  template:
    metadata:
      labels:
        app: testapp
    spec:
      containers:
      - name: testapp
        image: testapp:latest
        ports:
        - containerPort: 8080
EOF

# Validate deployment config
exec grep -q "testapp" k8s-deployment.yaml
exec grep -q "8080" k8s-deployment.yaml

# Security check simulation
exec grep -q "alpine" Dockerfile
! exec grep -i "root" k8s-deployment.yaml

# Cleanup
cd ..
rm -rf build`

	// Step 2: Set up synthetic coverage tracking
	tracker := synthetic.NewScriptTracker(
		synthetic.WithTestName("ci-cd-pipeline"),
		synthetic.WithLabels(map[string]string{
			"pipeline":    "ci-cd",
			"environment": "test",
			"stage":       "integration",
			"version":     "1.0.0",
		}),
	)

	err := tracker.ParseAndTrack(scripttestContent, "ci_cd_pipeline.txt", "scripttest", "ci-cd-pipeline")
	if err != nil {
		t.Fatalf("Failed to parse scripttest: %v", err)
	}

	// Step 3: Simulate scripttest execution with coverage tracking
	// In a real implementation, this would be integrated into the scripttest runner

	// Environment setup phase - fully executed
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 4) // env GO111MODULE
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 5) // env GOPROXY
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 6) // env GOSUMDB

	// Workspace setup - fully executed
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 9)  // rm -rf ./build
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 10) // mkdir build
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 11) // cd build

	// File creation - fully executed
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 14) // cat > main.go
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 35) // cat > main_test.go
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 50) // cat > go.mod

	// Build phase - fully executed
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 56) // go mod tidy
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 57) // go build
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 58) // exists app

	// Test phase - fully executed
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 61) // go test -v
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 65) // go test -cover

	// Integration test - partially executed (timeout scenario)
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 68) // exec timeout
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 69) // sleep 2
	// Note: curl command might fail in CI, so we track it as attempted but not successful

	// Deployment build - fully executed
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 73) // env GOOS
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 74) // env GOARCH
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 75) // go build linux

	// Artifact creation - fully executed
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 78) // cat > Dockerfile
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 84) // cat > k8s-deployment.yaml

	// Validation - fully executed
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 102) // grep testapp
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 103) // grep 8080

	// Security checks - partially executed
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 106) // grep alpine
	// Security check for root might be skipped in some runs

	// Cleanup - fully executed
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 110) // cd ..
	tracker.TrackExecution("ci_cd_pipeline.txt", "ci-cd-pipeline", 111) // rm -rf build

	// Step 4: Generate coverage report
	report := tracker.GetReport()
	t.Logf("Scripttest Coverage Report:\n%s", report)

	// Verify coverage metrics
	if !strings.Contains(report, "ci_cd_pipeline.txt") {
		t.Error("Report should contain scripttest filename")
	}

	// Step 5: Generate coverage pod for integration with Go coverage tools
	pod, err := tracker.GeneratePod()
	if err != nil {
		t.Fatalf("Failed to generate coverage pod: %v", err)
	}

	// Verify pod properties
	if pod.Labels["pipeline"] != "ci-cd" {
		t.Errorf("Expected pipeline=ci-cd, got %s", pod.Labels["pipeline"])
	}

	var totalFunctions int
	for _, pkg := range pod.Profile.Meta.Packages {
		totalFunctions += len(pkg.Functions)
	}
	if totalFunctions == 0 {
		t.Error("Pod should contain synthetic functions representing script commands")
	}

	t.Logf("Generated coverage pod with %d functions", totalFunctions)
	t.Logf("Pod labels: %v", pod.Labels)

	// Step 6: Test integration with covutil ecosystem
	profile, err := tracker.GenerateProfile("text")
	if err != nil {
		t.Fatalf("Failed to generate text profile: %v", err)
	}

	if len(profile) == 0 {
		t.Error("Generated profile should not be empty")
	}

	// Verify profile format (should be compatible with go tool cover)
	profileStr := string(profile)
	if !strings.Contains(profileStr, "ci_cd_pipeline.txt") {
		t.Error("Profile should contain artifact name")
	}

	t.Logf("Generated profile snippet:\n%s", profileStr[:min(200, len(profileStr))])
}

// TestScripttestCoverageIntegration demonstrates how to integrate synthetic coverage
// with real scripttest execution in a testing framework
func TestScripttestCoverageIntegration(t *testing.T) {
	// This test shows the pattern for integrating synthetic coverage into existing scripttest workflows

	tests := []struct {
		name             string
		scriptContent    string
		expectedSteps    int
		executedLines    []int
		expectedCoverage float64
	}{
		{
			name: "basic-build-test",
			scriptContent: `# Basic build and test
go version
go build .
go test ./...
exists main`,
			expectedSteps:    4,
			executedLines:    []int{2, 3, 4}, // Skip 'exists' command
			expectedCoverage: 75.0,
		},
		{
			name: "docker-workflow",
			scriptContent: `# Docker build workflow
docker version
docker build -t myapp .
docker run --rm myapp echo "test"
docker rmi myapp`,
			expectedSteps:    4,
			executedLines:    []int{2, 3, 4, 5}, // All commands executed
			expectedCoverage: 100.0,
		},
		{
			name: "failed-deployment",
			scriptContent: `# Deployment that partially fails
kubectl version
kubectl apply -f deployment.yaml
kubectl rollout status deployment/myapp
kubectl get pods`,
			expectedSteps:    4,
			executedLines:    []int{2}, // Only version check succeeds
			expectedCoverage: 25.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create tracker for this specific test
			tracker := synthetic.NewScriptTracker(
				synthetic.WithTestName(tt.name),
				synthetic.WithLabels(map[string]string{
					"test_type": "scripttest_integration",
					"scenario":  tt.name,
				}),
			)

			// Parse the scripttest content
			scriptFile := tt.name + ".txt"
			err := tracker.ParseAndTrack(tt.scriptContent, scriptFile, "scripttest", tt.name)
			if err != nil {
				t.Fatalf("Failed to parse script: %v", err)
			}

			// Simulate execution of specified lines
			for _, line := range tt.executedLines {
				tracker.TrackExecution(scriptFile, tt.name, line)
			}

			// Generate report and verify coverage
			report := tracker.GetReport()
			t.Logf("Coverage report for %s:\n%s", tt.name, report)

			// Verify expected coverage percentage is roughly correct
			// (allowing for small floating point differences)
			if !strings.Contains(report, fmt.Sprintf("%.1f%%", tt.expectedCoverage)) &&
				!strings.Contains(report, fmt.Sprintf("%.0f%%", tt.expectedCoverage)) {
				// Try alternative formatting
				alt1 := fmt.Sprintf("%.1f", tt.expectedCoverage)
				alt2 := fmt.Sprintf("%.0f", tt.expectedCoverage)
				if !strings.Contains(report, alt1) && !strings.Contains(report, alt2) {
					t.Errorf("Expected coverage around %.1f%%, but report was:\n%s", tt.expectedCoverage, report)
				}
			}

			// Verify pod generation works
			pod, err := tracker.GeneratePod()
			if err != nil {
				t.Fatalf("Failed to generate pod: %v", err)
			}

			if pod.Labels["scenario"] != tt.name {
				t.Errorf("Expected scenario=%s, got %s", tt.name, pod.Labels["scenario"])
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func BenchmarkScriptTracker(b *testing.B) {
	tracker := synthetic.NewScriptTracker()

	script := `#!/bin/bash
for i in {1..100}; do
    echo "Processing $i"
    if [ $((i % 10)) -eq 0 ]; then
        echo "Milestone: $i"
    fi
done`

	err := tracker.ParseAndTrack(script, "bench.sh", "bash", "benchmark")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.TrackExecution("bench.sh", "benchmark", (i%4)+2)
	}
}
