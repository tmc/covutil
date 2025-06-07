// Command covutil provides utilities for working with Go coverage data.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s [command] [args...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  version    Print version information\n")
		os.Exit(1)
	}

	switch flag.Arg(0) {
	case "version":
		fmt.Printf("covutil version 0.1.0\n")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", flag.Arg(0))
		os.Exit(1)
	}
}
