// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/tmc/covutil/covtree"
)

var cmdServe = &Command{
	UsageLine: "covtree serve -i=<directory> -http=<addr>",
	Short:     "start HTTP server for interactive coverage exploration",
	Long: `
Serve starts an HTTP server that provides an interactive web interface
for exploring coverage data.

The -i flag specifies a directory to scan recursively for coverage data.
The directory can contain nested subdirectories with coverage data files
produced by running "go build -cover" or similar. All found coverage
directories will be processed.

The -http flag specifies the address and port to listen on (e.g., ":8080").

Example:

	covtree serve -i=./coverage-repo -http=:8080
	covtree serve -i=/path/to/nested/coverage -http=localhost:9000
`,
}

var (
	serveInputDir = cmdServe.Flag.String("i", "", "input directory to scan recursively for coverage data")
	serveHTTPAddr = cmdServe.Flag.String("http", ":8080", "HTTP server address")
)

func init() {
	cmdServe.Run = runServe
}

func runServe(ctx context.Context, args []string) error {
	if *serveInputDir == "" {
		return fmt.Errorf("must specify input directory with -i flag")
	}

	// Validate directory exists
	if _, err := os.Stat(*serveInputDir); os.IsNotExist(err) {
		return fmt.Errorf("input directory does not exist: %s", *serveInputDir)
	}

	// Load coverage data from nested repository
	tree := covtree.NewCoverageTree()
	if err := tree.LoadFromNestedRepository(*serveInputDir); err != nil {
		return fmt.Errorf("failed to load coverage data from %s: %v", *serveInputDir, err)
	}

	// Set up HTTP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveIndex(w, r, tree)
	})
	mux.HandleFunc("/api/summary", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tree.Summary())
	})
	mux.HandleFunc("/api/packages", func(w http.ResponseWriter, r *http.Request) {
		filterObj := parseFilterFromQuery(r)
		packages := tree.FilterPackages(filterObj)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(packages)
	})
	mux.HandleFunc("/api/package/", func(w http.ResponseWriter, r *http.Request) {
		packagePath := strings.TrimPrefix(r.URL.Path, "/api/package/")
		pkg := tree.GetPackage(packagePath)
		if pkg == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pkg)
	})

	log.Printf("covtree: serving coverage data at http://%s", *serveHTTPAddr)
	log.Printf("covtree: loaded %d packages", len(tree.Packages))

	server := &http.Server{
		Addr:    *serveHTTPAddr,
		Handler: mux,
	}

	return server.ListenAndServe()
}

func parseFilterFromQuery(r *http.Request) covtree.Filter {
	filterObj := covtree.Filter{}

	if pattern := r.URL.Query().Get("pattern"); pattern != "" {
		filterObj.PackagePattern = pattern
	}

	if minCov := r.URL.Query().Get("min_coverage"); minCov != "" {
		if f, err := strconv.ParseFloat(minCov, 64); err == nil {
			filterObj.MinCoverage = f
		}
	}

	if maxCov := r.URL.Query().Get("max_coverage"); maxCov != "" {
		if f, err := strconv.ParseFloat(maxCov, 64); err == nil {
			filterObj.MaxCoverage = f
		}
	}

	return filterObj
}

func serveIndex(w http.ResponseWriter, r *http.Request, tree *covtree.CoverageTree) {
	tmpl := template.Must(template.New("index").Funcs(template.FuncMap{
		"mult": func(a, b float64) float64 { return a * b },
	}).Parse(indexTemplate))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, tree.Summary()); err != nil {
		log.Printf("covtree: template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

const indexTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Coverage Report</title>
	<style>
		body {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
			line-height: 1.6;
			margin: 0;
			padding: 20px;
			background-color: #f8f9fa;
		}
		.container {
			max-width: 1200px;
			margin: 0 auto;
			background: white;
			border-radius: 8px;
			box-shadow: 0 2px 10px rgba(0,0,0,0.1);
		}
		.header {
			background: #0f1419;
			color: white;
			padding: 20px 30px;
			border-radius: 8px 8px 0 0;
		}
		.header h1 {
			margin: 0;
			font-size: 1.8em;
			font-weight: 600;
		}
		.summary {
			padding: 30px;
			border-bottom: 1px solid #e1e4e8;
		}
		.summary h2 {
			margin-top: 0;
			color: #24292e;
			font-size: 1.4em;
		}
		.stats {
			display: grid;
			grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
			gap: 20px;
			margin-top: 20px;
		}
		.stat {
			background: #f6f8fa;
			padding: 15px;
			border-radius: 6px;
			border: 1px solid #e1e4e8;
		}
		.stat-label {
			font-size: 0.9em;
			color: #586069;
			margin-bottom: 5px;
		}
		.stat-value {
			font-size: 1.8em;
			font-weight: 600;
			color: #24292e;
		}
		.coverage-rate {
			color: {{if lt .CoverageRate 0.5}}#d73a49{{else if lt .CoverageRate 0.8}}#fb8500{{else}}#28a745{{end}};
		}
		.controls {
			padding: 30px;
			background: #f6f8fa;
			border-bottom: 1px solid #e1e4e8;
		}
		.controls h3 {
			margin-top: 0;
			margin-bottom: 15px;
			color: #24292e;
		}
		.filter-row {
			display: flex;
			gap: 10px;
			flex-wrap: wrap;
			align-items: center;
		}
		.filter-row input {
			padding: 8px 12px;
			border: 1px solid #d1d5da;
			border-radius: 6px;
			font-size: 14px;
		}
		.filter-row button {
			background: #0366d6;
			color: white;
			border: none;
			padding: 8px 16px;
			border-radius: 6px;
			cursor: pointer;
			font-size: 14px;
		}
		.filter-row button:hover {
			background: #0256cc;
		}
		.packages {
			padding: 30px;
		}
		.packages h3 {
			margin-top: 0;
			color: #24292e;
		}
		.package {
			border: 1px solid #e1e4e8;
			border-radius: 6px;
			margin-bottom: 15px;
			overflow: hidden;
		}
		.package-header {
			background: #f6f8fa;
			padding: 15px 20px;
			border-bottom: 1px solid #e1e4e8;
			cursor: pointer;
			display: flex;
			justify-content: space-between;
			align-items: center;
		}
		.package-header:hover {
			background: #f1f3f4;
		}
		.package-name {
			font-weight: 600;
			color: #0366d6;
		}
		.package-coverage {
			font-weight: 600;
		}
		.package-content {
			display: none;
			padding: 20px;
		}
		.function {
			padding: 8px 0;
			border-bottom: 1px solid #f1f3f4;
			display: flex;
			justify-content: space-between;
		}
		.function:last-child {
			border-bottom: none;
		}
		.function-name {
			font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
			font-size: 0.9em;
		}
		.coverage-high { color: #28a745; }
		.coverage-medium { color: #fb8500; }
		.coverage-low { color: #d73a49; }
		.loading {
			text-align: center;
			padding: 40px;
			color: #586069;
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1>Coverage Report</h1>
		</div>
		
		<div class="summary">
			<h2>Summary</h2>
			<div class="stats">
				<div class="stat">
					<div class="stat-label">Total Packages</div>
					<div class="stat-value">{{.TotalPackages}}</div>
				</div>
				<div class="stat">
					<div class="stat-label">Total Lines</div>
					<div class="stat-value">{{.TotalLines}}</div>
				</div>
				<div class="stat">
					<div class="stat-label">Covered Lines</div>
					<div class="stat-value">{{.CoveredLines}}</div>
				</div>
				<div class="stat">
					<div class="stat-label">Coverage</div>
					<div class="stat-value coverage-rate">{{printf "%.1f" (mult .CoverageRate 100)}}%</div>
				</div>
			</div>
		</div>

		<div class="controls">
			<h3>Filter Packages</h3>
			<div class="filter-row">
				<input type="text" id="filter" placeholder="Package pattern (e.g., github.com/*)">
				<input type="number" id="minCov" placeholder="Min coverage %" step="0.1" min="0" max="100">
				<input type="number" id="maxCov" placeholder="Max coverage %" step="0.1" min="0" max="100">
				<button onclick="loadPackages()">Apply Filter</button>
			</div>
		</div>

		<div class="packages">
			<h3>Packages</h3>
			<div id="packages-list">
				<div class="loading">Loading packages...</div>
			</div>
		</div>
	</div>

	<script>
		function mult(a, b) { return a * b; }

		function loadPackages() {
			const filter = document.getElementById('filter').value;
			const minCov = document.getElementById('minCov').value;
			const maxCov = document.getElementById('maxCov').value;
			
			let url = '/api/packages?';
			if (filter) url += 'pattern=' + encodeURIComponent(filter) + '&';
			if (minCov) url += 'min_coverage=' + (parseFloat(minCov) / 100) + '&';
			if (maxCov) url += 'max_coverage=' + (parseFloat(maxCov) / 100) + '&';
			
			fetch(url)
				.then(response => response.json())
				.then(data => {
					const container = document.getElementById('packages-list');
					if (data.length === 0) {
						container.innerHTML = '<div class="loading">No packages match the filter criteria.</div>';
						return;
					}
					
					let html = '';
					data.forEach(pkg => {
						const coverageClass = pkg.CoverageRate > 0.8 ? 'coverage-high' : 
											pkg.CoverageRate > 0.5 ? 'coverage-medium' : 'coverage-low';
						const packageId = pkg.ImportPath.replace(/[^a-zA-Z0-9]/g, '_');
						
						html += ` + "`" + `
							<div class="package">
								<div class="package-header" onclick="togglePackage('${packageId}')">
									<span class="package-name">${pkg.ImportPath}</span>
									<span class="package-coverage ${coverageClass}">
										${(pkg.CoverageRate * 100).toFixed(1)}% (${pkg.CoveredLines}/${pkg.TotalLines})
									</span>
								</div>
								<div class="package-content" id="content-${packageId}">
						` + "`" + `;
						
						if (pkg.Functions && pkg.Functions.length > 0) {
							pkg.Functions.forEach(fn => {
								const fnCoverageClass = fn.CoverageRate > 0.8 ? 'coverage-high' : 
													   fn.CoverageRate > 0.5 ? 'coverage-medium' : 'coverage-low';
								html += ` + "`" + `
									<div class="function">
										<span class="function-name">${fn.Name}</span>
										<span class="${fnCoverageClass}">${(fn.CoverageRate * 100).toFixed(1)}%</span>
									</div>
								` + "`" + `;
							});
						} else {
							html += '<div class="function">No functions found</div>';
						}
						
						html += '</div></div>';
					});
					
					container.innerHTML = html;
				})
				.catch(error => {
					console.error('Error loading packages:', error);
					document.getElementById('packages-list').innerHTML = 
						'<div class="loading">Error loading packages. Please try again.</div>';
				});
		}

		function togglePackage(packageId) {
			const content = document.getElementById('content-' + packageId);
			content.style.display = content.style.display === 'none' ? 'block' : 'none';
		}

		// Load packages on page load
		loadPackages();
	</script>
</body>
</html>`
