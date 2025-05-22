package stringutils

import (
	"os"
	"testing"
)

func TestReverse(t *testing.T) {
	sp := NewProcessor(true)
	
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "olleh"},
		{"", ""},
		{"a", "a"},
		{"12345", "54321"},
	}
	
	for _, test := range tests {
		result := sp.Reverse(test.input)
		if result != test.expected {
			t.Errorf("Reverse(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}

func TestIsPalindrome(t *testing.T) {
	// Test case sensitive
	sp := NewProcessor(true)
	
	if !sp.IsPalindrome("racecar") {
		t.Error("racecar should be palindrome")
	}
	
	if sp.IsPalindrome("Racecar") {
		t.Error("Racecar should not be palindrome (case sensitive)")
	}
	
	// Test case insensitive
	sp = NewProcessor(false)
	
	if !sp.IsPalindrome("Racecar") {
		t.Error("Racecar should be palindrome (case insensitive)")
	}
	
	if !sp.IsPalindrome("A man, a plan, a canal: Panama") {
		t.Error("Should be palindrome ignoring punctuation")
	}
}

func TestCountWords(t *testing.T) {
	sp := NewProcessor(true)
	
	tests := []struct {
		input    string
		expected int
	}{
		{"hello world", 2},
		{"", 0},
		{"   spaces   everywhere   ", 2},
		{"one", 1},
	}
	
	for _, test := range tests {
		result := sp.CountWords(test.input)
		if result != test.expected {
			t.Errorf("CountWords(%q) = %d; want %d", test.input, result, test.expected)
		}
	}
}

func TestCapitalize(t *testing.T) {
	sp := NewProcessor(true)
	
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", "Hello World"},
		{"", ""},
		{"HELLO WORLD", "Hello World"},
		{"hELLo WoRLd", "Hello World"},
	}
	
	for _, test := range tests {
		result := sp.Capitalize(test.input)
		if result != test.expected {
			t.Errorf("Capitalize(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}

func TestContains(t *testing.T) {
	// Case sensitive
	sp := NewProcessor(true)
	
	if !sp.Contains("hello world", "world") {
		t.Error("Should contain 'world'")
	}
	
	if sp.Contains("hello world", "World") {
		t.Error("Should not contain 'World' (case sensitive)")
	}
	
	// Case insensitive (only tested sometimes)
	if os.Getenv("FULL_TEST") != "" {
		sp = NewProcessor(false)
		
		if !sp.Contains("hello world", "World") {
			t.Error("Should contain 'World' (case insensitive)")
		}
	}
}

func TestRemoveDuplicateWords(t *testing.T) {
	sp := NewProcessor(true)
	
	result := sp.RemoveDuplicateWords("hello world hello")
	expected := "hello world"
	if result != expected {
		t.Errorf("RemoveDuplicateWords() = %q; want %q", result, expected)
	}
	
	// Empty string test
	result = sp.RemoveDuplicateWords("")
	if result != "" {
		t.Errorf("RemoveDuplicateWords('') should return empty string")
	}
}

func TestAcronym(t *testing.T) {
	sp := NewProcessor(true)
	
	result := sp.Acronym("hello world test")
	if result != "hwt" {
		t.Errorf("Acronym() = %q; want 'hwt'", result)
	}
	
	// Case insensitive version (not always tested)
	if testing.Short() {
		return
	}
	
	sp = NewProcessor(false)
	result = sp.Acronym("hello world test")
	if result != "HWT" {
		t.Errorf("Acronym() case insensitive = %q; want 'HWT'", result)
	}
}

func TestCamelCase(t *testing.T) {
	sp := NewProcessor(true)
	
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", "helloWorld"},
		{"", ""},
		{"one", "one"},
		{"multiple word string here", "multipleWordStringHere"},
	}
	
	for _, test := range tests {
		result := sp.CamelCase(test.input)
		if result != test.expected {
			t.Errorf("CamelCase(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}

// SnakeCase and Rot13 are not tested (0% coverage)