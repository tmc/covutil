package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CopyCoverageDataUp copies coverage data from current directory to parent directory
// This helps consolidate coverage data from multiple test scenarios
func CopyCoverageDataUp(prefix string) {
	coverDir := os.Getenv("GOCOVERDIR")
	if coverDir == "" {
		return
	}

	// Check if we're in integration coverage mode
	if os.Getenv("GO_INTEGRATION_COVERAGE") == "" {
		return
	}

	fmt.Printf("[%s] Copying coverage data up from %s\n", prefix, coverDir)

	// Find parent directory
	parentDir := filepath.Dir(coverDir)
	if parentDir == coverDir || parentDir == "/" || parentDir == "." {
		fmt.Printf("[%s] No valid parent directory to copy to\n", prefix)
		return
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		fmt.Printf("[%s] Failed to create parent directory: %v\n", prefix, err)
		return
	}

	// Read coverage files from current directory
	entries, err := os.ReadDir(coverDir)
	if err != nil {
		fmt.Printf("[%s] Failed to read coverage directory: %v\n", prefix, err)
		return
	}

	copiedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, "cov") {
			srcPath := filepath.Join(coverDir, name)

			// Create unique filename to avoid conflicts
			baseName := strings.TrimSuffix(name, filepath.Ext(name))
			ext := filepath.Ext(name)
			uniqueName := fmt.Sprintf("%s_%s%s", baseName, prefix, ext)
			dstPath := filepath.Join(parentDir, uniqueName)

			// If file already exists, append a counter
			counter := 1
			for {
				if _, err := os.Stat(dstPath); os.IsNotExist(err) {
					break
				}
				uniqueName = fmt.Sprintf("%s_%s_%d%s", baseName, prefix, counter, ext)
				dstPath = filepath.Join(parentDir, uniqueName)
				counter++
			}

			if err := copyFile(srcPath, dstPath); err != nil {
				fmt.Printf("[%s] Failed to copy %s: %v\n", prefix, name, err)
			} else {
				copiedCount++
				fmt.Printf("[%s] Copied %s -> %s\n", prefix, name, uniqueName)
			}
		}
	}

	if copiedCount > 0 {
		fmt.Printf("[%s] Copied %d coverage files to %s\n", prefix, copiedCount, parentDir)
	} else {
		fmt.Printf("[%s] No coverage files found to copy\n", prefix)
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = srcFile.WriteTo(dstFile)
	return err
}

// SetupCoverageExitHandler sets up a defer handler to copy coverage data on exit
func SetupCoverageExitHandler(prefix string) {
	// This can be called with defer in main() functions
	if os.Getenv("GO_INTEGRATION_COVERAGE") != "" {
		defer CopyCoverageDataUp(prefix)
	}
}
