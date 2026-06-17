package views

import (
	"context"
	"fmt"
	"strings"
	"time"

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

	return &BuildView{
		buildMode: bm,
		textInput: ti,
		viewport:  vp,
		renderer:  mr,
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
}

// Reset clears active logs and session metrics.
func (bv *BuildView) Reset() {
	bv.buildMode.Reset()
	bv.viewport.SetContent("Build mode cleared. Describe a task to execute.")
	bv.viewport.GotoTop()
	bv.StatusLog = ""
	bv.accumulatedThoughts.Reset()
	bv.isStreaming = false
}

// Update handles UI and agent updates.
func (bv *BuildView) Update(msg tea.Msg, modelID string, gitStatus, projectContext, memoryFacts string) (tea.Cmd, error) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
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
	bv.viewport, vpCmd = bv.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

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

func (bv *BuildView) renderToolActivity(width int) string {
	session := bv.buildMode.Session()
	if session == nil {
		return "No activity yet. Describe a task below to start building!"
	}

	history := session.GetToolCallHistory()
	if len(history) == 0 {
		return "Agent starting up..."
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(styles.DefaultTheme.AccentColor).Render("Tool Execution History\n\n"))

	for i, record := range history {
		statusIcon := "✓"
		statusColor := styles.DefaultTheme.SuccessColor
		if strings.Contains(strings.ToLower(record.Result), "error") || strings.Contains(strings.ToLower(record.Result), "failed") {
			statusIcon = "✗"
			statusColor = styles.DefaultTheme.DangerColor
		}

		statusStyled := lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(statusIcon)
		undoText := ""
		if record.Undoable {
			undoText = lipgloss.NewStyle().Foreground(styles.DefaultTheme.SecondaryColor).Render(" [undoable]")
		}

		sb.WriteString(fmt.Sprintf("[%d] %s  %s%s\n", i+1, statusStyled, record.ToolName, undoText))

		// Render targets
		if path, ok := record.Args["path"].(string); ok {
			sb.WriteString(fmt.Sprintf("    Target: %s\n", path))
		} else if cmd, ok := record.Args["cmd"].(string); ok {
			sb.WriteString(fmt.Sprintf("    Cmd: %s\n", cmd))
		}
		sb.WriteString(fmt.Sprintf("    Duration: %v\n\n", record.Duration.Round(time.Millisecond)))
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

	// Right panel tool activity
	activityText := bv.renderToolActivity(rightWidth)
	// Make right panel scrollable or simple viewport
	rightBox := theme.BoxStyle.Width(rightWidth).Height(bv.height - 10).Render(activityText)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)

	statusText := ""
	if bv.StatusLog != "" {
		statusText = theme.HelpStyle.Render("Status: " + bv.StatusLog)
	}

	inputBox := theme.BoxStyle.Width(bv.width - 4).Render(bv.textInput.View())

	return lipgloss.JoinVertical(lipgloss.Left,
		panels,
		statusText,
		inputBox,
	)
}
