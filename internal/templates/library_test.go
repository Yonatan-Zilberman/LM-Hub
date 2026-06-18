package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLibrary_BuiltinOnly(t *testing.T) {
	lib, err := NewLibrary(true, "")
	require.NoError(t, err)
	assert.Len(t, lib.Templates(), 20)

	// Verify a known template exists
	found := false
	for _, tmpl := range lib.Templates() {
		if tmpl.Name == "Debug Go error / panic" {
			found = true
			assert.Equal(t, "ask", tmpl.Mode)
			assert.Contains(t, tmpl.Tags, "go")
		}
	}
	assert.True(t, found)
}

func TestLibrary_UserDir(t *testing.T) {
	tempDir := t.TempDir()

	// Create a valid custom template file
	customYAML := `
name: "Custom Test Template"
description: "A simple template for testing"
tags: ["test", "custom"]
mode: "build"
prompt: "Testing {cursor} placeholder"
`
	err := os.WriteFile(filepath.Join(tempDir, "test.yaml"), []byte(customYAML), 0644)
	require.NoError(t, err)

	// Create an invalid YAML file that should be skipped
	invalidYAML := `
name: "Invalid Template"
description: "Missing prompt"
`
	err = os.WriteFile(filepath.Join(tempDir, "invalid.yaml"), []byte(invalidYAML), 0644)
	require.NoError(t, err)

	lib, err := NewLibrary(false, tempDir)
	require.NoError(t, err)
	assert.Len(t, lib.Templates(), 1)
	assert.Equal(t, "Custom Test Template", lib.Templates()[0].Name)
	assert.Equal(t, "build", lib.Templates()[0].Mode)
}

func TestLibrary_Search(t *testing.T) {
	lib, err := NewLibrary(true, "")
	require.NoError(t, err)

	// Test exact tag search
	results := lib.Search("cron")
	assert.NotEmpty(t, results)
	assert.Equal(t, "Write a cron job", results[0].Name)

	// Test tag case-insensitive search
	results2 := lib.Search("GO")
	assert.NotEmpty(t, results2)
	for _, tmpl := range results2 {
		assert.True(t, containsCaseInsensitive(tmpl.Tags, "go") || containsSubstring(tmpl.Name, "go") || containsSubstring(tmpl.Description, "go"))
	}

	// Empty query returns all
	all := lib.Search("")
	assert.Len(t, all, 20)
}

func TestApply(t *testing.T) {
	tmpl := Template{
		Name:   "Test",
		Prompt: "Prefix: {cursor}",
	}

	// Test cursor substitution
	out := Apply(tmpl, "User Content")
	assert.Equal(t, "Prefix: User Content", out)

	// Test fallback when cursor is missing
	tmplNoCursor := Template{
		Name:   "No Cursor",
		Prompt: "Fixed prompt",
	}
	out2 := Apply(tmplNoCursor, "Extra text")
	assert.Equal(t, "Fixed prompt\nExtra text", out2)

	// Test fallback with empty text
	out3 := Apply(tmplNoCursor, "")
	assert.Equal(t, "Fixed prompt", out3)
}

func containsCaseInsensitive(arr []string, val string) bool {
	for _, s := range arr {
		if filepath.Clean(filepath.Clean(strings.ToLower(s))) == strings.ToLower(val) {
			return true
		}
	}
	return false
}

func containsSubstring(source, target string) bool {
	return strings.Contains(strings.ToLower(source), strings.ToLower(target))
}
