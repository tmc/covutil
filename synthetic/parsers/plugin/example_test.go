package plugin_test

import (
	"fmt"
	"log"

	"github.com/tmc/covutil/synthetic/parsers"
	"github.com/tmc/covutil/synthetic/parsers/plugin"
)

func Example_loadPlugin() {
	// Load a parser plugin
	err := plugin.Load("/path/to/chromedp-parser.so")
	if err != nil {
		log.Printf("Failed to load plugin: %v", err)
		return
	}

	// The parser is now registered and can be used
	parser, ok := parsers.Get("chromedp")
	if !ok {
		log.Fatal("Parser not found after loading")
	}

	fmt.Printf("Loaded parser: %s\n", parser.Name())
	fmt.Printf("Description: %s\n", parser.Description())
}

func Example_conditionalLoading() {
	// Try to load optional parsers
	optionalParsers := []string{
		"/usr/local/lib/covutil/chromedp-parser.so",
		"/usr/local/lib/covutil/ruby-parser.so",
		"/usr/local/lib/covutil/rust-parser.so",
	}

	for _, pluginPath := range optionalParsers {
		if err := plugin.Load(pluginPath); err != nil {
			// Log but don't fail - these are optional
			log.Printf("Optional parser not available: %s", pluginPath)
		} else {
			log.Printf("Successfully loaded: %s", pluginPath)
		}
	}

	// List all available parsers
	available := parsers.List()
	fmt.Printf("Available parsers: %d\n", len(available))
	for name, parser := range available {
		fmt.Printf("- %s: %s\n", name, parser.Description())
	}
}
