package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tmc/covutil/covtree"
)

// WatchedWebServer extends WebServer with file watching capabilities
type WatchedWebServer struct {
	*WebServer
	watcher       *fsnotify.Watcher
	inputDir      string
	reloadChannel chan struct{}
}

// NewWatchedWebServer creates a new web server with file watching
func NewWatchedWebServer(server *WebServer, inputDir string) (*WatchedWebServer, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &WatchedWebServer{
		WebServer:     server,
		watcher:       watcher,
		inputDir:      inputDir,
		reloadChannel: make(chan struct{}, 1),
	}, nil
}

// StartWatching begins monitoring the input directory for coverage file changes
func (ws *WatchedWebServer) StartWatching(ctx context.Context) error {
	// Add the directory and all subdirectories to the watcher
	err := filepath.Walk(ws.inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return ws.watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	log.Printf("watching %s for coverage data changes", ws.inputDir)

	// Start watching in a goroutine
	go ws.watchLoop(ctx)
	go ws.reloadLoop(ctx)

	return nil
}

// Close closes the file watcher
func (ws *WatchedWebServer) Close() error {
	return ws.watcher.Close()
}

func (ws *WatchedWebServer) watchLoop(ctx context.Context) {
	debounceTimer := time.NewTimer(0)
	debounceTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-ws.watcher.Events:
			if !ok {
				return
			}

			// Only process coverage file changes
			if isCoverageFile(event.Name) {
				log.Printf("detected coverage file change: %s", event.Name)
				// Debounce rapid changes
				debounceTimer.Reset(500 * time.Millisecond)
			}

		case err, ok := <-ws.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)

		case <-debounceTimer.C:
			// Signal reload after debounce period
			select {
			case ws.reloadChannel <- struct{}{}:
			default:
				// Channel full, reload already pending
			}
		}
	}
}

func (ws *WatchedWebServer) reloadLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ws.reloadChannel:
			ws.reloadCoverageData()
		}
	}
}

func (ws *WatchedWebServer) reloadCoverageData() {
	log.Printf("reloading coverage data from %s...", ws.inputDir)

	// Create new tree and load data
	newTree := covtree.NewCoverageTree()
	if err := newTree.LoadFromNestedRepository(ws.inputDir); err != nil {
		log.Printf("failed to reload coverage data: %v", err)
		return
	}

	// Atomically replace the tree
	ws.Tree = newTree
	log.Printf("reloaded %d packages", len(newTree.Packages))
}

func isCoverageFile(filename string) bool {
	base := filepath.Base(filename)
	return strings.HasPrefix(base, "covmeta.") || strings.HasPrefix(base, "covcounters.")
}