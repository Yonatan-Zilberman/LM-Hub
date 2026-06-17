package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

// NewSpinner creates a standard styled spinner for LM Hub thinking states.
func NewSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.DefaultTheme.AccentColor)
	return s
}
