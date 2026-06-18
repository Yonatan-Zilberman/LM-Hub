package rag

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/bbolt"
)

func tempDB(t *testing.T) (*bbolt.DB, func()) {
	tmpDir, err := os.MkdirTemp("", "lmhub-rag-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestStore(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	store, err := NewStore(db)
	assert.NoError(t, err)

	chunk1 := &Chunk{
		ID:         "chunk-1",
		FilePath:   "main.go",
		StartLine:  1,
		EndLine:    10,
		Content:    "package main\n\nfunc main() {}",
		Embedding:  []float32{0.1, 0.2, 0.3},
		TokenCount: 15,
	}

	chunk2 := &Chunk{
		ID:         "chunk-2",
		FilePath:   "helper.go",
		StartLine:  5,
		EndLine:    15,
		Content:    "package main\n\nfunc helper() {}",
		Embedding:  []float32{0.4, 0.5, 0.6},
		TokenCount: 20,
	}

	// Test Put
	err = store.PutChunk(chunk1)
	assert.NoError(t, err)
	err = store.PutChunk(chunk2)
	assert.NoError(t, err)

	// Test Get
	got1, err := store.GetChunk("chunk-1")
	assert.NoError(t, err)
	assert.Equal(t, chunk1.FilePath, got1.FilePath)
	assert.Equal(t, chunk1.Content, got1.Content)
	assert.Equal(t, chunk1.Embedding, got1.Embedding)

	// Test GetAll
	all, err := store.GetAllChunks()
	assert.NoError(t, err)
	assert.Len(t, all, 2)

	// Test Stats
	files, chunks, err := store.GetStats()
	assert.NoError(t, err)
	assert.Equal(t, 2, files)
	assert.Equal(t, 2, chunks)

	// Test Meta
	err = store.SetMeta("test-key", "test-val")
	assert.NoError(t, err)
	val, err := store.GetMeta("test-key")
	assert.NoError(t, err)
	assert.Equal(t, "test-val", val)

	// Test LastIndexTime
	now := time.Now().Truncate(time.Second)
	err = store.SetLastIndexTime(now)
	assert.NoError(t, err)
	gotTime, err := store.GetLastIndexTime()
	assert.NoError(t, err)
	assert.True(t, now.Equal(gotTime))

	// Test Delete for file
	err = store.DeleteChunksForFile("main.go")
	assert.NoError(t, err)

	files, chunks, err = store.GetStats()
	assert.NoError(t, err)
	assert.Equal(t, 1, files)
	assert.Equal(t, 1, chunks)

	_, err = store.GetChunk("chunk-1")
	assert.Error(t, err)
}
