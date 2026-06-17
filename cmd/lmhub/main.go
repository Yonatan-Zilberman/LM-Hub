package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yonatanzilberman/lmhub/internal/agent"
	"github.com/yonatanzilberman/lmhub/internal/api"
	"github.com/yonatanzilberman/lmhub/internal/config"
	"github.com/yonatanzilberman/lmhub/internal/modes/ask"
	"github.com/yonatanzilberman/lmhub/internal/modelmanager"
	"github.com/yonatanzilberman/lmhub/internal/ui"
)

func main() {
	configFlag := flag.String("config", "", "Path to custom configuration file")
	flag.Parse()

	// 1. Load config
	cfg, err := config.Load(*configFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize API Client
	client := api.NewClient(cfg.LMStudio.BaseURL, cfg.LMStudio.TimeoutSeconds)

	// 3. Initialize Model Manager infrastructure
	metrics := &modelmanager.Metrics{}
	registry := modelmanager.NewRegistry(client)
	manager := modelmanager.NewManager(client, registry, metrics)

	// 4. Start model manager watcher in background
	watcherCtx, cancelWatcher := context.WithCancel(context.Background())
	defer cancelWatcher()
	watcher := modelmanager.NewWatcher(client, metrics, cfg.LMStudio.MetricsPollIntervalMs)
	go watcher.Start(watcherCtx)

	// 5. Initialize Context Manager
	ctxManager, err := agent.NewContextManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing tokenizer: %v\n", err)
		os.Exit(1)
	}

	// 6. Initialize Ask Mode
	askMode := ask.NewAskMode(client, manager, ctxManager)

	// 7. Initialize Bubbletea Application
	appModel, err := ui.NewApp(cfg, client, manager, askMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing TUI application: %v\n", err)
		os.Exit(1)
	}

	// 8. Launch Bubbletea TUI with Alt Screen
	p := tea.NewProgram(appModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI Execution Error: %v\n", err)
		os.Exit(1)
	}
}
