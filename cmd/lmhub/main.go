package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"go.etcd.io/bbolt"
	"github.com/yonatanzilberman/lmhub/internal/agent"
	"github.com/yonatanzilberman/lmhub/internal/api"
	"github.com/yonatanzilberman/lmhub/internal/config"
	"github.com/yonatanzilberman/lmhub/internal/modes/ask"
	"github.com/yonatanzilberman/lmhub/internal/modes/plan"
	"github.com/yonatanzilberman/lmhub/internal/modelmanager"
	"github.com/yonatanzilberman/lmhub/internal/rag"
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

	// Check for CLI subcommands
	args := flag.Args()
	if len(args) > 0 && args[0] == "index" {
		indexFlags := flag.NewFlagSet("index", flag.ExitOnError)
		watchOpt := indexFlags.Bool("watch", false, "Index + watch for changes")
		clearOpt := indexFlags.Bool("clear", false, "Wipe and re-index from scratch")
		statsOpt := indexFlags.Bool("stats", false, "Show indexing statistics")

		_ = indexFlags.Parse(args[1:])

		projectRoot, err := os.Getwd()
		if err != nil {
			projectRoot = "."
		}

		lmhubDir := filepath.Join(projectRoot, ".lmhub")
		if err := os.MkdirAll(lmhubDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating .lmhub directory: %v\n", err)
			os.Exit(1)
		}

		dbPath := filepath.Join(lmhubDir, "index.db")
		db, err := bbolt.Open(dbPath, 0600, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening RAG database: %v\n", err)
			os.Exit(1)
		}

		ragStore, err := rag.NewStore(db)
		if err != nil {
			db.Close()
			fmt.Fprintf(os.Stderr, "Error initializing RAG store: %v\n", err)
			os.Exit(1)
		}

		if *statsOpt {
			filesCount, chunksCount, err := ragStore.GetStats()
			db.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading statistics: %v\n", err)
				os.Exit(1)
			}
			lastTime, _ := ragStore.GetLastIndexTime()
			fmt.Printf("RAG Index Statistics:\n")
			fmt.Printf("  Files indexed:  %d\n", filesCount)
			fmt.Printf("  Chunks created: %d\n", chunksCount)
			if !lastTime.IsZero() {
				fmt.Printf("  Last indexed:   %s\n", lastTime.Format(time.RFC822))
			} else {
				fmt.Printf("  Last indexed:   Never\n")
			}
			return
		}

		if *clearOpt {
			err := db.Update(func(tx *bbolt.Tx) error {
				_ = tx.DeleteBucket([]byte("chunks"))
				_ = tx.DeleteBucket([]byte("meta"))
				_, _ = tx.CreateBucket([]byte("chunks"))
				_, _ = tx.CreateBucket([]byte("meta"))
				return nil
			})
			if err != nil {
				db.Close()
				fmt.Fprintf(os.Stderr, "Error clearing database: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("RAG database cleared.")
			if !*watchOpt {
				db.Close()
				return
			}
		}

		ragChunker, err := rag.NewChunker()
		if err != nil {
			db.Close()
			fmt.Fprintf(os.Stderr, "Error initializing RAG chunker: %v\n", err)
			os.Exit(1)
		}

		embModel := cfg.LMStudio.EmbeddingModel
		if embModel == "" {
			embModel = "text-embedding-nomic-embed-text-v1.5"
		}
		indexer := rag.NewIndexer(client, ragStore, ragChunker, embModel, cfg.RAG.ExcludePatterns, cfg.RAG.MaxTokens, 64)

		fmt.Printf("Ensuring embedding model '%s' is loaded...\n", embModel)
		metrics := &modelmanager.Metrics{}
		registry := modelmanager.NewRegistry(client)
		manager := modelmanager.NewManager(client, registry, metrics)

		statusChan := make(chan string, 100)
		go func() {
			for status := range statusChan {
				fmt.Printf("  %s\n", status)
			}
		}()
		err = manager.EnsureModel(context.Background(), embModel, 8192, statusChan)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load embedding model: %v. Attempting indexing anyway.\n", err)
		}

		fmt.Printf("Indexing codebase in %s...\n", projectRoot)
		err = indexer.IndexWalk(context.Background(), projectRoot, func(info rag.ProgressInfo) {
			if info.CurrentFile == "Complete" {
				fmt.Printf("\r[+] Indexing complete. Total files: %d\n", info.TotalFiles)
			} else {
				fmt.Printf("\rIndexing [%d/%d]: %s...", info.FilesProcessed+1, info.TotalFiles, info.CurrentFile)
			}
		})
		if err != nil {
			db.Close()
			fmt.Fprintf(os.Stderr, "\nError during indexing: %v\n", err)
			os.Exit(1)
		}

		if *watchOpt {
			fmt.Println("Starting file watcher for incremental indexing. Press Ctrl+C to stop.")
			watcher, err := rag.NewWatcher(projectRoot, indexer)
			if err != nil {
				db.Close()
				fmt.Fprintf(os.Stderr, "Error creating watcher: %v\n", err)
				os.Exit(1)
			}
			err = watcher.Start(context.Background())
			db.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Watcher exited with error: %v\n", err)
				os.Exit(1)
			}
		} else {
			db.Close()
		}

		return
	}

	// 3. Initialize Model Manager infrastructure
	metrics := &modelmanager.Metrics{}
	registry := modelmanager.NewRegistry(client)
	manager := modelmanager.NewManager(client, registry, metrics)

	// 4. Start model manager watcher in background
	watcherCtx, cancelWatcher := context.WithCancel(context.Background())
	defer cancelWatcher()
	watcher := modelmanager.NewWatcher(client, metrics, cfg.LMStudio.MetricsPollIntervalMs)
	go watcher.Start(watcherCtx)

	// Use current working directory as project root
	projectRoot, err := os.Getwd()
	if err != nil {
		projectRoot = "."
	}

	// Initialize RAG database
	lmhubDir := filepath.Join(projectRoot, ".lmhub")
	if err := os.MkdirAll(lmhubDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating .lmhub directory: %v\n", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(lmhubDir, "index.db")
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening RAG database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ragStore, err := rag.NewStore(db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing RAG store: %v\n", err)
		os.Exit(1)
	}



	embModel := cfg.LMStudio.EmbeddingModel
	if embModel == "" {
		embModel = "text-embedding-nomic-embed-text-v1.5"
	}
	retriever := rag.NewRetriever(client, ragStore, embModel, cfg.RAG.MinScore)

	// 5. Initialize Context Manager and Budget Manager
	ctxManager, err := agent.NewContextManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing tokenizer: %v\n", err)
		os.Exit(1)
	}
	budgetManager := agent.NewBudgetManager(ctxManager, &cfg.ContextBudget)

	// 6. Initialize Modes
	askMode := ask.NewAskMode(client, manager, ctxManager, budgetManager, cfg)
	planMode := plan.NewPlanMode(client, manager, ctxManager, budgetManager, cfg, retriever)

	// 7. Initialize Bubbletea Application
	appModel, err := ui.NewApp(cfg, client, manager, askMode, planMode, budgetManager, ctxManager, retriever, projectRoot)
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
