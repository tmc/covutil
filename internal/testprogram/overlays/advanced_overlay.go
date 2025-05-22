//go:build ignore

package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// OverlayConfig defines how to transform a source file
type OverlayConfig struct {
	SourceURL  string            // URL to fetch source from (e.g., GitHub)
	TargetPath string            // Path within Go module (e.g., "github.com/rsc/script/scripttest")
	FileName   string            // File to modify (e.g., "scripttest.go")
	Renames    map[string]string // Function/var renames (old -> new)
	AppendFile string            // File with content to append
	OutputPath string            // Where to write the modified file
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run advanced_overlay.go <config_name>")
	}

	// Ensure goimports is available
	if err := ensureGoimports(); err != nil {
		log.Fatalf("goimports is required but not available: %v", err)
	}

	configName := os.Args[1]

	switch configName {
	case "scripttest":
		generateScriptTestOverlay()
	default:
		log.Fatalf("Unknown config: %s", configName)
	}
}

func ensureGoimports() error {
	_, err := exec.LookPath("goimports")
	if err != nil {
		return fmt.Errorf("goimports not found in PATH. Install with: go install golang.org/x/tools/cmd/goimports@latest")
	}
	return nil
}

func generateScriptTestOverlay() {
	config := OverlayConfig{
		SourceURL:  "https://raw.githubusercontent.com/rsc/script/v0.0.2/scripttest/scripttest.go",
		TargetPath: "github.com/rsc/script/scripttest",
		FileName:   "scripttest.go",
		Renames: map[string]string{
			"Run": "run", // Rename public Run to private run
		},
		AppendFile: "scripttest_additions_clean.go.in",
		OutputPath: "scripttest/scripttest.go",
	}

	if err := processOverlay(config); err != nil {
		log.Fatalf("Failed to process overlay: %v", err)
	}

	fmt.Println("Successfully generated scripttest overlay")
}

func processOverlay(config OverlayConfig) error {
	// 1. Fetch the source file
	sourceCode, err := fetchSourceCode(config.SourceURL)
	if err != nil {
		return fmt.Errorf("failed to fetch source: %w", err)
	}

	// 2. Parse the Go source
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, config.FileName, sourceCode, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse source: %w", err)
	}

	// 3. Apply renames
	if err := applyRenames(file, config.Renames); err != nil {
		return fmt.Errorf("failed to apply renames: %w", err)
	}

	// 4. Read additional content to append
	additionalContent := ""
	if config.AppendFile != "" {
		content, err := os.ReadFile(config.AppendFile)
		if err == nil {
			additionalContent = string(content)
		}
	}

	// 5. Format and write the result
	if err := writeFormattedFile(fset, file, additionalContent, config.OutputPath); err != nil {
		return fmt.Errorf("failed to write formatted file: %w", err)
	}

	return nil
}

func fetchSourceCode(url string) (string, error) {
	cmd := exec.Command("curl", "-s", url)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	return string(output), nil
}

func applyRenames(file *ast.File, renames map[string]string) error {
	// Walk the AST and rename functions/variables
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if node.Name != nil {
				if newName, exists := renames[node.Name.Name]; exists {
					fmt.Printf("Renaming function %s -> %s\n", node.Name.Name, newName)
					node.Name.Name = newName
				}
			}
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				if valueSpec, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range valueSpec.Names {
						if newName, exists := renames[name.Name]; exists {
							fmt.Printf("Renaming variable %s -> %s\n", name.Name, newName)
							name.Name = newName
						}
					}
				}
			}
		}
		return true
	})

	return nil
}

func addRequiredImports(file *ast.File) {
	// Add imports needed for our enhanced functionality
	requiredImports := []string{
		"\"fmt\"",
		"\"runtime/coverage\"",
		"\"sync\"",
	}

	// Find the import declaration
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			// Add our required imports
			for _, imp := range requiredImports {
				// Check if import already exists
				exists := false
				for _, spec := range genDecl.Specs {
					if importSpec, ok := spec.(*ast.ImportSpec); ok {
						if importSpec.Path.Value == imp {
							exists = true
							break
						}
					}
				}

				if !exists {
					newImport := &ast.ImportSpec{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: imp,
						},
					}
					genDecl.Specs = append(genDecl.Specs, newImport)
				}
			}
			break
		}
	}
}

func writeFormattedFile(fset *token.FileSet, file *ast.File, additionalContent, outputPath string) error {
	// Create output directory
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Format the AST back to Go code
	var buf strings.Builder
	if err := format.Node(&buf, fset, file); err != nil {
		return fmt.Errorf("failed to format AST: %w", err)
	}

	// Add additional content if provided
	fullContent := buf.String()
	if additionalContent != "" {
		fullContent += "\n\n" + additionalContent
	}

	// Let goimports handle everything: imports, formatting, and organization
	formattedContent, err := runGoimports(fullContent)
	if err != nil {
		return fmt.Errorf("goimports failed: %w", err)
	}

	// Write to output file
	if err := os.WriteFile(outputPath, []byte(formattedContent), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

func runGoimports(content string) (string, error) {
	// Try to run goimports with various options
	cmd := exec.Command("goimports", "-local", "github.com/rsc/script")
	cmd.Stdin = strings.NewReader(content)

	var stderr strings.Builder
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		// Provide detailed error information
		return "", fmt.Errorf("goimports failed: %w (stderr: %s)", err, stderr.String())
	}

	result := string(output)

	// Verify the result is valid Go code
	if _, parseErr := parser.ParseFile(token.NewFileSet(), "", result, parser.ParseComments); parseErr != nil {
		return "", fmt.Errorf("goimports produced invalid Go code: %w", parseErr)
	}

	return result, nil
}
