//go:build ignore

package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "embed" // for embedding the template
)

func main() {
	log.SetPrefix("prepoverlay: ")
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

//go:embed runtime/coverage/coverage.go.in
var coverageTemplate string

//go:embed testing/testing.go.in
var testingTemplate string

var overlayTemplate = `{
    "Replace": {
        {{- $first := true }}
        {{- range $stdPath, $overlayPath := .Overlays }}
        {{- if not $first }},{{ end }}
        "{{$stdPath}}": "{{$overlayPath}}"
        {{- $first = false }}
        {{- end }}
    }
}`

type templateData struct {
	Workdir  string
	GoRoot   string
	Overlays map[string]string // maps Go standard library paths to our overlay paths
}

func run() error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Ensure goimports is available for advanced overlays
	if err := ensureGoimports(); err != nil {
		log.Printf("Warning: goimports not available, advanced overlays may fail: %v", err)
	}

	// Get GOROOT
	goroot, err := getGoRoot()
	if err != nil {
		return fmt.Errorf("failed to get GOROOT: %w", err)
	}

	// Create overlay mappings
	overlays := make(map[string]string)

	// Add runtime/coverage overlay
	coveragePath := filepath.Join(goroot, "src", "runtime", "coverage", "coverage.go")
	if _, err := os.Stat(coveragePath); err == nil {
		overlays[coveragePath] = filepath.Join(wd, "runtime", "coverage", "coverage.go")
	}

	// Add testing overlay if it exists
	testingPath := filepath.Join(goroot, "src", "testing", "testing.go")
	if _, err := os.Stat(testingPath); err == nil {
		if _, err := os.Stat(filepath.Join(wd, "testing", "testing.go")); err == nil {
			overlays[testingPath] = filepath.Join(wd, "testing", "testing.go")
		}
	}

	// Add scripttest overlay if it exists (for external packages)
	scripttestPath := filepath.Join(wd, "scripttest", "scripttest.go")
	if _, err := os.Stat(scripttestPath); err == nil {
		// This would be used with go mod replace directives
		overlays["github.com/rsc/script/scripttest/scripttest.go"] = scripttestPath
	}

	data := templateData{
		Workdir:  wd,
		GoRoot:   goroot,
		Overlays: overlays,
	}

	if err := generateCoverageFile(data); err != nil {
		return fmt.Errorf("failed to generate coverage file: %w", err)
	}

	if err := generateTestingFile(data); err != nil {
		return fmt.Errorf("failed to generate testing file: %w", err)
	}

	if err := generateOverlayFile(data); err != nil {
		return fmt.Errorf("failed to generate overlay file: %w", err)
	}

	log.Printf("overlay files generated in %s", wd)
	return nil
}

func generateCoverageFile(data templateData) error {
	t, err := template.New("coverage").Parse(coverageTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse coverage template: %w", err)
	}

	if err := os.MkdirAll("./runtime/coverage", 0755); err != nil {
		return fmt.Errorf("failed to create runtime/coverage directory: %w", err)
	}

	f, err := os.Create("./runtime/coverage/coverage.go")
	if err != nil {
		return fmt.Errorf("failed to create coverage.go file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("failed to close coverage.go file: %v", err)
		}
	}()

	if err := t.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute coverage template: %w", err)
	}

	return nil
}

func generateTestingFile(data templateData) error {
	// Check if testing overlay should be generated
	testingOverlayPath := filepath.Join(data.GoRoot, "src", "testing", "testing.go")
	if _, exists := data.Overlays[testingOverlayPath]; !exists {
		// Skip if testing overlay not configured
		return nil
	}

	// Create the complete testing.go file by merging original with our enhancements
	originalTestingFile := testingOverlayPath
	originalContent, err := os.ReadFile(originalTestingFile)
	if err != nil {
		return fmt.Errorf("failed to read original testing.go: %w", err)
	}

	// Read our template
	t, err := template.New("testing").Parse(testingTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse testing template: %w", err)
	}

	if err := os.MkdirAll("./testing", 0755); err != nil {
		return fmt.Errorf("failed to create testing directory: %w", err)
	}

	f, err := os.Create("./testing/testing.go")
	if err != nil {
		return fmt.Errorf("failed to create testing.go file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("failed to close testing.go file: %v", err)
		}
	}()

	// Write the original content first
	if _, err := f.Write(originalContent); err != nil {
		return fmt.Errorf("failed to write original testing content: %w", err)
	}

	// Write our enhanced imports and functions
	enhancedCode := `
// Enhanced testing package overlay additions
import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/coverage"
	"strings"
)

`
	if _, err := f.WriteString(enhancedCode); err != nil {
		return fmt.Errorf("failed to write enhanced code: %w", err)
	}

	// Execute our template to add the enhanced T.Run method
	if err := t.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute testing template: %w", err)
	}

	return nil
}

func generateOverlayFile(data templateData) error {
	t, err := template.New("overlay").Parse(overlayTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse overlay template: %w", err)
	}

	f, err := os.Create("overlay.json")
	if err != nil {
		return fmt.Errorf("failed to create overlay.json file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("failed to close overlay.json file: %v", err)
		}
	}()

	if err := t.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute overlay template: %w", err)
	}

	return nil
}

// getGoRoot gets the GOROOT environment variable
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

// ensureGoimports checks if goimports is available and tries to install it if not
func ensureGoimports() error {
	// Check if goimports is already available
	if _, err := exec.LookPath("goimports"); err == nil {
		return nil
	}

	// Try to install goimports using the module dependency
	log.Println("Installing goimports from module dependency...")
	cmd := exec.Command("go", "install", "golang.org/x/tools/cmd/goimports")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install goimports: %w", err)
	}

	// Verify it's now available
	if _, err := exec.LookPath("goimports"); err != nil {
		return fmt.Errorf("goimports still not available after installation: %w", err)
	}

	log.Println("Successfully installed goimports")
	return nil
}
