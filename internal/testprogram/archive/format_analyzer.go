package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

// analyzeRealCoverageFormat performs detailed analysis of real coverage files
func analyzeRealCoverageFormat() {
	fmt.Println("\n=== DETAILED REAL COVERAGE FORMAT ANALYSIS ===")

	realDir := "covdata2"
	entries, err := os.ReadDir(realDir)
	if err != nil {
		fmt.Printf("Error reading %s directory: %v\n", realDir, err)
		return
	}

	var metaFile, counterFile string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) > 8 && name[:8] == "covmeta." {
			metaFile = filepath.Join(realDir, name)
		} else if len(name) > 12 && name[:12] == "covcounters." {
			counterFile = filepath.Join(realDir, name)
		}
	}

	if metaFile != "" {
		fmt.Printf("\n=== ANALYZING METADATA FILE: %s ===\n", metaFile)
		analyzeMetadataFileDetailed(metaFile)
	}

	if counterFile != "" {
		fmt.Printf("\n=== ANALYZING COUNTER FILE: %s ===\n", counterFile)
		analyzeCounterFileDetailed(counterFile)
	}
}

func analyzeMetadataFileDetailed(filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	fmt.Printf("File size: %d bytes\n", len(data))

	if len(data) < 64 {
		fmt.Println("File too small for analysis")
		return
	}

	// Parse header (first 64 bytes)
	fmt.Println("\n--- FILE HEADER (64 bytes) ---")
	magic := data[0:4]
	version := binary.LittleEndian.Uint32(data[4:8])
	totalLength := binary.LittleEndian.Uint64(data[8:16])
	numPackages := binary.LittleEndian.Uint64(data[16:24])
	metaHash := data[24:40]
	strTabOffset := binary.LittleEndian.Uint32(data[40:44])
	strTabLength := binary.LittleEndian.Uint32(data[44:48])
	strTabEntries := binary.LittleEndian.Uint32(data[48:52])
	counterMode := data[52]
	granularity := data[53]

	fmt.Printf("Magic: %x ('%s')\n", magic, string(magic))
	fmt.Printf("Version: %d\n", version)
	fmt.Printf("Total Length: %d\n", totalLength)
	fmt.Printf("Number of Packages: %d\n", numPackages)
	fmt.Printf("Meta Hash: %x\n", metaHash)
	fmt.Printf("String Table Offset: %d\n", strTabOffset)
	fmt.Printf("String Table Length: %d\n", strTabLength)
	fmt.Printf("String Table Entries: %d\n", strTabEntries)
	fmt.Printf("Counter Mode: %d\n", counterMode)
	fmt.Printf("Granularity: %d\n", granularity)

	// Show hex dump of header
	fmt.Println("\nHeader hex dump:")
	for i := 0; i < 64; i += 16 {
		end := i + 16
		if end > 64 {
			end = 64
		}
		fmt.Printf("%04x: ", i)
		for j := i; j < end; j++ {
			fmt.Printf("%02x ", data[j])
		}
		fmt.Println()
	}

	// Parse package offset and length tables
	if len(data) >= int(64+8*numPackages*2) {
		fmt.Printf("\n--- PACKAGE TABLES (packages: %d) ---\n", numPackages)

		// Package offsets
		fmt.Println("Package Offsets:")
		for i := uint64(0); i < numPackages; i++ {
			offset := binary.LittleEndian.Uint64(data[64+i*8 : 64+(i+1)*8])
			fmt.Printf("  Package %d: offset %d\n", i, offset)
		}

		// Package lengths
		fmt.Println("Package Lengths:")
		lengthTableStart := 64 + numPackages*8
		for i := uint64(0); i < numPackages; i++ {
			length := binary.LittleEndian.Uint64(data[lengthTableStart+i*8 : lengthTableStart+(i+1)*8])
			fmt.Printf("  Package %d: length %d\n", i, length)
		}
	}

	// Parse string table
	if strTabOffset > 0 && int(strTabOffset+strTabLength) <= len(data) {
		fmt.Printf("\n--- STRING TABLE (offset: %d, length: %d) ---\n", strTabOffset, strTabLength)
		strTabData := data[strTabOffset : strTabOffset+strTabLength]

		fmt.Println("String table hex dump:")
		for i := 0; i < int(strTabLength) && i < 64; i += 16 {
			end := i + 16
			if end > int(strTabLength) {
				end = int(strTabLength)
			}
			fmt.Printf("%04x: ", i)
			for j := i; j < end; j++ {
				fmt.Printf("%02x ", strTabData[j])
			}
			fmt.Println()
		}
	}

	// Analyze package metadata
	if numPackages > 0 && len(data) > 64+int(numPackages*16) {
		fmt.Printf("\n--- PACKAGE METADATA ANALYSIS ---\n")

		// Get first package offset and length
		firstPkgOffset := binary.LittleEndian.Uint64(data[64:72])
		lengthTableStart := 64 + numPackages*8
		firstPkgLength := binary.LittleEndian.Uint64(data[lengthTableStart : lengthTableStart+8])

		fmt.Printf("First package: offset=%d, length=%d\n", firstPkgOffset, firstPkgLength)

		if int(firstPkgOffset+firstPkgLength) <= len(data) {
			pkgData := data[firstPkgOffset : firstPkgOffset+firstPkgLength]
			fmt.Printf("Package data (first 64 bytes):\n")
			for i := 0; i < 64 && i < len(pkgData); i += 16 {
				end := i + 16
				if end > len(pkgData) {
					end = len(pkgData)
				}
				fmt.Printf("%04x: ", i)
				for j := i; j < end; j++ {
					fmt.Printf("%02x ", pkgData[j])
				}
				fmt.Println()
			}
		}
	}
}

func analyzeCounterFileDetailed(filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	fmt.Printf("File size: %d bytes\n", len(data))

	if len(data) < 32 {
		fmt.Println("File too small for analysis")
		return
	}

	// Parse header (first 32 bytes)
	fmt.Println("\n--- FILE HEADER (32 bytes) ---")
	magic := data[0:4]
	version := binary.LittleEndian.Uint32(data[4:8])
	metaHash := data[8:24]
	numSegments := binary.LittleEndian.Uint64(data[24:32])

	fmt.Printf("Magic: %x ('%s')\n", magic, string(magic))
	fmt.Printf("Version: %d\n", version)
	fmt.Printf("Meta Hash: %x\n", metaHash)
	fmt.Printf("Number of Segments: %d\n", numSegments)

	// Show hex dump of header
	fmt.Println("\nHeader hex dump:")
	for i := 0; i < 32; i += 16 {
		end := i + 16
		if end > 32 {
			end = 32
		}
		fmt.Printf("%04x: ", i)
		for j := i; j < end; j++ {
			fmt.Printf("%02x ", data[j])
		}
		fmt.Println()
	}

	// Parse first segment if it exists
	if numSegments > 0 && len(data) > 32 {
		fmt.Printf("\n--- FIRST SEGMENT ANALYSIS ---\n")

		// Segment header should be at offset 32
		if len(data) >= 48 {
			segmentData := data[32:]
			numFunctions := binary.LittleEndian.Uint64(segmentData[0:8])
			strTableLen := binary.LittleEndian.Uint32(segmentData[8:12])
			argsTableLen := binary.LittleEndian.Uint32(segmentData[12:16])

			fmt.Printf("Functions in segment: %d\n", numFunctions)
			fmt.Printf("String table length: %d\n", strTableLen)
			fmt.Printf("Args table length: %d\n", argsTableLen)

			// Show segment header hex
			fmt.Println("Segment header hex dump:")
			for i := 0; i < 16 && i < len(segmentData); i += 16 {
				fmt.Printf("%04x: ", i+32)
				for j := i; j < i+16 && j < len(segmentData); j++ {
					fmt.Printf("%02x ", segmentData[j])
				}
				fmt.Println()
			}

			// Show string table if present
			if strTableLen > 0 && int(16+strTableLen) <= len(segmentData) {
				fmt.Printf("\nString table data (length %d):\n", strTableLen)
				strTable := segmentData[16 : 16+strTableLen]
				for i := 0; i < int(strTableLen) && i < 64; i += 16 {
					end := i + 16
					if end > int(strTableLen) {
						end = int(strTableLen)
					}
					fmt.Printf("%04x: ", i)
					for j := i; j < end; j++ {
						fmt.Printf("%02x ", strTable[j])
					}
					fmt.Println()
				}
			}
		}
	}

	// Check footer (last 16 bytes)
	if len(data) >= 16 {
		fmt.Printf("\n--- FOOTER (last 16 bytes) ---\n")
		footer := data[len(data)-16:]
		footerMagic := footer[0:4]
		footerSegments := binary.LittleEndian.Uint32(footer[8:12])

		fmt.Printf("Footer magic: %x ('%s')\n", footerMagic, string(footerMagic))
		fmt.Printf("Footer segments: %d\n", footerSegments)

		fmt.Println("Footer hex dump:")
		for i := 0; i < 16; i += 16 {
			fmt.Printf("%04x: ", len(data)-16+i)
			for j := i; j < 16; j++ {
				fmt.Printf("%02x ", footer[j])
			}
			fmt.Println()
		}
	}
}

// analyzeSyntheticCoverageFormat analyzes our synthetic coverage files
func analyzeSyntheticCoverageFormat() {
	fmt.Println("\n=== DETAILED SYNTHETIC COVERAGE FORMAT ANALYSIS ===")

	syntheticDir := "covdata"
	entries, err := os.ReadDir(syntheticDir)
	if err != nil {
		fmt.Printf("Error reading %s directory: %v\n", syntheticDir, err)
		return
	}

	var metaFile, counterFile string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) > 8 && name[:8] == "covmeta." {
			metaFile = filepath.Join(syntheticDir, name)
		} else if len(name) > 12 && name[:12] == "covcounters." {
			counterFile = filepath.Join(syntheticDir, name)
		}
	}

	if metaFile != "" {
		fmt.Printf("\n=== ANALYZING SYNTHETIC METADATA FILE: %s ===\n", metaFile)
		analyzeMetadataFileDetailed(metaFile)
	}

	if counterFile != "" {
		fmt.Printf("\n=== ANALYZING SYNTHETIC COUNTER FILE: %s ===\n", counterFile)
		analyzeCounterFileDetailed(counterFile)
	}
}
