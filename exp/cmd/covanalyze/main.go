package covanalyze

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
)

func main() {
	os.Exit(Main())
}

func Main() int {
	var (
		compare   = flag.Bool("compare", false, "Compare coverage between legacy and scripttest")
		analyze   = flag.Bool("analyze", false, "Run comprehensive coverage analysis")
		pattern   = flag.String("pattern", "", "Search for functions matching a pattern")
		missing   = flag.Bool("missing", false, "Show functions missing from scripttest coverage")
		uncovered = flag.Bool("uncovered", false, "Show uncovered functions")
		topTests  = flag.Bool("top-tests", false, "Show top tests by coverage contribution")
		goImpl    = flag.Bool("go", false, "Use Go-based implementation")
		help      = flag.Bool("help", false, "Show help")
	)

	flag.Parse()

	if *help || (!*compare && !*analyze && *pattern == "" && !*missing && !*uncovered && !*topTests) {
		printUsage()
		return 0
	}

	if *goImpl {
		return runGoImplementation(*compare, *analyze, *pattern, *missing, *uncovered, *topTests)
	}

	return runShellImplementation(*compare, *analyze, *pattern, *missing, *uncovered, *topTests)
}

func printUsage() {
	fmt.Println("Sprig Coverage Analysis Tools")
	fmt.Println("")
	fmt.Println("Usage: covanalyze [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  -compare                     Compare coverage between legacy and scripttest")
	fmt.Println("  -analyze                     Run comprehensive coverage analysis")
	fmt.Println("  -pattern <pattern>           Search for functions matching a pattern")
	fmt.Println("  -missing                     Show functions missing from scripttest coverage")
	fmt.Println("  -uncovered                   Show uncovered functions")
	fmt.Println("  -top-tests                   Show top tests by coverage contribution")
	fmt.Println("  -go                          Use Go-based implementation")
	fmt.Println("  -help                        Show this help")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  covanalyze -compare          # Compare coverage between test types")
	fmt.Println("  covanalyze -analyze          # Run comprehensive analysis")
	fmt.Println("  covanalyze -pattern crypto   # Find crypto functions coverage")
}

func runGoImplementation(compare, analyze bool, pattern string, missing, uncovered, topTests bool) int {
	fmt.Printf("Error: Go implementation not yet available\n")
	return 1
}

func runShellImplementation(compare, analyze bool, pattern string, missing, uncovered, topTests bool) int {
	var cmd *exec.Cmd

	switch {
	case compare:
		cmd = exec.Command("make", "-f", "Makefile.coverage", "coverage-compare")
	case analyze:
		cmd = exec.Command("make", "-f", "Makefile.coverage", "coverage-uncover-delta")
	case pattern != "":
		cmd = exec.Command("make", "-f", "Makefile.coverage", "coverage-for", "PATTERN="+pattern)
	case missing:
		cmd = exec.Command("make", "-f", "Makefile.coverage", "coverage-generate-missing-coverage-list")
	case uncovered:
		cmd = exec.Command("make", "-f", "Makefile.coverage", "coverage-uncover", "coverage-top-uncovered")
	case topTests:
		cmd = exec.Command("make", "-f", "Makefile.coverage", "coverage-list-significant")
	default:
		fmt.Printf("Error: No valid command specified\n")
		return 1
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running command: %v\n", err)
		return 1
	}
	return 0
}
