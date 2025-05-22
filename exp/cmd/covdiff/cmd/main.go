package main

import (
	"os"

	"github.com/tmc/covutil/exp/cmd/covdiff"
)

func main() {
	os.Exit(covdiff.Main())
}
