package main

import (
	"os"

	"github.com/tmc/covutil/exp/cmd/covered"
)

func main() {
	os.Exit(covered.Main())
}