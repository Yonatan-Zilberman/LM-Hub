package rag

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yonatanzilberman/lmhub/internal/api"
)

func TestIndexer(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/embeddings" {
			var req api.EmbeddingsRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			var data []api.EmbeddingObject
			for i := range req.Input {
				// Dummy embedding dimension 3
				data = append(data, api.EmbeddingObject{
					Object:    "embedding",
					Embedding: []float32{0.1, 0.2, 0.3},
					Index:     i,
				})
			}

			resp := api.EmbeddingsResponse{
				Object: "list",
				Data:   data,
				Model:  req.Model,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	// 2. Set up components
	client := api.NewClient(mockServer.URL, 10)
	db, dbCleanup := tempDB(t)
	defer dbCleanup()

	store, err := NewStore(db)
	assert.NoError(t, err)

	chunker, err := NewChunker()
	assert.NoError(t, err)

	excludes := []string{"*.lock", "ignored_dir/**"}
	indexer := NewIndexer(client, store, chunker, "mock-emb-model", excludes, 30, 5)

	// 3. Set up temp project directory
	tmpProj, err := os.MkdirTemp("", "lmhub-proj-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpProj)

	// File 1: to index
	file1Path := filepath.Join(tmpProj, "main.go")
	err = os.WriteFile(file1Path, []byte("package main\n\nfunc main() {\n\t// comment\n}"), 0644)
	assert.NoError(t, err)

	// File 2: ignored by default (node_modules)
	nodeDir := filepath.Join(tmpProj, "node_modules")
	err = os.MkdirAll(nodeDir, 0755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(nodeDir, "index.js"), []byte("console.log('ignored')"), 0644)
	assert.NoError(t, err)

	// File 3: ignored by exclude pattern (*.lock)
	err = os.WriteFile(filepath.Join(tmpProj, "cargo.lock"), []byte("lock content"), 0644)
	assert.NoError(t, err)

	// File 4: binary file (null bytes)
	binPath := filepath.Join(tmpProj, "exec.bin")
	err = os.WriteFile(binPath, []byte{0x00, 0x01, 0x02, 0x03}, 0644)
	assert.NoError(t, err)

	// 4. Perform index walk
	var progressEvents []ProgressInfo
	progressCallback := func(info ProgressInfo) {
		progressEvents = append(progressEvents, info)
	}

	err = indexer.IndexWalk(context.Background(), tmpProj, progressCallback)
	assert.NoError(t, err)
	assert.NotEmpty(t, progressEvents)

	// 5. Verify store results
	files, chunks, err := store.GetStats()
	assert.NoError(t, err)
	assert.Equal(t, 1, files) // Only main.go is indexed
	assert.Equal(t, 1, chunks)

	// Verify meta last index time exists
	lastTime, err := store.GetLastIndexTime()
	assert.NoError(t, err)
	assert.False(t, lastTime.IsZero())
}
