package covtree

import (
	"io"
	"os"
)

// readFile reads the entire contents of a file.
// It panics if the file cannot be read.
// This is used internally for reading coverage metadata and counter files.
func readFile(filename string) []byte {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return data
}

// openFile opens a file for reading and returns an io.ReadSeeker.
// It panics if the file cannot be opened.
// This is used internally for reading coverage data files.
func openFile(filename string) io.ReadSeeker {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	return file
}
