package rag

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yonatanzilberman/lmhub/internal/api"
)

func TestRetriever(t *testing.T) {
	// 1. Mock API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/embeddings" {
			var req api.EmbeddingsRequest
			json.NewDecoder(r.Body).Decode(&req)

			// Simple mapping: return embedding depending on query
			var emb []float32
			if req.Input[0] == "test query" {
				emb = []float32{1.0, 0.0, 0.0}
			} else {
				emb = []float32{0.0, 1.0, 0.0}
			}

			resp := api.EmbeddingsResponse{
				Object: "list",
				Data: []api.EmbeddingObject{
					{
						Object:    "embedding",
						Embedding: emb,
						Index:     0,
					},
				},
				Model: req.Model,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
			return
		}
	}))
	defer mockServer.Close()

	client := api.NewClient(mockServer.URL, 10)
	db, cleanup := tempDB(t)
	defer cleanup()

	store, err := NewStore(db)
	assert.NoError(t, err)

	// Populate store with pre-indexed chunks
	chunk1 := &Chunk{
		ID:         "chunk-1",
		FilePath:   "foo.go",
		Content:    "foo content",
		Embedding:  []float32{0.9, 0.1, 0.0}, // very close to [1.0, 0.0, 0.0]
		TokenCount: 10,
	}
	chunk2 := &Chunk{
		ID:         "chunk-2",
		FilePath:   "bar.go",
		Content:    "bar content",
		Embedding:  []float32{0.1, 0.9, 0.0}, // far from query [1.0, 0.0, 0.0]
		TokenCount: 20,
	}

	assert.NoError(t, store.PutChunk(chunk1))
	assert.NoError(t, store.PutChunk(chunk2))

	retriever := NewRetriever(client, store, "mock-model", 0.5)

	// Test Retrieve ranking
	results, err := retriever.Retrieve(context.Background(), "test query", 5, 100)
	assert.NoError(t, err)
	assert.Len(t, results, 1) // chunk2 should be filtered out by minScore (cosine similarity will be low)
	assert.Equal(t, "chunk-1", results[0].ID)

	// Test Retrieve budget capping (maxTokens)
	chunk3 := &Chunk{
		ID:         "chunk-3",
		FilePath:   "baz.go",
		Content:    "baz content",
		Embedding:  []float32{0.8, 0.2, 0.0}, // also close
		TokenCount: 100,
	}
	assert.NoError(t, store.PutChunk(chunk3))

	// If we limit to 50 tokens, chunk3 (100 tokens) should not be returned, only chunk1
	results2, err := retriever.Retrieve(context.Background(), "test query", 5, 50)
	assert.NoError(t, err)
	assert.Len(t, results2, 1)
	assert.Equal(t, "chunk-1", results2[0].ID)
}
