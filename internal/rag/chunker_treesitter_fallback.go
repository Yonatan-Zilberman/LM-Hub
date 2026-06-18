//go:build !treesitter

package rag

// ChunkFileAST falls back to standard line-based sliding-window chunking.
func (c *Chunker) ChunkFileAST(filePath string, content string, chunkSize, overlap int) ([]*Chunk, error) {
	return c.ChunkFile(filePath, content, chunkSize, overlap)
}
