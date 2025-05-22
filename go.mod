module github.com/tmc/covutil

go 1.23.0

toolchain go1.24.3

tool (
	golang.org/x/tools/cmd/fiximports
	golang.org/x/tools/cmd/goimports
)

require github.com/google/go-cmp v0.7.0 // indirect

require (
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
	rsc.io/script v0.0.2 // indirect
)
