package tools

import (
	"context"
	"time"
)

// PermissionLevel defines the safety tier of a tool call.
type PermissionLevel int

const (
	// Safe tools execute immediately with no confirmation.
	Safe PermissionLevel = iota
	// Warn tools require confirmation or notice based on settings.
	Warn
	// Dangerous tools always require explicit user confirmation.
	Dangerous
)

// ToolSchema defines the JSON schema for tool arguments.
type ToolSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

// ToolResult represents the output of a tool execution.
type ToolResult struct {
	Content  string                 `json:"content"`
	IsError  bool                   `json:"is_error"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UndoRecord stores the inverse action required to roll back a tool execution.
type UndoRecord struct {
	ToolName    string                 `json:"tool_name"`
	Args        map[string]interface{} `json:"args"`
	InverseOp   string                 `json:"inverse_op"`
	InverseArgs map[string]interface{} `json:"inverse_args"`
	Snapshot    []byte                 `json:"snapshot,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Description string                 `json:"description"`
}

// Tool defines the behavior that all agent tools must implement.
type Tool interface {
	// Name returns the unique identifier of the tool.
	Name() string
	// Description returns a brief description of what the tool does.
	Description() string
	// Schema returns the JSON schema structure for the tool's input arguments.
	Schema() ToolSchema
	// Permission returns the default permission level of the tool.
	Permission() PermissionLevel
	// Undoable returns whether this tool supports rolling back its changes.
	Undoable() bool
	// Execute performs the tool action with the given arguments.
	Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error)
	// Snapshot captures pre-execution state for undo capabilities.
	Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error)
}
