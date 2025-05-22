package main

import (
	"testing"

	"github.com/tmc/covutil/exp/cmd/covanalyze"
	"github.com/tmc/covutil/exp/cmd/covcompare"
	"github.com/tmc/covutil/exp/cmd/covdiff"
	"github.com/tmc/covutil/exp/cmd/covdup"
	"github.com/tmc/covutil/exp/cmd/covshow"
	"github.com/tmc/covutil/exp/cmd/covzero"
)

func TestCmds(t *testing.T) {
	// Basic smoke test that commands can be imported
	cmds := map[string]func() int{
		"covanalyze": covanalyze.Main,
		"covcompare": covcompare.Main,
		"covdiff":    covdiff.Main,
		"covdup":     covdup.Main,
		"covshow":    covshow.Main,
		"covzero":    covzero.Main,
	}

	if len(cmds) == 0 {
		t.Error("no commands found")
	}
}
