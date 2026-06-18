package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/modes/build"
	"github.com/yonatanzilberman/lmhub/internal/ui/components"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

// BuildChannelReaderMsg bridges background agent updates into the Bubbletea event loop.
type BuildChannelReaderMsg struct {
	Stream chan build.AgentStepMsg
}

// NextStepCmd waits for the next step update from the background thread and returns it.
func NextStepCmd(stream chan build.AgentStepMsg) tea.Cmd {
	return func() tea.Msg {
		step, ok := <-stream
		if !ok {
			return build.AgentStepMsg{Done: true, StepType: "finished", Content: "Stream closed"}
		}
		return step
	}
}

// BuildView manages the interactive agent execution layout for Build Mode.
type BuildView struct {
	buildMode   *build.BuildMode
	viewport    viewport.Model
	diffView    components.DiffView
	showDiff    bool
	textInput   textinput.Model
	renderer    *components.MarkdownRenderer
	width       int
	height      int
	isStreaming bool
	stream      chan build.AgentStepMsg
	StatusLog   string

	// Current streaming content
	accumulatedThoughts strings.Builder
}

// NewBuildView creates a new BuildView view.
func NewBuildView(bm *build.BuildMode) (*BuildView, error) {
	ti := textinput.New()
	ti.Placeholder = "Enter a task description (e.g. 'implement test router') and press Enter..."
	ti.Focus()
	ti.Prompt = " build > "

	mr, err := components.NewMarkdownRenderer()
	if err != nil {
		return nil, err
	}

	vp := viewport.New(40, 20)
	vp.SetContent("Build mode ready. Describe a task to execute.")

	dv := components.NewDiffView("", 40, 20)

	return &BuildView{
		buildMode: bm,
		textInput: ti,
		viewport:  vp,
		renderer:  mr,
		diffView:  dv,
		showDiff:  false,
	}, nil
}

// SetSize updates widths and heights.
func (bv *BuildView) SetSize(w, h int) {
	bv.width = w
	bv.height = h

	leftWidth := (w / 2) - 2
	bv.viewport.Width = leftWidth
	bv.viewport.Height = h - 10
	bv.textInput.Width = w - 10

	rightWidth := w - leftWidth - 4
	bv.diffView.SetSize(rightWidth, h - 10)
}

// SetInputValue updates the text input value.
func (bv *BuildView) SetInputValue(val string) {
	bv.textInput.SetValue(val)
	bv.textInput.CursorEnd()
}

// Reset clears active logs and session metrics.
func (bv *BuildView) Reset() {
	bv.buildMode.Reset()
	bv.viewport.SetContent("Build mode cleared. Describe a task to execute.")
	bv.viewport.GotoTop()
	bv.StatusLog = ""
	bv.accumulatedThoughts.Reset()
	bv.isStreaming = false
	bv.showDiff = false
	bv.diffView.SetContent("")
}

// SetDiff updates the diff viewer content and shows it.
func (bv *BuildView) SetDiff(diffText string) {
	bv.diffView.SetContent(diffText)
	bv.showDiff = true
}

// Update handles UI and agent updates.
func (bv *BuildView) Update(msg tea.Msg, modelID string, gitStatus, projectContext, memoryFacts string) (tea.Cmd, error) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlD {
			bv.showDiff = !bv.showDiff
			break
		}

		if bv.isStreaming {
			if msg.Type == tea.KeyCtrlC {
				bv.isStreaming = false
				bv.StatusLog = "Build session interrupted."
			}
			break
		}

		switch msg.Type {
		case tea.KeyEnter:
			input := strings.TrimSpace(bv.textInput.Value())
			if input == "" {
				break
			}

			bv.textInput.SetValue("")
			bv.isStreaming = true
			bv.StatusLog = "Starting build session..."
			bv.accumulatedThoughts.Reset()
			bv.viewport.SetContent("")

			cmds = append(cmds, bv.startBuildTaskCmd(modelID, input, gitStatus, projectContext, memoryFacts))
		}

	case BuildChannelReaderMsg:
		bv.stream = msg.Stream
		cmds = append(cmds, NextStepCmd(msg.Stream))

	case build.AgentStepMsg:
		bv.isStreaming = !msg.Done
		bv.handleAgentStep(msg)

		if bv.isStreaming {
			cmds = append(cmds, NextStepCmd(bv.stream))
		} else {
			bv.StatusLog = "Build completed: " + msg.Content
		}
	}

	// Update inputs
	var tiCmd tea.Cmd
	bv.textInput, tiCmd = bv.textInput.Update(msg)
	cmds = append(cmds, tiCmd)

	var vpCmd tea.Cmd
	if !bv.showDiff {
		bv.viewport, vpCmd = bv.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
	} else {
		var dvCmd tea.Cmd
		bv.diffView, dvCmd = bv.diffView.Update(msg)
		cmds = append(cmds, dvCmd)
	}

	return tea.Batch(cmds...), nil
}

func (bv *BuildView) startBuildTaskCmd(modelID, task, gitStatus, projectContext, memoryFacts string) tea.Cmd {
	return func() tea.Msg {
		ch := make(chan build.AgentStepMsg, 100)
		bv.buildMode.SetUpdateCallback(func(msg build.AgentStepMsg) {
			ch <- msg
		})

		// Temperature: 0.5, MaxTokens: 8192 for build mode
		bv.buildMode.ExecuteTask(context.Background(), modelID, task, projectContext, memoryFacts, gitStatus, 0.5, 8192)

		return BuildChannelReaderMsg{Stream: ch}
	}
}

func (bv *BuildView) handleAgentStep(msg build.AgentStepMsg) {
	switch msg.StepType {
	case "thought":
		if msg.Content == "Thinking..." {
			bv.accumulatedThoughts.WriteString("\n🤖 [Thinking...]\n")
		} else {
			bv.accumulatedThoughts.WriteString(msg.Content)
		}
		bv.refreshViewport()

	case "tool_call":
		bv.accumulatedThoughts.WriteString(fmt.Sprintf("\n🛠️  Executing tool: **%s** with args: `%v`\n", msg.ToolName, msg.ToolArgs))
		bv.refreshViewport()

	case "tool_result":
		// Truncate long results for clean chat view output
		resultPreview := msg.Content
		if len(resultPreview) > 200 {
			resultPreview = resultPreview[:200] + "... (truncated)"
		}
		bv.accumulatedThoughts.WriteString(fmt.Sprintf("📝 Result: *%s*\n", resultPreview))
		bv.refreshViewport()

		if msg.ToolName == "git_diff" {
			bv.diffView.SetContent(msg.Content)
			bv.showDiff = true
		}

	case "finished":
		bv.accumulatedThoughts.WriteString("\n🏁 **Agent Finished**\n")
		bv.refreshViewport()

	case "error":
		bv.accumulatedThoughts.WriteString(fmt.Sprintf("\n❌ **Error**: %s\n", msg.Content))
		bv.refreshViewport()

	case "warning":
		bv.accumulatedThoughts.WriteString(fmt.Sprintf("\n⚠️  **Warning**: %s\n", msg.Content))
		bv.refreshViewport()
	}
}

func (bv *BuildView) refreshViewport() {
	rendered, err := bv.renderer.Render(bv.accumulatedThoughts.String())
	if err != nil {
		bv.viewport.SetContent(bv.accumulatedThoughts.String())
	} else {
		bv.viewport.SetContent(rendered)
	}
	bv.viewport.GotoBottom()
}

func (bv *BuildView) renderToolActivity() string {
	session := bv.buildMode.Session()
	if session == nil {
		return "No activity yet. Describe a task below to start building!"
	}

	var sb strings.Builder
	theme := styles.DefaultTheme

	// 1. Session Scope & Stats Panel
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(theme.PrimaryColor).Render("📊 SESSION METRICS\n"))
	sb.WriteString(fmt.Sprintf("Scope: %s\n", session.ScopeRoot))
	sb.WriteString(fmt.Sprintf("Iteration: %d/%d\n", session.GetIteration(), session.MaxIterations))

	modifiedFiles := session.GetFilesModified()
	sb.WriteString(fmt.Sprintf("Changed Files: %d\n", len(modifiedFiles)))
	for _, f := range modifiedFiles {
		sb.WriteString(fmt.Sprintf("  - %s\n", f))
	}
	sb.WriteString("\n")

	// 2. Plan Progress Panel (if plan exists)
	planRef := session.GetPlan()
	if planRef != nil {
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(theme.AccentColor).Render("📋 PLAN PROGRESS\n"))
		sb.WriteString(fmt.Sprintf("Plan: %s\n", planRef.Title))
		currentStep := session.GetCurrentStep()

		// Render step-by-step progress
		for i, step := range planRef.Steps {
			statusSymbol := "○" // Pending
			statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			if i < currentStep {
				statusSymbol = "✓" // Done
				statusStyle = lipgloss.NewStyle().Foreground(theme.SuccessColor).Bold(true)
			} else if i == currentStep {
				statusSymbol = "➔" // Active
				statusStyle = lipgloss.NewStyle().Foreground(theme.SecondaryColor).Bold(true)
			}

			sb.WriteString(statusStyle.Render(fmt.Sprintf("%s Step %d: %s\n", statusSymbol, i+1, step.Description)))
		}
		sb.WriteString("\n")
	}

	// 3. Tool Execution History Panel
	history := session.GetToolCallHistory()
	if len(history) > 0 {
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(theme.AccentColor).Render("🛠️  TOOL HISTORY\n"))
		for i, record := range history {
			statusIcon := "✓"
			statusColor := theme.SuccessColor
			if strings.Contains(strings.ToLower(record.Result), "error") || strings.Contains(strings.ToLower(record.Result), "failed") {
				statusIcon = "✗"
				statusColor = theme.DangerColor
			}

			statusStyled := lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(statusIcon)
			undoText := ""
			if record.Undoable {
				undoText = lipgloss.NewStyle().Foreground(theme.SecondaryColor).Render(" [undo]")
			}

			sb.WriteString(fmt.Sprintf("[%d] %s %s%s\n", i+1, statusStyled, record.ToolName, undoText))

			// Render targets/commands concisely
			if path, ok := record.Args["path"].(string); ok {
				sb.WriteString(fmt.Sprintf("    Path: %s\n", path))
			} else if cmd, ok := record.Args["cmd"].(string); ok {
				truncatedCmd := cmd
				if len(truncatedCmd) > 30 {
					truncatedCmd = truncatedCmd[:27] + "..."
				}
				sb.WriteString(fmt.Sprintf("    Cmd: %s\n", truncatedCmd))
			}
		}
	} else {
		sb.WriteString("No tool calls executed yet in this step.\n")
	}

	return sb.String()
}

// View renders the horizontal split screen panels.
func (bv *BuildView) View() string {
	theme := styles.DefaultTheme

	leftWidth := (bv.width / 2) - 2
	rightWidth := bv.width - leftWidth - 4

	// Left panel chat content
	leftBox := theme.BoxStyle.Width(leftWidth).Height(bv.height - 10).Render(bv.viewport.View())

	// Right panel content (toggled between diff and tool activity)
	var rightContent string
	if bv.showDiff {
		rightContent = bv.diffView.View()
	} else {
		rightContent = bv.renderToolActivity()
	}
	rightBox := theme.BoxStyle.Width(rightWidth).Height(bv.height - 10).Render(rightContent)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)

	statusText := ""
	helpMsg := " [Ctrl+D] Toggle Diff"
	if bv.StatusLog != "" {
		statusText = theme.HelpStyle.Render("Status: " + bv.StatusLog + " | " + helpMsg)
	} else {
		statusText = theme.HelpStyle.Render(helpMsg)
	}

	inputBox := theme.BoxStyle.Width(bv.width - 4).Render(bv.textInput.View())

	return lipgloss.JoinVertical(lipgloss.Left,
		panels,
		statusText,
		inputBox,
	)
}
