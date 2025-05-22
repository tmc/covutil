//go:build gocoverruntime

package cfile

import "github.com/tmc/covutil/internal/coverage/rtcov"

// getCovCounterList returns a list of counter-data blobs registered
// for the currently executing instrumented program. It is defined in the
// runtime.
//
//go:linkname getCovCounterList
func getCovCounterList() []rtcov.CovCounterBlob
