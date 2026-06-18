// Package templates implements the prompt template library.
package templates

// Template represents a reusable prompt template.
type Template struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	Tags        []string `yaml:"tags" json:"tags"`
	Mode        string   `yaml:"mode" json:"mode"`     // "ask" | "plan" | "build"
	Prompt      string   `yaml:"prompt" json:"prompt"` // "{cursor}" represents user input insertion point
}
