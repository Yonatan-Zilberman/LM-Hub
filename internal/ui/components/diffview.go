// Package components contains reusable TUI components.
package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

// DiffView is a component that renders colored unified diffs in a scrollable viewport.
type DiffView struct {
	viewport viewport.Model
	width    int
	height   int
	rawDiff  string
}

// NewDiffView creates a new DiffView instance.
func NewDiffView(diffText string, width, height int) DiffView {
	vp := viewport.New(width, height)
	dv := DiffView{
		viewport: vp,
		width:    width,
		height:   height,
		rawDiff:  diffText,
	}
	dv.SetContent(diffText)
	return dv
}

// SetContent updates the diff text content and applies Lipgloss color styling.
func (dv *DiffView) SetContent(diffText string) {
	dv.rawDiff = diffText
	theme := styles.DefaultTheme

	// Style lines:
	// + -> green
	// - -> red
	// @@ -> cyan
	// file headers -> bold/purple
	lines := strings.Split(diffText, "\n")
	var styledLines []string

	plusStyle := lipgloss.NewStyle().Foreground(theme.SuccessColor)
	minusStyle := lipgloss.NewStyle().Foreground(theme.DangerColor)
	hunkStyle := lipgloss.NewStyle().Foreground(theme.AccentColor)
	headerStyle := lipgloss.NewStyle().Foreground(theme.PrimaryColor).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(theme.FgColor)

	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			styledLines = append(styledLines, plusStyle.Render(line))
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			styledLines = append(styledLines, minusStyle.Render(line))
		} else if strings.HasPrefix(line, "@@") {
			styledLines = append(styledLines, hunkStyle.Render(line))
		} else if strings.HasPrefix(line, "diff ") || strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
			styledLines = append(styledLines, headerStyle.Render(line))
		} else {
			styledLines = append(styledLines, normalStyle.Render(line))
		}
	}

	dv.viewport.SetContent(strings.Join(styledLines, "\n"))
}

// SetSize updates the viewport size.
func (dv *DiffView) SetSize(w, h int) {
	dv.width = w
	dv.height = h
	dv.viewport.Width = w
	dv.viewport.Height = h
}

// Update handles scroll events.
func (dv *DiffView) Update(msg tea.Msg) (DiffView, tea.Cmd) {
	var cmd tea.Cmd
	dv.viewport, cmd = dv.viewport.Update(msg)
	return *dv, cmd
}

// View renders the styled viewport.
func (dv *DiffView) View() string {
	return dv.viewport.View()
}
