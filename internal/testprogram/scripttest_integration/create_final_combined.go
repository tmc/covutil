package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Combine real Go coverage with synthetic scripttest coverage
	output, err := os.Create("final_combined.cov")
	if err != nil {
		panic(err)
	}
	defer output.Close()

	fmt.Fprintln(output, "mode: set")

	// Add real Go coverage
	addCoverageFile(output, "cov_both.out")
	
	// Add synthetic coverage (with path fixes)
	addSyntheticCoverage(output)

	fmt.Println("Created final_combined.cov")
}

func addCoverageFile(output *os.File, filename string) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Warning: couldn't open %s: %v\n", filename, err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") {
			continue
		}
		if strings.TrimSpace(line) != "" {
			fmt.Fprintln(output, line)
		}
	}
}

func addSyntheticCoverage(output *os.File) {
	cwd, _ := os.Getwd()
	scriptPath := filepath.Join(cwd, "testdata", "TestSyntheticCoverage_synthetic_test.txt")
	
	// Add synthetic coverage lines (some executed, some not)
	lines := []struct {
		line    int
		command string
		covered bool
	}{
		{3, "exec main hello World", true},
		{4, "exec cmd1 greet Universe", true},
		{5, "exec cmd2 elaborate Testing", false}, // skipped
		{6, "exec cmd3 flourish Coverage", false}, // skipped
		{9, "go mod init testproject", false},      // skipped
		{10, "go mod tidy", false},                 // skipped
		{11, "mkdir testdir", true},
		{12, `echo "Hello" > test.txt`, true},
		{13, "cat test.txt", true},
		{16, "! exec main invalid-command", true},
		{17, "! exec cmd1 unknown-operation", true},
		{20, "go version", true},
		{21, "go env", true},
	}

	for _, l := range lines {
		executed := 0
		if l.covered {
			executed = 1
		}
		fmt.Fprintf(output, "%s:%d.1,%d.%d 1 %d\n",
			scriptPath, l.line, l.line, len(l.command)+1, executed)
	}
}