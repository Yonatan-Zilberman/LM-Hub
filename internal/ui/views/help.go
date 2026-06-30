package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

// HelpView renders the overlay displaying keyboard shortcuts and slash commands.
type HelpView struct {
	width  int
	height int
}

// NewHelpView creates a new HelpView overlay.
func NewHelpView() *HelpView {
	return &HelpView{}
}

// SetSize updates layout size.
func (hv *HelpView) SetSize(w, h int) {
	hv.width = w
	hv.height = h
}

// View renders the help panel.
func (hv *HelpView) View() string {
	theme := styles.DefaultTheme

	var sb strings.Builder
	sb.WriteString(theme.TitleStyle.Render("❓ LMHUB HELP (Ctrl+H) ❓\n\n"))

	sb.WriteString(theme.HighlightStyle.Render("Keyboard Shortcuts:\n"))
	shortcuts := [][]string{
		{"Ctrl+A", "Switch to Ask (Chat) mode"},
		{"Ctrl+P", "Switch to Plan mode"},
		{"Ctrl+B", "Switch to Build mode"},
		{"Ctrl+M", "Open Model Browser"},
		{"Ctrl+G", "Open Inference Metrics overlay"},
		{"Ctrl+E", "Open Agent Memory Facts Center"},
		{"Ctrl+T", "Open Prompt Template Browser"},
		{"Ctrl+Z", "Open Undo/Rollback History (Build mode)"},
		{"Ctrl+S", "Save current session history to disk"},
		{"Ctrl+L", "Clear active chat/session history"},
		{"Ctrl+C", "Cancel streaming / interrupt agent"},
		{"Ctrl+H", "Toggle this Help overlay"},
		{"Ctrl+Q", "Quit (unloads all LM Studio models)"},
		{"Tab", "Cycle through active modes"},
	}

	for _, s := range shortcuts {
		key := lipgloss.NewStyle().Foreground(theme.AccentColor).Bold(true).Width(10).Render(s[0])
		desc := theme.NormalTextStyle.Render(s[1])
		sb.WriteString(key + " " + desc + "\n")
	}

	sb.WriteString("\n" + theme.HighlightStyle.Render("Slash Commands (Input box):\n"))
	slashCmds := [][]string{
		{"/save [name]", "Save conversation history to disk"},
		{"/load <id>", "Restore a saved conversation session by ID"},
		{"/clear", "Reset chat history"},
		{"/mem", "Toggle the Memory Facts Center"},
		{"/context", "Edit project context.md in $EDITOR"},
		{"/help", "Print available slash commands"},
	}

	for _, sc := range slashCmds {
		cmd := lipgloss.NewStyle().Foreground(theme.SecondaryColor).Bold(true).Width(15).Render(sc[0])
		desc := theme.NormalTextStyle.Render(sc[1])
		sb.WriteString(cmd + " " + desc + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(theme.HelpStyle.Render("[Esc] / [Ctrl+H] Close Help\n"))

	boxWidth := 65
	if hv.width > 0 && hv.width < 75 {
		boxWidth = hv.width - 10
	}

	modal := theme.FloatingModalStyle.Width(boxWidth).Render(sb.String())

	if hv.width > 0 && hv.height > 0 {
		return lipgloss.Place(hv.width, hv.height, lipgloss.Center, lipgloss.Center, modal)
	}

	return modal
}
