package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestUndoStack(t *testing.T) {
	stack := NewUndoStack()

	r1 := UndoRecord{ToolName: "write_file", Description: "First"}
	r2 := UndoRecord{ToolName: "create_dir", Description: "Second"}

	stack.Push(r1)
	stack.Push(r2)

	if stack.Len() != 2 {
		t.Errorf("expected length 2, got %d", stack.Len())
	}

	// Peek
	peeked, ok := stack.Peek()
	if !ok || peeked.Description != "Second" {
		t.Errorf("peek failed")
	}

	// List
	list := stack.List()
	if len(list) != 2 || list[0].Description != "Second" || list[1].Description != "First" {
		t.Errorf("list order incorrect")
	}

	// Pop
	popped, ok := stack.Pop()
	if !ok || popped.Description != "Second" {
		t.Errorf("pop failed")
	}

	if stack.Len() != 1 {
		t.Errorf("expected length 1, got %d", stack.Len())
	}
}

func TestUndoStack_UndoOperations(t *testing.T) {
	tmpDir := t.TempDir()
	r := NewRegistry(tmpDir)

	writeTool := NewWriteFileTool(tmpDir)
	deleteTool := NewDeleteFileTool(tmpDir)
	r.Register(writeTool)
	r.Register(deleteTool)

	stack := NewUndoStack()
	ctx := context.Background()

	// Perform an action: write a file
	filePath := "hello.txt"
	snapshot, err := writeTool.Snapshot(ctx, map[string]interface{}{
		"path":    filePath,
		"content": "hello world",
		"mode":    "overwrite",
	})
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	_, err = writeTool.Execute(ctx, map[string]interface{}{
		"path":    filePath,
		"content": "hello world",
		"mode":    "overwrite",
	})
	if err != nil {
		t.Fatalf("failed to execute write: %v", err)
	}

	stack.Push(snapshot)

	// Verify file exists
	fullPath := filepath.Join(tmpDir, filePath)
	if _, err := os.Stat(fullPath); err != nil {
		t.Fatalf("file not found before undo")
	}

	// Perform Undo
	if err := stack.UndoLast(ctx, r); err != nil {
		t.Fatalf("undo failed: %v", err)
	}

	// Verify file is deleted (rollback target for new file is delete_file)
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Errorf("file still exists after undo")
	}
}
