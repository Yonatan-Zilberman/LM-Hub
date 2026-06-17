package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

// HomeView is the home/welcome screen of the application.
type HomeView struct {
	width  int
	height int
}

// NewHomeView creates a new HomeView instance.
func NewHomeView() *HomeView {
	return &HomeView{}
}

// SetSize updates layout sizes.
func (hv *HomeView) SetSize(w, h int) {
	hv.width = w
	hv.height = h
}

// View renders the welcome splash screen.
func (hv *HomeView) View() string {
	theme := styles.DefaultTheme

	title := theme.TitleStyle.Render("⚡ L M   H U B ⚡")
	subtitle := theme.SubtitleStyle.Render("Local-first AI Agent Harness for LM Studio")

	description := "LM Hub allows you to run local agents inside your workspace safely.\n" +
		"It connects to LM Studio and automates code execution with multi-mode isolation."

	// Active and Disabled mode badges
	askBadge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#000000")).Background(theme.SuccessColor).Padding(0, 1).Render(" ASK [ACTIVE] ")
	planBadge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#888888")).Background(lipgloss.Color("#333333")).Padding(0, 1).Render(" PLAN [DISABLED] ")
	buildBadge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#888888")).Background(lipgloss.Color("#333333")).Padding(0, 1).Render(" BUILD [DISABLED] ")

	modesBlock := lipgloss.JoinHorizontal(lipgloss.Center,
		askBadge, "   ", planBadge, "   ", buildBadge,
	)

	// Boxed menu/instructions
	instructions := "Press " + theme.HighlightStyle.Render("Ctrl+A") + " to open Ask mode chat\n" +
		"Press " + theme.HighlightStyle.Render("Ctrl+M") + " to open Model Browser\n" +
		"Press " + theme.HighlightStyle.Render("Ctrl+I") + " to open Inference Metrics overlay\n" +
		"Press " + theme.HighlightStyle.Render("Ctrl+L") + " to clear active chat context\n" +
		"Press " + theme.HighlightStyle.Render("Ctrl+Q") + " to quit the application"

	boxedInstructions := theme.BoxStyle.Width(hv.width - 10).Render(instructions)

	// Center content vertically and horizontally
	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		subtitle,
		"",
		description,
		"",
		modesBlock,
		"",
		boxedInstructions,
	)

	contentHeight := lipgloss.Height(content)
	topPadding := (hv.height - contentHeight) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	return strings.Repeat("\n", topPadding) + lipgloss.PlaceHorizontal(hv.width, lipgloss.Center, content)
}
