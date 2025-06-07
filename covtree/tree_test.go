package covtree

import (
	"os"
	"testing"
)

func TestLoadFromFS(t *testing.T) {
	// Test with nil options (default behavior)
	t.Run("nil options", func(t *testing.T) {
		tree := NewCoverageTree()
		fsys := os.DirFS(".")

		// This should work with nil options
		err := tree.LoadFromFS(fsys, ".", nil)
		if err != nil {
			// Not an error if no coverage data exists
			if _, ok := err.(*NoCoverageDataError); !ok {
				t.Errorf("LoadFromFS with nil options failed: %v", err)
			}
		}
	})

	// Test with custom options
	t.Run("custom options", func(t *testing.T) {
		tree := NewCoverageTree()
		fsys := os.DirFS(".")

		opts := &LoadOptions{
			MaxDepth:       2,
			FollowSymlinks: false,
		}

		err := tree.LoadFromFS(fsys, ".", opts)
		if err != nil {
			// Not an error if no coverage data exists
			if _, ok := err.(*NoCoverageDataError); !ok {
				t.Errorf("LoadFromFS with custom options failed: %v", err)
			}
		}
	})

	// Test LoadFromDirectory (which internally uses nil options)
	t.Run("LoadFromDirectory", func(t *testing.T) {
		tree := NewCoverageTree()

		err := tree.LoadFromDirectory(".")
		if err != nil {
			// Not an error if no coverage data exists
			if _, ok := err.(*NoCoverageDataError); !ok {
				t.Errorf("LoadFromDirectory failed: %v", err)
			}
		}
	})
}
