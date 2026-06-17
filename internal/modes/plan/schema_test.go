package plan

import (
	"strings"
	"testing"
)

func TestParsePlanJSON_Valid(t *testing.T) {
	raw := `{
		"title": "Add Auth",
		"summary": "Add JWT authentication to endpoints",
		"confidence": 0.95,
		"estimated_steps": 2,
		"risks": ["break existing tests"],
		"files_affected": ["auth.go", "main.go"],
		"rollback_strategy": "git checkout main.go auth.go",
		"steps": [
			{
				"id": 1,
				"description": "Create auth.go",
				"type": "file_edit",
				"target": "auth.go",
				"reversible": true,
				"requires_confirm": false
			},
			{
				"id": 2,
				"description": "Register middleware",
				"type": "file_edit",
				"target": "main.go",
				"reversible": true,
				"requires_confirm": true
			}
		]
	}`

	p, err := ParsePlanJSON(raw)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if p.Title != "Add Auth" {
		t.Errorf("expected Title 'Add Auth', got '%s'", p.Title)
	}
	if p.Confidence != 0.95 {
		t.Errorf("expected Confidence 0.95, got %f", p.Confidence)
	}
	if len(p.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(p.Steps))
	}
}

func TestParsePlanJSON_CodeFences(t *testing.T) {
	raw := "```json\n{\n\t\"title\": \"Add Auth\",\n\t\"summary\": \"Add JWT\",\n\t\"confidence\": 0.95,\n\t\"estimated_steps\": 1,\n\t\"risks\": [],\n\t\"files_affected\": [],\n\t\"rollback_strategy\": \"\",\n\t\"steps\": [\n\t\t{\n\t\t\t\"id\": 1,\n\t\t\t\"description\": \"Do it\",\n\t\t\t\"type\": \"info\",\n\t\t\t\"target\": \"\",\n\t\t\t\"reversible\": true,\n\t\t\t\"requires_confirm\": false\n\t\t}\n\t]\n}\n```"

	p, err := ParsePlanJSON(raw)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if p.Title != "Add Auth" {
		t.Errorf("expected Title 'Add Auth', got '%s'", p.Title)
	}
}

func TestParsePlanJSON_OptionalDefaults(t *testing.T) {
	raw := `{
		"title": "Minimal Plan",
		"summary": "This plan has minimal fields",
		"steps": [
			{
				"id": 1,
				"description": "Just run this info",
				"type": "info",
				"target": "",
				"reversible": false,
				"requires_confirm": false
			}
		]
	}`

	p, err := ParsePlanJSON(raw)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if p.Confidence != 0.7 {
		t.Errorf("expected default Confidence 0.7, got %f", p.Confidence)
	}
	if p.RollbackStrategy != "No specific rollback strategy provided." {
		t.Errorf("expected default RollbackStrategy, got '%s'", p.RollbackStrategy)
	}
	if p.EstimatedSteps != 1 {
		t.Errorf("expected EstimatedSteps default to step count (1), got %d", p.EstimatedSteps)
	}
	if len(p.Risks) != 0 {
		t.Errorf("expected empty Risks slice, got %v", p.Risks)
	}
}

func TestParsePlanJSON_MissingRequired(t *testing.T) {
	// Missing title
	rawNoTitle := `{
		"summary": "No title",
		"steps": [{"id": 1, "description": "test", "type": "info", "target": "", "reversible": false, "requires_confirm": false}]
	}`
	_, err := ParsePlanJSON(rawNoTitle)
	if err == nil || !strings.Contains(err.Error(), "missing required field: title") {
		t.Errorf("expected title missing error, got: %v", err)
	}

	// Missing steps
	rawNoSteps := `{
		"title": "No steps",
		"summary": "No steps"
	}`
	_, err = ParsePlanJSON(rawNoSteps)
	if err == nil || !strings.Contains(err.Error(), "missing required field: steps") {
		t.Errorf("expected steps missing error, got: %v", err)
	}
}

func TestParsePlanJSON_InvalidSteps(t *testing.T) {
	rawInvalidStepType := `{
		"title": "Invalid Step Type",
		"summary": "Invalid Step Type",
		"steps": [{"id": 1, "description": "test", "type": "unknown_type", "target": "", "reversible": false, "requires_confirm": false}]
	}`
	_, err := ParsePlanJSON(rawInvalidStepType)
	if err == nil || !strings.Contains(err.Error(), "invalid step type") {
		t.Errorf("expected invalid step type error, got: %v", err)
	}
}
