package main

import (
	"os"
	"testing"

	"rsc.io/script/scripttest"

	"github.com/tmc/covutil/exp/cmd/covanalyze"
	"github.com/tmc/covutil/exp/cmd/covcompare"
	"github.com/tmc/covutil/exp/cmd/covdiff"
	"github.com/tmc/covutil/exp/cmd/covdup"
	"github.com/tmc/covutil/exp/cmd/covered"
	"github.com/tmc/covutil/exp/cmd/covshow"
	"github.com/tmc/covutil/exp/cmd/covzero"
)

func TestMain(m *testing.M) {
	os.Exit(scripttest.RunMain(m, map[string]func() int{
		"covanalyze": covanalyze.Main,
		"covcompare": covcompare.Main,
		"covdiff":    covdiff.Main,
		"covdup":     covdup.Main,
		"covered":    covered.Main,
		"covshow":    covshow.Main,
		"covzero":    covzero.Main,
	}))
}
