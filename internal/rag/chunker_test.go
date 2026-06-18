package rag

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunker(t *testing.T) {
	chunker, err := NewChunker()
	assert.NoError(t, err)

	content := `line 1: this is a very long line that should serve to take up some tokens and test our token-based chunking boundaries
line 2: here is another line of text to continue the document
line 3: third line is short
line 4: fourth line
line 5: fifth line of text
line 6: sixth line
line 7: seventh line`

	// Test chunk size small enough to trigger multiple chunks
	chunks, err := chunker.ChunkFile("test.txt", content, 30, 5)
	assert.NoError(t, err)
	assert.NotEmpty(t, chunks)

	// Ensure chunks are non-empty
	for _, chunk := range chunks {
		assert.Equal(t, "test.txt", chunk.FilePath)
		assert.True(t, chunk.StartLine > 0)
		assert.True(t, chunk.EndLine >= chunk.StartLine)
		assert.NotEmpty(t, chunk.Content)
		assert.True(t, chunk.TokenCount > 0)
		assert.Contains(t, chunk.ID, "test.txt:")
	}

	// Verify line alignment
	firstLines := strings.Split(chunks[0].Content, "\n")
	assert.Equal(t, "line 1: this is a very long line that should serve to take up some tokens and test our token-based chunking boundaries", firstLines[0])
}
