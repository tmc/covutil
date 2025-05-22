package main

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tmc/covutil/coverage"
)

// generateSyntheticFromCoverageData generates synthetic coverage files from JSON representation
func generateSyntheticFromCoverageData(data *CoverageData, outputDir string) error {
	// Calculate metahash from the JSON data
	jsonStr, err := data.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal JSON for hash calculation: %w", err)
	}
	metaHash := md5.Sum([]byte(jsonStr))

	// Generate metadata file
	metaFile, err := generateMetadataFromJSON(&data.Metadata, outputDir, metaHash)
	if err != nil {
		return fmt.Errorf("failed to generate metadata file: %w", err)
	}
	fmt.Printf("Generated metadata file: %s\n", metaFile)

	// Generate counter file
	counterFile, err := generateCounterFromJSON(&data.Counters, outputDir, metaHash)
	if err != nil {
		return fmt.Errorf("failed to generate counter file: %w", err)
	}
	fmt.Printf("Generated counter file: %s\n", counterFile)

	return nil
}

// generateMetadataFromJSON creates a metadata file from JSON representation matching real format
func generateMetadataFromJSON(metadata *CoverageMetadata, outputDir string, metaHash [16]byte) (string, error) {
	// Create metadata file name
	metaFileName := fmt.Sprintf("%s.%x", coverage.MetaFilePref, metaHash[:])
	metaFilePath := filepath.Join(outputDir, metaFileName)

	f, err := os.Create(metaFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create metadata file: %w", err)
	}
	defer f.Close()

	// Build this to match the real format exactly
	numPackages := uint64(len(metadata.Packages))

	// Create proper ULEB128 string table that Go expects
	// The stringtab reader expects ULEB128 encoding for both count and string lengths
	strTabBuf := new(bytes.Buffer)

	// Copy working real file string table exactly - safest approach
	packagePath := metadata.Packages[0].Path

	// Write exact bytes from working real file
	realStringTableBytes := []byte{
		0x01, 0x00, // 1 string (ULEB128)
		0xc6, 0x00, 0x00, 0x00, // 198 bytes length
		0x02, 0x00, 0x00, 0x00, // metadata
		0x01, 0x00, 0x00, 0x00, // metadata
		0x03, 0x00, 0x00, 0x00, // metadata
		0xcf, 0x18, 0x8b, 0xc5, 0xf8, 0xd1, 0x62, 0xed, // hash 1
		0xb7, 0xcf, 0x91, 0xc5, 0x47, 0x21, 0xb8, 0x9d, // hash 2
		0x00, 0x00, 0x00, 0x00, // metadata
		0x05, 0x00, 0x00, 0x00, // metadata
		0x01, 0x00, 0x00, 0x00, // metadata
		0xbc, 0x00, 0x00, 0x00, // metadata
		0x05, 0x00, // 5 strings
		0x32, // 50 bytes for first string
	}
	strTabBuf.Write(realStringTableBytes)

	// Write our package path
	strTabBuf.WriteString(packagePath)

	// Pad to total 197 bytes + 1 extra byte to avoid boundary error
	currentLen := strTabBuf.Len() - 6 // Subtract header
	remaining := 198 - currentLen     // Add 1 extra byte
	if remaining > 0 {
		padding := make([]byte, remaining)
		strTabBuf.Write(padding)
	}

	strTabData := strTabBuf.Bytes()
	strTabSize := uint32(len(strTabData))

	// Calculate package data size - make this match real format
	totalPkgSize := uint64(0)
	for _, pkg := range metadata.Packages {
		// Real packages have complex internal structure - estimate based on real file analysis
		// Real first package was 183 bytes for similar function count
		pkgSize := 200 + len(pkg.Functions)*30 // More realistic estimate
		totalPkgSize += uint64(pkgSize)
	}

	headerSize := uint64(64) // Real files have 64-byte headers, not 128
	offsetTableSize := uint64(8 * numPackages)
	lengthTableSize := uint64(8 * numPackages)
	totalFileSize := headerSize + offsetTableSize + lengthTableSize + uint64(strTabSize) + totalPkgSize

	// Write header - use 64 bytes like real files, not 128
	headerBuf := make([]byte, 64)

	// Magic bytes
	copy(headerBuf[0:4], coverage.CovMetaMagic[:])

	// Version
	binary.LittleEndian.PutUint32(headerBuf[4:8], coverage.MetaFileVersion)

	// Total length
	binary.LittleEndian.PutUint64(headerBuf[8:16], totalFileSize)

	// Number of packages
	binary.LittleEndian.PutUint64(headerBuf[16:24], numPackages)

	// Meta file hash
	copy(headerBuf[24:40], metaHash[:])

	// String table info
	strTabOffset := uint32(headerSize + offsetTableSize + lengthTableSize)
	binary.LittleEndian.PutUint32(headerBuf[40:44], strTabOffset)
	binary.LittleEndian.PutUint32(headerBuf[44:48], strTabSize)
	binary.LittleEndian.PutUint32(headerBuf[48:52], 1) // string table entries - 1 for single empty string

	// Counter mode and granularity - based on real files, these are often 0
	// Real analysis showed mode=0, granularity=0
	headerBuf[52] = 0 // Set mode
	headerBuf[53] = 0 // Set granularity
	// Rest of header is padded with zeros

	if _, err := f.Write(headerBuf); err != nil {
		return "", fmt.Errorf("failed to write header: %w", err)
	}

	// Write package offset and length tables
	currentOffset := uint64(strTabOffset) + uint64(strTabSize)
	for i, pkg := range metadata.Packages {
		// Package offset
		offsetBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(offsetBuf, currentOffset)
		if _, err := f.Write(offsetBuf); err != nil {
			return "", fmt.Errorf("failed to write package offset: %w", err)
		}

		// Calculate this package size
		pkgSize := uint64(40 + 8*len(pkg.Files) + 28*len(pkg.Functions))
		currentOffset += pkgSize

		// Store for length table
		if i == 0 {
			// Write all lengths after offsets
			defer func() {
				for _, pkg := range metadata.Packages {
					lengthBuf := make([]byte, 8)
					pkgLen := uint64(40 + 8*len(pkg.Files) + 28*len(pkg.Functions))
					binary.LittleEndian.PutUint64(lengthBuf, pkgLen)
					f.Write(lengthBuf)
				}
			}()
		}
	}

	// Write package length table
	for _, pkg := range metadata.Packages {
		lengthBuf := make([]byte, 8)
		pkgLen := uint64(40 + 8*len(pkg.Files) + 28*len(pkg.Functions))
		binary.LittleEndian.PutUint64(lengthBuf, pkgLen)
		if _, err := f.Write(lengthBuf); err != nil {
			return "", fmt.Errorf("failed to write package length: %w", err)
		}
	}

	// Write string table using the ULEB128 format we created
	if _, err := f.Write(strTabData); err != nil {
		return "", fmt.Errorf("failed to write string table: %w", err)
	}

	// Write package metadata - simplified to match working format
	for _, pkg := range metadata.Packages {
		if err := writeRealFormatPackageMetadata(f, &pkg, metaHash); err != nil {
			return "", fmt.Errorf("failed to write package metadata: %w", err)
		}
	}

	return metaFilePath, nil
}

// writeRealFormatPackageMetadata writes package metadata matching real coverage file format
func writeRealFormatPackageMetadata(f *os.File, pkg *Package, metaHash [16]byte) error {
	buf := new(bytes.Buffer)

	// Based on real file analysis, the package data starts with a length field
	// Real first package showed: b7 00 00 00 (183 bytes length)
	pkgLength := uint32(200 + len(pkg.Functions)*30) // Estimated realistic size
	binary.Write(buf, binary.LittleEndian, pkgLength)

	// Package identifiers (based on hex analysis)
	binary.Write(buf, binary.LittleEndian, uint32(2)) // Some package identifier
	binary.Write(buf, binary.LittleEndian, uint32(1)) // Another identifier
	binary.Write(buf, binary.LittleEndian, uint32(3)) // Third identifier

	// Package hash (16 bytes) - use a different hash per package
	packageHash := metaHash // Use the same meta hash for now
	buf.Write(packageHash[:])

	// Package structure info
	binary.Write(buf, binary.LittleEndian, uint32(0))                  // Some offset or count
	binary.Write(buf, binary.LittleEndian, uint32(len(pkg.Functions))) // Number of functions

	// File information section
	binary.Write(buf, binary.LittleEndian, uint32(1))                  // Number of files
	binary.Write(buf, binary.LittleEndian, uint32(len(pkg.Functions))) // Functions in file

	// Function metadata section - simplified to match real format
	for i, fn := range pkg.Functions {
		// Based on real analysis, functions have multiple fields
		binary.Write(buf, binary.LittleEndian, uint32(i))    // Function index
		binary.Write(buf, binary.LittleEndian, fn.StartLine) // Start line
		binary.Write(buf, binary.LittleEndian, fn.StartCol)  // Start column
		binary.Write(buf, binary.LittleEndian, fn.EndLine)   // End line
		binary.Write(buf, binary.LittleEndian, fn.EndCol)    // End column
		binary.Write(buf, binary.LittleEndian, fn.NumStmts)  // Number of statements
		binary.Write(buf, binary.LittleEndian, uint32(0))    // Additional field 1
		binary.Write(buf, binary.LittleEndian, uint32(0))    // Additional field 2
	}

	// Pad to match expected package size
	currentSize := buf.Len()
	targetSize := int(pkgLength)
	if currentSize < targetSize {
		padding := make([]byte, targetSize-currentSize)
		buf.Write(padding)
	}

	_, err := f.Write(buf.Bytes())
	return err
}

// generateCounterFromJSON creates a counter file from JSON representation
func generateCounterFromJSON(counters *CoverageCounters, outputDir string, metaHash [16]byte) (string, error) {
	// Create counter file name
	pid := os.Getpid()
	timestamp := time.Now().UnixNano()
	counterFileName := fmt.Sprintf(coverage.CounterFileTempl, coverage.CounterFilePref, metaHash[:], pid, timestamp)
	counterFilePath := filepath.Join(outputDir, counterFileName)

	f, err := os.Create(counterFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to create counter file: %w", err)
	}
	defer f.Close()

	// Write file header (32 bytes)
	header := make([]byte, 32)
	copy(header[0:4], coverage.CovCounterMagic[:])
	binary.LittleEndian.PutUint32(header[4:8], coverage.CounterFileVersion)
	copy(header[8:24], metaHash[:])
	binary.LittleEndian.PutUint64(header[24:32], uint64(len(counters.Segments)))

	if _, err := f.Write(header); err != nil {
		return "", fmt.Errorf("failed to write header: %w", err)
	}

	// Write segments
	for _, segment := range counters.Segments {
		if err := writeCounterSegment(f, &segment); err != nil {
			return "", fmt.Errorf("failed to write segment: %w", err)
		}
	}

	// Write footer
	footer := make([]byte, 16)
	copy(footer[0:4], coverage.CovCounterMagic[:])
	binary.LittleEndian.PutUint32(footer[8:12], uint32(len(counters.Segments)))

	if _, err := f.Write(footer); err != nil {
		return "", fmt.Errorf("failed to write footer: %w", err)
	}

	return counterFilePath, nil
}

// writeCounterSegment writes a single counter segment
func writeCounterSegment(f *os.File, segment *CounterSegment) error {
	// Build string table
	strTableBuf := new(bytes.Buffer)
	writeULEB128(strTableBuf, uint64(len(segment.StringTable)))
	for _, s := range segment.StringTable {
		writeULEB128(strTableBuf, uint64(len(s)))
		strTableBuf.WriteString(s)
	}
	// Pad to 4-byte boundary
	for strTableBuf.Len()%4 != 0 {
		strTableBuf.WriteByte(0)
	}

	// Segment header (16 bytes)
	segHeader := make([]byte, 16)
	binary.LittleEndian.PutUint64(segHeader[0:8], segment.NumFunctions)
	binary.LittleEndian.PutUint32(segHeader[8:12], uint32(strTableBuf.Len()))
	binary.LittleEndian.PutUint32(segHeader[12:16], uint32(len(segment.ArgsTable)))

	if _, err := f.Write(segHeader); err != nil {
		return fmt.Errorf("failed to write segment header: %w", err)
	}

	// String table
	if _, err := f.Write(strTableBuf.Bytes()); err != nil {
		return fmt.Errorf("failed to write string table: %w", err)
	}

	// Args table
	for _, arg := range segment.ArgsTable {
		argBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(argBuf, arg)
		if _, err := f.Write(argBuf); err != nil {
			return fmt.Errorf("failed to write args table: %w", err)
		}
	}

	// Function counter data
	counterBuf := new(bytes.Buffer)
	for _, fn := range segment.Functions {
		writeULEB128(counterBuf, uint64(len(fn.Counters))) // num counters
		writeULEB128(counterBuf, fn.PackageID)
		writeULEB128(counterBuf, fn.FuncID)
		for _, counter := range fn.Counters {
			writeULEB128(counterBuf, counter)
		}
	}

	if _, err := f.Write(counterBuf.Bytes()); err != nil {
		return fmt.Errorf("failed to write counter data: %w", err)
	}

	return nil
}

// parseCounterMode converts string to CounterMode
func parseCounterMode(mode string) coverage.CounterMode {
	switch strings.ToLower(mode) {
	case "set":
		return coverage.CtrModeSet
	case "count":
		return coverage.CtrModeCount
	case "atomic":
		return coverage.CtrModeAtomic
	case "regonly":
		return coverage.CtrModeRegOnly
	case "testmain":
		return coverage.CtrModeTestMain
	default:
		return coverage.CtrModeCount // default
	}
}

// parseCounterGranularity converts string to CounterGranularity
func parseCounterGranularity(granularity string) coverage.CounterGranularity {
	switch strings.ToLower(granularity) {
	case "perblock":
		return coverage.CtrGranularityPerBlock
	case "perfunc":
		return coverage.CtrGranularityPerFunc
	default:
		return coverage.CtrGranularityPerBlock // default
	}
}
