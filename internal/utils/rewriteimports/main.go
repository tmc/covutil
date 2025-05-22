// Command rewriteimports rewrites import paths in Go files.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	dryRun = flag.Bool("n", false, "Print changes without modifying files")
)

func main() {
	flag.Parse()

	// Map of old import paths to new import paths
	replacements := map[string]string{
		"internal/coverage/":      "github.com/tmc/covutil/internal/coverage/",
		"internal/coverage":       "github.com/tmc/covutil/internal/coverage",
		"internal/runtime/atomic": "sync/atomic",
	}

	// Find all Go files in internal/coverage
	var goFiles []string
	err := filepath.Walk("./internal/coverage", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			goFiles = append(goFiles, path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Error walking directory: %v", err)
	}

	// Process each Go file
	for _, file := range goFiles {
		fmt.Printf("Processing %s\n", file)

		// Read file content
		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("Error reading file %s: %v", file, err)
		}

		// Apply replacements
		fileContent := string(content)
		changed := false

		for oldImport, newImport := range replacements {
			// Check for imports with quotes
			oldQuoted := fmt.Sprintf("\"%s\"", oldImport)
			newQuoted := fmt.Sprintf("\"%s\"", newImport)

			if strings.Contains(fileContent, oldQuoted) {
				fileContent = strings.ReplaceAll(fileContent, oldQuoted, newQuoted)
				changed = true
				fmt.Printf("  Replaced %s with %s\n", oldQuoted, newQuoted)
			}
		}

		if !changed {
			continue
		}

		// Only print changes for dry run
		if *dryRun {
			continue
		}

		// Write changes back to file
		err = os.WriteFile(file, []byte(fileContent), 0644)
		if err != nil {
			log.Fatalf("Error writing file %s: %v", file, err)
		}
	}

	// Format the files with gofmt
	if !*dryRun {
		cmd := exec.Command("gofmt", "-w", "./internal/coverage")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("Error running gofmt: %v", err)
		}
	}
}
