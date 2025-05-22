// Package synthetic provides functionality for generating synthetic coverage data
// for non-Go artifacts such as scripts, configuration files, and other resources.
//
// The package offers different types of trackers for various artifact types:
//
//   - BasicTracker: Generic tracker for any type of artifact
//   - ScriptTracker: Specialized tracker for shell scripts and scripttest files
//
// # Basic Usage
//
//	// Create a basic tracker
//	tracker := synthetic.NewBasicTracker(
//		synthetic.WithLabels(map[string]string{"test": "my-test"}),
//	)
//
//	// Track execution of artifact locations
//	tracker.Track("my-script.sh", "5", true)  // Line 5 executed
//	tracker.Track("my-script.sh", "10", false) // Line 10 not executed
//
//	// Generate coverage data
//	pod, err := tracker.GeneratePod()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Write coverage data
//	err = covutil.WritePodToDirectory("coverage", pod)
//
// # Script Tracking
//
//	// Create a script tracker
//	tracker := synthetic.NewScriptTracker(
//		synthetic.WithTestName("integration-test"),
//	)
//
//	// Parse and set up tracking for a script
//	err := tracker.ParseAndTrack(scriptContent, "test.sh", "shell", "my-test")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Track execution of specific lines
//	tracker.TrackExecution("test.sh", "my-test", 5)
//	tracker.TrackExecution("test.sh", "my-test", 10)
//
//	// Generate report
//	report := tracker.GetReport()
//	fmt.Println(report)
//
// # Integration with covutil
//
// The synthetic package is designed to work seamlessly with the main covutil
// package. Generated coverage data can be merged with real Go coverage data:
//
//	// Load real coverage data
//	realSet, err := covutil.LoadCoverageSetFromDirectory("real-coverage")
//
//	// Generate synthetic coverage
//	syntheticPod, err := tracker.GeneratePod()
//
//	// Combine them
//	combinedSet := &covutil.CoverageSet{
//		Pods: []*covutil.Pod{realSet.Pods[0], syntheticPod},
//	}
//
//	// Generate combined report
//	formatter := covutil.NewFormatter(covutil.ModeSet)
//	formatter.AddPodProfile(realSet.Pods[0])
//	formatter.AddPodProfile(syntheticPod)
//
// # Custom Parsers
//
// You can implement custom parsers for different artifact types:
//
//	type MyParser struct{}
//
//	func (p *MyParser) ParseScript(content string) map[int]string {
//		// Parse content and return map of line numbers to commands
//		return commands
//	}
//
//	func (p *MyParser) IsExecutable(line string) bool {
//		// Return true if line is executable
//		return isExecutable
//	}
//
//	// Register the parser
//	tracker.RegisterParser("myformat", &MyParser{})
//
// # Output Formats
//
// The package supports multiple output formats:
//
//   - Binary format: Compatible with Go's coverage tools via covutil.Pod
//   - Text format: Traditional Go coverage profile format
//   - JSON format: Machine-readable format for integration
//
// The synthetic coverage data can be consumed by any tool that works with
// Go coverage data, including the go tool cover command and various coverage
// visualization tools.
package synthetic
