package tools

import (
	"context"
	"errors"
)

// AskUserTool is an agent tool that prompts the user for input or clarification.
type AskUserTool struct {
	askCallback func(question string) (string, error)
}

// NewAskUserTool creates a new AskUserTool instance.
func NewAskUserTool(cb func(question string) (string, error)) *AskUserTool {
	return &AskUserTool{askCallback: cb}
}

// Name returns the unique identifier of the tool.
func (t *AskUserTool) Name() string {
	return "ask_user"
}

// Description returns a brief description of what the tool does.
func (t *AskUserTool) Description() string {
	return "Prompt the user for clarification or input when needed."
}

// Schema returns the JSON schema structure for the tool's input arguments.
func (t *AskUserTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"question": map[string]interface{}{
				"type":        "string",
				"description": "The question to ask the user.",
			},
		},
		Required: []string{"question"},
	}
}

// Permission returns the default permission level of the tool.
func (t *AskUserTool) Permission() PermissionLevel {
	return Safe
}

// Undoable returns whether this tool supports rolling back its changes.
func (t *AskUserTool) Undoable() bool {
	return false
}

// Execute performs the tool action with the given arguments.
func (t *AskUserTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	question, ok := args["question"].(string)
	if !ok || question == "" {
		return ToolResult{IsError: true, Content: "missing or empty 'question' argument"}, nil
	}

	if t.askCallback == nil {
		return ToolResult{IsError: true, Content: "ask_user tool callback is not configured"}, nil
	}

	answer, err := t.askCallback(question)
	if err != nil {
		return ToolResult{IsError: true, Content: err.Error()}, nil
	}

	return ToolResult{Content: answer}, nil
}

// Snapshot captures pre-execution state for undo capabilities.
func (t *AskUserTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{}, errors.New("undo not supported for ask_user")
}
