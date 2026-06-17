package tools

import (
	"context"
	"strings"
	"testing"
)

func TestShellTool(t *testing.T) {
	tmpDir := t.TempDir()
	blocklist := []string{"rm -rf /", "forbidden_command"}

	shellTool := NewRunCommandTool(tmpDir, 2, []string{"zsh", "bash"}, blocklist)
	ctx := context.Background()

	// 1. Run valid command
	res, err := shellTool.Execute(ctx, map[string]interface{}{
		"cmd": "echo 'hello world'",
	})
	if err != nil {
		t.Fatalf("shell error: %v", err)
	}
	if res.IsError {
		t.Fatalf("shell execution failed: %s", res.Content)
	}
	if !strings.Contains(res.Content, "hello world") {
		t.Errorf("expected output to contain 'hello world', got: %q", res.Content)
	}

	// 2. Test blocklist command
	res, err = shellTool.Execute(ctx, map[string]interface{}{
		"cmd": "forbidden_command",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content, "blocklisted") {
		t.Errorf("expected blocklist error, got: %v", res)
	}

	// 3. Test timeout command
	// Sleep for 5 seconds, but timeout is set to 1 second
	res, err = shellTool.Execute(ctx, map[string]interface{}{
		"cmd":     "sleep 5",
		"timeout": 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content, "timed out") {
		t.Errorf("expected timeout error, got: %v", res)
	}

	// 4. Test invalid cwd
	res, err = shellTool.Execute(ctx, map[string]interface{}{
		"cmd": "ls",
		"cwd": "/invalid/path/outside/scope",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError || !strings.Contains(res.Content, "escapes scope") {
		t.Errorf("expected escapes scope error, got: %v", res)
	}
}
