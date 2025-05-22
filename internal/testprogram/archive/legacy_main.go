// Test program demonstrating the usage of the coverage package
package main

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tmc/covutil/coverage"
	"github.com/tmc/covutil/internal/testprogram/pkga"
)

var (
	outputDir        = flag.String("output", "covdata", "Directory to write coverage data to")
	skipCovdata      = flag.Bool("skip-covdata", false, "Skip running the go tool covdata command")
	analyzeReal      = flag.Bool("analyze-real", false, "Analyze real coverage data in covdata2 directory")
	analyzeDetailed  = flag.Bool("analyze-detailed", false, "Detailed analysis of real coverage file format")
	analyzeSynthetic = flag.Bool("analyze-synthetic", false, "Detailed analysis of synthetic coverage file format")
	toJson           = flag.Bool("to-json", false, "Convert coverage data to JSON format")
	fromJson         = flag.String("from-json", "", "Generate synthetic coverage data from JSON file")
	compareDir       = flag.String("compare", "", "Compare coverage data with another directory")
	demoJson         = flag.Bool("demo-json", false, "Demonstrate JSON functionality with sample data")
	demoJsonSimple   = flag.Bool("demo-json-simple", false, "Demonstrate JSON functionality without covdata calls")
	enableDebugEnv   = flag.Bool("debug-env", false, "Enable GOCOVERDEBUG=1 for go tool covdata commands")
)

// writeULEB128 writes a uint64 value using ULEB128 encoding
func writeULEB128(buf *bytes.Buffer, value uint64) {
	for {
		b := byte(value & 0x7F)
		value >>= 7
		if value != 0 {
			b |= 0x80
		}
		buf.WriteByte(b)
		if value == 0 {
			break
		}
	}
}

func displayBasicInfo() {
	fmt.Println("Coverage Package Test Program")
	fmt.Println("=============================")
	fmt.Println("Command-line flags:")
	fmt.Printf("  -output=%s      Directory to write coverage data to\n", *outputDir)
	fmt.Printf("  -skip-covdata=%t   Skip running the go tool covdata command\n", *skipCovdata)
	fmt.Printf("  -analyze-real=%t    Analyze real coverage data in covdata2 directory\n", *analyzeReal)
	fmt.Println()
	pkga.Yo()

	fmt.Printf("Package Path: %s\n", coverage.PkgPath)
	fmt.Println()

	fmt.Println("Coverage File Constants:")
	fmt.Printf("Meta File Prefix: %s\n", coverage.MetaFilePref)
	fmt.Printf("Counter File Prefix: %s\n", coverage.CounterFilePref)
	fmt.Printf("Meta File Version: %d\n", coverage.MetaFileVersion)
	fmt.Printf("Counter File Version: %d\n", coverage.CounterFileVersion)
	fmt.Println()
	fmt.Println()

	fmt.Println("Counter Modes:")
	modes := []string{"set", "count", "atomic", "regonly", "testmain", "invalid"}
	for _, mode := range modes {
		counterMode := coverage.ParseCounterMode(mode)
		fmt.Printf("Mode '%s': %v\n", mode, counterMode)
	}
	fmt.Println()

	fmt.Println("Magic Constants:")
	fmt.Printf("Meta Magic: %#v\n", coverage.CovMetaMagic)
	fmt.Printf("Counter Magic: %#v\n", coverage.CovCounterMagic)
	fmt.Println()
}

// createMetaDataFile creates a simple coverage metadata file with the proper header
func createMetaDataFile(dir string) (string, error) {
	// Create a simple meta-data hash (would normally be generated from actual code coverage data)
	metaHash := md5.Sum([]byte("sample-coverage-metadata"))

	// Create the metadata file name
	metaFileName := fmt.Sprintf("%s.%x", coverage.MetaFilePref, metaHash[:])
	metaFilePath := filepath.Join(dir, metaFileName)

	// Create metadata file
	fmt.Printf("Creating metadata file: %s\n", metaFilePath)
	f, err := os.Create(metaFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create metadata file: %w", err)
	}
	defer f.Close()

	// Create a simple metafile that matches the exact Go runtime format
	// We'll create a file with 0 packages to prevent hanging due to metadata/counter mismatch
	numPackages := uint64(0)

	// Calculate total file size based on actual content (just header and string table)
	totalFileSize := uint64(128 + 8*int(numPackages) + 8*int(numPackages) + 8) // header + offset table + length table + string table

	// Define string table - it will have one entry (an empty string)
	strTabOffset := uint32(128 + 8*int(numPackages) + 8*int(numPackages)) // After header and package tables
	strTabLength := uint32(8)                                             // 1 byte for ULEB length + 7 bytes padding
	strTabEntries := uint32(1)

	// Set up header
	headerBuf := make([]byte, 128) // First 128 bytes for header

	// Magic bytes
	copy(headerBuf[0:4], coverage.CovMetaMagic[:])

	// Version (4 bytes)
	binary.LittleEndian.PutUint32(headerBuf[4:8], coverage.MetaFileVersion)

	// Total length (8 bytes)
	binary.LittleEndian.PutUint64(headerBuf[8:16], totalFileSize)

	// Number of packages (8 bytes)
	binary.LittleEndian.PutUint64(headerBuf[16:24], numPackages)

	// Meta file hash (16 bytes) - use our hash
	copy(headerBuf[24:40], metaHash[:])

	// String table offset (4 bytes)
	binary.LittleEndian.PutUint32(headerBuf[40:44], strTabOffset)

	// String table length (4 bytes)
	binary.LittleEndian.PutUint32(headerBuf[44:48], strTabLength)

	// Number of entries in string table (4 bytes)
	binary.LittleEndian.PutUint32(headerBuf[48:52], strTabEntries)

	// Counter mode (1 byte)
	headerBuf[52] = byte(coverage.CtrModeCount)

	// Counter granularity (1 byte)
	headerBuf[53] = byte(coverage.CtrGranularityPerBlock)

	// Padding (6 bytes) - already zeroed in the headerBuf

	// Write the header
	if _, err := f.Write(headerBuf); err != nil {
		return "", fmt.Errorf("failed to write header: %w", err)
	}

	// With 0 packages, we don't need offset or length tables

	// Write string table - very simple with one entry (empty string)
	strTabData := []byte{
		0x00,                                     // ULEB128 length of 0 (empty string)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // padding
	}
	if _, err := f.Write(strTabData); err != nil {
		return "", fmt.Errorf("failed to write string table: %w", err)
	}

	// With 0 packages, no package metadata needs to be written

	return metaFilePath, nil
}

// createCounterDataFile creates a proper coverage counter data file with synthetic data
func createCounterDataFile(dir string, metaHash [16]byte) (string, error) {
	// Create the counter data file name (using meta hash, pid, and timestamp)
	pid := os.Getpid()
	now := time.Now().UnixNano()
	counterFileName := fmt.Sprintf(coverage.CounterFileTempl, coverage.CounterFilePref, metaHash[:], pid, now)
	counterFilePath := filepath.Join(dir, counterFileName)

	// Create counter data file
	fmt.Printf("Creating counter data file: %s\n", counterFilePath)
	f, err := os.Create(counterFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create counter data file: %w", err)
	}
	defer f.Close()

	// Write a counter file with synthetic coverage data
	// This format should match what the Go runtime produces

	// -- HEADER (32 bytes) --
	header := make([]byte, 32)

	// Magic bytes (4 bytes)
	copy(header[0:4], coverage.CovCounterMagic[:])

	// Version (4 bytes)
	binary.LittleEndian.PutUint32(header[4:8], coverage.CounterFileVersion)

	// Meta hash (16 bytes)
	copy(header[8:24], metaHash[:])

	// Counter flavor (1 byte) - Use CtrULeb128 like real files
	header[24] = byte(coverage.CtrULeb128)

	// Big endian flag (1 byte)
	header[25] = 0 // Little endian

	// Padding (6 bytes) already zeroed

	if _, err := f.Write(header); err != nil {
		return "", fmt.Errorf("failed to write file header: %w", err)
	}

	// -- SEGMENT (based on real coverage data analysis) --
	// Create a segment that mimics the structure of real coverage files
	numFunctions := uint64(2) // We'll have 2 synthetic functions

	// Build string table similar to real files
	strTableBuf := new(bytes.Buffer)

	// Add strings that would be typical in a counter file
	strings := []string{
		"GOARCH", "arm64",
		"argc", "1",
		"argv0", "/path/to/testprogram",
		"GOOS", "darwin",
	}

	// First write the number of strings (ULEB128)
	writeULEB128(strTableBuf, uint64(len(strings)))

	// Then write each string with its ULEB128 length
	for _, s := range strings {
		writeULEB128(strTableBuf, uint64(len(s)))
		strTableBuf.WriteString(s)
	}

	// Pad to 4-byte boundary
	for strTableBuf.Len()%4 != 0 {
		strTableBuf.WriteByte(0)
	}

	strTableBytes := strTableBuf.Bytes()
	strTableSize := len(strTableBytes)

	// Args table size - we'll reference some of the strings above
	argsTableSize := len(strings) // One entry per string

	// -- SEGMENT HEADER --
	segmentHeader := make([]byte, 16)

	// Function entries (8 bytes)
	binary.LittleEndian.PutUint64(segmentHeader[0:8], numFunctions)

	// String table length (4 bytes)
	binary.LittleEndian.PutUint32(segmentHeader[8:12], uint32(strTableSize))

	// Args table length (4 bytes)
	binary.LittleEndian.PutUint32(segmentHeader[12:16], uint32(argsTableSize))

	if _, err := f.Write(segmentHeader); err != nil {
		return "", fmt.Errorf("failed to write segment header: %w", err)
	}

	// -- STRING TABLE --
	if _, err := f.Write(strTableBytes); err != nil {
		return "", fmt.Errorf("failed to write string table: %w", err)
	}

	// -- ARGS TABLE --
	// Write indices that reference our strings (simple sequential indices)
	argsTable := make([]byte, argsTableSize)
	for i := 0; i < argsTableSize; i++ {
		argsTable[i] = byte(i)
	}
	if _, err := f.Write(argsTable); err != nil {
		return "", fmt.Errorf("failed to write args table: %w", err)
	}

	// -- COUNTER DATA (for our 2 functions using ULEB128 format) --
	// Function 1: Write counter data using ULEB128 encoding
	counterDataBuf := new(bytes.Buffer)

	// Function 1
	writeULEB128(counterDataBuf, 3) // numCounters = 3
	writeULEB128(counterDataBuf, 0) // pkgID = 0
	writeULEB128(counterDataBuf, 0) // funcID = 0
	// Counter values
	writeULEB128(counterDataBuf, 10) // counter 1
	writeULEB128(counterDataBuf, 5)  // counter 2
	writeULEB128(counterDataBuf, 20) // counter 3

	// Function 2
	writeULEB128(counterDataBuf, 2) // numCounters = 2
	writeULEB128(counterDataBuf, 0) // pkgID = 0
	writeULEB128(counterDataBuf, 1) // funcID = 1
	// Counter values
	writeULEB128(counterDataBuf, 15) // counter 1
	writeULEB128(counterDataBuf, 8)  // counter 2

	if _, err := f.Write(counterDataBuf.Bytes()); err != nil {
		return "", fmt.Errorf("failed to write counter data: %w", err)
	}

	// -- FOOTER --
	// Write footer (16 bytes)
	footer := make([]byte, 16)
	// Magic bytes (4 bytes)
	copy(footer[0:4], coverage.CovCounterMagic[:])
	// 4 bytes padding
	binary.LittleEndian.PutUint32(footer[8:12], 1) // 1 segment
	// 4 more bytes padding

	if _, err := f.Write(footer); err != nil {
		return "", fmt.Errorf("failed to write footer: %w", err)
	}

	return counterFilePath, nil
}

// analyzeRealCoverageData examines the structure of real coverage files
func analyzeRealCoverageData() {
	fmt.Println("\n=== ANALYZING REAL COVERAGE DATA ===")

	realDir := "covdata2"
	entries, err := os.ReadDir(realDir)
	if err != nil {
		fmt.Printf("Error reading %s directory: %v\n", realDir, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(realDir, entry.Name())
		fmt.Printf("\nAnalyzing file: %s\n", entry.Name())

		// Read first 64 bytes to examine the header
		f, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("  Error opening file: %v\n", err)
			continue
		}

		header := make([]byte, 64)
		n, err := f.Read(header)
		f.Close()

		if err != nil && n == 0 {
			fmt.Printf("  Error reading file: %v\n", err)
			continue
		}

		fmt.Printf("  File size: %d bytes\n", getFileSize(filePath))
		fmt.Printf("  Header (first %d bytes):\n", n)

		// Check magic bytes
		if n >= 4 {
			magic := header[0:4]
			if bytes.Equal(magic, coverage.CovMetaMagic[:]) {
				fmt.Printf("    Magic: Meta data file (CovMetaMagic)\n")
				analyzeMetaHeader(header, n)
			} else if bytes.Equal(magic, coverage.CovCounterMagic[:]) {
				fmt.Printf("    Magic: Counter data file (CovCounterMagic)\n")
				analyzeCounterHeader(header, n)
			} else {
				fmt.Printf("    Magic: Unknown (%#v)\n", magic)
			}
		}

		// Show hex dump of first 32 bytes
		fmt.Printf("    Hex dump (first 32 bytes):\n")
		for i := 0; i < 32 && i < n; i += 16 {
			end := i + 16
			if end > n {
				end = n
			}
			fmt.Printf("      %04x: ", i)
			for j := i; j < end; j++ {
				fmt.Printf("%02x ", header[j])
			}
			fmt.Printf("\n")
		}
	}
}

func getFileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

func analyzeMetaHeader(header []byte, n int) {
	if n >= 8 {
		version := binary.LittleEndian.Uint32(header[4:8])
		fmt.Printf("    Version: %d\n", version)
	}
	if n >= 16 {
		totalLength := binary.LittleEndian.Uint64(header[8:16])
		fmt.Printf("    Total Length: %d bytes\n", totalLength)
	}
	if n >= 24 {
		numPackages := binary.LittleEndian.Uint64(header[16:24])
		fmt.Printf("    Number of Packages: %d\n", numPackages)
	}
	if n >= 44 {
		strTabOffset := binary.LittleEndian.Uint32(header[40:44])
		fmt.Printf("    String Table Offset: %d\n", strTabOffset)
	}
	if n >= 48 {
		strTabLength := binary.LittleEndian.Uint32(header[44:48])
		fmt.Printf("    String Table Length: %d\n", strTabLength)
	}
	if n >= 52 {
		strTabEntries := binary.LittleEndian.Uint32(header[48:52])
		fmt.Printf("    String Table Entries: %d\n", strTabEntries)
	}
	if n >= 53 {
		counterMode := coverage.CounterMode(header[52])
		fmt.Printf("    Counter Mode: %v\n", counterMode)
	}
	if n >= 54 {
		counterGranularity := coverage.CounterGranularity(header[53])
		fmt.Printf("    Counter Granularity: %v\n", counterGranularity)
	}
}

func analyzeCounterHeader(header []byte, n int) {
	if n >= 8 {
		version := binary.LittleEndian.Uint32(header[4:8])
		fmt.Printf("    Version: %d\n", version)
	}
	if n >= 25 {
		flavor := coverage.CounterFlavor(header[24])
		fmt.Printf("    Counter Flavor: %d\n", flavor)
	}
	if n >= 26 {
		bigEndian := header[25] != 0
		fmt.Printf("    Big Endian: %t\n", bigEndian)
	}

	// Look for segment header starting at offset 32
	if n >= 32+16 {
		funcEntries := binary.LittleEndian.Uint64(header[32:40])
		strTabLen := binary.LittleEndian.Uint32(header[40:44])
		argsLen := binary.LittleEndian.Uint32(header[44:48])
		fmt.Printf("    Segment - Function Entries: %d\n", funcEntries)
		fmt.Printf("    Segment - String Table Length: %d\n", strTabLen)
		fmt.Printf("    Segment - Args Length: %d\n", argsLen)
	}
}

// runCoverageCommand executes the go tool covdata command
func runCoverageCommand(dir string) {
	cmdArgs := []string{"tool", "covdata", "percent", "-i=" + dir}
	envInfo := ""
	if *enableDebugEnv {
		envInfo = " (with GOCOVERDEBUG=1)"
	}
	fmt.Printf("\nRunning command: go %s%s\n", strings.Join(cmdArgs, " "), envInfo)

	cmd := exec.Command("go", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment
	if *enableDebugEnv {
		cmd.Env = append(os.Environ(), "GOCOVERDEBUG=1")
	}

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running go tool covdata: %v\n", err)
		fmt.Println("\nNOTE: This is expected because we're generating minimal sample files for demonstration purposes.")
		fmt.Println("The actual Go coverage tool expects more complete data.")
		fmt.Println("Use the -skip-covdata flag to skip running this command.")
	}
}

func main() {
	flag.Parse()

	// Display basic package information
	displayBasicInfo()

	// If analyze-real flag is set, analyze the real coverage data and exit
	if *analyzeReal {
		analyzeRealCoverageData()
		return
	}

	// If analyze-detailed flag is set, perform detailed format analysis and exit
	if *analyzeDetailed {
		analyzeRealCoverageFormat()
		return
	}

	// If analyze-synthetic flag is set, analyze synthetic coverage files and exit
	if *analyzeSynthetic {
		analyzeSyntheticCoverageFormat()
		return
	}

	// If to-json flag is set, convert coverage data to JSON and exit
	if *toJson {
		convertToJson()
		return
	}

	// If from-json flag is set, generate synthetic data from JSON and exit
	if *fromJson != "" {
		generateFromJson(*fromJson)
		return
	}

	// If compare flag is set, compare two coverage directories and exit
	if *compareDir != "" {
		compareCoverageData(*outputDir, *compareDir)
		return
	}

	// If demo-json flag is set, demonstrate JSON functionality and exit
	if *demoJson {
		demonstrateJsonFunctionality()
		return
	}

	// If demo-json-simple flag is set, demonstrate JSON functionality without covdata calls and exit
	if *demoJsonSimple {
		demonstrateJsonFunctionalitySimple()
		return
	}

	// Clear existing covdata directory if it exists
	if _, err := os.Stat(*outputDir); !os.IsNotExist(err) {
		fmt.Printf("Removing existing directory: %s\n", *outputDir)
		if err := os.RemoveAll(*outputDir); err != nil {
			fmt.Printf("Error removing directory: %v\n", err)
			os.Exit(1)
		}
	}

	// Create coverage data directory
	fmt.Printf("Creating directory: %s\n", *outputDir)
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Create a meta-data hash (would normally be generated from actual code coverage data)
	metaHash := md5.Sum([]byte("sample-coverage-metadata"))

	// Create metadata file
	metaFilePath, err := createMetaDataFile(*outputDir)
	if err != nil {
		fmt.Printf("Error creating metadata file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Successfully created metadata file: %s\n", metaFilePath)

	// Create counter data file
	counterFilePath, err := createCounterDataFile(*outputDir, metaHash)
	if err != nil {
		fmt.Printf("Error creating counter data file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Successfully created counter data file: %s\n", counterFilePath)

	fmt.Println("\nCoverage data files have been successfully created in:", *outputDir)

	// Run the go tool covdata command if not skipped
	if !*skipCovdata {
		runCoverageCommand(*outputDir)
	} else {
		fmt.Println("\nSkipped running go tool covdata (use -skip-covdata=false to run it)")
	}
}
