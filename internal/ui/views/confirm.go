package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/safety"
	"github.com/yonatanzilberman/lmhub/internal/ui/components"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

// ConfirmView renders a confirmation modal dialogue for user-approval gates.
type ConfirmView struct {
	width  int
	height int
	msg    safety.ConfirmMsg
}

// NewConfirmView creates a new ConfirmView overlay.
func NewConfirmView(msg safety.ConfirmMsg) *ConfirmView {
	return &ConfirmView{msg: msg}
}

// SetSize updates the dimensions.
func (cv *ConfirmView) SetSize(w, h int) {
	cv.width = w
	cv.height = h
}

// View renders the confirmation modal box.
func (cv *ConfirmView) View() string {
	theme := styles.DefaultTheme

	// Determine coloring by permission tier
	levelText := "WARNING"
	levelStyle := lipgloss.NewStyle().Foreground(theme.WarningColor).Bold(true)
	// tools.Dangerous = 2
	if int(cv.msg.Level) == 2 {
		levelText = "CRITICAL / DANGEROUS ACTION"
		levelStyle = lipgloss.NewStyle().Foreground(theme.DangerColor).Bold(true)
	}

	boxWidth := 60
	if cv.width > 0 && cv.width < 70 {
		boxWidth = cv.width - 10
	}

	var sb strings.Builder
	sb.WriteString(levelStyle.Render(fmt.Sprintf("⚠️  %s REQUIRED  ⚠️\n\n", levelText)))
	sb.WriteString(theme.NormalTextStyle.Render(fmt.Sprintf("%s\n\n", cv.msg.Description)))

	// Display arguments
	sb.WriteString(theme.HelpStyle.Render("Parameters:\n"))
	for k, v := range cv.msg.Args {
		if k == "content" {
			contentStr, _ := v.(string)
			if len(contentStr) > 60 {
				contentStr = contentStr[:60] + "... (truncated)"
			}
			sb.WriteString(theme.NormalTextStyle.Render(fmt.Sprintf("  %s: %q\n", k, contentStr)))
		} else {
			sb.WriteString(theme.NormalTextStyle.Render(fmt.Sprintf("  %s: %v\n", k, v)))
		}
	}

	if cv.msg.Diff != "" {
		sb.WriteString("\n" + theme.HelpStyle.Render("Proposed Changes:") + "\n")
		dv := components.NewDiffView(cv.msg.Diff, boxWidth-4, 10)
		sb.WriteString(dv.View() + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(theme.HighlightStyle.Render("Proceed with execution? [y] Yes  |  [n/Esc] No\n"))

	modal := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(levelStyle.GetForeground()).
		Padding(1, 2).
		Width(boxWidth).
		Render(sb.String())

	// Center the modal on screen if size is known
	if cv.width > 0 && cv.height > 0 {
		return lipgloss.Place(cv.width, cv.height, lipgloss.Center, lipgloss.Center, modal)
	}

	return modal
}
