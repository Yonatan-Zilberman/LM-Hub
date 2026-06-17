package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/tools"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

// UndoHistoryView renders the overlay displaying build session execution steps that can be rolled back.
type UndoHistoryView struct {
	width         int
	height        int
	records       []tools.UndoRecord
	selectedIndex int
}

// NewUndoHistoryView creates a new UndoHistoryView overlay.
func NewUndoHistoryView(records []tools.UndoRecord) *UndoHistoryView {
	return &UndoHistoryView{
		records:       records,
		selectedIndex: 0,
	}
}

// SetSize updates layout size.
func (uv *UndoHistoryView) SetSize(w, h int) {
	uv.width = w
	uv.height = h
}

// SelectedIndex returns the index of the currently selected record.
func (uv *UndoHistoryView) SelectedIndex() int {
	return uv.selectedIndex
}

// MoveSelection moves the highlight up or down.
func (uv *UndoHistoryView) MoveSelection(delta int) {
	if len(uv.records) == 0 {
		return
	}
	uv.selectedIndex += delta
	if uv.selectedIndex < 0 {
		uv.selectedIndex = 0
	}
	if uv.selectedIndex >= len(uv.records) {
		uv.selectedIndex = len(uv.records) - 1
	}
}

// View renders the undo history panel.
func (uv *UndoHistoryView) View() string {
	theme := styles.DefaultTheme

	var sb strings.Builder
	sb.WriteString(theme.TitleStyle.Render("⏪ UNDO HISTORY (Ctrl+Z) ⏪\n\n"))

	if len(uv.records) == 0 {
		sb.WriteString(theme.NormalTextStyle.Render("No actions have been executed in this session yet.\n\n"))
	} else {
		for i, r := range uv.records {
			marker := "  "
			style := theme.NormalTextStyle
			if i == uv.selectedIndex {
				marker = "➔ "
				style = theme.HighlightStyle
			}

			undoableMarker := ""
			if r.InverseOp == "" {
				undoableMarker = lipgloss.NewStyle().Foreground(theme.DangerColor).Render(" [⚠ non-undoable]")
			}

			sb.WriteString(style.Render(fmt.Sprintf("%s[%d] %s%s\n", marker, len(uv.records)-i, r.Description, undoableMarker)))
		}
		sb.WriteString("\n")
		sb.WriteString(theme.HelpStyle.Render("[Enter] Rollback Selected  |  [U] Rollback All  |  [Esc] Close\n"))
	}

	boxWidth := 65
	if uv.width > 0 && uv.width < 75 {
		boxWidth = uv.width - 10
	}

	modal := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.PrimaryColor).
		Padding(1, 2).
		Width(boxWidth).
		Render(sb.String())

	if uv.width > 0 && uv.height > 0 {
		return lipgloss.Place(uv.width, uv.height, lipgloss.Center, lipgloss.Center, modal)
	}

	return modal
}
