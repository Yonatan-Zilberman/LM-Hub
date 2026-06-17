package styles

import "github.com/charmbracelet/lipgloss"

// Theme contains colors and styles for the TUI components.
type Theme struct {
	PrimaryColor   lipgloss.Color
	SecondaryColor lipgloss.Color
	AccentColor    lipgloss.Color
	BgColor        lipgloss.Color
	FgColor        lipgloss.Color
	
	SuccessColor lipgloss.Color
	WarningColor lipgloss.Color
	DangerColor  lipgloss.Color
	
	BorderColor lipgloss.Color

	// Common reusable styles
	TitleStyle        lipgloss.Style
	SubtitleStyle     lipgloss.Style
	NormalTextStyle   lipgloss.Style
	HighlightStyle    lipgloss.Style
	ModeBadgeStyle    lipgloss.Style
	StatusBarLeft     lipgloss.Style
	StatusBarRight    lipgloss.Style
	StatusBarActive   lipgloss.Style
	StatusBarInactive lipgloss.Style
	
	BoxStyle          lipgloss.Style
	ActiveBoxStyle    lipgloss.Style
	HelpStyle         lipgloss.Style
}

// DefaultTheme returns the standard dark theme for LM Hub.
var DefaultTheme Theme

func init() {
	// Sleek Dracula/Tokyonight-inspired dark palette
	DefaultTheme = Theme{
		PrimaryColor:   lipgloss.Color("#bd93f9"), // Purple
		SecondaryColor: lipgloss.Color("#ff79c6"), // Pink
		AccentColor:    lipgloss.Color("#8be9fd"), // Cyan
		BgColor:        lipgloss.Color("#282a36"), // Dark BG
		FgColor:        lipgloss.Color("#f8f8f2"), // Warm White
		
		SuccessColor: lipgloss.Color("#50fa7b"), // Green
		WarningColor: lipgloss.Color("#f1fa8c"), // Yellow
		DangerColor:  lipgloss.Color("#ff5555"), // Red
		
		BorderColor: lipgloss.Color("#44475a"), // Muted Purple-Gray
	}

	theme := &DefaultTheme

	// Titles & Headers
	theme.TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.PrimaryColor).
		Padding(0, 1)

	theme.SubtitleStyle = lipgloss.NewStyle().
		Foreground(theme.SecondaryColor).
		Italic(true)

	theme.NormalTextStyle = lipgloss.NewStyle().
		Foreground(theme.FgColor)

	theme.HighlightStyle = lipgloss.NewStyle().
		Foreground(theme.AccentColor).
		Bold(true)

	theme.ModeBadgeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#000000")).
		Background(theme.PrimaryColor).
		Padding(0, 1)

	// Status Bar
	theme.StatusBarLeft = lipgloss.NewStyle().
		Foreground(theme.FgColor).
		Background(lipgloss.Color("#343746")).
		Padding(0, 1)

	theme.StatusBarRight = lipgloss.NewStyle().
		Foreground(theme.FgColor).
		Background(lipgloss.Color("#44475a")).
		Padding(0, 1)

	theme.StatusBarActive = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(theme.SuccessColor).
		Bold(true).
		Padding(0, 1)

	theme.StatusBarInactive = lipgloss.NewStyle().
		Foreground(theme.FgColor).
		Background(theme.DangerColor).
		Padding(0, 1)

	// Box / Containers
	theme.BoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderColor).
		Padding(1)

	theme.ActiveBoxStyle = theme.BoxStyle.Copy().
		BorderForeground(theme.PrimaryColor)

	// Help Text
	theme.HelpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6272a4")).
		Italic(true)
}
