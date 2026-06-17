package ui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/api"
	"github.com/yonatanzilberman/lmhub/internal/config"
	"github.com/yonatanzilberman/lmhub/internal/modes/ask"
	"github.com/yonatanzilberman/lmhub/internal/modelmanager"
	"github.com/yonatanzilberman/lmhub/internal/ui/components"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
	"github.com/yonatanzilberman/lmhub/internal/ui/views"
)

// ActiveView represents which view/screen is currently displayed.
type ActiveView int

const (
	ViewHome ActiveView = iota
	ViewChat
	ViewModelSelect
	ViewMetrics
)

type tickOnlineStatusMsg struct {
	Online bool
}

// App is the root Bubbletea model for the LM Hub TUI application.
type App struct {
	cfg             *config.Config
	apiClient       *api.Client
	modelManager    *modelmanager.Manager
	statusBar       *components.StatusBar
	contextBar      *components.ContextBar
	
	homeView        *views.HomeView
	chatView        *views.ChatView
	modelSelectView *views.ModelSelectView
	metricsView     *views.MetricsView

	activeView      ActiveView
	previousView    ActiveView
	selectedModelID string
	isOnline        bool

	width  int
	height int
}

// NewApp creates a new App root Bubbletea model.
func NewApp(cfg *config.Config, client *api.Client, mm *modelmanager.Manager, am *ask.AskMode) (*App, error) {
	chat, err := views.NewChatView(am)
	if err != nil {
		return nil, err
	}

	app := &App{
		cfg:             cfg,
		apiClient:       client,
		modelManager:    mm,
		statusBar:       components.NewStatusBar(),
		contextBar:      components.NewContextBar(),
		homeView:        views.NewHomeView(),
		chatView:        chat,
		modelSelectView: views.NewModelSelectView(mm),
		metricsView:     views.NewMetricsView(mm.Metrics()),
		activeView:      ViewHome,
		isOnline:        false,
	}

	// Try using pinned model if set in config
	if cfg.ModeModels.Ask != "" {
		app.selectedModelID = cfg.ModeModels.Ask
	}

	return app, nil
}

// Init initializes the application.
func (a *App) Init() tea.Cmd {
	// First check online status and start watcher
	return a.checkOnlineStatusCmd()
}

// checkOnlineStatusCmd returns a command checking if server is active.
func (a *App) checkOnlineStatusCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		err := a.apiClient.Ping(ctx)
		return tickOnlineStatusMsg{Online: err == nil}
	}
}

// scheduleOnlineCheck returns a command scheduling another online status check.
func (a *App) scheduleOnlineCheck() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return a.checkOnlineStatusCmd()()
	})
}

// Update handles standard keyboard shortcuts and routing.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		
		a.statusBar.SetWidth(msg.Width)
		a.contextBar.SetWidth(msg.Width)
		a.homeView.SetSize(msg.Width, msg.Height)
		a.chatView.SetSize(msg.Width, msg.Height)
		a.modelSelectView.SetSize(msg.Width, msg.Height)
		a.metricsView.SetSize(msg.Width, msg.Height)

	case tickOnlineStatusMsg:
		a.isOnline = msg.Online
		cmds = append(cmds, a.scheduleOnlineCheck())

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlQ:
			return a, tea.Quit

		case tea.KeyCtrlA:
			if a.activeView != ViewChat {
				a.activeView = ViewChat
				a.chatView.Reset()
			}

		case tea.KeyCtrlM:
			if a.activeView == ViewModelSelect {
				a.activeView = a.previousView
			} else {
				a.previousView = a.activeView
				a.activeView = ViewModelSelect
			}

		case tea.KeyCtrlI:
			if a.activeView == ViewMetrics {
				a.activeView = a.previousView
			} else {
				a.previousView = a.activeView
				a.activeView = ViewMetrics
			}

		case tea.KeyCtrlL:
			if a.activeView == ViewChat {
				a.chatView.Reset()
			}
		}

	case views.ChannelReaderMsg:
		// Route connection stream messages from chatView
		cmd := a.chatView.HandleChannelReader(msg)
		return a, cmd

	case views.ModelLoadDoneMsg:
		// Model loaded via model select view
		a.selectedModelID = msg.ModelID
	}

	// Update the active view
	switch a.activeView {
	case ViewChat:
		// ModelID pin override
		cmd, _ := a.chatView.Update(msg, a.selectedModelID)
		cmds = append(cmds, cmd)
	case ViewModelSelect:
		cmd, _ := a.modelSelectView.Update(msg)
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

// View renders the entire application window.
func (a *App) View() string {
	theme := styles.DefaultTheme

	// 1. Top Mode Tab Selector Header
	tabs := []string{
		" [Ctrl+A] ASK (Chat) ",
		" PLAN (Disabled) ",
		" BUILD (Disabled) ",
	}
	
	activeTab := 0
	if a.activeView == ViewChat {
		activeTab = 0
	}

	var renderedTabs []string
	for i, tab := range tabs {
		if i == activeTab && a.activeView == ViewChat {
			renderedTabs = append(renderedTabs, lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#000000")).
				Background(theme.SuccessColor).
				Render(tab))
		} else {
			renderedTabs = append(renderedTabs, lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
				Background(lipgloss.Color("#333333")).
				Render(tab))
		}
	}

	headerLine := lipgloss.JoinHorizontal(lipgloss.Left, renderedTabs...)
	modelSelectHelp := theme.HelpStyle.Render("   [Ctrl+M] Models  |  [Ctrl+I] Metrics  |  [Ctrl+Q] Exit")
	header := lipgloss.JoinHorizontal(lipgloss.Left, headerLine, modelSelectHelp)

	// 2. Active Screen Content
	var content string
	switch a.activeView {
	case ViewHome:
		content = a.homeView.View()
	case ViewChat:
		content = a.chatView.View()
	case ViewModelSelect:
		content = a.modelSelectView.View()
	case ViewMetrics:
		content = a.metricsView.View()
	}

	// 3. Context Bar (Visible only in Chat screen)
	ctxBar := ""
	if a.activeView == ViewChat && a.cfg.UI.ShowContextBar {
		m := a.modelManager.Metrics().Get()
		
		// Map current metrics to context breakdown estimate
		// Using token estimations
		sysTokens := 400  // baseline
		histTokens := m.TokensUsed - sysTokens
		if histTokens < 0 {
			histTokens = 0
		}
		ctxBar = a.contextBar.Render(m.TokensUsed, m.ContextLimit, sysTokens, histTokens, 0, 0)
	}

	// 4. Status Bar (Always on bottom)
	m := a.modelManager.Metrics().Get()
	activeModeStr := "home"
	switch a.activeView {
	case ViewChat:
		activeModeStr = "ask"
	case ViewModelSelect:
		activeModeStr = "models"
	case ViewMetrics:
		activeModeStr = "metrics"
	}
	
	loadedID := a.selectedModelID
	if m.ModelID != "" {
		loadedID = m.ModelID
	}
	sBar := a.statusBar.Render(activeModeStr, loadedID, m.RAMUsedGB, a.chatView.CurrentSpeed, a.isOnline)

	// Combine components
	res := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n",
		content,
	)

	// Make sure the screen layout fits window height nicely
	remainingNewlines := a.height - lipgloss.Height(res) - lipgloss.Height(ctxBar) - lipgloss.Height(sBar) - 2
	paddingStr := ""
	if remainingNewlines > 0 {
		paddingStr = strings.Repeat("\n", remainingNewlines)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		res,
		paddingStr,
		ctxBar,
		"\n",
		sBar,
	)
}
