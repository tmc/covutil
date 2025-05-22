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
	"strings"

	"github.com/tmc/covutil/internal/covforest"
)

var cmdServe = &Command{
	UsageLine: "covforest serve [-http=<addr>] [-forest=<path>]",
	Short:     "start HTTP server for exploring the forest",
	Long: `
Serve starts an HTTP server that provides an interactive web interface
for exploring coverage data across multiple trees in the forest.

The -http flag specifies the address and port to listen on (default: ":8080").
The -forest flag specifies the forest file path (default: ~/.covforest/forest.json).

Example:

	covforest serve
	covforest serve -http=:9000
	covforest serve -forest=/path/to/forest.json
`,
}

var (
	serveHTTPAddr = cmdServe.Flag.String("http", ":8080", "HTTP server address")
	serveForest   = cmdServe.Flag.String("forest", "", "forest file path (default: ~/.covforest/forest.json)")
)

func init() {
	cmdServe.Run = runServe
}

func runServe(ctx context.Context, args []string) error {
	forestPath := *serveForest
	if forestPath == "" {
		forestPath = covforest.DefaultForestPath()
	}

	forest, err := covforest.LoadFromFile(forestPath)
	if err != nil {
		return fmt.Errorf("failed to load forest: %v", err)
	}

	// Set up HTTP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveForestIndex(w, r, forest)
	})
	mux.HandleFunc("/api/summary", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(forest.Summary())
	})
	mux.HandleFunc("/api/trees", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(forest.ListTrees())
	})
	mux.HandleFunc("/api/tree/", func(w http.ResponseWriter, r *http.Request) {
		treeID := strings.TrimPrefix(r.URL.Path, "/api/tree/")
		tree, err := forest.GetTree(treeID)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tree)
	})

	log.Printf("covforest: serving forest at http://%s", *serveHTTPAddr)
	log.Printf("covforest: loaded %d trees from %s", len(forest.Trees), forestPath)

	server := &http.Server{
		Addr:    *serveHTTPAddr,
		Handler: mux,
	}

	return server.ListenAndServe()
}

func serveForestIndex(w http.ResponseWriter, r *http.Request, forest *covforest.Forest) {
	tmpl := template.Must(template.New("index").Funcs(template.FuncMap{
		"mult": func(a, b float64) float64 { return a * b },
	}).Parse(forestIndexTemplate))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, forest.Summary()); err != nil {
		log.Printf("covforest: template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

const forestIndexTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Coverage Forest</title>
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
			background: #2f3349;
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
		.trees-section {
			padding: 30px;
		}
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
			<h1>Coverage Forest</h1>
			<p>Multi-tree coverage analysis across machines, repositories, and timelines</p>
		</div>
		
		<div class="summary">
			<h2>Forest Summary</h2>
			<div class="stats">
				<div class="stat">
					<div class="stat-label">Coverage Trees</div>
					<div class="stat-value">{{.TreeCount}}</div>
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
					<div class="stat-label">Overall Coverage</div>
					<div class="stat-value">{{printf "%.1f" (mult .CoverageRate 100)}}%</div>
				</div>
			</div>
		</div>

		<div class="trees-section">
			<h3>Coverage Trees</h3>
			<div id="trees-list">
				<div class="loading">Loading trees...</div>
			</div>
		</div>
	</div>

	<script>
		function loadTrees() {
			fetch('/api/trees')
				.then(response => response.json())
				.then(data => {
					const container = document.getElementById('trees-list');
					if (data.length === 0) {
						container.innerHTML = '<div class="loading">No trees found in forest.</div>';
						return;
					}
					
					let html = '<div style="display: grid; gap: 15px;">';
					data.forEach(tree => {
						const coverage = tree.CoverageTree ? 
							(tree.CoverageTree.Summary ? tree.CoverageTree.Summary.CoverageRate * 100 : 0) : 0;
						
						html += ` + "`" + `
							<div style="border: 1px solid #e1e4e8; border-radius: 6px; padding: 20px; background: white;">
								<h4 style="margin: 0 0 10px 0; color: #0366d6;">${tree.Name}</h4>
								<div style="font-size: 0.9em; color: #586069; margin-bottom: 10px;">
									<strong>ID:</strong> ${tree.ID}<br>
									<strong>Machine:</strong> ${tree.Source.Machine || 'Unknown'}<br>
									<strong>Repository:</strong> ${tree.Source.Repository || 'Unknown'}<br>
									<strong>Branch:</strong> ${tree.Source.Branch || 'Unknown'}<br>
									<strong>Updated:</strong> ${new Date(tree.LastUpdated).toLocaleString()}
								</div>
								<div style="font-size: 1.2em; font-weight: 600; color: ${coverage > 80 ? '#28a745' : coverage > 50 ? '#fb8500' : '#d73a49'};">
									Coverage: ${coverage.toFixed(1)}%
								</div>
							</div>
						` + "`" + `;
					});
					html += '</div>';
					
					container.innerHTML = html;
				})
				.catch(error => {
					console.error('Error loading trees:', error);
					document.getElementById('trees-list').innerHTML = 
						'<div class="loading">Error loading trees.</div>';
				});
		}

		// Load trees on page load
		loadTrees();
	</script>
</body>
</html>`
