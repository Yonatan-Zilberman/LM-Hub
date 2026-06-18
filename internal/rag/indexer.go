// Package rag implements retrieval-augmented generation.
package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yonatanzilberman/lmhub/internal/api"
)

// ProgressInfo tracks progress of a codebase indexing session.
type ProgressInfo struct {
	FilesProcessed int
	TotalFiles     int
	CurrentFile    string
}

// Indexer coordinates walking, reading, chunking, and embedding files into the vector store.
type Indexer struct {
	client    *api.Client
	store     *Store
	chunker   *Chunker
	model     string
	excludes  []string
	chunkSize int
	overlap   int
}

// NewIndexer creates a new Indexer instance.
func NewIndexer(client *api.Client, store *Store, chunker *Chunker, model string, excludes []string, chunkSize, overlap int) *Indexer {
	return &Indexer{
		client:    client,
		store:     store,
		chunker:   chunker,
		model:     model,
		excludes:  excludes,
		chunkSize: chunkSize,
		overlap:   overlap,
	}
}

// IndexWalk scans the project directory and updates the vector database.
// It reports progress via a callback.
func (idx *Indexer) IndexWalk(ctx context.Context, projectRoot string, progress func(ProgressInfo)) error {
	var filesToIndex []string

	// Walk to find all files to index
	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			if idx.shouldIgnore(rel) {
				return filepath.SkipDir
			}
			return nil
		}

		if idx.shouldIgnore(rel) || idx.isBinary(path) {
			return nil
		}

		filesToIndex = append(filesToIndex, rel)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan project directory: %w", err)
	}

	total := len(filesToIndex)
	for i, rel := range filesToIndex {
		if progress != nil {
			progress(ProgressInfo{
				FilesProcessed: i,
				TotalFiles:     total,
				CurrentFile:    rel,
			})
		}

		absPath := filepath.Join(projectRoot, rel)
		err := idx.IndexFile(ctx, rel, absPath)
		if err != nil {
			return fmt.Errorf("failed to index file %s: %w", rel, err)
		}
	}

	if progress != nil {
		progress(ProgressInfo{
			FilesProcessed: total,
			TotalFiles:     total,
			CurrentFile:    "Complete",
		})
	}

	err = idx.store.SetLastIndexTime(time.Now())
	if err != nil {
		return fmt.Errorf("failed to save index timestamp: %w", err)
	}

	return nil
}

// IndexFile chunks a single file and persists its embeddings to the store.
func (idx *Indexer) IndexFile(ctx context.Context, relPath, absPath string) error {
	// 1. Delete existing chunks for this file
	err := idx.store.DeleteChunksForFile(relPath)
	if err != nil {
		return fmt.Errorf("failed to clear old chunks: %w", err)
	}

	// 2. Read file content
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)
	if len(strings.TrimSpace(content)) == 0 {
		return nil
	}

	// 3. Chunk the file
	chunks, err := idx.chunker.ChunkFile(relPath, content, idx.chunkSize, idx.overlap)
	if err != nil {
		return fmt.Errorf("failed to chunk file: %w", err)
	}

	if len(chunks) == 0 {
		return nil
	}

	// 4. Batch request embeddings to minimize API calls
	var texts []string
	for _, chunk := range chunks {
		texts = append(texts, chunk.Content)
	}

	resp, err := idx.client.CreateEmbeddings(ctx, idx.model, texts)
	if err != nil {
		return fmt.Errorf("embeddings generation failed: %w", err)
	}

	if len(resp.Data) != len(chunks) {
		return fmt.Errorf("received %d embeddings for %d chunks", len(resp.Data), len(chunks))
	}

	// 5. Save chunks to vector store
	for i, embObj := range resp.Data {
		chunks[i].Embedding = embObj.Embedding
		err = idx.store.PutChunk(chunks[i])
		if err != nil {
			return fmt.Errorf("failed to save chunk: %w", err)
		}
	}

	return nil
}

func (idx *Indexer) shouldIgnore(path string) bool {
	// Standard system/library directories
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") && part != "." && part != ".." {
			return true
		}
		if part == "node_modules" || part == "vendor" || part == "dist" || part == "build" {
			return true
		}
	}

	for _, pattern := range idx.excludes {
		pat := filepath.ToSlash(pattern)
		match, err := filepath.Match(pat, filepath.ToSlash(path))
		if err == nil && match {
			return true
		}
		if strings.Contains(filepath.ToSlash(path), pat) {
			return true
		}
	}
	return false
}

func (idx *Indexer) isBinary(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true // Treat unreadable files as skipped/binary
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil {
		return false // Empty file is not binary
	}

	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}
	return false
}
