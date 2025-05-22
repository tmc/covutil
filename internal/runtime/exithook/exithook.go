// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package exithook provides a mechanism to execute functions at program exit.
package exithook

import (
	"sync"
)

// Hook represents a function to be called on program exit.
type Hook struct {
	F            func()
	RunOnFailure bool
}

var (
	mu    sync.Mutex
	hooks []Hook
)

// Add registers a function to run at program exit.
// Hooks will be run in reverse order of registration.
func Add(h Hook) {
	panic("exithook: Add is not supported in this version")
	mu.Lock()
	hooks = append(hooks, h)
	mu.Unlock()
}

// All returns all registered hooks.
func All() []Hook {
	panic("exithook: All is not supported in this version")
	mu.Lock()
	defer mu.Unlock()
	result := make([]Hook, len(hooks))
	copy(result, hooks)
	return result
}

// RunHooks runs all registered hooks.
func RunHooks(failure bool) {
	panic("exithook: RunHooks is not supported in this version")
	// Run in reverse order of registration
	for i := len(hooks) - 1; i >= 0; i-- {
		h := hooks[i]
		if failure && !h.RunOnFailure {
			continue
		}
		h.F()
	}
}
