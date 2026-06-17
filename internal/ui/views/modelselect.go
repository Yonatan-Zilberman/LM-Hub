package views

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/modelmanager"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
)

// ModelLoadStatusMsg is sent during model loading to display progress logs.
type ModelLoadStatusMsg struct {
	Status string
}

// ModelLoadDoneMsg is sent when model loading completes.
type ModelLoadDoneMsg struct {
	ModelID string
}

// ModelLoadErrorMsg is sent if model loading fails.
type ModelLoadErrorMsg struct {
	Err error
}

// ModelSelectView manages the model browsing and loading list.
type ModelSelectView struct {
	manager     *modelmanager.Manager
	width       int
	height      int
	selectedIndex int
	isLoading   bool
	loadStatus  string
	statusChan  chan string
}

// NewModelSelectView creates a new ModelSelectView instance.
func NewModelSelectView(mm *modelmanager.Manager) *ModelSelectView {
	return &ModelSelectView{
		manager: mm,
	}
}

// SetSize updates layout sizes.
func (mv *ModelSelectView) SetSize(w, h int) {
	mv.width = w
	mv.height = h
}

// IsLoading returns whether a model load is currently in progress.
func (mv *ModelSelectView) IsLoading() bool {
	return mv.isLoading
}

// LoadStatus returns the current model load progress text.
func (mv *ModelSelectView) LoadStatus() string {
	return mv.loadStatus
}

// Update handles interactions in the model selector.
func (mv *ModelSelectView) Update(msg tea.Msg) (tea.Cmd, error) {
	var cmds []tea.Cmd

	if mv.isLoading {
		switch msg := msg.(type) {
		case ModelLoadStatusMsg:
			mv.loadStatus = msg.Status
			// Continue reading next status update
			cmds = append(cmds, mv.readNextStatusCmd())
		case ModelLoadDoneMsg:
			mv.isLoading = false
			mv.loadStatus = "Model loaded successfully!"
		case ModelLoadErrorMsg:
			mv.isLoading = false
			mv.loadStatus = fmt.Sprintf("Error loading model: %v", msg.Err)
		}
		return tea.Batch(cmds...), nil
	}

	models := mv.manager.Registry().List()
	if len(models) == 0 {
		// Try refreshing available models
		return func() tea.Msg {
			ctx := context.Background()
			_ = mv.manager.Registry().Refresh(ctx)
			return nil
		}, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if mv.selectedIndex > 0 {
				mv.selectedIndex--
			}
		case "down", "j":
			if mv.selectedIndex < len(models)-1 {
				mv.selectedIndex++
			}
		case "enter":
			selectedModel := models[mv.selectedIndex]
			
			// If already loaded, unload it. Otherwise, load it.
			if len(selectedModel.LoadedInstances) > 0 {
				mv.isLoading = true
				mv.loadStatus = fmt.Sprintf("Unloading model: %s...", selectedModel.DisplayName)
				cmds = append(cmds, mv.unloadModelCmd(selectedModel.LoadedInstances[0].ID))
			} else {
				mv.isLoading = true
				mv.loadStatus = "Initializing model load..."
				mv.statusChan = make(chan string, 10)
				
				// Standard context length default to 8192 for local loading
				cmds = append(cmds, mv.loadModelCmd(selectedModel.Key, 8192))
				cmds = append(cmds, mv.readNextStatusCmd())
			}
		}
	}

	return tea.Batch(cmds...), nil
}

func (mv *ModelSelectView) unloadModelCmd(_ string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := mv.manager.UnloadAll(ctx) // Unload everything for simplicity
		if err != nil {
			return ModelLoadErrorMsg{Err: err}
		}
		return ModelLoadDoneMsg{ModelID: ""}
	}
}

func (mv *ModelSelectView) loadModelCmd(key string, contextLength int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		// EnsureModel unloads any current models and loads the target model
		err := mv.manager.EnsureModel(ctx, key, contextLength, mv.statusChan)
		close(mv.statusChan)
		if err != nil {
			return ModelLoadErrorMsg{Err: err}
		}
		return ModelLoadDoneMsg{ModelID: key}
	}
}

func (mv *ModelSelectView) readNextStatusCmd() tea.Cmd {
	return func() tea.Msg {
		status, ok := <-mv.statusChan
		if !ok {
			return nil // Stream closed
		}
		return ModelLoadStatusMsg{Status: status}
	}
}

// View renders the model selector screen.
func (mv *ModelSelectView) View() string {
	theme := styles.DefaultTheme

	var sb strings.Builder
	sb.WriteString(theme.TitleStyle.Render("🤖 Model Browser (LM Studio)"))
	sb.WriteString("\n")
	sb.WriteString(theme.HelpStyle.Render("Use [↑↓] or [kj] to navigate, [Enter] to Load/Unload, [Ctrl+M] to close"))
	sb.WriteString("\n\n")

	if mv.isLoading {
		loadingBox := theme.BoxStyle.Width(mv.width - 10).Render(
			fmt.Sprintf("Loading Status:\n\n %s", mv.loadStatus),
		)
		sb.WriteString(loadingBox)
		return sb.String()
	}

	models := mv.manager.Registry().List()
	if len(models) == 0 {
		sb.WriteString("No models found in LM Studio registry. Is the server running?")
		return sb.String()
	}

	for i, m := range models {
		prefix := "  "
		style := theme.NormalTextStyle
		if i == mv.selectedIndex {
			prefix = "❯ "
			style = theme.HighlightStyle
		}

		statusLabel := "[Not Loaded]"
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		if len(m.LoadedInstances) > 0 {
			statusLabel = "[LOADED]"
			statusStyle = lipgloss.NewStyle().Foreground(theme.SuccessColor).Bold(true)
		}

		mSize := float64(m.SizeBytes) / (1024 * 1024 * 1024)
		modelLine := fmt.Sprintf("%s%s (%s, %.2f GB) %s", 
			prefix, m.DisplayName, m.Architecture, mSize, statusStyle.Render(statusLabel))
		
		sb.WriteString(style.Render(modelLine))
		sb.WriteString("\n")
	}

	if mv.loadStatus != "" && !mv.isLoading {
		sb.WriteString("\n")
		sb.WriteString(theme.HelpStyle.Render("Last status: " + mv.loadStatus))
	}

	return sb.String()
}
