//go:build ignore

// Generator for integration coverage overlay that addresses Go issue #60182
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run generate_integration_overlay.go <mode>\nModes: coverage, testing, full")
	}

	mode := os.Args[1]

	switch mode {
	case "coverage":
		generateIntegrationCoverageOverlay()
	case "testing":
		generateIntegrationTestingOverlay()
	case "full":
		generateIntegrationCoverageOverlay()
		generateIntegrationTestingOverlay()
		generateIntegrationOverlayConfig()
	default:
		log.Fatalf("Unknown mode: %s", mode)
	}
}

func generateIntegrationCoverageOverlay() {
	fmt.Println("Generating integration coverage overlay...")

	// Read the integration coverage template
	template, err := os.ReadFile("runtime/coverage/coverage_integration.go.in")
	if err != nil {
		log.Fatalf("Failed to read integration coverage template: %v", err)
	}

	// Create the output directory
	outputDir := "runtime/coverage"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Write the coverage overlay file
	outputFile := filepath.Join(outputDir, "coverage.go")
	if err := os.WriteFile(outputFile, template, 0644); err != nil {
		log.Fatalf("Failed to write coverage overlay: %v", err)
	}

	fmt.Printf("Generated integration coverage overlay: %s\n", outputFile)
}

func generateIntegrationTestingOverlay() {
	fmt.Println("Generating integration testing overlay...")

	// For the testing overlay, we need to merge with the original testing.go
	goroot, err := getGoRoot()
	if err != nil {
		log.Fatalf("Failed to get GOROOT: %v", err)
	}

	originalTestingFile := filepath.Join(goroot, "src", "testing", "testing.go")
	originalContent, err := os.ReadFile(originalTestingFile)
	if err != nil {
		log.Fatalf("Failed to read original testing.go: %v", err)
	}

	// Read our integration testing template
	template, err := os.ReadFile("testing/testing_integration.go.in")
	if err != nil {
		log.Fatalf("Failed to read integration testing template: %v", err)
	}

	// Create the output directory
	outputDir := "testing"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Combine original content with our enhancements
	outputFile := filepath.Join(outputDir, "testing.go")
	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Failed to create testing overlay file: %v", err)
	}
	defer f.Close()

	// Write original content
	if _, err := f.Write(originalContent); err != nil {
		log.Fatalf("Failed to write original testing content: %v", err)
	}

	// Write our enhancements
	if _, err := f.Write([]byte("\n\n// Integration coverage enhancements\n")); err != nil {
		log.Fatalf("Failed to write separator: %v", err)
	}

	if _, err := f.Write(template); err != nil {
		log.Fatalf("Failed to write integration testing template: %v", err)
	}

	fmt.Printf("Generated integration testing overlay: %s\n", outputFile)
}

func generateIntegrationOverlayConfig() {
	fmt.Println("Generating integration overlay configuration...")

	goroot, err := getGoRoot()
	if err != nil {
		log.Fatalf("Failed to get GOROOT: %v", err)
	}

	// Create overlay.json for integration coverage
	config := fmt.Sprintf(`{
    "Replace": {
        "%s/src/runtime/coverage/coverage.go": "runtime/coverage/coverage.go",
        "%s/src/testing/testing.go": "testing/testing.go"
    }
}`, goroot, goroot)

	if err := os.WriteFile("overlay_integration.json", []byte(config), 0644); err != nil {
		log.Fatalf("Failed to write integration overlay config: %v", err)
	}

	fmt.Println("Generated integration overlay configuration: overlay_integration.json")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go test -overlay=overlay_integration.json -coverprofile=coverage.out ./...")
	fmt.Println("  go test -overlay=overlay_integration.json -coverpkg=./... ./...")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Println("  GO_INTEGRATION_COVERAGE=1  # Enable integration coverage mode")
	fmt.Println("  GOCOVERDIR=./coverage      # Directory for coverage data")
}

func getGoRoot() (string, error) {
	cmd := exec.Command("go", "env", "GOROOT")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get GOROOT: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
