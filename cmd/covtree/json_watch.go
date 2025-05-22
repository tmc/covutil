package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

func watchAndStreamJSON(ctx context.Context, dir string, encoder *json.Encoder) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %v", err)
	}
	defer watcher.Close()

	// Add the directory and all subdirectories to the watcher
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to setup directory watching: %v", err)
	}

	log.Printf("covtree: watching %s for coverage data changes", dir)

	// Process initial state
	if err := processDirectoryToJSON(dir, encoder); err != nil {
		log.Printf("covtree: initial processing failed: %v", err)
	}

	// Watch for changes
	debounceTimer := time.NewTimer(0)
	debounceTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("watcher closed unexpectedly")
			}

			// Only process coverage file changes
			if isCoverageFile(event.Name) {
				// Debounce rapid changes
				debounceTimer.Reset(500 * time.Millisecond)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher error channel closed")
			}
			log.Printf("covtree: watcher error: %v", err)

		case <-debounceTimer.C:
			// Process directory after debounce period
			if err := processDirectoryToJSON(dir, encoder); err != nil {
				log.Printf("covtree: processing failed: %v", err)
			}
		}
	}
}
