package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectContext(t *testing.T) {
	cm, err := NewContextManager()
	if err != nil {
		t.Fatalf("failed to create context manager: %v", err)
	}

	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "lmhub-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Case 1: Missing context.md should return empty string, no error
	content, err := LoadProjectContext(tempDir, cm, 100)
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if content != "" {
		t.Errorf("expected empty content, got: %s", content)
	}

	// Create .lmhub folder and context.md
	lmhubDir := filepath.Join(tempDir, ".lmhub")
	if err := os.Mkdir(lmhubDir, 0755); err != nil {
		t.Fatalf("failed to create .lmhub dir: %v", err)
	}

	rawContent := "This is the project context file content."
	if err := os.WriteFile(filepath.Join(lmhubDir, "context.md"), []byte(rawContent), 0644); err != nil {
		t.Fatalf("failed to write context.md: %v", err)
	}

	// Case 2: Loading existing context.md
	content, err = LoadProjectContext(tempDir, cm, 100)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if content != rawContent {
		t.Errorf("expected content '%s', got '%s'", rawContent, content)
	}

	// Case 3: Truncating to low token limit
	truncatedContent, err := LoadProjectContext(tempDir, cm, 2)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(truncatedContent) >= len(rawContent) {
		t.Errorf("expected truncated content to be shorter than raw, got '%s'", truncatedContent)
	}
}
