// Package stringutils provides string manipulation utilities
package stringutils

import (
	"strings"
	"unicode"
)

// StringProcessor handles various string operations
type StringProcessor struct {
	caseSensitive bool
}

// NewProcessor creates a new string processor
func NewProcessor(caseSensitive bool) *StringProcessor {
	return &StringProcessor{
		caseSensitive: caseSensitive,
	}
}

// Reverse reverses a string
func (sp *StringProcessor) Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// IsPalindrome checks if a string is a palindrome
func (sp *StringProcessor) IsPalindrome(s string) bool {
	if !sp.caseSensitive {
		s = strings.ToLower(s)
	}
	
	// Remove non-alphanumeric characters
	cleaned := ""
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cleaned += string(r)
		}
	}
	
	return cleaned == sp.Reverse(cleaned)
}

// CountWords counts words in a string
func (sp *StringProcessor) CountWords(s string) int {
	if s == "" {
		return 0
	}
	
	words := strings.Fields(s)
	return len(words)
}

// Capitalize capitalizes the first letter of each word
func (sp *StringProcessor) Capitalize(s string) string {
	if s == "" {
		return s
	}
	
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	
	return strings.Join(words, " ")
}

// Contains checks if a string contains a substring
func (sp *StringProcessor) Contains(s, substr string) bool {
	if !sp.caseSensitive {
		s = strings.ToLower(s)
		substr = strings.ToLower(substr)
	}
	return strings.Contains(s, substr)
}

// RemoveDuplicateWords removes duplicate words from a string
func (sp *StringProcessor) RemoveDuplicateWords(s string) string {
	if s == "" {
		return s
	}
	
	words := strings.Fields(s)
	seen := make(map[string]bool)
	var result []string
	
	for _, word := range words {
		key := word
		if !sp.caseSensitive {
			key = strings.ToLower(word)
		}
		
		if !seen[key] {
			seen[key] = true
			result = append(result, word)
		}
	}
	
	return strings.Join(result, " ")
}

// Acronym creates an acronym from a string
func (sp *StringProcessor) Acronym(s string) string {
	words := strings.Fields(s)
	var acronym strings.Builder
	
	for _, word := range words {
		if len(word) > 0 {
			if sp.caseSensitive {
				acronym.WriteString(string(word[0]))
			} else {
				acronym.WriteString(strings.ToUpper(string(word[0])))
			}
		}
	}
	
	return acronym.String()
}

// CamelCase converts a string to camelCase
func (sp *StringProcessor) CamelCase(s string) string {
	if s == "" {
		return s
	}
	
	words := strings.Fields(s)
	if len(words) == 0 {
		return s
	}
	
	var result strings.Builder
	
	// First word in lowercase
	if len(words[0]) > 0 {
		result.WriteString(strings.ToLower(words[0]))
	}
	
	// Remaining words capitalized
	for i := 1; i < len(words); i++ {
		if len(words[i]) > 0 {
			result.WriteString(strings.ToUpper(string(words[i][0])) + strings.ToLower(words[i][1:]))
		}
	}
	
	return result.String()
}

// SnakeCase converts a string to snake_case
func (sp *StringProcessor) SnakeCase(s string) string {
	if s == "" {
		return s
	}
	
	words := strings.Fields(s)
	for i, word := range words {
		words[i] = strings.ToLower(word)
	}
	
	return strings.Join(words, "_")
}

// Rot13 applies ROT13 encoding (rarely tested)
func (sp *StringProcessor) Rot13(s string) string {
	var result strings.Builder
	
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			result.WriteRune('a' + (r-'a'+13)%26)
		} else if r >= 'A' && r <= 'Z' {
			result.WriteRune('A' + (r-'A'+13)%26)
		} else {
			result.WriteRune(r)
		}
	}
	
	return result.String()
}