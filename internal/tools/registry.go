package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
)

// Registry manages the registration, retrieval, and execution of tools.
type Registry struct {
	tools     map[string]Tool
	scopeRoot string
}

// NewRegistry creates a new tool registry.
func NewRegistry(scopeRoot string) *Registry {
	return &Registry{
		tools:     make(map[string]Tool),
		scopeRoot: scopeRoot,
	}
}

// Register registers a tool with the registry.
// It panics if a tool with the same name is already registered.
func (r *Registry) Register(t Tool) {
	if _, exists := r.tools[t.Name()]; exists {
		panic(fmt.Sprintf("tool already registered: %s", t.Name()))
	}
	r.tools[t.Name()] = t
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, exists := r.tools[name]
	return t, exists
}

// List returns all registered tools, sorted alphabetically by name.
func (r *Registry) List() []Tool {
	list := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name() < list[j].Name()
	})
	return list
}

// ScopeRoot returns the working directory scope root.
func (r *Registry) ScopeRoot() string {
	return r.scopeRoot
}

// SetScopeRoot updates the working directory scope root.
func (r *Registry) SetScopeRoot(path string) {
	r.scopeRoot = path
}

// SchemaJSON returns all registered tool schemas as a JSON array string.
func (r *Registry) SchemaJSON() (string, error) {
	type ToolDef struct {
		Name        string     `json:"name"`
		Description string     `json:"description"`
		Schema      ToolSchema `json:"schema"`
	}

	tools := r.List()
	defs := make([]ToolDef, len(tools))
	for i, t := range tools {
		defs[i] = ToolDef{
			Name:        t.Name(),
			Description: t.Description(),
			Schema:      t.Schema(),
		}
	}

	bytes, err := json.MarshalIndent(defs, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal schemas: %w", err)
	}
	return string(bytes), nil
}

// ValidateArgs checks if the arguments conform to the tool's schema.
func (r *Registry) ValidateArgs(t Tool, args map[string]interface{}) error {
	schema := t.Schema()
	for _, req := range schema.Required {
		if _, exists := args[req]; !exists {
			return fmt.Errorf("missing required parameter: %s", req)
		}
	}

	for key, val := range args {
		prop, ok := schema.Properties[key]
		if !ok {
			// Extra parameter, can be ignored or warned, but let's allow it for robustness
			continue
		}

		propMap, ok := prop.(map[string]interface{})
		if !ok {
			continue
		}

		expectedType, ok := propMap["type"].(string)
		if !ok {
			continue
		}

		switch expectedType {
		case "string":
			if _, ok := val.(string); !ok {
				return fmt.Errorf("parameter %s must be a string", key)
			}
		case "boolean":
			if _, ok := val.(bool); !ok {
				return fmt.Errorf("parameter %s must be a boolean", key)
			}
		case "number":
			switch v := val.(type) {
			case float64, float32, int, int64, int32:
				// Valid number
			default:
				_ = v // prevent unused var warning
				return fmt.Errorf("parameter %s must be a number", key)
			}
		case "integer":
			switch v := val.(type) {
			case int, int64, int32:
				// Valid integer
			case float64:
				// JSON numbers are parsed as float64, check if it's a whole number
				if v != float64(int(v)) {
					return fmt.Errorf("parameter %s must be an integer", key)
				}
			default:
				return fmt.Errorf("parameter %s must be an integer", key)
			}
		case "array":
			if _, ok := val.([]interface{}); !ok {
				return fmt.Errorf("parameter %s must be an array", key)
			}
		}
	}

	return nil
}

// Execute validates and runs a tool by name.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]interface{}) (ToolResult, error) {
	t, exists := r.Get(name)
	if !exists {
		return ToolResult{IsError: true}, fmt.Errorf("tool not found: %s", name)
	}

	if err := r.ValidateArgs(t, args); err != nil {
		return ToolResult{IsError: true, Content: err.Error()}, fmt.Errorf("validate args: %w", err)
	}

	return t.Execute(ctx, args)
}
