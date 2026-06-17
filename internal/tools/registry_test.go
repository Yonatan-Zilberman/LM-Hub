package tools

import (
	"context"
	"testing"
)

type mockTool struct {
	name        string
	description string
	schema      ToolSchema
	permission  PermissionLevel
	undoable    bool
	executeFunc func(ctx context.Context, args map[string]interface{}) (ToolResult, error)
}

func (m *mockTool) Name() string { return m.name }
func (m *mockTool) Description() string { return m.description }
func (m *mockTool) Schema() ToolSchema { return m.schema }
func (m *mockTool) Permission() PermissionLevel { return m.permission }
func (m *mockTool) Undoable() bool { return m.undoable }
func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, args)
	}
	return ToolResult{Content: "success"}, nil
}
func (m *mockTool) Snapshot(ctx context.Context, args map[string]interface{}) (UndoRecord, error) {
	return UndoRecord{ToolName: m.name}, nil
}

func TestRegistry(t *testing.T) {
	r := NewRegistry("/tmp/mock-scope")

	tool1 := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		schema: ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"required_arg": map[string]interface{}{"type": "string"},
				"number_arg":   map[string]interface{}{"type": "number"},
			},
			Required: []string{"required_arg"},
		},
	}

	r.Register(tool1)

	// Test duplicate registration panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on duplicate registration")
		}
	}()
	r.Register(tool1)
}

func TestRegistry_GetAndList(t *testing.T) {
	r := NewRegistry("/tmp/mock-scope")
	t1 := &mockTool{name: "b_tool"}
	t2 := &mockTool{name: "a_tool"}
	r.Register(t1)
	r.Register(t2)

	// Test Get
	got, ok := r.Get("b_tool")
	if !ok || got.Name() != "b_tool" {
		t.Errorf("Get failed")
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Errorf("Get non-existent tool succeeded")
	}

	// Test List ordering
	list := r.List()
	if len(list) != 2 || list[0].Name() != "a_tool" || list[1].Name() != "b_tool" {
		t.Errorf("List or ordering failed: %v", list)
	}
}

func TestRegistry_ValidateArgs(t *testing.T) {
	r := NewRegistry("/tmp/mock-scope")
	tool := &mockTool{
		name: "validator_tool",
		schema: ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"str":  map[string]interface{}{"type": "string"},
				"num":  map[string]interface{}{"type": "number"},
				"bool": map[string]interface{}{"type": "boolean"},
			},
			Required: []string{"str"},
		},
	}
	r.Register(tool)

	// Valid args
	err := r.ValidateArgs(tool, map[string]interface{}{
		"str":  "hello",
		"num":  42.0,
		"bool": true,
	})
	if err != nil {
		t.Errorf("expected valid args: %v", err)
	}

	// Missing required
	err = r.ValidateArgs(tool, map[string]interface{}{
		"num": 42.0,
	})
	if err == nil {
		t.Errorf("expected error for missing required parameter")
	}

	// Invalid type
	err = r.ValidateArgs(tool, map[string]interface{}{
		"str": 123,
	})
	if err == nil {
		t.Errorf("expected error for invalid type")
	}
}

func TestRegistry_Execute(t *testing.T) {
	r := NewRegistry("/tmp/mock-scope")
	executed := false
	tool := &mockTool{
		name: "exec_tool",
		executeFunc: func(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
			executed = true
			return ToolResult{Content: "done"}, nil
		},
	}
	r.Register(tool)

	res, err := r.Execute(context.Background(), "exec_tool", nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if res.Content != "done" || !executed {
		t.Errorf("execution failed")
	}

	// Execute non-existent
	_, err = r.Execute(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Errorf("expected error executing non-existent tool")
	}
}

func TestRegistry_SchemaJSON(t *testing.T) {
	r := NewRegistry("/tmp/mock-scope")
	tool := &mockTool{
		name:        "schema_tool",
		description: "Desc",
		schema: ToolSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"param": map[string]interface{}{"type": "string"},
			},
			Required: []string{"param"},
		},
	}
	r.Register(tool)

	schema, err := r.SchemaJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema == "" {
		t.Errorf("expected non-empty schema JSON")
	}
}
