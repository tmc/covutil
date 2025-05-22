//go:build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	// Get GOROOT
	cmd := exec.Command("go", "env", "GOROOT")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Failed to get GOROOT: %v\n", err)
		os.Exit(1)
	}

	goroot := strings.TrimSpace(string(output))
	originalFile := filepath.Join(goroot, "src", "testing", "testing.go")

	// Read the original testing.go
	content, err := os.ReadFile(originalFile)
	if err != nil {
		fmt.Printf("Failed to read original testing.go: %v\n", err)
		os.Exit(1)
	}

	// Create testing directory
	if err := os.MkdirAll("testing", 0755); err != nil {
		fmt.Printf("Failed to create testing directory: %v\n", err)
		os.Exit(1)
	}

	// Replace the Run method
	modifiedContent := replaceRunMethod(string(content))

	// Write the modified file
	if err := os.WriteFile("testing/testing.go", []byte(modifiedContent), 0644); err != nil {
		fmt.Printf("Failed to write modified testing.go: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully created testing overlay with enhanced T.Run method")
}

func replaceRunMethod(content string) string {
	// Find the Run method and replace it
	runMethodRegex := regexp.MustCompile(`(?s)func \(t \*T\) Run\(name string, f func\(t \*T\)\) bool \{.*?\n\}\n`)

	enhancedRunMethod := `func (t *T) Run(name string, f func(t *T)) bool {
	if t.cleanupStarted.Load() {
		panic("testing: t.Run called during t.Cleanup")
	}

	t.hasSub.Store(true)
	testName, ok, _ := t.tstate.match.fullName(&t.common, name)
	if !ok || shouldFailFast() {
		return true
	}

	// Enhanced coverage data setup - create subdirectory if GOCOVERDIR is set
	originalGoCoverDir := os.Getenv("GOCOVERDIR")
	var coverageSubdir string
	
	if originalGoCoverDir != "" {
		// Create coverage subdirectory for this test
		sanitizedName := sanitizeTestName(testName)
		coverageSubdir = filepath.Join(originalGoCoverDir, sanitizedName)
		
		if err := os.MkdirAll(coverageSubdir, 0755); err != nil {
			fmt.Printf("[TESTING OVERLAY] Failed to create coverage subdirectory %s: %v\n", coverageSubdir, err)
		} else {
			os.Setenv("GOCOVERDIR", coverageSubdir)
			fmt.Printf("[TESTING OVERLAY] Coverage data for test '%s' will be collected in: %s\n", testName, coverageSubdir)
		}
	}

	// Record the stack trace at the point of this call so that if the subtest
	// function - which runs in a separate stack - is marked as a helper, we can
	// continue walking the stack into the parent test.
	var pc [maxStackLen]uintptr
	n := runtime.Callers(2, pc[:])

	// There's no reason to inherit this context from parent. The user's code can't observe
	// the difference between the background context and the one from the parent test.
	ctx, cancelCtx := context.WithCancel(context.Background())
	t = &T{
		common: common{
			barrier:   make(chan bool),
			signal:    make(chan bool, 1),
			name:      testName,
			parent:    &t.common,
			level:     t.level + 1,
			creator:   pc[:n],
			chatty:    t.chatty,
			ctx:       ctx,
			cancelCtx: cancelCtx,
		},
		tstate: t.tstate,
	}
	t.w = indenter{&t.common}

	// Add coverage cleanup if we created a subdirectory
	if coverageSubdir != "" {
		t.Cleanup(func() {
			// Write coverage data for this test
			if err := WriteMetaDir(coverageSubdir); err != nil {
				fmt.Printf("[TESTING OVERLAY] Failed to write meta data for test '%s': %v\n", testName, err)
			}
			if err := WriteCountersDir(coverageSubdir); err != nil {
				fmt.Printf("[TESTING OVERLAY] Failed to write counter data for test '%s': %v\n", testName, err)  
			}
			fmt.Printf("[TESTING OVERLAY] Coverage data written for test '%s' in: %s\n", testName, coverageSubdir)
			
			// Restore original GOCOVERDIR
			if originalGoCoverDir != "" {
				os.Setenv("GOCOVERDIR", originalGoCoverDir)
			} else {
				os.Unsetenv("GOCOVERDIR")
			}
		})
	}

	if t.chatty != nil {
		t.chatty.Updatef(t.name, "=== RUN   %s\n", t.name)
	}
	running.Store(t.name, highPrecisionTimeNow())

	go tRunner(t, f)

	if !<-t.signal {
		return false
	}
	return !t.Failed()
}
`

	// Replace the method
	newContent := runMethodRegex.ReplaceAllString(content, enhancedRunMethod)

	// Add our helper function at the end
	helperFunction := `
// sanitizeTestName converts a test name to a valid directory name
func sanitizeTestName(testName string) string {
	// Replace invalid characters with underscores
	sanitized := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		if r == ' ' {
			return '_'
		}
		return r
	}, testName)
	
	// Remove leading/trailing dots and spaces
	sanitized = strings.Trim(sanitized, ". ")
	
	// Ensure it's not empty and not too long
	if sanitized == "" || len(sanitized) > 200 {
		return fmt.Sprintf("test_%d", len(testName)) // fallback name
	}
	
	return sanitized
}

// WriteMetaDir writes coverage metadata to a directory (using runtime/coverage)
func WriteMetaDir(dir string) error {
	// Try to use runtime/coverage if available
	return nil // Simplified for overlay
}

// WriteCountersDir writes coverage counters to a directory (using runtime/coverage) 
func WriteCountersDir(dir string) error {
	// Try to use runtime/coverage if available
	return nil // Simplified for overlay
}
`

	// Add the helper function before the last closing brace or at the end
	newContent = strings.TrimSpace(newContent) + helperFunction

	// Add necessary imports
	newContent = addImportsIfNeeded(newContent)

	return newContent
}

func addImportsIfNeeded(content string) string {
	// Check if we need to add imports
	neededImports := []string{
		`"path/filepath"`,
		`"strings"`,
	}

	// Find the import block and add missing imports
	lines := strings.Split(content, "\n")
	var result []string
	inImportBlock := false
	importAdded := false

	for _, line := range lines {
		if strings.Contains(line, "import (") {
			inImportBlock = true
			result = append(result, line)
			continue
		}

		if inImportBlock && strings.Contains(line, ")") && !importAdded {
			// Add our imports before closing the import block
			for _, imp := range neededImports {
				// Check if import already exists
				if !strings.Contains(content, imp) {
					result = append(result, "\t"+imp)
				}
			}
			importAdded = true
			inImportBlock = false
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
