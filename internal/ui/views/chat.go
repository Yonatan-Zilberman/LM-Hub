package views

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/api"
	"github.com/yonatanzilberman/lmhub/internal/memory"
	"github.com/yonatanzilberman/lmhub/internal/modes/ask"
	"github.com/yonatanzilberman/lmhub/internal/ui/components"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

// ChatChunkMsg is sent to update the UI with a new streamed token.
type ChatChunkMsg struct {
	Content  string
	TokSpeed float64
	TTFTMs   int
	Usage    *api.Usage
}

// ChatDoneMsg is sent when streaming chat finishes.
type ChatDoneMsg struct{}

// ChatErrorMsg is sent when an error occurs during chat.
type ChatErrorMsg struct {
	Err error
}

// ChatView manages the chat screen, history viewport, and message input.
type ChatView struct {
	askMode     *ask.AskMode
	viewport    viewport.Model
	textInput   textinput.Model
	renderer    *components.MarkdownRenderer
	width       int
	height      int
	isStreaming bool
	stream      <-chan api.StreamChunk
	memManager  *memory.MemoryManager
	projectRoot string

	// Streaming metrics and state
	CurrentSpeed float64
	CurrentTTFT  int
	streamBuffer strings.Builder

	// Log warnings/trim messages
	StatusLog string

	// Cancellation function for the active stream
	cancel context.CancelFunc
}

// NewChatView creates a new ChatView instance.
func NewChatView(am *ask.AskMode, mm *memory.MemoryManager, projectRoot string) (*ChatView, error) {
	ti := textinput.New()
	ti.Placeholder = "Type a message and press Enter..."
	ti.Focus()
	ti.Prompt = " > "

	mr, err := components.NewMarkdownRenderer()
	if err != nil {
		return nil, err
	}

	vp := viewport.New(80, 20)
	vp.SetContent("Conversation started. Ask anything!")

	return &ChatView{
		askMode:    am,
		textInput:  ti,
		viewport:   vp,
		renderer:   mr,
		memManager: mm,
		projectRoot: projectRoot,
	}, nil
}

// SetSize updates layout sizes.
func (cv *ChatView) SetSize(w, h int) {
	cv.width = w
	cv.height = h
	
	// Reserve space for input, borders, and status bars (approx 8 lines)
	cv.viewport.Width = w
	cv.viewport.Height = h - 8
	cv.textInput.Width = w - 10
}

// SetInputValue updates the text input value.
func (cv *ChatView) SetInputValue(val string) {
	cv.textInput.SetValue(val)
	cv.textInput.CursorEnd()
}

// Reset clears current chat view logs and ask mode history.
func (cv *ChatView) Reset() {
	cv.askMode.Reset()
	cv.viewport.SetContent("Conversation cleared. Ask anything!")
	cv.viewport.GotoTop()
	cv.StatusLog = ""
	cv.CurrentSpeed = 0
	cv.CurrentTTFT = 0
}

// AskMode returns the underlying AskMode instance.
func (cv *ChatView) AskMode() *ask.AskMode {
	return cv.askMode
}

// SetHistory replaces the history and refreshes the view content.
func (cv *ChatView) SetHistory(messages []api.Message) {
	cv.askMode.SetHistory(messages)
	cv.refreshContent()
}

// Update handles message updates.
func (cv *ChatView) Update(msg tea.Msg, modelID string) (tea.Cmd, error) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if cv.isStreaming {
			// Ignore typing while streaming, but allow standard navigation/quit keys
			if msg.Type == tea.KeyCtrlC {
				cv.isStreaming = false
				if cv.cancel != nil {
					cv.cancel()
				}
			}
			break
		}

		switch msg.Type {
		case tea.KeyEnter:
			input := strings.TrimSpace(cv.textInput.Value())
			if input == "" {
				break
			}

			if strings.HasPrefix(input, "/") {
				parts := strings.Fields(input)
				cmdName := parts[0]
				arg := ""
				if len(parts) > 1 {
					arg = strings.Join(parts[1:], " ")
				}
				cv.textInput.SetValue("")
				cmds = append(cmds, func() tea.Msg {
					return SlashCmdMsg{CmdType: cmdName, Arg: arg}
				})
				break
			}

			// Clear text input
			cv.textInput.SetValue("")
			cv.isStreaming = true
			cv.streamBuffer.Reset()
			cv.StatusLog = "Waiting for model..."

			// Start streaming logic
			cmds = append(cmds, cv.startChatStreamCmd(modelID, input))
		}

	case ChatChunkMsg:
		cv.CurrentSpeed = msg.TokSpeed
		cv.CurrentTTFT = msg.TTFTMs
		cv.StatusLog = fmt.Sprintf("Streaming... Speed: %.1f tok/s | TTFT: %dms", msg.TokSpeed, msg.TTFTMs)
		
		cv.streamBuffer.WriteString(msg.Content)
		
		// Update viewport content dynamically
		cv.refreshContent()
		
		// Continue reading next token from stream
		if cv.isStreaming {
			cmds = append(cmds, NextChunkCmd(cv.stream))
		}

	case ChatDoneMsg:
		cv.isStreaming = false
		cv.streamBuffer.Reset()
		cv.StatusLog = "Response complete."
		cv.refreshContent()

	case ChatErrorMsg:
		cv.isStreaming = false
		cv.StatusLog = fmt.Sprintf("Error: %v", msg.Err)
		cv.viewport.SetContent(cv.viewport.View() + "\n\nError: " + msg.Err.Error())
		cv.viewport.GotoBottom()
	}

	// Update components
	var tiCmd tea.Cmd
	cv.textInput, tiCmd = cv.textInput.Update(msg)
	cmds = append(cmds, tiCmd)

	var vpCmd tea.Cmd
	cv.viewport, vpCmd = cv.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return tea.Batch(cmds...), nil
}

func (cv *ChatView) startChatStreamCmd(modelID, text string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		cv.cancel = cancel
		
		// Setup call options
		// Default temperature 0.7, max tokens 8192
		var memoryFacts string
		if cv.memManager != nil {
			memoryFacts = cv.memManager.InjectFacts()
		}
		stream, logMsg, err := cv.askMode.SendUserMessage(ctx, modelID, text, cv.projectRoot, runtime.GOOS, "zsh", "", memoryFacts, 0.0)
		if err != nil {
			return ChatErrorMsg{Err: err}
		}

		if logMsg != "" {
			cv.viewport.SetContent(cv.viewport.View() + "\n\n> [!WARNING]\n> " + logMsg + "\n\n")
		}

		// Read from the streaming channel asynchronously and dispatch messages
		// To send them to Bubbletea, we need to dispatch chunks individually.
		// Since a tea.Cmd is synchronous, we run a goroutine that calls a callback or we return the final result.
		// Wait! Bubbletea expects commands to return tea.Msg. If we are in a goroutine, how do we send a message?
		// Bubbletea allows passing program handle, but we don't have it directly.
		// Wait, a clean design for streaming in Bubbletea is to use a sub-command loop or a channel reader command!
		// Let's create a command generator that reads next item from a channel and schedules itself recursively!
		// Let's do that!
		return ChannelReaderMsg{Stream: stream}
	}
}

// ChannelReaderMsg is used to bridge the channel to Bubbletea event loop.
type ChannelReaderMsg struct {
	Stream <-chan api.StreamChunk
}

// NextChunkCmd waits for the next chunk from the channel and returns it.
func NextChunkCmd(stream <-chan api.StreamChunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-stream
		if !ok {
			return ChatDoneMsg{}
		}
		if chunk.Error != nil {
			return ChatErrorMsg{Err: chunk.Error}
		}
		if chunk.Done {
			return ChatDoneMsg{}
		}
		return ChatChunkMsg{
			Content:  chunk.Content,
			TokSpeed: chunk.TokSpeed,
			TTFTMs:   chunk.TTFTMs,
			Usage:    chunk.UsageInfo,
		}
	}
}

// HandleChannelReader resolves the channel reader message and triggers NextChunkCmd.
func (cv *ChatView) HandleChannelReader(msg ChannelReaderMsg) tea.Cmd {
	cv.stream = msg.Stream
	return NextChunkCmd(msg.Stream)
}

func (cv *ChatView) refreshContent() {
	var sb strings.Builder
	for _, m := range cv.askMode.History() {
		roleTitle := "You"
		roleStyle := lipgloss.NewStyle().Foreground(styles.DefaultTheme.AccentColor).Bold(true)
		if m.Role == "assistant" {
			roleTitle = "LM Hub"
			roleStyle = lipgloss.NewStyle().Foreground(styles.DefaultTheme.PrimaryColor).Bold(true)
		}
		
		sb.WriteString(fmt.Sprintf("%s\n", roleStyle.Render(roleTitle)))
		
		// Render markdown for each block
		rendered, err := cv.renderer.Render(m.Content)
		if err != nil {
			sb.WriteString(m.Content)
		} else {
			sb.WriteString(rendered)
		}
		sb.WriteString("\n")
	}

	// Append the active stream buffer if we are currently streaming
	if cv.isStreaming && cv.streamBuffer.Len() > 0 {
		roleStyle := lipgloss.NewStyle().Foreground(styles.DefaultTheme.PrimaryColor).Bold(true)
		sb.WriteString(fmt.Sprintf("%s\n", roleStyle.Render("LM Hub")))
		
		rendered, err := cv.renderer.Render(cv.streamBuffer.String() + " █")
		if err != nil {
			sb.WriteString(cv.streamBuffer.String())
			sb.WriteString(" █")
		} else {
			sb.WriteString(rendered)
		}
		sb.WriteString("\n")
	}

	cv.viewport.SetContent(sb.String())
	cv.viewport.GotoBottom()
}

// History returns the conversation history from Ask mode.
func (cv *ChatView) History() []api.Message {
	return cv.askMode.History()
}

// View renders the chat screen.
func (cv *ChatView) View() string {
	theme := styles.DefaultTheme

	// Viewport display
	vpBox := cv.viewport.View()

	// Text input box
	inputBox := theme.BoxStyle.Width(cv.width - 4).Render(
		cv.textInput.View(),
	)

	// Status log info
	statusText := ""
	if cv.StatusLog != "" {
		statusText = theme.HelpStyle.Render("Status: " + cv.StatusLog)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		vpBox,
		statusText,
		inputBox,
	)
}

// SlashCmdMsg represents a slash command entered in the chat interface.
type SlashCmdMsg struct {
	CmdType string
	Arg     string
}
