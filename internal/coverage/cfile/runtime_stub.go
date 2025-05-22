//go:build !gocoverruntime

package cfile

import "github.com/tmc/covutil/internal/coverage/rtcov"

// getCovCounterList is a stub implementation for when runtime linkage is not available
func getCovCounterList() []rtcov.CovCounterBlob {
	return []rtcov.CovCounterBlob{}
}
