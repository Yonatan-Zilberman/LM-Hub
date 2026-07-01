package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

// StatusBar renders the bottom bar showing modes, model, speed, and status.
type StatusBar struct {
	width int
}

// NewStatusBar creates a new StatusBar instance.
func NewStatusBar() *StatusBar {
	return &StatusBar{}
}

// SetWidth updates the status bar width.
func (sb *StatusBar) SetWidth(w int) {
	sb.width = w
}

// Render returns the styled string representation of the status bar.
func (sb *StatusBar) Render(mode string, model string, ramUsed float64, speed float64, isOnline bool) string {
	theme := styles.DefaultTheme

	// Mode Segment
	modeText := fmt.Sprintf(" %s ", strings.ToUpper(mode))
	modeStyle := theme.ModeBadgeStyle
	if mode == "build" {
		modeStyle = modeStyle.Copy().Background(theme.SecondaryColor)
	} else if strings.HasPrefix(mode, "plan") {
		modeStyle = modeStyle.Copy().Background(theme.AccentColor)
	}
	modeSection := modeStyle.Render(modeText)

	// Model Segment
	modelText := " No Model Loaded "
	if model != "" {
		modelText = fmt.Sprintf(" %s (%.2f GB) ", model, ramUsed)
	}
	modelSection := theme.StatusBarLeft.Render(modelText)

	// Speed Segment
	speedSection := ""
	if speed > 0 {
		speedSection = theme.StatusBarRight.Render(fmt.Sprintf(" %.1f tok/s ", speed))
	}

	// Status Indicator Segment
	statusText := " ● DISCONNECTED "
	statusStyle := theme.StatusBarInactive
	if isOnline {
		statusText = " ● ONLINE "
		statusStyle = theme.StatusBarActive
	}
	statusSection := statusStyle.Render(statusText)

	// Calculate spacing
	modeWidth := lipgloss.Width(modeSection)
	modelWidth := lipgloss.Width(modelSection)
	speedWidth := lipgloss.Width(speedSection)
	statusWidth := lipgloss.Width(statusSection)

	remainingSpace := sb.width - (modeWidth + modelWidth + speedWidth + statusWidth)
	if remainingSpace < 0 {
		remainingSpace = 0
	}

	// Dynamic middle filler
	filler := lipgloss.NewStyle().
		Background(theme.BgColor).
		Render(strings.Repeat(" ", remainingSpace))

	return lipgloss.JoinHorizontal(lipgloss.Bottom,
		modeSection,
		modelSection,
		filler,
		speedSection,
		statusSection,
	)
}
