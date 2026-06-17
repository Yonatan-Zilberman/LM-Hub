package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesystemTools(t *testing.T) {
	tmpDir := t.TempDir()

	readTool := NewReadFileTool(tmpDir)
	writeTool := NewWriteFileTool(tmpDir)
	createDirTool := NewCreateDirTool(tmpDir)
	listTool := NewListDirTool(tmpDir)
	deleteTool := NewDeleteFileTool(tmpDir)
	moveTool := NewMoveFileTool(tmpDir)
	searchTool := NewSearchFilesTool(tmpDir)

	ctx := context.Background()

	// 1. Create a directory
	res, err := createDirTool.Execute(ctx, map[string]interface{}{
		"path":      "src",
		"recursive": true,
	})
	if err != nil {
		t.Fatalf("create dir error: %v", err)
	}
	if res.IsError {
		t.Fatalf("create dir returned error: %s", res.Content)
	}

	// 2. Write a file
	res, err = writeTool.Execute(ctx, map[string]interface{}{
		"path":    "src/hello.go",
		"content": "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n",
		"mode":    "overwrite",
	})
	if err != nil {
		t.Fatalf("write file error: %v", err)
	}
	if res.IsError {
		t.Fatalf("write file returned error: %s", res.Content)
	}

	// 3. Read the file
	res, err = readTool.Execute(ctx, map[string]interface{}{
		"path": "src/hello.go",
	})
	if err != nil {
		t.Fatalf("read file error: %v", err)
	}
	if res.IsError {
		t.Fatalf("read file returned error: %s", res.Content)
	}
	if !strings.Contains(res.Content, "fmt.Println") {
		t.Errorf("read file content missing text, got: %s", res.Content)
	}

	// Read with line range
	res, err = readTool.Execute(ctx, map[string]interface{}{
		"path":       "src/hello.go",
		"start_line": 3,
		"end_line":   5,
	})
	if err != nil {
		t.Fatalf("read file range error: %v", err)
	}
	expectedLines := "import \"fmt\"\n\nfunc main() {"
	if strings.TrimSpace(res.Content) != strings.TrimSpace(expectedLines) {
		t.Errorf("expected line range: %q, got: %q", expectedLines, res.Content)
	}

	// 4. List directory
	res, err = listTool.Execute(ctx, map[string]interface{}{
		"path":      "src",
		"recursive": false,
	})
	if err != nil {
		t.Fatalf("list dir error: %v", err)
	}
	if !strings.Contains(res.Content, "hello.go") {
		t.Errorf("list dir output missing file: %s", res.Content)
	}

	// 5. Search files (by content)
	res, err = searchTool.Execute(ctx, map[string]interface{}{
		"path":    "src",
		"pattern": "Println",
		"type":    "content",
	})
	if err != nil {
		t.Fatalf("search content error: %v", err)
	}
	if !strings.Contains(res.Content, "hello.go:") {
		t.Errorf("search content missing match details: %s", res.Content)
	}

	// Search files (by name)
	res, err = searchTool.Execute(ctx, map[string]interface{}{
		"path":    "src",
		"pattern": "hello",
		"type":    "name",
	})
	if err != nil {
		t.Fatalf("search name error: %v", err)
	}
	if !strings.Contains(res.Content, "hello.go") {
		t.Errorf("search name missing match details: %s", res.Content)
	}

	// 6. Move file
	res, err = moveTool.Execute(ctx, map[string]interface{}{
		"src": "src/hello.go",
		"dst": "src/main.go",
	})
	if err != nil {
		t.Fatalf("move file error: %v", err)
	}

	// Verify move
	if _, err := os.Stat(filepath.Join(tmpDir, "src/main.go")); err != nil {
		t.Errorf("moved file main.go not found")
	}

	// 7. Delete file
	res, err = deleteTool.Execute(ctx, map[string]interface{}{
		"path": "src/main.go",
	})
	if err != nil {
		t.Fatalf("delete file error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "src/main.go")); !os.IsNotExist(err) {
		t.Errorf("deleted file main.go still exists")
	}
}

func TestFilesystemTools_ScopeViolations(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := t.TempDir()

	readTool := NewReadFileTool(tmpDir)
	writeTool := NewWriteFileTool(tmpDir)

	ctx := context.Background()

	// Try reading outside scope
	relativeEscape := "../" + filepath.Base(outDir) + "/secret.txt"
	res, err := readTool.Execute(ctx, map[string]interface{}{
		"path": relativeEscape,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Errorf("expected error reading outside scope: %s", relativeEscape)
	}

	// Try writing outside scope with absolute path
	absSecret := filepath.Join(outDir, "secret.txt")
	res, err = writeTool.Execute(ctx, map[string]interface{}{
		"path":    absSecret,
		"content": "hacked",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Errorf("expected error writing to absolute path outside scope: %s", absSecret)
	}
}

func TestFilesystemTools_UndoSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	writeTool := NewWriteFileTool(tmpDir)
	ctx := context.Background()

	filePath := "test.txt"

	// Write new file snapshot
	snapshot1, err := writeTool.Snapshot(ctx, map[string]interface{}{
		"path":    filePath,
		"content": "hello",
	})
	if err != nil {
		t.Fatalf("failed snapshot for new file: %v", err)
	}
	if snapshot1.InverseOp != "delete_file" {
		t.Errorf("expected delete_file inverse for new file, got %s", snapshot1.InverseOp)
	}

	// Actually write the file
	_, err = writeTool.Execute(ctx, map[string]interface{}{
		"path":    filePath,
		"content": "hello",
	})
	if err != nil {
		t.Fatalf("failed writing test file: %v", err)
	}

	// Overwrite file snapshot
	snapshot2, err := writeTool.Snapshot(ctx, map[string]interface{}{
		"path":    filePath,
		"content": "world",
	})
	if err != nil {
		t.Fatalf("failed snapshot for overwrite: %v", err)
	}
	if snapshot2.InverseOp != "write_file" {
		t.Errorf("expected write_file inverse for overwrite, got %s", snapshot2.InverseOp)
	}
	if string(snapshot2.Snapshot) != "hello" {
		t.Errorf("expected snapshotted content to be 'hello', got %s", string(snapshot2.Snapshot))
	}
}
