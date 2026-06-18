// Package rag implements retrieval-augmented generation.
package rag

import (
	"context"
	"fmt"
	"sort"

	"github.com/yonatanzilberman/lmhub/internal/api"
)

// ChunkWithScore wraps a Chunk with its similarity score.
type ChunkWithScore struct {
	*Chunk
	Score float32
}

// Retriever handles querying the vector store and ranking results.
type Retriever struct {
	client   *api.Client
	store    *Store
	model    string
	minScore float64
}

// NewRetriever creates a new Retriever instance.
func NewRetriever(client *api.Client, store *Store, model string, minScore float64) *Retriever {
	return &Retriever{
		client:   client,
		store:    store,
		model:    model,
		minScore: minScore,
	}
}

// Store returns the underlying store instance.
func (r *Retriever) Store() *Store {
	return r.store
}

// Retrieve searches the vector store for chunks matching the query.
// It ranks chunks by cosine similarity, filters by minScore, and respects topK and maxTokens budgets.
func (r *Retriever) Retrieve(ctx context.Context, query string, topK int, maxTokens int) ([]*Chunk, error) {
	if len(query) == 0 {
		return nil, nil
	}

	// 1. Generate embedding for query
	resp, err := r.client.CreateEmbeddings(ctx, r.model, []string{query})
	if err != nil {
		return nil, fmt.Errorf("retriever failed to embed query: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("retriever received empty embedding data for query")
	}

	queryVec := resp.Data[0].Embedding

	// 2. Fetch all chunks
	allChunks, err := r.store.GetAllChunks()
	if err != nil {
		return nil, fmt.Errorf("retriever failed to load chunks: %w", err)
	}

	// 3. Score all chunks
	var scored []ChunkWithScore
	for _, chunk := range allChunks {
		similarity, err := CosineSimilarity(queryVec, chunk.Embedding)
		if err != nil {
			// Skip chunks with vector dimensions that do not match the query
			continue
		}

		if float64(similarity) >= r.minScore {
			scored = append(scored, ChunkWithScore{
				Chunk: chunk,
				Score: similarity,
			})
		}
	}

	// 4. Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// 5. Select topK chunks within maxTokens budget
	var results []*Chunk
	totalTokens := 0

	for _, s := range scored {
		if len(results) >= topK {
			break
		}

		tokens := s.TokenCount
		if tokens <= 0 {
			tokens = len(s.Content) / 4
		}

		if totalTokens+tokens > maxTokens && len(results) > 0 {
			// Stop if adding this chunk would exceed the token limit, unless we have nothing yet.
			break
		}

		results = append(results, s.Chunk)
		totalTokens += tokens
	}

	return results, nil
}
