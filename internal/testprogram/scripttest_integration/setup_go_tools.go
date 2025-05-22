//go:build ignore

// Setup script to build Go tools and prepare curated PATH for scripttest integration testing
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run setup_go_tools.go <output_dir>")
	}

	outputDir := os.Args[1]

	fmt.Printf("Setting up Go tools and binaries in: %s\n", outputDir)

	// Create the output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Build our test binaries with coverage
	if err := buildTestBinaries(outputDir); err != nil {
		log.Fatalf("Failed to build test binaries: %v", err)
	}

	// Build selected Go tools with coverage
	if err := buildGoTools(outputDir); err != nil {
		log.Fatalf("Failed to build Go tools: %v", err)
	}

	// Create PATH setup script
	if err := createPathScript(outputDir); err != nil {
		log.Fatalf("Failed to create PATH script: %v", err)
	}

	fmt.Printf("✅ Setup complete!\n")
	fmt.Printf("   - Test binaries: main, cmd1, cmd2, cmd3\n")
	fmt.Printf("   - Go tools: go, gofmt, vet, doc\n")
	fmt.Printf("   - PATH script: %s/setup_path.sh\n", outputDir)
	fmt.Printf("\nUsage:\n")
	fmt.Printf("   source %s/setup_path.sh\n", outputDir)
	fmt.Printf("   # Now run your scripttest integration tests\n")
}

// buildTestBinaries builds our main test binaries with coverage
func buildTestBinaries(outputDir string) error {
	fmt.Println("🔨 Building test binaries with coverage...")

	binaries := []struct {
		name   string
		srcDir string
	}{
		{"main", "main"},
		{"cmd1", "cmd1"},
		{"cmd2", "cmd2"},
		{"cmd3", "cmd3"},
	}

	for _, binary := range binaries {
		binaryPath := filepath.Join(outputDir, binary.name)
		if runtime.GOOS == "windows" {
			binaryPath += ".exe"
		}

		// Get absolute path to avoid overlay conflicts
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		srcPath := filepath.Join(wd, binary.srcDir)
		fmt.Printf("  Building %s from %s\n", binary.name, srcPath)

		cmd := exec.Command("go", "build", "-cover", "-o", binaryPath, srcPath)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to build %s: %w", binary.name, err)
		}
	}

	fmt.Printf("✅ Built %d test binaries\n", len(binaries))
	return nil
}

// buildGoTools builds selected Go tools with coverage for integration testing
func buildGoTools(outputDir string) error {
	fmt.Println("🔨 Building Go tools with coverage...")

	// Get GOROOT to find tool sources
	goroot, err := getGoRoot()
	if err != nil {
		return fmt.Errorf("failed to get GOROOT: %w", err)
	}

	// Tools to build - these are commonly used and good for testing
	tools := []struct {
		name    string
		srcPath string
		alias   string // Optional alias for the binary
	}{
		{"go", "cmd/go", "go"},
		{"gofmt", "cmd/gofmt", "gofmt"},
		{"vet", "cmd/vet", "vet"},
		{"doc", "cmd/doc", "doc"},
		{"fix", "cmd/fix", "fix"},
		{"test2json", "cmd/test2json", "test2json"},
	}

	for _, tool := range tools {
		binaryName := tool.alias
		if binaryName == "" {
			binaryName = tool.name
		}

		binaryPath := filepath.Join(outputDir, binaryName)
		if runtime.GOOS == "windows" {
			binaryPath += ".exe"
		}

		srcPath := filepath.Join(goroot, "src", tool.srcPath)
		fmt.Printf("  Building %s from %s\n", tool.name, srcPath)

		// Build the tool with coverage enabled
		cmd := exec.Command("go", "build", "-cover", "-o", binaryPath, srcPath)
		if err := cmd.Run(); err != nil {
			// Some tools might not build with coverage, that's ok
			fmt.Printf("    Warning: failed to build %s with coverage, trying without: %v\n", tool.name, err)

			// Try without coverage
			cmd = exec.Command("go", "build", "-o", binaryPath, srcPath)
			if err := cmd.Run(); err != nil {
				fmt.Printf("    Warning: failed to build %s: %v\n", tool.name, err)
				continue
			}
		}

		fmt.Printf("    ✅ Built %s\n", tool.name)
	}

	return nil
}

// createPathScript creates a script to set up the PATH with our binaries
func createPathScript(outputDir string) error {
	fmt.Println("📝 Creating PATH setup script...")

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	scriptPath := filepath.Join(outputDir, "setup_path.sh")

	scriptContent := fmt.Sprintf(`#!/bin/bash
# PATH setup script for scripttest integration testing
# This script sets up a curated PATH with test binaries and Go tools

export ORIGINAL_PATH="$PATH"
export CURATED_BIN_DIR="%s"
export PATH="$CURATED_BIN_DIR:$PATH"

echo "🔧 Curated PATH setup complete"
echo "   Binary directory: $CURATED_BIN_DIR"
echo "   Available binaries:"

# List available binaries
for binary in "$CURATED_BIN_DIR"/*; do
    if [ -x "$binary" ] && [ -f "$binary" ]; then
        basename "$binary"
    fi
done | sort | sed 's/^/     - /'

echo ""
echo "Environment variables for integration coverage:"
echo "   export GO_INTEGRATION_COVERAGE=1"
echo "   export GOCOVERDIR=./coverage_data"
echo ""
echo "Ready for scripttest integration testing!"

# Function to restore original PATH
restore_path() {
    export PATH="$ORIGINAL_PATH"
    echo "🔄 PATH restored to original"
}

# Export the restore function for use
export -f restore_path
`, absOutputDir)

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to write PATH script: %w", err)
	}

	fmt.Printf("✅ Created PATH setup script: %s\n", scriptPath)
	return nil
}

// getGoRoot gets the GOROOT path
func getGoRoot() (string, error) {
	cmd := exec.Command("go", "env", "GOROOT")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get GOROOT: %w", err)
	}

	goroot := strings.TrimSpace(string(output))
	if goroot == "" {
		return "", fmt.Errorf("GOROOT is empty")
	}

	return goroot, nil
}
