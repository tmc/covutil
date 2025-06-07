package covtree_test

import (
	"os"
	"testing"

	"github.com/tmc/covutil/covtree"
)

func TestMetadataExtensions(t *testing.T) {
	// Create a new coverage tree
	tree := covtree.NewCoverageTree()

	// Set metadata manually
	tree.SetMetadata("GoTestName", "TestAPIIntegration")
	tree.SetMetadata("GoModuleName", "github.com/example/service-a")
	tree.SetMetadata("GoTestPackage", "github.com/example/service-a/integration")
	tree.SetMetadata("TestType", "integration")
	tree.SetMetadata("TestRunID", "run-2024-01-15-001")

	// Verify metadata
	if got := tree.GetMetadata("GoTestName"); got != "TestAPIIntegration" {
		t.Errorf("Expected GoTestName to be TestAPIIntegration, got %s", got)
	}

	if got := tree.GetMetadata("TestType"); got != "integration" {
		t.Errorf("Expected TestType to be integration, got %s", got)
	}
}

func TestMetadataFromEnvironment(t *testing.T) {
	// Set environment variables
	os.Setenv("COVUTIL_TEST_RUN_ID", "env-run-001")
	os.Setenv("COVUTIL_MODULE", "github.com/example/service-b")
	os.Setenv("COVUTIL_TEST_NAME", "TestEnvironmentLoading")
	os.Setenv("COVUTIL_TEST_PACKAGE", "github.com/example/service-b/test")
	os.Setenv("COVUTIL_TEST_TYPE", "unit")
	defer func() {
		// Clean up
		os.Unsetenv("COVUTIL_TEST_RUN_ID")
		os.Unsetenv("COVUTIL_MODULE")
		os.Unsetenv("COVUTIL_TEST_NAME")
		os.Unsetenv("COVUTIL_TEST_PACKAGE")
		os.Unsetenv("COVUTIL_TEST_TYPE")
	}()

	// Create tree and load from environment
	tree := covtree.NewCoverageTree()
	tree.LoadMetadataFromEnv()

	// Verify metadata was loaded
	tests := []struct {
		key      string
		expected string
	}{
		{"TestRunID", "env-run-001"},
		{"GoModuleName", "github.com/example/service-b"},
		{"GoTestName", "TestEnvironmentLoading"},
		{"GoTestPackage", "github.com/example/service-b/test"},
		{"TestType", "unit"},
	}

	for _, tt := range tests {
		if got := tree.GetMetadata(tt.key); got != tt.expected {
			t.Errorf("Expected %s to be %s, got %s", tt.key, tt.expected, got)
		}
	}
}

func TestMetadataFiltering(t *testing.T) {
	// Create a mock tree with packages having different metadata
	tree := covtree.NewCoverageTree()

	// Simulate packages with metadata
	pkg1 := &covtree.PackageNode{
		ImportPath: "github.com/example/pkg1",
		Name:       "pkg1",
		Metadata: map[string]string{
			"GoTestName": "TestUnit",
			"TestType":   "unit",
		},
	}

	pkg2 := &covtree.PackageNode{
		ImportPath: "github.com/example/pkg2",
		Name:       "pkg2",
		Metadata: map[string]string{
			"GoTestName": "TestIntegration",
			"TestType":   "integration",
		},
	}

	pkg3 := &covtree.PackageNode{
		ImportPath: "github.com/example/pkg3",
		Name:       "pkg3",
		Metadata: map[string]string{
			"GoTestName": "TestUnitAdvanced",
			"TestType":   "unit",
		},
	}

	tree.Packages[pkg1.ImportPath] = pkg1
	tree.Packages[pkg2.ImportPath] = pkg2
	tree.Packages[pkg3.ImportPath] = pkg3

	// Test filtering by metadata
	filter := covtree.Filter{
		Metadata: map[string]string{
			"TestType": "unit",
		},
	}

	filtered := tree.FilterPackages(filter)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 packages with TestType=unit, got %d", len(filtered))
	}

	// Test wildcard filtering
	filter2 := covtree.Filter{
		Metadata: map[string]string{
			"GoTestName": "TestUnit*",
		},
	}

	filtered2 := tree.FilterPackages(filter2)
	if len(filtered2) != 2 {
		t.Errorf("Expected 2 packages matching TestUnit*, got %d", len(filtered2))
	}
}

func ExampleCoverageTree_SetMetadata() {
	// Create a coverage tree for cross-module testing
	tree := covtree.NewCoverageTree()

	// Set metadata to track which test generated this coverage
	tree.SetMetadata("GoTestName", "TestAPIEndToEnd")
	tree.SetMetadata("GoModuleName", "github.com/myorg/api-gateway")
	tree.SetMetadata("GoTestPackage", "github.com/myorg/api-gateway/e2e")
	tree.SetMetadata("TestType", "e2e")
	tree.SetMetadata("TestRunID", "e2e-2024-01-15-42")

	// In a dependent service, you can correlate coverage
	// by using the same TestRunID
}

func ExampleCoverageTree_LoadMetadataFromEnv() {
	// Set up environment for cross-module coverage tracking
	os.Setenv("COVUTIL_TEST_RUN_ID", "integration-2024-01-15-001")
	os.Setenv("COVUTIL_MODULE", "github.com/myorg/service-a")
	os.Setenv("COVUTIL_TEST_NAME", "TestServiceIntegration")

	// Create tree and automatically load metadata
	tree := covtree.NewCoverageTree()
	tree.LoadMetadataFromEnv()

	// The tree now has metadata from the environment
	// This is useful when running tests across multiple modules
}
