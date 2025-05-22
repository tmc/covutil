// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file provides support for analyzing function-level coverage.

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
)

// FunctionCoverage represents coverage metrics for a specific function.
type FunctionCoverage struct {
	File         string            // File containing function
	Name         string            // Function name
	StartLine    int               // Starting line in source
	EndLine      int               // Ending line in source
	TotalLines   int               // Total lines in function
	CoveredLines int               // Covered lines in function
	Coverage     float64           // Coverage percentage (0-100)
	IsAnonymous  bool              // Whether this is an anonymous function
	Parent       *FunctionCoverage // Parent function for anonymous functions
	Children     []*FunctionCoverage // Child anonymous functions
}

// findFunctions parses the Go source files and identifies function boundaries.
func findFunctions(file string, src []byte) ([]FunctionCoverage, error) {
	fset := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fset, file, src, 0)
	if err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}

	var functions []FunctionCoverage
	// Stack to track function nesting for anonymous functions
	var funcStack []*FunctionCoverage

	// Visit all function declarations
	ast.Inspect(parsedFile, func(n ast.Node) bool {
		var name string
		var pos, end token.Pos
		var isAnonymous bool

		switch x := n.(type) {
		case *ast.FuncDecl:
			// Regular function or method
			name = x.Name.Name
			if x.Recv != nil {
				// This is a method, get the receiver type
				if len(x.Recv.List) > 0 {
					recvType := x.Recv.List[0].Type
					if star, ok := recvType.(*ast.StarExpr); ok {
						recvType = star.X
					}
					if ident, ok := recvType.(*ast.Ident); ok {
						name = fmt.Sprintf("(%s).%s", ident.Name, name)
					}
				}
			}
			pos = x.Pos()
			end = x.End()
			isAnonymous = false
		case *ast.FuncLit:
			// Anonymous function
			name = "<anonymous>"
			pos = x.Pos()
			end = x.End()
			isAnonymous = true
		default:
			return true
		}

		if name != "" {
			startPos := fset.Position(pos)
			endPos := fset.Position(end)

			fc := FunctionCoverage{
				File:        file,
				Name:        name,
				StartLine:   startPos.Line,
				EndLine:     endPos.Line,
				TotalLines:  endPos.Line - startPos.Line + 1,
				IsAnonymous: isAnonymous,
				Children:    make([]*FunctionCoverage, 0),
			}

			// Add to functions slice
			functions = append(functions, fc)
			fcPtr := &functions[len(functions)-1]

			// Handle parent-child relationship for anonymous functions
			if isAnonymous && len(funcStack) > 0 {
				// Find the nearest named parent or use the top of the stack
				parent := funcStack[len(funcStack)-1]
				var namedParent *FunctionCoverage = parent

				// Navigate up the stack to find the nearest named parent
				for namedParent.IsAnonymous && namedParent.Parent != nil {
					namedParent = namedParent.Parent
				}

				// Set parent reference and add to parent's children
				fcPtr.Parent = namedParent
				namedParent.Children = append(namedParent.Children, fcPtr)
			}

			// Push this function onto the stack before processing its children
			funcStack = append(funcStack, fcPtr)

			// NOTE: Anonymous functions will be visited in Inspect AFTER we complete the block
			// So when Inspect exits a function (pops), we'll handle that specially
			return true
		}

		// If n is not a function node, check if we've exited a function block
		if n == nil && len(funcStack) > 0 {
			// Pop function from stack when exiting its scope
			funcStack = funcStack[:len(funcStack)-1]
		}

		return true
	})

	return functions, nil
}

// calculateFunctionCoverage updates coverage statistics for functions based on profile data.
func calculateFunctionCoverage(functions []FunctionCoverage, profile *Profile) []FunctionCoverage {
	// Create a map of covered lines
	coveredLines := make(map[int]bool)
	for _, block := range profile.Blocks {
		if block.Count > 0 {
			for line := block.StartLine; line <= block.EndLine; line++ {
				coveredLines[line] = true
			}
		}
	}

	// Map to store aggregate statistics for named functions
	funcMap := make(map[*FunctionCoverage]struct{
		totalLines   int
		coveredLines int
	})

	// Calculate direct coverage for each function first
	for i := range functions {
		covered := 0
		for line := functions[i].StartLine; line <= functions[i].EndLine; line++ {
			if coveredLines[line] {
				covered++
			}
		}
		functions[i].CoveredLines = covered
		if functions[i].TotalLines > 0 {
			functions[i].Coverage = float64(covered) / float64(functions[i].TotalLines) * 100.0
		}

		// For named functions, initialize in the map
		if !functions[i].IsAnonymous {
			funcMap[&functions[i]] = struct {
				totalLines   int
				coveredLines int
			}{
				totalLines:   functions[i].TotalLines,
				coveredLines: functions[i].CoveredLines,
			}
		}
	}

	// Build a list of named functions with pointers
	var namedFunctions []*FunctionCoverage
	for i := range functions {
		if !functions[i].IsAnonymous {
			namedFunctions = append(namedFunctions, &functions[i])
		}
	}

	// Aggregate coverage data from anonymous functions to their parent named functions
	for i := range functions {
		if functions[i].IsAnonymous && functions[i].Parent != nil {
			// Add this anonymous function's coverage to its named parent
			stats := funcMap[functions[i].Parent]
			stats.totalLines += functions[i].TotalLines
			stats.coveredLines += functions[i].CoveredLines
			funcMap[functions[i].Parent] = stats
		}
	}

	// Update named functions with aggregated coverage including their anonymous children
	for fc, stats := range funcMap {
		// Update the coverage stats for named functions
		if stats.totalLines > 0 {
			fc.TotalLines = stats.totalLines
			fc.CoveredLines = stats.coveredLines
			fc.Coverage = float64(stats.coveredLines) / float64(stats.totalLines) * 100.0
		}
	}

	return functions
}

// byIncreasingCoverage sorts FunctionCoverage by increasing coverage percentage
type byIncreasingCoverage []FunctionCoverage

func (s byIncreasingCoverage) Len() int           { return len(s) }
func (s byIncreasingCoverage) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byIncreasingCoverage) Less(i, j int) bool { return s[i].Coverage < s[j].Coverage }

// listUncoveredFunctions returns a report of functions with 0% coverage.
func listUncoveredFunctions(functions []FunctionCoverage) string {
	if *sortable {
		return listUncoveredFunctionsSortable(functions)
	}

	var buf bytes.Buffer

	// Find completely uncovered functions, excluding anonymous functions
	var uncovered []FunctionCoverage
	for _, fc := range functions {
		if fc.Coverage == 0 && !fc.IsAnonymous {
			uncovered = append(uncovered, fc)
		}
	}

	// Sort by file, then by line number
	sort.Slice(uncovered, func(i, j int) bool {
		if uncovered[i].File != uncovered[j].File {
			return uncovered[i].File < uncovered[j].File
		}
		return uncovered[i].StartLine < uncovered[j].StartLine
	})

	// Group by file
	currentFile := ""
	for _, fc := range uncovered {
		file := fc.File
		if !*longNames && pwd != "" {
			rel, err := filepath.Rel(pwd, file)
			if err == nil && !strings.HasPrefix(rel, "..") {
				file = rel
			}
		}

		if file != currentFile {
			if currentFile != "" {
				buf.WriteString("\n")
			}
			buf.WriteString(fmt.Sprintf("%s:\n", file))
			currentFile = file
		}

		buf.WriteString(fmt.Sprintf("  %s (lines %d-%d)\n", fc.Name, fc.StartLine, fc.EndLine))
	}

	return buf.String()
}

// listUncoveredFunctionsSortable returns a report of functions with 0% coverage in a sortable/diffable format.
func listUncoveredFunctionsSortable(functions []FunctionCoverage) string {
	var buf bytes.Buffer
	buf.WriteString("# Uncovered functions (0% coverage)\n")

	// Find completely uncovered functions, excluding anonymous functions
	var uncovered []FunctionCoverage
	for _, fc := range functions {
		if fc.Coverage == 0 && !fc.IsAnonymous {
			uncovered = append(uncovered, fc)
		}
	}

	// Sort by file, then by function name, then by line number
	sort.Slice(uncovered, func(i, j int) bool {
		if uncovered[i].File != uncovered[j].File {
			return uncovered[i].File < uncovered[j].File
		}
		if uncovered[i].Name != uncovered[j].Name {
			return uncovered[i].Name < uncovered[j].Name
		}
		return uncovered[i].StartLine < uncovered[j].StartLine
	})

	// List each uncovered function on a single line
	for _, fc := range uncovered {
		file := fc.File
		if !*longNames && pwd != "" {
			rel, err := filepath.Rel(pwd, file)
			if err == nil && !strings.HasPrefix(rel, "..") {
				file = rel
			}
		}

		buf.WriteString(fmt.Sprintf("%s:%s:%d-%d:0.00%%\n",
			file, fc.Name, fc.StartLine, fc.EndLine))
	}

	return buf.String()
}

// listTopUncoveredFunctions returns a report of the top N least-covered functions.
func listTopUncoveredFunctions(functions []FunctionCoverage, n int) string {
	if *sortable {
		return listTopUncoveredFunctionsSortable(functions, n)
	}

	var buf bytes.Buffer

	// Filter out anonymous functions
	var namedFunctions []FunctionCoverage
	for _, fc := range functions {
		if !fc.IsAnonymous {
			namedFunctions = append(namedFunctions, fc)
		}
	}

	// Sort by coverage percentage (ascending)
	sorted := make([]FunctionCoverage, len(namedFunctions))
	copy(sorted, namedFunctions)
	sort.Sort(byIncreasingCoverage(sorted))

	// Limit to N entries
	if n > len(sorted) {
		n = len(sorted)
	}

	// Generate report
	buf.WriteString(fmt.Sprintf("Top %d functions by lowest coverage percentage:\n\n", n))

	for i := 0; i < n; i++ {
		fc := sorted[i]
		file := fc.File
		if !*longNames && pwd != "" {
			rel, err := filepath.Rel(pwd, file)
			if err == nil && !strings.HasPrefix(rel, "..") {
				file = rel
			}
		}

		buf.WriteString(fmt.Sprintf("%s: %s (%.2f%% covered, %d/%d lines)\n",
			file, fc.Name, fc.Coverage, fc.CoveredLines, fc.TotalLines))
	}

	return buf.String()
}

// listTopUncoveredFunctionsSortable returns a report of the top N least-covered functions in a sortable/diffable format.
func listTopUncoveredFunctionsSortable(functions []FunctionCoverage, n int) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# Top %d functions by lowest coverage percentage\n", n))

	// Filter out anonymous functions
	var namedFunctions []FunctionCoverage
	for _, fc := range functions {
		if !fc.IsAnonymous {
			namedFunctions = append(namedFunctions, fc)
		}
	}

	// Sort by coverage percentage (ascending)
	sorted := make([]FunctionCoverage, len(namedFunctions))
	copy(sorted, namedFunctions)
	sort.Sort(byIncreasingCoverage(sorted))

	// Limit to N entries
	if n > len(sorted) {
		n = len(sorted)
	}

	// List each function on a single line with stable sorting for diffability
	for i := 0; i < n; i++ {
		fc := sorted[i]
		file := fc.File
		if !*longNames && pwd != "" {
			rel, err := filepath.Rel(pwd, file)
			if err == nil && !strings.HasPrefix(rel, "..") {
				file = rel
			}
		}

		buf.WriteString(fmt.Sprintf("%s:%s:%d-%d:%.2f%%:%d/%d\n",
			file, fc.Name, fc.StartLine, fc.EndLine,
			fc.Coverage, fc.CoveredLines, fc.TotalLines))
	}

	return buf.String()
}