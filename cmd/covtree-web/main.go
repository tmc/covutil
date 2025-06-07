// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Covtree-web is a standalone web server for interactive coverage data exploration.
//
// Usage:
//
//	covtree-web [flags]
//
// The flags are:
//
//	-i directory    input directory to scan recursively for coverage data
//	-http address   HTTP server address (default :8080)
//	-title string   custom title for the web interface
//	-open           open browser automatically after starting server
//
// Example:
//
//	covtree-web -i=./coverage -http=:9000 -title="My Project Coverage"
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/tmc/covutil/covtree"
)

var (
	inputDir    = flag.String("i", "", "input directory to scan recursively for coverage data")
	httpAddr    = flag.String("http", ":8080", "HTTP server address")
	title       = flag.String("title", "Coverage Report", "custom title for the web interface")
	openBrowser = flag.Bool("open", false, "open browser automatically after starting server")
	watch       = flag.Bool("watch", false, "watch directory for changes and reload automatically")
)

func main() {
	log.SetPrefix("covtree-web: ")
	log.SetFlags(0)

	flag.Usage = usage
	flag.Parse()

	if *inputDir == "" {
		fmt.Fprintf(os.Stderr, "covtree-web: must specify input directory with -i flag\n")
		flag.Usage()
		os.Exit(2)
	}

	// Validate directory exists
	if _, err := os.Stat(*inputDir); os.IsNotExist(err) {
		log.Fatalf("input directory does not exist: %s", *inputDir)
	}

	// Load coverage data from nested repository
	tree := covtree.NewCoverageTree()
	log.Printf("loading coverage data from %s...", *inputDir)
	if err := tree.LoadFromNestedRepository(*inputDir); err != nil {
		log.Fatalf("failed to load coverage data from %s: %v", *inputDir, err)
	}

	// Create web server
	server := &WebServer{
		Tree:     tree,
		Title:    *title,
		HTTPAddr: *httpAddr,
	}

	// Set up file watching if requested
	var watchedServer *WatchedWebServer
	if *watch {
		var err error
		watchedServer, err = NewWatchedWebServer(server, *inputDir)
		if err != nil {
			log.Fatalf("failed to create watcher: %v", err)
		}
		defer watchedServer.Close()

		ctx := context.Background()
		if err := watchedServer.StartWatching(ctx); err != nil {
			log.Fatalf("failed to start watching: %v", err)
		}
	}

	// Set up HTTP handlers
	mux := http.NewServeMux()
	server.SetupRoutes(mux)

	httpServer := &http.Server{
		Addr:    *httpAddr,
		Handler: mux,
	}

	log.Printf("loaded %d packages from %s", len(tree.Packages), *inputDir)
	if *watch {
		log.Printf("serving coverage web interface at http://localhost%s (with auto-reload)", *httpAddr)
	} else {
		log.Printf("serving coverage web interface at http://localhost%s", *httpAddr)
	}

	// Open browser if requested
	if *openBrowser {
		go func() {
			time.Sleep(500 * time.Millisecond) // Give server time to start
			openURL(fmt.Sprintf("http://localhost%s", *httpAddr))
		}()
	}

	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Covtree-web is a standalone web server for interactive coverage data exploration.

Usage:

	covtree-web [flags]

The flags are:

	-i directory    input directory to scan recursively for coverage data
	-http address   HTTP server address (default :8080)
	-title string   custom title for the web interface
	-open           open browser automatically after starting server
	-watch          watch directory for changes and reload automatically

Example:

	covtree-web -i=./coverage -http=:9000 -title="My Project Coverage" -open -watch
`)
}

// WebServer handles HTTP requests for coverage data visualization
type WebServer struct {
	Tree     *covtree.CoverageTree
	Title    string
	HTTPAddr string
}

// SetupRoutes configures HTTP routes for the web server
func (s *WebServer) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/summary", s.handleAPISummary)
	mux.HandleFunc("/api/packages", s.handleAPIPackages)
	mux.HandleFunc("/api/package/", s.handleAPIPackage)
	mux.HandleFunc("/api/health", s.handleAPIHealth)
	mux.HandleFunc("/favicon.ico", s.handleFavicon)
}

func (s *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl := template.Must(template.New("index").Funcs(template.FuncMap{
		"mult": func(a, b float64) float64 { return a * b },
	}).Parse(indexTemplate))

	data := struct {
		Title   string
		Summary interface{}
	}{
		Title:   s.Title,
		Summary: s.Tree.Summary(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (s *WebServer) handleAPISummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.Tree.Summary())
}

func (s *WebServer) handleAPIPackages(w http.ResponseWriter, r *http.Request) {
	filterObj := parseFilterFromQuery(r)
	packages := s.Tree.FilterPackages(filterObj)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(packages)
}

func (s *WebServer) handleAPIPackage(w http.ResponseWriter, r *http.Request) {
	packagePath := strings.TrimPrefix(r.URL.Path, "/api/package/")
	pkg := s.Tree.GetPackage(packagePath)
	if pkg == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pkg)
}

func (s *WebServer) handleAPIHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "ok",
		"total_packages": len(s.Tree.Packages),
		"title":          s.Title,
		"timestamp":      time.Now().Unix(),
	})
}

func (s *WebServer) handleFavicon(w http.ResponseWriter, r *http.Request) {
	// Return a simple SVG favicon
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">
		<rect width="32" height="32" fill="#0f1419"/>
		<text x="16" y="22" font-family="monospace" font-size="14" text-anchor="middle" fill="white">C</text>
	</svg>`))
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

// openURL opens the specified URL in the default browser
func openURL(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Printf("failed to open browser: %v", err)
	} else {
		log.Printf("opened browser to %s", url)
	}
}
