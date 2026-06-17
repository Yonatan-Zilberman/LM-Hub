package views

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/modes/plan"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

// PlanApproveMsg is sent when the user approves the generated plan.
type PlanApproveMsg struct {
	Filename string
}

// PlanRejectMsg is sent when the user rejects the generated plan.
type PlanRejectMsg struct{}

// PlanView is the interactive plan review view.
type PlanView struct {
	plan        *plan.Plan
	viewport    viewport.Model
	width       int
	height      int
	projectRoot string
	statusMsg   string
}

// NewPlanView creates a new PlanView.
func NewPlanView(projectRoot string) *PlanView {
	vp := viewport.New(80, 20)
	vp.SetContent("No plan loaded.")
	return &PlanView{
		viewport:    vp,
		projectRoot: projectRoot,
	}
}

// SetPlan updates the view with the loaded Plan and refreshes content.
func (pv *PlanView) SetPlan(p *plan.Plan) {
	pv.plan = p
	pv.statusMsg = ""
	pv.refreshContent()
}

// SetSize updates the view sizes.
func (pv *PlanView) SetSize(w, h int) {
	pv.width = w
	pv.height = h
	pv.viewport.Width = w
	pv.viewport.Height = h - 8
}

// refreshContent formats and updates the viewport contents.
func (pv *PlanView) refreshContent() {
	if pv.plan == nil {
		pv.viewport.SetContent("No plan loaded.")
		return
	}

	var sb strings.Builder
	theme := styles.DefaultTheme

	// Confidence color styling
	confStyle := lipgloss.NewStyle().Bold(true)
	if pv.plan.Confidence >= 0.8 {
		confStyle = confStyle.Foreground(theme.SuccessColor)
	} else if pv.plan.Confidence >= 0.6 {
		confStyle = confStyle.Foreground(theme.WarningColor)
	} else {
		confStyle = confStyle.Foreground(theme.DangerColor)
	}

	sb.WriteString(theme.TitleStyle.Render("Plan: " + pv.plan.Title))
	sb.WriteString("\n")
	sb.WriteString(theme.NormalTextStyle.Render(fmt.Sprintf("Confidence: %s (%.0f%%)", confStyle.Render(fmt.Sprintf("%.2f", pv.plan.Confidence)), pv.plan.Confidence*100)))
	sb.WriteString("\n")
	sb.WriteString(theme.NormalTextStyle.Render(fmt.Sprintf("Summary: %s", pv.plan.Summary)))
	sb.WriteString("\n\n")

	sb.WriteString(theme.HighlightStyle.Render("Steps:"))
	sb.WriteString("\n")
	for _, s := range pv.plan.Steps {
		icon := "ℹ"
		switch s.Type {
		case "file_edit":
			icon = "✎"
		case "shell":
			icon = "⌘"
		case "git":
			icon = "🌿"
		case "docker":
			icon = "🐳"
		}

		revStr := "[reversible]"
		if !s.Reversible {
			revStr = lipgloss.NewStyle().Foreground(theme.WarningColor).Render("⚠️ [non-reversible]")
		}

		confirmStr := ""
		if s.RequiresConfirm {
			confirmStr = lipgloss.NewStyle().Foreground(theme.SecondaryColor).Render(" (Requires Confirm)")
		}

		sb.WriteString(fmt.Sprintf("  [%d] %s  %s  %s  %s%s\n", s.ID, icon, lipgloss.NewStyle().Bold(true).Render(s.Type), s.Target, revStr, confirmStr))
		sb.WriteString(fmt.Sprintf("      Description: %s\n\n", s.Description))
	}

	if len(pv.plan.Risks) > 0 {
		sb.WriteString(theme.HighlightStyle.Foreground(theme.DangerColor).Render("Risks:"))
		sb.WriteString("\n")
		for _, r := range pv.plan.Risks {
			sb.WriteString(fmt.Sprintf("  • %s\n", r))
		}
		sb.WriteString("\n")
	}

	if len(pv.plan.FilesAffected) > 0 {
		sb.WriteString(theme.HighlightStyle.Render("Files Affected:"))
		sb.WriteString("\n")
		sb.WriteString("  ")
		sb.WriteString(strings.Join(pv.plan.FilesAffected, ", "))
		sb.WriteString("\n\n")
	}

	sb.WriteString(theme.HighlightStyle.Render("Rollback Strategy:"))
	sb.WriteString("\n")
	sb.WriteString("  ")
	sb.WriteString(pv.plan.RollbackStrategy)
	sb.WriteString("\n")

	pv.viewport.SetContent(sb.String())
}

// Update handles interactive keyboard review shortcuts.
func (pv *PlanView) Update(msg tea.Msg) (tea.Cmd, error) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			filename, err := pv.savePlan()
			if err != nil {
				pv.statusMsg = fmt.Sprintf("Failed to save plan: %v", err)
				break
			}
			return func() tea.Msg {
				return PlanApproveMsg{Filename: filename}
			}, nil

		case "n", "N", "esc":
			return func() tea.Msg {
				return PlanRejectMsg{}
			}, nil

		case "s", "S":
			filename, err := pv.savePlan()
			if err != nil {
				pv.statusMsg = fmt.Sprintf("Failed to save plan: %v", err)
			} else {
				pv.statusMsg = fmt.Sprintf("Plan saved to %s", filename)
			}

		case "e", "E":
			pv.statusMsg = "Inline editing is not yet implemented."
		}
	}

	var vpCmd tea.Cmd
	pv.viewport, vpCmd = pv.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return tea.Batch(cmds...), nil
}

// savePlan serializes and writes the plan to .lmhub/plan-{timestamp}.json.
func (pv *PlanView) savePlan() (string, error) {
	dir := filepath.Join(pv.projectRoot, ".lmhub")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	filename := fmt.Sprintf("plan-%d.json", time.Now().Unix())
	path := filepath.Join(dir, filename)

	data, err := json.MarshalIndent(pv.plan, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal plan: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write plan file: %w", err)
	}

	return filename, nil
}

// View renders the Plan view.
func (pv *PlanView) View() string {
	if pv.plan == nil {
		return "No plan loaded."
	}

	theme := styles.DefaultTheme

	// Viewport display
	vpBox := pv.viewport.View()

	// Bottom action bar
	actions := " [Y/Enter] Approve & Save   [N/Esc] Reject   [S] Save Only   [E] Edit Inline"
	actionsBar := theme.HighlightStyle.Render(actions)

	statusLine := ""
	if pv.statusMsg != "" {
		statusLine = theme.HelpStyle.Render("Status: "+pv.statusMsg) + "\n"
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		vpBox,
		"\n",
		statusLine,
		actionsBar,
	)
}
