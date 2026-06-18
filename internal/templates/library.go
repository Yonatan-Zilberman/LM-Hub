package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Library manages built-in and user-defined templates.
type Library struct {
	templates []Template
}

// NewLibrary initializes a template library.
func NewLibrary(builtinEnabled bool, userDir string) (*Library, error) {
	lib := &Library{
		templates: make([]Template, 0),
	}

	if builtinEnabled {
		lib.templates = append(lib.templates, BuiltinTemplates()...)
	}

	if userDir != "" {
		err := lib.LoadUserTemplates(userDir)
		if err != nil {
			// Don't crash the app if user templates directory has a bad file, but return warning/error
			return lib, fmt.Errorf("failed to load user templates: %w", err)
		}
	}

	return lib, nil
}

// Templates returns the complete list of loaded templates.
func (l *Library) Templates() []Template {
	return l.templates
}

// Get searches for a template by name (case-insensitive).
func (l *Library) Get(name string) (Template, error) {
	for _, t := range l.templates {
		if strings.EqualFold(t.Name, name) {
			return t, nil
		}
	}
	return Template{}, fmt.Errorf("template not found: %s", name)
}

// LoadUserTemplates walks the specified directory and parses all YAML files.
func (l *Library) LoadUserTemplates(dir string) error {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		// Dir doesn't exist yet, just skip without error
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("user templates path is not a directory: %s", dir)
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := strings.ToLower(file.Name())
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			filePath := filepath.Join(dir, file.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			var tmpl Template
			if err := yaml.Unmarshal(data, &tmpl); err != nil {
				// Log or skip invalid YAML
				continue
			}

			// Validate basic fields
			if tmpl.Name == "" || tmpl.Prompt == "" {
				continue
			}
			if tmpl.Mode != "ask" && tmpl.Mode != "plan" && tmpl.Mode != "build" {
				tmpl.Mode = "ask" // default fallback
			}

			l.templates = append(l.templates, tmpl)
		}
	}

	return nil
}

// Search filters the templates based on query matching name, description, or tags.
func (l *Library) Search(query string) []Template {
	trimmed := strings.TrimSpace(strings.ToLower(query))
	if trimmed == "" {
		return l.templates
	}

	var matched []Template
	for _, t := range l.templates {
		name := strings.ToLower(t.Name)
		desc := strings.ToLower(t.Description)

		tagMatch := false
		for _, tag := range t.Tags {
			if strings.Contains(strings.ToLower(tag), trimmed) {
				tagMatch = true
				break
			}
		}

		if strings.Contains(name, trimmed) || strings.Contains(desc, trimmed) || tagMatch {
			matched = append(matched, t)
		}
	}

	return matched
}

// Apply replaces the {cursor} placeholder in the template with user input.
// If {cursor} is not present, it appends the input to the end of the prompt.
func Apply(tmpl Template, userInput string) string {
	if !strings.Contains(tmpl.Prompt, "{cursor}") {
		if userInput == "" {
			return tmpl.Prompt
		}
		return tmpl.Prompt + "\n" + userInput
	}
	return strings.ReplaceAll(tmpl.Prompt, "{cursor}", userInput)
}
