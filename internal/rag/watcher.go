// Package rag implements retrieval-augmented generation.
package rag

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors the codebase filesystem for changes and updates the vector database incrementally.
type Watcher struct {
	indexer     *Indexer
	projectRoot string
	fsWatcher   *fsnotify.Watcher
}

// NewWatcher initializes a new Watcher instance.
func NewWatcher(projectRoot string, indexer *Indexer) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	return &Watcher{
		indexer:     indexer,
		projectRoot: projectRoot,
		fsWatcher:   fw,
	}, nil
}

// Start runs the watcher event loop. It blocks until the context is cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	// Add project root and all subdirectories recursively, skipping ignored paths
	err := filepath.Walk(w.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			rel, err := filepath.Rel(w.projectRoot, path)
			if err != nil {
				return err
			}
			if w.indexer.shouldIgnore(rel) {
				return filepath.SkipDir
			}
			err = w.fsWatcher.Add(path)
			if err != nil {
				return fmt.Errorf("failed to watch directory %s: %w", path, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	defer w.fsWatcher.Close()

	debounce := make(map[string]time.Time)

	for {
		select {
		case <-ctx.Done():
			return nil
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("RAG watcher error: %v", err)
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return nil
			}

			rel, err := filepath.Rel(w.projectRoot, event.Name)
			if err != nil {
				continue
			}

			if w.indexer.shouldIgnore(rel) {
				continue
			}

			// Debounce: ignore changes to the same file within 500ms
			if last, ok := debounce[event.Name]; ok && time.Since(last) < 500*time.Millisecond {
				continue
			}
			debounce[event.Name] = time.Now()

			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(event.Name)
				if err != nil {
					continue
				}

				if info.IsDir() {
					_ = w.fsWatcher.Add(event.Name)
					continue
				}

				if !w.indexer.isBinary(event.Name) {
					// Perform background update
					_ = w.indexer.IndexFile(ctx, rel, event.Name)
				}
			} else if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
				_ = w.fsWatcher.Remove(event.Name)
				_ = w.indexer.store.DeleteChunksForFile(rel)
			}
		}
	}
}
