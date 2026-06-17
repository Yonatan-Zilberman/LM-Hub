package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

// ContextBar renders the live token fill and budget breakdown of the context window.
type ContextBar struct {
	width int
}

// NewContextBar creates a new ContextBar instance.
func NewContextBar() *ContextBar {
	return &ContextBar{}
}

// SetWidth updates the component width.
func (cb *ContextBar) SetWidth(w int) {
	cb.width = w
}

// Render returns the visual context bar string.
func (cb *ContextBar) Render(used, total int, sys, hist, mem, rag int) string {
	if total == 0 {
		total = 128000 // Safe default if unset
	}

	theme := styles.DefaultTheme
	pct := float64(used) / float64(total)
	pctVal := int(pct * 100)

	// Determine bar color based on thresholds
	barColor := theme.SuccessColor // Green
	if pctVal >= 90 {
		barColor = theme.DangerColor // Red
	} else if pctVal >= 85 {
		barColor = lipgloss.Color("#ffb86c") // Orange
	} else if pctVal >= 70 {
		barColor = theme.WarningColor // Yellow
	}

	// Visual progress bar
	barLength := cb.width - 25 // Reserved space for text
	if barLength < 10 {
		barLength = 10
	}

	filledLength := int(pct * float64(barLength))
	if filledLength > barLength {
		filledLength = barLength
	}
	emptyLength := barLength - filledLength

	filledStyle := lipgloss.NewStyle().Foreground(barColor)
	emptyStyle := lipgloss.NewStyle().Foreground(theme.BorderColor)

	progressBar := filledStyle.Render(strings.Repeat("█", filledLength)) +
		emptyStyle.Render(strings.Repeat("░", emptyLength))

	// Header line
	headerText := fmt.Sprintf("Context [%s] %d / %d (%d%%)", 
		progressBar, used, total, pctVal)

	// Breakdown labels line
	sysStyle := lipgloss.NewStyle().Foreground(theme.PrimaryColor)
	histStyle := lipgloss.NewStyle().Foreground(theme.SecondaryColor)
	memStyle := lipgloss.NewStyle().Foreground(theme.AccentColor)
	ragStyle := lipgloss.NewStyle().Foreground(theme.SuccessColor)

	breakdownText := fmt.Sprintf("          %s sys: %d  |  %s hist: %d  |  %s mem: %d  |  %s rag: %d",
		sysStyle.Render("█"), sys,
		histStyle.Render("█"), hist,
		memStyle.Render("█"), mem,
		ragStyle.Render("█"), rag,
	)

	return headerText + "\n" + breakdownText
}
