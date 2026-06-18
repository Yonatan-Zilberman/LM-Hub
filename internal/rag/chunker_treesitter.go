//go:build treesitter

package rag

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

// ChunkFileAST uses tree-sitter to parse source files and partition them into logical function/struct chunks.
func (c *Chunker) ChunkFileAST(filePath string, content string, chunkSize, overlap int) ([]*Chunk, error) {
	// Fall back to sliding window chunking if not Go file
	if !strings.HasSuffix(filePath, ".go") {
		return c.ChunkFile(filePath, content, chunkSize, overlap)
	}

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(content))
	if err != nil || tree == nil {
		return c.ChunkFile(filePath, content, chunkSize, overlap)
	}

	root := tree.RootNode()
	if root == nil {
		return c.ChunkFile(filePath, content, chunkSize, overlap)
	}

	var chunks []*Chunk
	lines := strings.Split(content, "\n")

	// Helper function to recursively traverse nodes and extract functions/struct declarations
	var visit func(*sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}

		// Look for function_declaration, method_declaration, type_declaration
		t := n.Type()
		if t == "function_declaration" || t == "method_declaration" || t == "type_declaration" {
			startRow := int(n.StartPoint().Row)
			endRow := int(n.EndPoint().Row)

			if startRow < len(lines) && endRow <= len(lines) {
				chunkLines := lines[startRow:endRow]
				chunkContent := strings.Join(chunkLines, "\n")
				tokCount := c.CountTokens(chunkContent)

				// Only save as standalone chunk if it's within limits, otherwise fallback chunker handles it
				if tokCount > 0 && tokCount <= chunkSize {
					chunkID := fmt.Sprintf("%s:%d-%d", filePath, startRow+1, endRow)
					chunks = append(chunks, &Chunk{
						ID:         chunkID,
						FilePath:   filePath,
						StartLine:  startRow + 1,
						EndLine:    endRow,
						Content:    chunkContent,
						TokenCount: tokCount,
					})
					return // Don't traverse deeper inside this declaration
				}
			}
		}

		for i := 0; i < int(n.ChildCount()); i++ {
			visit(n.Child(i))
		}
	}

	visit(root)

	// If no function/struct chunks were found, run fallback
	if len(chunks) == 0 {
		return c.ChunkFile(filePath, content, chunkSize, overlap)
	}

	return chunks, nil
}
