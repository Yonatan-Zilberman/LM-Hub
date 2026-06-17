package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yonatanzilberman/lmhub/internal/agent"
	"github.com/yonatanzilberman/lmhub/internal/api"
	"github.com/yonatanzilberman/lmhub/internal/config"
	"github.com/yonatanzilberman/lmhub/internal/modes/ask"
	"github.com/yonatanzilberman/lmhub/internal/modes/plan"
	"github.com/yonatanzilberman/lmhub/internal/modes/build"
	"github.com/yonatanzilberman/lmhub/internal/modelmanager"
	"github.com/yonatanzilberman/lmhub/internal/safety"
	"github.com/yonatanzilberman/lmhub/internal/tools"
	"github.com/yonatanzilberman/lmhub/internal/ui/components"
	"github.com/yonatanzilberman/lmhub/internal/ui/styles"
	"github.com/yonatanzilberman/lmhub/internal/ui/views"
)

// ActiveView represents which view/screen is currently displayed.
type ActiveView int

const (
	ViewHome ActiveView = iota
	ViewChat
	ViewPlanChat
	ViewPlan
	ViewBuild
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
	contextManager  *agent.ContextManager
	budgetManager   *agent.BudgetManager
	statusBar       *components.StatusBar
	contextBar      *components.ContextBar
	registry        *tools.Registry
	
	homeView        *views.HomeView
	chatView        *views.ChatView
	planChatView    *views.PlanChatView
	planView        *views.PlanView
	buildView       *views.BuildView
	buildMode       *build.BuildMode
	modelSelectView *views.ModelSelectView
	metricsView     *views.MetricsView

	activeView      ActiveView
	previousView    ActiveView
	selectedModelID string
	isOnline        bool

	isLoadingModel  bool
	modelLoadStatus string
	modelStatusChan chan string

	// Safety overlays
	showConfirm     bool
	confirmMsg      safety.ConfirmMsg
	confirmView     *views.ConfirmView

	// Undo overlays
	showUndoHistory bool
	undoHistoryView *views.UndoHistoryView

	// Warning banners
	parseWarningMsg string

	width  int
	height int
}

// NewApp creates a new App root Bubbletea model.
func NewApp(
	cfg *config.Config,
	client *api.Client,
	mm *modelmanager.Manager,
	am *ask.AskMode,
	pm *plan.PlanMode,
	bm *agent.BudgetManager,
	cm *agent.ContextManager,
	projectRoot string,
) (*App, error) {
	chat, err := views.NewChatView(am)
	if err != nil {
		return nil, err
	}

	planChat := views.NewPlanChatView(pm, projectRoot)
	planView := views.NewPlanView(projectRoot)

	// Initialize Build Mode and Tools Registry
	reg := tools.NewRegistry(projectRoot)
	reg.Register(tools.NewReadFileTool(projectRoot))
	reg.Register(tools.NewWriteFileTool(projectRoot))
	reg.Register(tools.NewCreateDirTool(projectRoot))
	reg.Register(tools.NewListDirTool(projectRoot))
	reg.Register(tools.NewDeleteFileTool(projectRoot))
	reg.Register(tools.NewMoveFileTool(projectRoot))
	reg.Register(tools.NewSearchFilesTool(projectRoot))
	reg.Register(tools.NewRunCommandTool(projectRoot, cfg.Tools.Shell.TimeoutSeconds, cfg.Tools.Shell.AllowedShells, cfg.Tools.Shell.Blocklist))

	// Git tools
	reg.Register(tools.NewGitStatusTool(projectRoot))
	reg.Register(tools.NewGitDiffTool(projectRoot))
	reg.Register(tools.NewGitAddTool(projectRoot))
	reg.Register(tools.NewGitRestoreStagedTool(projectRoot))
	reg.Register(tools.NewGitCommitTool(projectRoot))
	reg.Register(tools.NewGitResetCommitTool(projectRoot))
	reg.Register(tools.NewGitLogTool(projectRoot))
	reg.Register(tools.NewGitBranchTool(projectRoot))
	reg.Register(tools.NewGitStashTool(projectRoot))

	// Docker tools
	reg.Register(tools.NewDockerPSTool(projectRoot, cfg.Tools.Docker.Socket))
	reg.Register(tools.NewDockerLogsTool(projectRoot, cfg.Tools.Docker.Socket))
	reg.Register(tools.NewDockerExecTool(projectRoot, cfg.Tools.Docker.Socket))
	reg.Register(tools.NewDockerBuildTool(projectRoot, cfg.Tools.Docker.Socket))
	reg.Register(tools.NewDockerComposeTool(projectRoot))
	reg.Register(tools.NewDockerPullTool(projectRoot, cfg.Tools.Docker.Socket))

	// Web tools
	reg.Register(tools.NewWebSearchTool(cfg.Tools.Web.SearchProvider, cfg.Tools.Web.SerperAPIKey))
	reg.Register(tools.NewWebFetchTool(cfg.Tools.Web.FetchTimeoutSeconds, cfg.Tools.Web.CacheTTLMinutes))

	buildMode := build.NewBuildMode(client, mm, cm, bm, cfg, reg, nil, nil)
	buildView, err := views.NewBuildView(buildMode)
	if err != nil {
		return nil, err
	}

	app := &App{
		cfg:             cfg,
		apiClient:       client,
		modelManager:    mm,
		contextManager:  cm,
		budgetManager:   bm,
		statusBar:       components.NewStatusBar(),
		contextBar:      components.NewContextBar(),
		registry:        reg,
		homeView:        views.NewHomeView(),
		chatView:        chat,
		planChatView:    planChat,
		planView:        planView,
		buildView:       buildView,
		buildMode:       buildMode,
		modelSelectView: views.NewModelSelectView(mm),
		metricsView:     views.NewMetricsView(mm.Metrics()),
		activeView:      ViewHome,
		isOnline:        false,
	}

	// Configure build mode confirmation callback to hook TUI overlays
	app.buildMode.SetConfirmCallback(func(msg safety.ConfirmMsg) bool {
		app.confirmMsg = msg
		app.confirmView = views.NewConfirmView(msg)
		app.showConfirm = true

		if msg.Diff != "" {
			app.buildView.SetDiff(msg.Diff)
		}

		// Wait blocks the background execution loop until user sets the channel response
		approved := <-msg.ResponseChan
		return approved
	})

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

	// 1. Check Safety Confirmation Modal overlay input
	if a.showConfirm {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "y", "Y":
				a.confirmMsg.ResponseChan <- true
				a.showConfirm = false
			case "n", "N":
				a.confirmMsg.ResponseChan <- false
				a.showConfirm = false
			default:
				if keyMsg.Type == tea.KeyEsc {
					a.confirmMsg.ResponseChan <- false
					a.showConfirm = false
				}
			}
		}
		return a, nil
	}

	// 2. Check Undo History Modal overlay input
	if a.showUndoHistory {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.Type {
			case tea.KeyUp:
				a.undoHistoryView.MoveSelection(-1)
			case tea.KeyDown:
				a.undoHistoryView.MoveSelection(1)
			case tea.KeyEsc:
				a.showUndoHistory = false
			case tea.KeyEnter:
				a.showUndoHistory = false
				idx := a.undoHistoryView.SelectedIndex()
				cmds = append(cmds, func() tea.Msg {
					for i := 0; i <= idx; i++ {
						_ = a.buildMode.Session().UndoStack.UndoLast(context.Background(), a.registry)
					}
					return nil
				})
			case tea.KeyRunes:
				if keyMsg.String() == "u" || keyMsg.String() == "U" {
					a.showUndoHistory = false
					cmds = append(cmds, func() tea.Msg {
						_ = a.buildMode.Session().UndoStack.UndoAll(context.Background(), a.registry)
						return nil
					})
				}
			}
		}
		return a, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		
		a.statusBar.SetWidth(msg.Width)
		a.contextBar.SetWidth(msg.Width)
		a.homeView.SetSize(msg.Width, msg.Height)
		a.chatView.SetSize(msg.Width, msg.Height)
		a.planChatView.SetSize(msg.Width, msg.Height)
		a.planView.SetSize(msg.Width, msg.Height)
		a.buildView.SetSize(msg.Width, msg.Height)
		a.modelSelectView.SetSize(msg.Width, msg.Height)
		a.metricsView.SetSize(msg.Width, msg.Height)

		if a.confirmView != nil {
			a.confirmView.SetSize(msg.Width, msg.Height)
		}
		if a.undoHistoryView != nil {
			a.undoHistoryView.SetSize(msg.Width, msg.Height)
		}

	case tickOnlineStatusMsg:
		a.isOnline = msg.Online
		cmds = append(cmds, a.scheduleOnlineCheck())

	case views.ModelLoadStatusMsg:
		if a.isLoadingModel {
			a.modelLoadStatus = msg.Status
			cmds = append(cmds, a.readNextModelStatusCmd())
		} else {
			cmd, _ := a.modelSelectView.Update(msg)
			cmds = append(cmds, cmd)
		}

	case views.ModelLoadDoneMsg:
		if a.isLoadingModel {
			a.isLoadingModel = false
			a.selectedModelID = msg.ModelID
		} else {
			a.selectedModelID = msg.ModelID
			cmd, _ := a.modelSelectView.Update(msg)
			cmds = append(cmds, cmd)
		}

	case views.ModelLoadErrorMsg:
		if a.isLoadingModel {
			a.isLoadingModel = false
			a.modelLoadStatus = fmt.Sprintf("Error: %v", msg.Err)
		} else {
			cmd, _ := a.modelSelectView.Update(msg)
			cmds = append(cmds, cmd)
		}

	case views.PlanGenerateMsg:
		a.planView.SetPlan(msg.Plan)
		a.activeView = ViewPlan

	case views.PlanApproveMsg:
		a.activeView = ViewChat
		a.chatView.Reset()
		a.chatView.StatusLog = fmt.Sprintf("Plan approved and saved as %s. Ready to build!", msg.Filename)

	case views.PlanRejectMsg:
		a.activeView = ViewPlanChat
		a.planChatView.Reset()

	case tea.KeyMsg:
		if a.parseWarningMsg != "" {
			a.parseWarningMsg = ""
		}
		if a.isLoadingModel {
			break
		}

		switch msg.Type {
		case tea.KeyCtrlQ:
			return a, tea.Quit

		case tea.KeyCtrlA:
			if a.activeView != ViewChat {
				a.activeView = ViewChat
				a.chatView.Reset()
			}

		case tea.KeyCtrlP:
			if a.activeView != ViewPlanChat && a.activeView != ViewPlan {
				a.activeView = ViewPlanChat
				a.planChatView.Reset()
				
				planModel := a.cfg.ModeModels.Plan
				if planModel != "" && planModel != a.selectedModelID {
					a.isLoadingModel = true
					a.modelLoadStatus = fmt.Sprintf("Auto-switching to Plan mode model: %s...", planModel)
					a.modelStatusChan = make(chan string, 10)
					
					cmds = append(cmds, a.loadModelCmd(planModel, 8192))
					cmds = append(cmds, a.readNextModelStatusCmd())
				}
			}

		case tea.KeyCtrlB:
			if a.activeView != ViewBuild {
				a.activeView = ViewBuild
				a.buildView.Reset()

				buildModel := a.cfg.ModeModels.Build
				if buildModel != "" && buildModel != a.selectedModelID {
					a.isLoadingModel = true
					a.modelLoadStatus = fmt.Sprintf("Auto-switching to Build mode model: %s...", buildModel)
					a.modelStatusChan = make(chan string, 10)

					cmds = append(cmds, a.loadModelCmd(buildModel, 32768))
					cmds = append(cmds, a.readNextModelStatusCmd())
				}
			}

		case tea.KeyCtrlZ:
			if a.activeView == ViewBuild {
				session := a.buildMode.Session()
				if session != nil {
					list := session.UndoStack.List()
					a.undoHistoryView = views.NewUndoHistoryView(list)
					a.undoHistoryView.SetSize(a.width, a.height)
					a.showUndoHistory = true
				}
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
			} else if a.activeView == ViewBuild {
				a.buildView.Reset()
			}
		}

	case views.ChannelReaderMsg:
		cmd := a.chatView.HandleChannelReader(msg)
		return a, cmd

	case views.BuildChannelReaderMsg:
		gitStatus := ""
		projectContext, _ := agent.LoadProjectContext(".", a.contextManager, a.cfg.ContextBudget.ProjectContextMaxTokens)
		cmd, _ := a.buildView.Update(msg, a.selectedModelID, gitStatus, projectContext, "")
		return a, cmd

	case build.AgentStepMsg:
		if agent.GlobalParseMetrics.ShouldWarn() {
			a.parseWarningMsg = fmt.Sprintf("Model %s is producing unreliable tool calls (3+ consecutive failures). Consider switching models.", a.selectedModelID)
		}
		gitStatus := ""
		projectContext, _ := agent.LoadProjectContext(".", a.contextManager, a.cfg.ContextBudget.ProjectContextMaxTokens)
		cmd, _ := a.buildView.Update(msg, a.selectedModelID, gitStatus, projectContext, "")
		return a, cmd
	}

	// Update active component
	if !a.isLoadingModel {
		switch a.activeView {
		case ViewChat:
			cmd, _ := a.chatView.Update(msg, a.selectedModelID)
			cmds = append(cmds, cmd)
		case ViewPlanChat:
			cmd, _ := a.planChatView.Update(msg, a.selectedModelID)
			cmds = append(cmds, cmd)
		case ViewPlan:
			cmd, _ := a.planView.Update(msg)
			cmds = append(cmds, cmd)
		case ViewBuild:
			gitStatus := ""
			projectContext, _ := agent.LoadProjectContext(".", a.contextManager, a.cfg.ContextBudget.ProjectContextMaxTokens)
			cmd, _ := a.buildView.Update(msg, a.selectedModelID, gitStatus, projectContext, "")
			cmds = append(cmds, cmd)
		case ViewModelSelect:
			cmd, _ := a.modelSelectView.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return a, tea.Batch(cmds...)
}

func (a *App) loadModelCmd(key string, contextLength int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := a.modelManager.EnsureModel(ctx, key, contextLength, a.modelStatusChan)
		close(a.modelStatusChan)
		if err != nil {
			return views.ModelLoadErrorMsg{Err: err}
		}
		return views.ModelLoadDoneMsg{ModelID: key}
	}
}

func (a *App) readNextModelStatusCmd() tea.Cmd {
	return func() tea.Msg {
		status, ok := <-a.modelStatusChan
		if !ok {
			return nil
		}
		return views.ModelLoadStatusMsg{Status: status}
	}
}

// View renders the entire application window.
func (a *App) View() string {
	theme := styles.DefaultTheme

	// 1. Overlay Overrides
	if a.showConfirm && a.confirmView != nil {
		return a.confirmView.View()
	}
	if a.showUndoHistory && a.undoHistoryView != nil {
		return a.undoHistoryView.View()
	}

	// 2. Top Mode Tab Selector Header
	tabs := []string{
		" [Ctrl+A] ASK (Chat) ",
		" [Ctrl+P] PLAN ",
		" [Ctrl+B] BUILD ",
	}
	
	activeTab := 0
	if a.activeView == ViewChat {
		activeTab = 0
	} else if a.activeView == ViewPlanChat || a.activeView == ViewPlan {
		activeTab = 1
	} else if a.activeView == ViewBuild {
		activeTab = 2
	}

	var renderedTabs []string
	for i, tab := range tabs {
		isActive := i == activeTab && (a.activeView == ViewChat || a.activeView == ViewPlanChat || a.activeView == ViewPlan || a.activeView == ViewBuild)
		if isActive {
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

	// 3. Active Screen Content
	var content string
	if a.isLoadingModel {
		content = lipgloss.JoinVertical(lipgloss.Center,
			"\n\n\n",
			"🤖 Model Auto-Swapping...",
			theme.SubtitleStyle.Render(a.modelLoadStatus),
			"\n\n\n",
		)
	} else {
		switch a.activeView {
		case ViewHome:
			content = a.homeView.View()
		case ViewChat:
			content = a.chatView.View()
		case ViewPlanChat:
			content = a.planChatView.View()
		case ViewPlan:
			content = a.planView.View()
		case ViewBuild:
			content = a.buildView.View()
		case ViewModelSelect:
			content = a.modelSelectView.View()
		case ViewMetrics:
			content = a.metricsView.View()
		}
	}

	// 4. Context Bar (Visible in Chat and Build screens)
	ctxBar := ""
	if (a.activeView == ViewChat || a.activeView == ViewBuild) && a.cfg.UI.ShowContextBar {
		m := a.modelManager.Metrics().Get()
		
		sysTokens := 400  // baseline estimate
		
		// Load actual project context tokens
		projectCtx, _ := agent.LoadProjectContext(".", a.contextManager, a.cfg.ContextBudget.ProjectContextMaxTokens)
		alloc := a.budgetManager.Allocate(projectCtx, "", "")

		histTokens := m.TokensUsed - sysTokens - alloc.ProjectTokens - alloc.MemoryTokens - alloc.RAGTokens
		if histTokens < 0 {
			histTokens = 0
		}
		
		ctxBar = a.contextBar.Render(
			m.TokensUsed,
			m.ContextLimit,
			sysTokens,
			histTokens,
			alloc.MemoryTokens,
			alloc.RAGTokens,
		)
	}

	// 5. Status Bar (Always on bottom)
	m := a.modelManager.Metrics().Get()
	activeModeStr := "home"
	switch a.activeView {
	case ViewChat:
		activeModeStr = "ask"
	case ViewPlanChat:
		activeModeStr = "plan-input"
	case ViewPlan:
		activeModeStr = "plan-review"
	case ViewBuild:
		activeModeStr = "build"
	case ViewModelSelect:
		activeModeStr = "models"
	case ViewMetrics:
		activeModeStr = "metrics"
	}
	
	loadedID := a.selectedModelID
	if m.ModelID != "" {
		loadedID = m.ModelID
	}
	
	speed := a.chatView.CurrentSpeed
	sBar := a.statusBar.Render(activeModeStr, loadedID, m.RAMUsedGB, speed, a.isOnline)

	// Combine components
	var res string
	if a.parseWarningMsg != "" {
		bannerStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#121212")).
			Background(theme.WarningColor).
			Padding(0, 1)
		if a.width > 0 {
			bannerStyle = bannerStyle.Width(a.width)
		}
		banner := bannerStyle.Render("⚠️  " + a.parseWarningMsg + " (Press any key to dismiss)")
		res = lipgloss.JoinVertical(lipgloss.Left,
			header,
			"\n",
			banner,
			"\n",
			content,
		)
	} else {
		res = lipgloss.JoinVertical(lipgloss.Left,
			header,
			"\n",
			content,
		)
	}

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
