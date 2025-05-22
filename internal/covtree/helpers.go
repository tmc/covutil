package covtree

import (
	"io"
	"os"
)

func readFile(filename string) []byte {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return data
}

func openFile(filename string) io.ReadSeeker {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	return file
}
