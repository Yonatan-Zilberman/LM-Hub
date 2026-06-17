package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
)

// MarkdownRenderer wraps Glamour to render markdown and code blocks inside the terminal.
type MarkdownRenderer struct {
	renderer *glamour.TermRenderer
}

// NewMarkdownRenderer creates a new MarkdownRenderer.
func NewMarkdownRenderer() (*MarkdownRenderer, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(0), // Wrap handled by viewports
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Glamour renderer: %w", err)
	}

	return &MarkdownRenderer{renderer: r}, nil
}

// Render compiles markdown text into styled terminal ANSI codes.
func (mr *MarkdownRenderer) Render(markdown string) (string, error) {
	// Glamour expects trailing newlines to render blocks correctly
	if !strings.HasSuffix(markdown, "\n") {
		markdown += "\n"
	}
	
	rendered, err := mr.renderer.Render(markdown)
	if err != nil {
		return "", err
	}
	return rendered, nil
}
