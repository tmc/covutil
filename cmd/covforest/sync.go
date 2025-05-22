// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
)

var cmdSync = &Command{
	UsageLine: "covforest sync [-config=<file>]",
	Short:     "synchronize trees from remote sources",
	Long: `
Sync synchronizes coverage trees from remote sources defined in a configuration file.

The -config flag specifies the sync configuration file path.

Example:

	covforest sync -config=sync.yaml

The sync configuration file defines remote sources such as:
- Git repositories with coverage data
- CI/CD artifact storage
- Remote file systems
- HTTP endpoints serving coverage data

This command is currently a placeholder for future implementation.
`,
}

var (
	syncConfig = cmdSync.Flag.String("config", "", "sync configuration file path")
)

func init() {
	cmdSync.Run = runSync
}

func runSync(ctx context.Context, args []string) error {
	return fmt.Errorf("sync command is not yet implemented - coming soon!")
}
