package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	verbose  = flag.Bool("v", false, "Verbose output")
	timeout  = flag.String("timeout", "10m", "Test timeout per package")
	coverDir = flag.String("coverdir", ".coverage", "Coverage data directory")
	clean    = flag.Bool("clean", true, "Remove coverage directory before running")
	include  = flag.String("include", ".", "Regular expression to match module paths to include (e.g., '^(cmd/|exp/)' to test only cmd/ and exp/ subdirectories)")
)

func main() {
	flag.Parse()

	// Use current directory as root
	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	// Create coverage directory
	coverageDir := filepath.Join(root, *coverDir)
	if *clean {
		if err := os.RemoveAll(coverageDir); err != nil {
			log.Fatalf("Failed to remove old coverage directory: %v", err)
		}
	}
	if err := os.MkdirAll(coverageDir, 0755); err != nil {
		log.Fatalf("Failed to create coverage directory: %v", err)
	}

	fmt.Printf("Finding all Go modules under %s...\n", root)

	// Find all go.mod files
	modules, err := findGoModules(root)
	if err != nil {
		log.Fatalf("Failed to find Go modules: %v", err)
	}

	// Compile include regexp
	includeRe, err := regexp.Compile(*include)
	if err != nil {
		log.Fatalf("Invalid include regexp: %v", err)
	}

	// Filter modules based on include pattern
	var filteredModules []string
	for _, modPath := range modules {
		modDir := filepath.Dir(modPath)
		relPath, _ := filepath.Rel(root, modDir)
		if relPath == "" {
			relPath = "."
		}

		if includeRe.MatchString(relPath) {
			filteredModules = append(filteredModules, modPath)
		} else if *verbose {
			fmt.Printf("Skipping module: %s (doesn't match include pattern)\n", relPath)
		}
	}

	modules = filteredModules
	fmt.Printf("Found %d Go modules (after filtering)\n", len(modules))

	// Run tests for each module
	failed := false
	for i, modPath := range modules {
		modDir := filepath.Dir(modPath)
		relPath, _ := filepath.Rel(root, modDir)
		if relPath == "" {
			relPath = "."
		}

		fmt.Printf("\n[%d/%d] Testing module: %s\n", i+1, len(modules), relPath)

		// Create module-specific coverage directory
		safeName := strings.ReplaceAll(relPath, string(os.PathSeparator), "_")
		if safeName == "." {
			safeName = "root"
		}
		modCoverDir := filepath.Join(coverageDir, "modules", safeName)
		if err := os.MkdirAll(modCoverDir, 0755); err != nil {
			log.Printf("Failed to create module coverage directory: %v", err)
			continue
		}

		// Run tests with coverage using -args -test.gocoverdir
		cmd := exec.Command("go", "test",
			"-timeout", *timeout,
			"-cover",
			"./...",
			"-args", "-test.gocoverdir="+modCoverDir,
		)
		cmd.Dir = modDir

		var output []byte
		var err error

		if *verbose {
			fmt.Printf("Running tests with -args -test.gocoverdir=%s\n", modCoverDir)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
		} else {
			output, err = cmd.CombinedOutput()
		}
		if err != nil {
			log.Printf("⚠️  Tests failed for %s: %v", relPath, err)
			if !*verbose && len(output) > 0 {
				// Show last few lines of output for context
				lines := strings.Split(string(output), "\n")
				start := len(lines) - 10
				if start < 0 {
					start = 0
				}
				fmt.Printf("    Last %d lines of output:\n", len(lines)-start)
				for i := start; i < len(lines); i++ {
					if lines[i] != "" {
						fmt.Printf("    %s\n", lines[i])
					}
				}
			}
			failed = true
		} else {
			fmt.Printf("✓ Tests passed for %s\n", relPath)
			if !*verbose && len(output) > 0 {
				// Check if any coverage was actually generated
				if strings.Contains(string(output), "no test files") {
					fmt.Printf("    Note: No test files found\n")
				} else if strings.Contains(string(output), "no packages to test") {
					fmt.Printf("    Note: No packages to test\n")
				}
			}
		}
	}

	// Find all coverage data
	fmt.Printf("\nLooking for coverage data...\n")
	var coverDirs []string
	err = filepath.Walk(coverageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Check if directory contains coverage data
			entries, _ := os.ReadDir(path)
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), "covcounters.") || strings.HasPrefix(entry.Name(), "covmeta.") {
					coverDirs = append(coverDirs, path)
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to walk coverage directory: %v", err)
	}

	if len(coverDirs) > 0 {
		fmt.Printf("Found coverage data in %d directories\n", len(coverDirs))

		// Copy all coverage files to root directory
		fmt.Printf("\nCopying coverage data to root...\n")
		copiedFiles := 0
		for _, dir := range coverDirs {
			entries, err := os.ReadDir(dir)
			if err != nil {
				log.Printf("Warning: Failed to read directory %s: %v", dir, err)
				continue
			}

			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), "covcounters.") || strings.HasPrefix(entry.Name(), "covmeta.") {
					src := filepath.Join(dir, entry.Name())
					dst := filepath.Join(coverageDir, entry.Name())

					// Check if file already exists with same name
					if _, err := os.Stat(dst); err == nil {
						// File exists, create unique name
						base := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
						ext := filepath.Ext(entry.Name())
						for i := 1; ; i++ {
							dst = filepath.Join(coverageDir, fmt.Sprintf("%s_%d%s", base, i, ext))
							if _, err := os.Stat(dst); os.IsNotExist(err) {
								break
							}
						}
					}

					// Read source file
					data, err := os.ReadFile(src)
					if err != nil {
						log.Printf("Warning: Failed to read %s: %v", src, err)
						continue
					}

					// Write to destination
					if err := os.WriteFile(dst, data, 0644); err != nil {
						log.Printf("Warning: Failed to write %s: %v", dst, err)
						continue
					}

					copiedFiles++
					if *verbose {
						relDir, _ := filepath.Rel(coverageDir, dir)
						fmt.Printf("  Copied %s from %s\n", entry.Name(), relDir)
					}
				}
			}
		}
		fmt.Printf("✓ Copied %d coverage files to %s\n", copiedFiles, coverageDir)

		// Calculate coverage percentage from root directory
		fmt.Printf("\nDetailed coverage report:\n")
		percentCmd := exec.Command("go", "tool", "covdata", "percent", "-i="+coverageDir)
		percentCmd.Stdout = os.Stdout
		percentCmd.Stderr = os.Stderr
		if err := percentCmd.Run(); err != nil {
			log.Printf("Warning: Failed to calculate coverage: %v", err)
		}

		// Calculate and display total coverage
		fmt.Printf("\nTotal coverage summary:\n")
		totalOut, err := exec.Command("go", "tool", "covdata", "percent", "-i="+coverageDir).Output()
		if err != nil {
			log.Printf("Warning: Failed to get total coverage: %v", err)
		} else {
			coverage := calculateTotalCoverage(string(totalOut))
			fmt.Printf("Overall coverage: %.1f%% of statements\n", coverage)
		}
	} else {
		fmt.Printf("No coverage data found. This is expected if tests don't have coverage instrumentation.\n")
		fmt.Printf("To enable coverage collection, ensure test binaries are built with -cover flag.\n")
	}

	if failed {
		fmt.Printf("\n⚠️  Some tests failed. Coverage data was collected where possible.\n")
		// Don't exit with error - we still collected coverage data
	}
}

// findGoModules finds all Go modules under the root directory.
// It first checks if the root itself is a Go module, then looks for nested modules.
func findGoModules(root string) ([]string, error) {
	var modules []string

	// First, check if the current directory is part of a Go module
	if isGoModule(root) {
		modFile := filepath.Join(root, "go.mod")
		// If go.mod doesn't exist in root, find it in parent directories
		if _, err := os.Stat(modFile); os.IsNotExist(err) {
			modFile = findModuleRoot(root)
		}
		if modFile != "" {
			modules = append(modules, modFile)
		}
	}

	// Then walk to find any nested modules (submodules)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor and hidden directories
		if info.IsDir() && (info.Name() == "vendor" || (strings.HasPrefix(info.Name(), ".") && info.Name() != ".")) {
			return filepath.SkipDir
		}

		// Found a go.mod file
		if !info.IsDir() && info.Name() == "go.mod" {
			// Don't add if it's the root module we already added
			if len(modules) == 0 || path != modules[0] {
				modules = append(modules, path)
			}
		}

		return nil
	})

	// If still no modules found, and we're in a Go project, use the root
	if len(modules) == 0 && hasGoFiles(root) {
		// Create a synthetic entry for the current directory
		modules = append(modules, filepath.Join(root, "go.mod"))
	}

	return modules, err
}

// isGoModule checks if a directory is part of a Go module by running go list
func isGoModule(dir string) bool {
	cmd := exec.Command("go", "list", "-m")
	cmd.Dir = dir
	output, err := cmd.Output()
	return err == nil && len(output) > 0
}

// findModuleRoot finds the go.mod file in the current or parent directories
func findModuleRoot(dir string) string {
	current := dir
	for {
		modFile := filepath.Join(current, "go.mod")
		if _, err := os.Stat(modFile); err == nil {
			return modFile
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached the root
			break
		}
		current = parent
	}
	return ""
}

// hasGoFiles checks if a directory contains any .go files
func hasGoFiles(dir string) bool {
	hasGo := false
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor and hidden directories
		if info.IsDir() && (info.Name() == "vendor" || strings.HasPrefix(info.Name(), ".")) {
			return filepath.SkipDir
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go") {
			hasGo = true
			return filepath.SkipAll // Stop walking, we found what we need
		}

		return nil
	})
	return hasGo
}

// calculateTotalCoverage gets the total coverage across all packages
// by using 'go tool covdata func' which provides a total summary
func calculateTotalCoverage(output string) float64 {
	// First try to get the total from 'go tool covdata func' output
	funcOut, err := exec.Command("go", "tool", "covdata", "func", "-i="+*coverDir).Output()
	if err == nil {
		lines := strings.Split(string(funcOut), "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if strings.HasPrefix(line, "total") && strings.Contains(line, "(statements)") {
				// Parse line like: "total			(statements)	92.3%"
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					percentStr := strings.TrimSuffix(parts[len(parts)-1], "%")
					percent, err := strconv.ParseFloat(percentStr, 64)
					if err == nil {
						return percent
					}
				}
			}
		}
	}

	// Fallback: calculate average from package percentages
	lines := strings.Split(output, "\n")
	var totalPercent float64
	var packageCount int

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse lines like: "github.com/tmc/covutil		coverage: 20.2% of statements"
		parts := strings.Fields(line)
		if len(parts) >= 4 && parts[1] == "coverage:" && strings.HasSuffix(parts[3], "%") {
			percentStr := strings.TrimSuffix(parts[3], "%")
			percent, err := strconv.ParseFloat(percentStr, 64)
			if err != nil {
				continue
			}

			totalPercent += percent
			packageCount++
		}
	}

	if packageCount == 0 {
		return 0.0
	}

	return totalPercent / float64(packageCount)
}
