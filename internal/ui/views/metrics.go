package views

import (
	"fmt"
	"strings"

	"github.com/yonatanzilberman/lmhub/internal/modelmanager"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

// MetricsView displays the Ctrl+I Inference Metrics overlay panel.
type MetricsView struct {
	metrics *modelmanager.Metrics
	width   int
	height  int
}

// NewMetricsView creates a new MetricsView instance.
func NewMetricsView(m *modelmanager.Metrics) *MetricsView {
	return &MetricsView{
		metrics: m,
	}
}

// SetSize updates layout sizes.
func (mv *MetricsView) SetSize(w, h int) {
	mv.width = w
	mv.height = h
}

// View renders the metrics overlay panel.
func (mv *MetricsView) View() string {
	theme := styles.DefaultTheme
	m := mv.metrics.Get()

	var sb strings.Builder
	sb.WriteString(theme.TitleStyle.Render("📊 Inference Metrics"))
	sb.WriteString("\n")
	sb.WriteString(theme.HelpStyle.Render("Press [Ctrl+I] to close"))
	sb.WriteString("\n\n")

	if m.ModelID == "" {
		sb.WriteString("No model currently loaded.\nMetrics will be populated when you start a conversation.")
		return theme.BoxStyle.Width(mv.width - 6).Render(sb.String())
	}

	totalTimeSecs := 0.0
	if m.TokensPerSecond > 0 {
		totalTimeSecs = float64(m.TotalTokens) / m.TokensPerSecond
	}

	modelInfo := fmt.Sprintf(
		"Model ID:     %s\n"+
			"RAM Size:     %.2f GB\n",
		theme.HighlightStyle.Render(m.ModelID), m.RAMUsedGB,
	)
	sb.WriteString(modelInfo)
	sb.WriteString("\n")

	sb.WriteString(theme.SubtitleStyle.Render("Last Chat Stream Completion\n"))
	sb.WriteString(strings.Repeat("─", mv.width-12))
	sb.WriteString("\n")
	completionInfo := fmt.Sprintf(
		"Time to first token (TTFT): %d ms\n"+
			"Generation speed:          %.1f tok/s\n"+
			"Tokens generated:          %d\n"+
			"Approx. stream duration:   %.1fs\n",
		m.TTFTMs, m.TokensPerSecond, m.TotalTokens, totalTimeSecs,
	)
	sb.WriteString(completionInfo)
	sb.WriteString("\n")

	sb.WriteString(theme.SubtitleStyle.Render("Context Window Allocation\n"))
	sb.WriteString(strings.Repeat("─", mv.width-12))
	sb.WriteString("\n")
	
	pct := 0.0
	if m.ContextLimit > 0 {
		pct = (float64(m.TokensUsed) / float64(m.ContextLimit)) * 100
	}

	contextInfo := fmt.Sprintf(
		"Capacity:                  %d tokens\n"+
			"Used (Estimate):           %d tokens (%.1f%%)\n"+
			"Remaining:                 %d tokens\n",
		m.ContextLimit, m.TokensUsed, pct, m.ContextLimit-m.TokensUsed,
	)
	sb.WriteString(contextInfo)

	return theme.BoxStyle.Width(mv.width - 6).Render(sb.String())
}
