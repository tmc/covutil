package main

import (
	"fmt"
	"testing"
)

func TestCombineCoverage(t *testing.T) {
	// Test combining real coverage with synthetic coverage
	profiles := []string{
		"cov_both.out",           // Real Go coverage
		"demo_synthetic_combined.cov", // Has both real and synthetic (with old paths)
	}
	
	err := CombineCoverageProfiles(profiles, "test_real_combined.cov")
	if err != nil {
		t.Fatal(err)
	}
	
	fmt.Println("Combined coverage profile created: test_real_combined.cov")
}