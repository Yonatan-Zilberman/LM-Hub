# Project Status: LM Hub TUI

This file tracks the active status of the LM Hub codebase, detailing completed packages, directory structure, and immediate upcoming work.

---

## Active Status

* **Current Phase**: Phase 8 — Production Polish (Complete)
* **Status**: Compilation is clean, unit tests pass. Primary CLI is `lmh` (`cmd/lmh/`). TUI redesign, distribution pipeline, bug fixes, and documentation sync are complete.

---

## Codebase Directory & Component Map

### `cmd/lmh/`
* [main.go](file:///Users/yonatanzilberman/Documents/LM-Hub/cmd/lmh/main.go) / [cli.go](file:///Users/yonatanzilberman/Documents/LM-Hub/cmd/lmh/cli.go) / [tui.go](file:///Users/yonatanzilberman/Documents/LM-Hub/cmd/lmh/tui.go): App CLI routing and TUI bootstrap.
* Per-command files: `cmd_ask.go`, `cmd_plan.go`, `cmd_build.go`, `cmd_memory.go`, `cmd_index.go`, `cmd_config.go`, `cmd_models.go`, `cmd_sessions.go`, `cmd_init.go`.

### `internal/api/`
* [client.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/api/client.go) / [chat.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/api/chat.go) / [models.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/api/models.go) / [embeddings.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/api/embeddings.go): LM Studio API v1 integration. SSE streaming completions, loaded model configuration queries, client-side metrics calculation (TTFT, tokens/sec), and embeddings request client.

### `internal/config/`
* [schema.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/config/schema.go) / [config.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/config/config.go): Application schema defaults and Viper configuration parser.

### `internal/modelmanager/`
* [manager.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modelmanager/manager.go) / [watcher.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modelmanager/watcher.go): Manages active model unloading/loading with status channels. Telemetry Metrics tracker.

### `internal/agent/`
* [context.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/agent/context.go): Token counting (`tiktoken-go`) and 4-stage context escalation strategy.
* [budget.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/agent/budget.go): Budget allocator for Project Context, Memory, and RAG.
* [projectctx.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/agent/projectctx.go): Loads `.lmhub/context.md` from the project root.
* [prompts.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/agent/prompts.go): Prompt templates for Ask, Plan, and Build modes.
* [parser.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/agent/parser.go): 5-layer tool call parser with parsing failure metrics.

### `internal/modes/`
* `ask/` - [mode.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modes/ask/mode.go): Ask chat loop with context summarization.
* `plan/` - [schema.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modes/plan/schema.go) / [mode.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modes/plan/mode.go): Plan generation, JSON validation, retry loop.
* `build/` - [mode.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modes/build/mode.go) / [session.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modes/build/session.go): Build ReAct loop with thread-safe history.

### `internal/memory/`
* [store.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/memory/store.go) / [memory.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/memory/memory.go) / [extractor.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/memory/extractor.go): Project and global bbolt memory stores with LLM fact extraction.

### `internal/templates/`
* [library.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/templates/library.go): Built-in and user YAML prompt template loader.

### `internal/tools/`
* [types.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/types.go) / [registry.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/registry.go): Tool interfaces and registry.
* [filesystem.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/filesystem.go), [shell.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/shell.go), [git.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/git.go), [docker.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/docker.go), [web.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/web.go): Tool implementations.
* [ask_user.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/ask_user.go): Interactive user prompt tool for Build mode.
* [undo.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/undo.go): Thread-safe undo stack.

### `internal/rag/`
* [store.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/rag/store.go) / [chunker.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/rag/chunker.go) / [indexer.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/rag/indexer.go) / [retriever.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/rag/retriever.go) / [watcher.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/rag/watcher.go): RAG indexing and retrieval.

### `internal/safety/`
* [guardrails.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/safety/guardrails.go) / [confirm.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/safety/confirm.go): Safety classification and confirmation modals.

### `internal/session/`
* [session.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/session/session.go) / [history.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/session/history.go): Session persistence (JSON).

### `pkg/platform/`
* [platform.go](file:///Users/yonatanzilberman/Documents/LM-Hub/pkg/platform/platform.go) / [darwin.go](file:///Users/yonatanzilberman/Documents/LM-Hub/pkg/platform/darwin.go) / [linux.go](file:///Users/yonatanzilberman/Documents/LM-Hub/pkg/platform/linux.go) / [windows.go](file:///Users/yonatanzilberman/Documents/LM-Hub/pkg/platform/windows.go): Cross-platform paths and shells.

### `internal/ui/`
* [app.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/ui/app.go): Bubbletea root model, mode routing, overlay delegation.
* [layout.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/ui/layout.go): LayoutManager for single/split-right panel layouts.
* [overlay.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/ui/overlay.go): OverlayManager for floating modal state and dimmed rendering.
* `views/` — ChatView, PlanChatView, PlanView, BuildView, ConfirmView, UndoHistoryView, ModelSelectView, MetricsView, HomeView, [help.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/ui/views/help.go), MemoryView, TemplatesView.
* `components/` — context bar, status bar, spinner, codeblock, diffview.
* `styles/` — [theme.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/ui/styles/theme.go) with PanelHeaderStyle, KeybindBarStyle, DimmedOverlayStyle, FloatingModalStyle.

---

## Technical Debt & Deferred Items

* **Named Plan Files**: Plan mode saves plans using timestamped names. Custom-named plans are deferred.
* **Multi-language Tree-sitter**: Go-only behind `treesitter` build tag; Python/JS/TS deferred.

---

## Verification Commands

* **Compile Codebase**: `go build ./...`
* **Static Analysis**: `go vet ./...`
* **Run Unit Tests**: `go test ./... -v`
