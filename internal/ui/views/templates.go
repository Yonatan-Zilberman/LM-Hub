package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/templates"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

// TemplateApplyMsg is sent to the app when a user selects a template to apply.
type TemplateApplyMsg struct {
	Template templates.Template
}

// TemplatesView displays the Ctrl+T Prompt Templates browser.
type TemplatesView struct {
	library       *templates.Library
	width         int
	height        int
	textInput     textinput.Model
	filtered      []templates.Template
	selectedIndex int
}

// NewTemplatesView creates a new TemplatesView overlay.
func NewTemplatesView(lib *templates.Library) *TemplatesView {
	ti := textinput.New()
	ti.Placeholder = "Search prompt templates (e.g. go error, docker)..."
	ti.Prompt = " Search: "
	ti.Focus()
	ti.Width = 45

	tv := &TemplatesView{
		library:   lib,
		textInput: ti,
	}
	tv.Refresh()
	return tv
}

// SetSize updates layout sizes.
func (tv *TemplatesView) SetSize(w, h int) {
	tv.width = w
	tv.height = h
}

// Refresh updates the filtered templates list based on search query.
func (tv *TemplatesView) Refresh() {
	query := tv.textInput.Value()
	tv.filtered = tv.library.Search(query)

	if tv.selectedIndex >= len(tv.filtered) {
		tv.selectedIndex = len(tv.filtered) - 1
	}
	if tv.selectedIndex < 0 {
		tv.selectedIndex = 0
	}
}

// MoveSelection moves list cursor up or down.
func (tv *TemplatesView) MoveSelection(delta int) {
	if len(tv.filtered) == 0 {
		return
	}
	tv.selectedIndex += delta
	if tv.selectedIndex < 0 {
		tv.selectedIndex = 0
	}
	if tv.selectedIndex >= len(tv.filtered) {
		tv.selectedIndex = len(tv.filtered) - 1
	}
}

// HandleKey processes keyboard input when the template overlay is active.
func (tv *TemplatesView) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool, tea.Msg) {
	// Returns: (cmd, shouldClose, applyMsg)
	switch msg.Type {
	case tea.KeyEsc:
		return nil, true, nil
	case tea.KeyUp:
		tv.MoveSelection(-1)
		return nil, false, nil
	case tea.KeyDown:
		tv.MoveSelection(1)
		return nil, false, nil
	case tea.KeyEnter:
		if len(tv.filtered) > 0 {
			selected := tv.filtered[tv.selectedIndex]
			return nil, true, TemplateApplyMsg{Template: selected}
		}
		return nil, false, nil
	}

	var cmd tea.Cmd
	tv.textInput, cmd = tv.textInput.Update(msg)
	tv.Refresh()
	return cmd, false, nil
}

// View renders the prompt templates browser overlay.
func (tv *TemplatesView) View() string {
	theme := styles.DefaultTheme

	var sb strings.Builder
	sb.WriteString(theme.TitleStyle.Render("📋 Prompt Templates Browser"))
	sb.WriteString("\n")
	sb.WriteString(theme.HelpStyle.Render("Select a template to pre-fill input and switch operational mode"))
	sb.WriteString("\n\n")

	sb.WriteString(tv.textInput.View())
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", tv.width-12))
	sb.WriteString("\n\n")

	if len(tv.filtered) == 0 {
		sb.WriteString("  No matching templates found.\n\n")
	} else {
		// Limit display to fit height nicely
		maxDisplay := 6
		start := 0
		if tv.selectedIndex >= maxDisplay {
			start = tv.selectedIndex - maxDisplay + 1
		}

		end := start + maxDisplay
		if end > len(tv.filtered) {
			end = len(tv.filtered)
		}

		for i := start; i < end; i++ {
			t := tv.filtered[i]
			marker := "  "
			style := theme.NormalTextStyle
			if i == tv.selectedIndex {
				marker = "➔ "
				style = theme.HighlightStyle
			}

			modeBadge := fmt.Sprintf("[%s]", t.Mode)
			modeStyle := lipgloss.NewStyle().Foreground(theme.SuccessColor).Bold(true)
			switch t.Mode {
			case "plan":
				modeStyle = lipgloss.NewStyle().Foreground(theme.SecondaryColor).Bold(true)
			case "build":
				modeStyle = lipgloss.NewStyle().Foreground(theme.PrimaryColor).Bold(true)
			}

			sb.WriteString(style.Render(fmt.Sprintf("%s%s %s\n", marker, modeStyle.Render(modeBadge), t.Name)))
			sb.WriteString(theme.HelpStyle.Render(fmt.Sprintf("    %s\n", t.Description)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(theme.HelpStyle.Render("[↑/↓] Navigate  |  [Enter] Apply template  |  [Esc] Close\n"))

	boxWidth := 75
	if tv.width > 0 && tv.width < 85 {
		boxWidth = tv.width - 10
	}

	modal := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.PrimaryColor).
		Padding(1, 2).
		Width(boxWidth).
		Render(sb.String())

	if tv.width > 0 && tv.height > 0 {
		return lipgloss.Place(tv.width, tv.height, lipgloss.Center, lipgloss.Center, modal)
	}

	return modal
}
