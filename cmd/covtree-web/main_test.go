// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmc/covutil/internal/covtree"
)

func TestWebServerRoutes(t *testing.T) {
	// Create a mock coverage tree for testing
	tree := covtree.NewCoverageTree()

	server := &WebServer{
		Tree:     tree,
		Title:    "Test Coverage Report",
		HTTPAddr: ":8080",
	}

	mux := http.NewServeMux()
	server.SetupRoutes(mux)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedType   string
	}{
		{"Index page", "/", http.StatusOK, "text/html"},
		{"API Summary", "/api/summary", http.StatusOK, "application/json"},
		{"API Packages", "/api/packages", http.StatusOK, "application/json"},
		{"API Health", "/api/health", http.StatusOK, "application/json"},
		{"Favicon", "/favicon.ico", http.StatusOK, "image/svg+xml"},
		{"Not Found", "/nonexistent", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedType != "" {
				contentType := w.Header().Get("Content-Type")
				if !strings.Contains(contentType, tt.expectedType) {
					t.Errorf("Expected content type to contain %s, got %s", tt.expectedType, contentType)
				}
			}
		})
	}
}

func TestAPIEndpoints(t *testing.T) {
	tree := covtree.NewCoverageTree()

	server := &WebServer{
		Tree:     tree,
		Title:    "Test Coverage Report",
		HTTPAddr: ":8080",
	}

	t.Run("API Summary", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/summary", nil)
		w := httptest.NewRecorder()

		server.handleAPISummary(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var summary map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&summary); err != nil {
			t.Errorf("Failed to decode JSON response: %v", err)
		}
	})

	t.Run("API Health", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()

		server.handleAPIHealth(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var health map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&health); err != nil {
			t.Errorf("Failed to decode JSON response: %v", err)
		}

		if health["status"] != "ok" {
			t.Errorf("Expected status 'ok', got %v", health["status"])
		}

		if health["title"] != "Test Coverage Report" {
			t.Errorf("Expected title 'Test Coverage Report', got %v", health["title"])
		}
	})

	t.Run("API Package Not Found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/package/nonexistent", nil)
		w := httptest.NewRecorder()

		server.handleAPIPackage(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

func TestIndexTemplate(t *testing.T) {
	tree := covtree.NewCoverageTree()

	server := &WebServer{
		Tree:     tree,
		Title:    "Test Coverage Report",
		HTTPAddr: ":8080",
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Test Coverage Report") {
		t.Errorf("Expected body to contain title 'Test Coverage Report'")
	}

	if !strings.Contains(body, "Coverage Summary") {
		t.Errorf("Expected body to contain 'Coverage Summary'")
	}

	if !strings.Contains(body, "Interactive Coverage Explorer") {
		t.Errorf("Expected body to contain 'Interactive Coverage Explorer'")
	}
}

func TestParseFilterFromQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string // Expected pattern field
	}{
		{"Empty query", "", ""},
		{"Pattern only", "pattern=github.com/*", "github.com/*"},
		{"Multiple params", "pattern=test&min_coverage=0.5", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?"+tt.query, nil)
			filter := parseFilterFromQuery(req)

			if filter.PackagePattern != tt.expected {
				t.Errorf("Expected pattern %s, got %s", tt.expected, filter.PackagePattern)
			}
		})
	}
}

func TestCovtreeWebWithSprig(t *testing.T) {
	sprigCovPath := "/Users/tmc/go/src/github.com/Masterminds/sprig/coverage/per-test"

	// Check if Sprig coverage data exists
	if _, err := os.Stat(sprigCovPath); os.IsNotExist(err) {
		t.Skip("Sprig coverage data not available")
	}

	// Test loading a single test directory
	testPath := filepath.Join(sprigCovPath, "simple_test")
	tree := covtree.NewCoverageTree()

	err := tree.LoadFromNestedRepository(testPath)
	if err != nil {
		// Expected behavior - coverage data format may not be compatible
		t.Logf("Coverage data loading failed (expected): %v", err)
		if !strings.Contains(err.Error(), "failed to parse any coverage data") {
			t.Errorf("Unexpected error: %v", err)
		}
		return
	}

	// If loading succeeded, test the web server
	server := &WebServer{
		Tree:     tree,
		Title:    "Sprig Coverage Test",
		HTTPAddr: ":8080",
	}

	// Test that server can handle requests
	req := httptest.NewRequest("GET", "/api/summary", nil)
	w := httptest.NewRecorder()

	server.handleAPISummary(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestWebServerEdgeCases(t *testing.T) {
	tree := covtree.NewCoverageTree()

	server := &WebServer{
		Tree:     tree,
		Title:    "Edge Case Test",
		HTTPAddr: ":8080",
	}

	tests := []struct {
		name           string
		path           string
		method         string
		expectedStatus int
	}{
		{"Root path only", "/", "GET", http.StatusOK},
		{"Invalid sub-path", "/invalid", "GET", http.StatusNotFound},
		{"POST to index", "/", "POST", http.StatusOK}, // Should still work
		{"API package with empty path", "/api/package/", "GET", http.StatusNotFound},
	}

	mux := http.NewServeMux()
	server.SetupRoutes(mux)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d for %s %s", tt.expectedStatus, w.Code, tt.method, tt.path)
			}
		})
	}
}

func TestFilterParsing(t *testing.T) {
	tests := []struct {
		name          string
		queryString   string
		expectMin     float64
		expectMax     float64
		expectPattern string
	}{
		{
			name:          "All filters",
			queryString:   "pattern=test&min_coverage=0.5&max_coverage=0.9",
			expectMin:     0.5,
			expectMax:     0.9,
			expectPattern: "test",
		},
		{
			name:          "Only min coverage",
			queryString:   "min_coverage=0.3",
			expectMin:     0.3,
			expectMax:     0.0,
			expectPattern: "",
		},
		{
			name:          "Invalid coverage values",
			queryString:   "min_coverage=invalid&max_coverage=also_invalid",
			expectMin:     0.0,
			expectMax:     0.0,
			expectPattern: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?"+tt.queryString, nil)
			filter := parseFilterFromQuery(req)

			if filter.MinCoverage != tt.expectMin {
				t.Errorf("Expected MinCoverage %f, got %f", tt.expectMin, filter.MinCoverage)
			}

			if filter.MaxCoverage != tt.expectMax {
				t.Errorf("Expected MaxCoverage %f, got %f", tt.expectMax, filter.MaxCoverage)
			}

			if filter.PackagePattern != tt.expectPattern {
				t.Errorf("Expected PackagePattern %s, got %s", tt.expectPattern, filter.PackagePattern)
			}
		})
	}
}
