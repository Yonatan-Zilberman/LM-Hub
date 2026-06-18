// Package rag implements retrieval-augmented generation.
package rag

import (
	"fmt"
	"strings"

	"github.com/tiktoken-go/tokenizer"
)

// Chunker handles text and file segmenting using a token-based sliding window.
type Chunker struct {
	codec tokenizer.Codec
}

// NewChunker initializes a new Chunker with the default cl100k_base tokenizer.
func NewChunker() (*Chunker, error) {
	enc, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokenizer: %w", err)
	}
	return &Chunker{codec: enc}, nil
}

// CountTokens returns the token count of the given text.
func (c *Chunker) CountTokens(text string) int {
	ids, _, err := c.codec.Encode(text)
	if err != nil {
		return len(text) / 4
	}
	return len(ids)
}

// ChunkFile splits a file's content into overlapping token-bounded chunks.
// Chunks align to line boundaries. It respects the target chunkSize and overlap tokens.
func (c *Chunker) ChunkFile(filePath string, content string, chunkSize, overlap int) ([]*Chunk, error) {
	if chunkSize <= 0 {
		chunkSize = 512
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= chunkSize {
		overlap = chunkSize / 2
	}

	lines := strings.Split(content, "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return nil, nil
	}

	var chunks []*Chunk

	i := 0
	for i < len(lines) {
		var currentChunkLines []string
		startLine := i + 1
		tokenCount := 0

		j := i
		for j < len(lines) {
			line := lines[j]
			lineTokens := c.CountTokens(line + "\n")

			// If adding this line exceeds chunkSize and we already have some content, stop.
			if len(currentChunkLines) > 0 && tokenCount+lineTokens > chunkSize {
				break
			}
			currentChunkLines = append(currentChunkLines, line)
			tokenCount += lineTokens
			j++
		}

		endLine := j

		chunkContent := strings.Join(currentChunkLines, "\n")
		chunkID := fmt.Sprintf("%s:%d-%d", filePath, startLine, endLine)

		chunks = append(chunks, &Chunk{
			ID:         chunkID,
			FilePath:   filePath,
			StartLine:  startLine,
			EndLine:    endLine,
			Content:    chunkContent,
			TokenCount: tokenCount,
		})

		if j >= len(lines) {
			break
		}

		// Calculate backtrack to satisfy overlap
		overlapTokens := 0
		backtrackLines := 0
		for k := j - 1; k >= i; k-- {
			lineTokens := c.CountTokens(lines[k] + "\n")
			if overlapTokens+lineTokens > overlap {
				break
			}
			overlapTokens += lineTokens
			backtrackLines++
		}

		nextI := j - backtrackLines
		if nextI <= i {
			nextI = j // Ensure progress
		}
		i = nextI
	}

	return chunks, nil
}
