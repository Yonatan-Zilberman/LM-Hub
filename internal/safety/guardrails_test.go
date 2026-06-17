package safety

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yonatanzilberman/lmhub/internal/tools"
)

type dummyTool struct {
	name       string
	permission tools.PermissionLevel
}

func (d *dummyTool) Name() string                     { return d.name }
func (d *dummyTool) Description() string              { return "" }
func (d *dummyTool) Schema() tools.ToolSchema         { return tools.ToolSchema{} }
func (d *dummyTool) Permission() tools.PermissionLevel { return d.permission }
func (d *dummyTool) Undoable() bool                   { return false }
func (d *dummyTool) Execute(ctx context.Context, args map[string]interface{}) (tools.ToolResult, error) {
	return tools.ToolResult{}, nil
}
func (d *dummyTool) Snapshot(ctx context.Context, args map[string]interface{}) (tools.UndoRecord, error) {
	return tools.UndoRecord{}, nil
}

func TestClassifier_Classify(t *testing.T) {
	classifier := NewClassifier([]string{"rm -rf", "DROP TABLE"})

	// Safe tool
	safeTool := &dummyTool{name: "read_file", permission: tools.Safe}
	if classifier.Classify(safeTool, nil) != tools.Safe {
		t.Errorf("expected Safe level")
	}

	// Warn tool
	warnTool := &dummyTool{name: "write_file", permission: tools.Warn}
	if classifier.Classify(warnTool, nil) != tools.Warn {
		t.Errorf("expected Warn level")
	}

	// Non-blocklisted run_command (Warn)
	shellTool := &dummyTool{name: "run_command", permission: tools.Warn}
	if classifier.Classify(shellTool, map[string]interface{}{"cmd": "echo hello"}) != tools.Warn {
		t.Errorf("expected Warn level for harmless command")
	}

	// Blocklisted run_command (Dangerous)
	if classifier.Classify(shellTool, map[string]interface{}{"cmd": "rm -rf /tmp"}) != tools.Dangerous {
		t.Errorf("expected Dangerous level for blocklisted command")
	}
}

func TestFileSizeGuard(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")

	// Write 10 bytes
	data := []byte("0123456789")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Test under limit
	if err := FileSizeGuard(filePath, 20); err != nil {
		t.Errorf("unexpected error under limit: %v", err)
	}

	// Test over limit
	if err := FileSizeGuard(filePath, 5); err == nil {
		t.Errorf("expected error for file exceeding limit")
	}

	// Test non-existent file
	if err := FileSizeGuard(filepath.Join(tmpDir, "missing.txt"), 5); err != nil {
		t.Errorf("unexpected error for missing file: %v", err)
	}
}
