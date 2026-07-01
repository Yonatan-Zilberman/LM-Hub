package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/memory"
	"github.com/yonatanzilberman/lmhub/internal/modes/plan"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

// PlanGenerateMsg is sent when a plan is successfully generated and parsed.
type PlanGenerateMsg struct {
	Plan        *plan.Plan
	RawResponse string
}

// PlanErrorMsg is sent when plan generation fails or parsing fails twice.
type PlanErrorMsg struct {
	Err         error
	RawResponse string
}

// PlanChatView is the one-shot input screen for launching plan generation.
type PlanChatView struct {
	planMode     *plan.PlanMode
	textInput    textinput.Model
	spinner      spinner.Model
	viewport     viewport.Model
	width        int
	height       int
	isGenerating bool
	projectRoot  string
	errorMsg     string
	rawResponse  string
	epoch        int
	memManager   *memory.MemoryManager
	cancel       context.CancelFunc
}

// NewPlanChatView creates a new PlanChatView.
func NewPlanChatView(pm *plan.PlanMode, projectRoot string, mm *memory.MemoryManager) *PlanChatView {
	ti := textinput.New()
	ti.Placeholder = "Enter a task description to generate a structured implementation plan..."
	ti.Focus()
	ti.Prompt = " Plan Task > "

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.DefaultTheme.PrimaryColor)

	vp := viewport.New(80, 10)
	vp.SetContent("Provide a task to generate a step-by-step implementation plan.")

	return &PlanChatView{
		planMode:    pm,
		textInput:   ti,
		spinner:     s,
		viewport:    vp,
		projectRoot: projectRoot,
		memManager:  mm,
	}
}

// SetSize updates the view dimensions.
func (pcv *PlanChatView) SetSize(w, h int) {
	pcv.width = w
	pcv.height = h
	pcv.viewport.Width = w
	pcv.viewport.Height = h - 10
	pcv.textInput.Width = w - 16
}

// SetInputValue updates the text input value.
func (pcv *PlanChatView) SetInputValue(val string) {
	pcv.textInput.SetValue(val)
	pcv.textInput.CursorEnd()
}

// Reset clears the input and previous error status.
func (pcv *PlanChatView) Reset() {
	pcv.textInput.SetValue("")
	pcv.isGenerating = false
	pcv.errorMsg = ""
	pcv.rawResponse = ""
	pcv.viewport.SetContent("Provide a task to generate a step-by-step implementation plan.")
}

// Update handles incoming messages.
func (pcv *PlanChatView) Update(msg tea.Msg, modelID string) (tea.Cmd, error) {
	var cmds []tea.Cmd

	if pcv.isGenerating {
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyCtrlC {
			if pcv.cancel != nil {
				pcv.cancel()
				pcv.cancel = nil
			}
			pcv.isGenerating = false
			pcv.errorMsg = "Plan generation cancelled."
			pcv.viewport.SetContent("Plan generation was cancelled. Enter a new task to try again.")
			return tea.Batch(cmds...), nil
		}

		var spinCmd tea.Cmd
		pcv.spinner, spinCmd = pcv.spinner.Update(msg)
		cmds = append(cmds, spinCmd)

		switch msg := msg.(type) {
		case PlanGenerateMsg:
			pcv.isGenerating = false
			pcv.errorMsg = ""
			pcv.rawResponse = ""
			// Forward the message to App so it can switch views
			return func() tea.Msg { return msg }, nil

		case PlanErrorMsg:
			pcv.isGenerating = false
			pcv.errorMsg = msg.Err.Error()
			pcv.rawResponse = msg.RawResponse
			pcv.viewport.SetContent(fmt.Sprintf("%s Error generating plan: %v\n\nRaw output from model:\n%s", styles.SymbolError, msg.Err, msg.RawResponse))
			pcv.viewport.GotoTop()
		}
		return tea.Batch(cmds...), nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			task := strings.TrimSpace(pcv.textInput.Value())
			if task == "" {
				break
			}
			pcv.textInput.SetValue("")
			pcv.isGenerating = true
			pcv.errorMsg = ""
			pcv.rawResponse = ""
			pcv.epoch++
			cmds = append(cmds, pcv.spinner.Tick, pcv.generatePlanCmd(modelID, task, pcv.epoch))
		}
	}

	var tiCmd tea.Cmd
	pcv.textInput, tiCmd = pcv.textInput.Update(msg)
	cmds = append(cmds, tiCmd)

	var vpCmd tea.Cmd
	pcv.viewport, vpCmd = pcv.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return tea.Batch(cmds...), nil
}

// generatePlanCmd calls GeneratePlan in a background goroutine.
func (pcv *PlanChatView) generatePlanCmd(modelID, task string, targetEpoch int) tea.Cmd {
	return func() tea.Msg {
		if pcv.cancel != nil {
			pcv.cancel()
		}
		ctx, cancel := context.WithCancel(context.Background())
		pcv.cancel = cancel
		defer func() {
			if pcv.epoch == targetEpoch {
				pcv.cancel = nil
			}
			cancel()
		}()

		var memoryFacts string
		if pcv.memManager != nil {
			memoryFacts = pcv.memManager.InjectFacts()
		}
		p, raw, err := pcv.planMode.GeneratePlan(ctx, modelID, task, pcv.projectRoot, "", memoryFacts)
		if pcv.epoch != targetEpoch {
			return nil // Stale response
		}
		if err != nil {
			return PlanErrorMsg{Err: err, RawResponse: raw}
		}
		return PlanGenerateMsg{Plan: p, RawResponse: raw}
	}
}

// View renders the Plan input / error view.
func (pcv *PlanChatView) View() string {
	theme := styles.DefaultTheme

	if pcv.isGenerating {
		return lipgloss.JoinVertical(lipgloss.Center,
			"\n\n\n",
			pcv.spinner.View()+"  Generating structured plan... Please stand by.",
			"\n\n\n",
		)
	}

	vpBox := pcv.viewport.View()
	inputBox := theme.BoxStyle.Width(pcv.width - 4).Render(
		pcv.textInput.View(),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		vpBox,
		"\n",
		inputBox,
	)
}
