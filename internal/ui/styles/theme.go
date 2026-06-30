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

	PanelHeaderStyle    lipgloss.Style
	KeybindBarStyle     lipgloss.Style
	DimmedOverlayStyle  lipgloss.Style
	FloatingModalStyle  lipgloss.Style
}

// DefaultTheme returns the standard dark theme for LM Hub.
var DefaultTheme Theme

func init() {
	// Sleek dark palette with muted grays and single cyan accent
	DefaultTheme = Theme{
		PrimaryColor:   lipgloss.Color("#888888"), // Muted Gray
		SecondaryColor: lipgloss.Color("#555555"), // Darker Muted Gray
		AccentColor:    lipgloss.Color("#00d2ff"), // Bright Cyan Accent
		BgColor:        lipgloss.Color("#161616"), // Very Dark Charcoal BG
		FgColor:        lipgloss.Color("#cccccc"), // Soft Gray Fg
		
		SuccessColor: lipgloss.Color("#22aa55"), // Green
		WarningColor: lipgloss.Color("#ddaa22"), // Orange/Yellow
		DangerColor:  lipgloss.Color("#cc3333"), // Red
		
		BorderColor: lipgloss.Color("#333333"), // Dark gray borders
	}

	theme := &DefaultTheme

	// Titles & Headers
	theme.TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.AccentColor).
		Padding(0, 1)

	theme.SubtitleStyle = lipgloss.NewStyle().
		Foreground(theme.PrimaryColor).
		Italic(true)

	theme.NormalTextStyle = lipgloss.NewStyle().
		Foreground(theme.FgColor)

	theme.HighlightStyle = lipgloss.NewStyle().
		Foreground(theme.AccentColor).
		Bold(true)

	theme.ModeBadgeStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#000000")).
		Background(theme.AccentColor).
		Padding(0, 1)

	// Status Bar
	theme.StatusBarLeft = lipgloss.NewStyle().
		Foreground(theme.FgColor).
		Background(lipgloss.Color("#242424")).
		Padding(0, 1)

	theme.StatusBarRight = lipgloss.NewStyle().
		Foreground(theme.FgColor).
		Background(lipgloss.Color("#333333")).
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

	// Box / Containers (Sharp borders as requested)
	theme.BoxStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.BorderColor).
		Padding(1)

	theme.ActiveBoxStyle = theme.BoxStyle.Copy().
		BorderForeground(theme.AccentColor)

	// Help Text
	theme.HelpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Italic(true)

	theme.PanelHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.AccentColor).
		Background(lipgloss.Color("#242424")).
		Padding(0, 1)

	theme.KeybindBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#777777")).
		Background(theme.BgColor).
		Padding(0, 1)

	theme.DimmedOverlayStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#444444"))

	theme.FloatingModalStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.AccentColor).
		Padding(1, 2)
}
