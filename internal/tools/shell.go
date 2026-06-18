package tools

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/yonatanzilberman/lmhub/pkg/platform"
)

// RunCommandTool executes shell commands in a scoped directory.
type RunCommandTool struct {
	scopeRoot    string
	timeoutSec   int
	allowedShell []string
	blocklist    []string
}

// NewRunCommandTool creates a new run_command tool.
func NewRunCommandTool(scopeRoot string, timeoutSec int, allowedShell []string, blocklist []string) *RunCommandTool {
	if timeoutSec <= 0 {
		timeoutSec = 30 // Default fallback
	}
	return &RunCommandTool{
		scopeRoot:    scopeRoot,
		timeoutSec:   timeoutSec,
		allowedShell: allowedShell,
		blocklist:    blocklist,
	}
}

// Name returns the name of the tool.
func (t *RunCommandTool) Name() string { return "run_command" }

// Description returns the description of the tool.
func (t *RunCommandTool) Description() string {
	return "Run a shell command in the workspace directory."
}

// Schema returns the JSON schema for the tool.
func (t *RunCommandTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"cmd":     map[string]interface{}{"type": "string", "description": "The shell command to run"},
			"cwd":     map[string]interface{}{"type": "string", "description": "Optional subdirectory within the workspace directory"},
			"timeout": map[string]interface{}{"type": "integer", "description": "Optional command execution timeout in seconds"},
		},
		Required: []string{"cmd"},
	}
}

// Permission returns the default permission level.
func (t *RunCommandTool) Permission() PermissionLevel { return Warn }

// Undoable returns whether this tool is undoable (always false for command execution).
func (t *RunCommandTool) Undoable() bool { return false }

// Snapshot is a no-op for run_command.
func (t *RunCommandTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{}, nil
}

// IsBlocklisted checks if the command string matches any blocklist patterns.
func (t *RunCommandTool) IsBlocklisted(cmd string) bool {
	for _, pattern := range t.blocklist {
		if strings.Contains(cmd, pattern) {
			return true
		}
	}
	return false
}

// Execute runs the shell command.
func (t *RunCommandTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	cmdStr, _ := args["cmd"].(string)
	if t.IsBlocklisted(cmdStr) {
		return ToolResult{
			IsError: true,
			Content: fmt.Sprintf("Command is blocklisted for safety: %s", cmdStr),
		}, nil
	}

	runCwd := t.scopeRoot
	if subCwd, ok := args["cwd"].(string); ok && subCwd != "" {
		resolvedCwd, err := PathInScope(subCwd, t.scopeRoot)
		if err != nil {
			return ToolResult{IsError: true, Content: fmt.Sprintf("cwd scope error: %s", err.Error())}, nil
		}
		runCwd = resolvedCwd
	}

	timeout := time.Duration(t.timeoutSec) * time.Second
	if tVal, ok := args["timeout"]; ok {
		if tFloat, ok := tVal.(float64); ok && tFloat > 0 {
			timeout = time.Duration(tFloat) * time.Second
		} else if tInt, ok := tVal.(int); ok && tInt > 0 {
			timeout = time.Duration(tInt) * time.Second
		}
	}

	// Determine the shell interpreter using platform abstractions.
	var p platform.Platform
	switch runtime.GOOS {
	case "darwin":
		p = platform.NewDarwinPlatform()
	case "linux":
		p = platform.NewLinuxPlatform()
	case "windows":
		p = platform.NewWindowsPlatform()
	default:
		p = platform.NewLinuxPlatform()
	}
	shellCmd, shellArgs := p.ShellArgs(cmdStr)

	// Create command with context for timeout/cancellation
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, shellCmd, shellArgs...)
	cmd.Dir = runCwd

	output, err := cmd.CombinedOutput()
	if cmdCtx.Err() == context.DeadlineExceeded {
		return ToolResult{
			IsError: true,
			Content: fmt.Sprintf("Command timed out after %v: %s\nOutput so far:\n%s", timeout, cmdStr, string(output)),
		}, nil
	}

	if err != nil {
		return ToolResult{
			IsError: true,
			Content: fmt.Sprintf("Command failed: %v\nOutput:\n%s", err, string(output)),
			Metadata: map[string]interface{}{
				"exit_code": getExitCode(err),
			},
		}, nil
	}

	return ToolResult{
		Content: string(output),
		Metadata: map[string]interface{}{
			"exit_code": 0,
		},
	}, nil
}

func getExitCode(err error) int {
	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode()
	}
	return -1
}
