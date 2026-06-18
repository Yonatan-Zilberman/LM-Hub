# Project Status: LM Hub TUI

This file tracks the active status of the LM Hub codebase, detailing completed packages, directory structure, and immediate upcoming work.

---

## Active Status

* **Current Phase**: Phase 7 Complete (Polish & Platform)
* **Status**: Compilation is clean, 100% of unit tests are passing. CLI commands suite, platform support config files, first-run wizard, and session persistence are fully implemented and integrated.

---

## Codebase Directory & Component Map

### `cmd/lmhub/`
* [main.go](file:///Users/yonatanzilberman/Documents/LM-Hub/cmd/lmhub/main.go) / [cli.go](file:///Users/yonatanzilberman/Documents/LM-Hub/cmd/lmhub/cli.go) / [commands.go](file:///Users/yonatanzilberman/Documents/LM-Hub/cmd/lmhub/commands.go): App CLI routing and non-interactive executions. Boots TUI, runs ask/plan/build, handles memory and index subcommands, and configures setup wizard.

### `internal/api/`
* [client.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/api/client.go) / [chat.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/api/chat.go) / [models.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/api/models.go) / [embeddings.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/api/embeddings.go): LM Studio API v1 integration. SSE streaming completions, loaded model configuration queries, client-side metrics calculation (TTFT, tokens/sec), and embeddings request client.

### `internal/config/`
* [schema.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/config/schema.go) / [config.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/config/config.go): Application schema defaults and Viper configuration parser.

### `internal/modelmanager/`
* [manager.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modelmanager/manager.go) / [watcher.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modelmanager/watcher.go): Manages active model unloading/loading with status channels. Telemetry Metrics tracker.

### `internal/agent/`
* [context.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/agent/context.go): Token counting (`tiktoken-go`) and 4-stage context escalation strategy (`ContextOK`, `ContextWarn`, `ContextTrimmed`, `ContextNeedsSummarize`, `ContextHardStop`).
* [budget.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/agent/budget.go): Budget allocator that maps and limits context inputs dynamically based on priority (Project Context -> Memory -> RAG).
* [projectctx.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/agent/projectctx.go): Loads `.lmhub/context.md` from the project root and truncates to budget limits.
* [prompts.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/agent/prompts.go): Prompt templates for Ask, Plan, and Build modes.
* [parser.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/agent/parser.go): Extracts tool calls from model output using a 5-layer fallback parsing strategy with parsing failure metrics.

### `internal/modes/`
* `ask/` - [mode.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modes/ask/mode.go): Coordinates Ask stateful conversation chat loop.
* `plan/` - [schema.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modes/plan/schema.go) / [mode.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modes/plan/mode.go): Coordinates Plan structured reasoning, JSON schema parser and validation, retry correction loop, and Gemma instruction safety injection.
* `build/` - [mode.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modes/build/mode.go) / [session.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modes/build/session.go): Coordinates Build autonomous loop, updates updates callback, tracks session state, and executes sequential plan steps.

### `internal/tools/`
* [types.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/types.go) / [registry.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/registry.go) / [path.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/path.go): Defines core tool interfaces, manages tool execution validation, scope checking.
* [filesystem.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/filesystem.go): Implements files tools (read_file, write_file, list_dir, search_files, delete_file, move_file, create_dir).
* [shell.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/shell.go): Implements run_command tool with blocklist protection and process execution.
* [undo.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/undo.go): Thread-safe undo operations stack.
* [git.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/git.go): Implements 7 git tools (status, diff, add, commit, log, branch, stash) via go-git.
* [docker.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/docker.go): Implements 6 docker tools (ps, logs, exec, build, compose, pull) via Docker SDK.
* [web.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/web.go): Implements web search and page fetch.

### `internal/rag/`
* [store.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/rag/store.go): bbolt-backed vector + metadata store.
* [chunker.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/rag/chunker.go): Token-based sliding-window chunker aligning to line boundaries.
* [indexer.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/rag/indexer.go): Handles walking project directory, binary checks, and embedding batches.
* [retriever.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/rag/retriever.go): Semantic retriever matching and ranking chunks under token budgets.
* [watcher.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/rag/watcher.go): fsnotify-based recursive project watcher updating store incrementally.
* [cosine.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/rag/cosine.go): Pure-Go float32 cosine similarity calculation.

### `internal/safety/`
* [guardrails.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/safety/guardrails.go) / [confirm.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/safety/confirm.go): Classifies execution safety tiers and handles confirmation queries structure.

### `internal/session/`
* [session.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/session/session.go) / [history.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/session/history.go): Local conversation history logging and JSON serializations.

### `pkg/platform/`
* [platform.go](file:///Users/yonatanzilberman/Documents/LM-Hub/pkg/platform/platform.go) / [darwin.go](file:///Users/yonatanzilberman/Documents/LM-Hub/pkg/platform/darwin.go) / [linux.go](file:///Users/yonatanzilberman/Documents/LM-Hub/pkg/platform/linux.go) / [windows.go](file:///Users/yonatanzilberman/Documents/LM-Hub/pkg/platform/windows.go): Cross-platform configuration paths and shell executors.

### `internal/ui/`
* [app.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/ui/app.go): Bubbletea program root coordinate, tab layout selection (`Ctrl+A` -> Ask, `Ctrl+P` -> Plan, `Ctrl+B` -> Build), model auto-swapping, overlays routing, and parse warning banner.
* `views/` - ChatView, PlanChatView, PlanView, BuildView, ConfirmView, UndoHistoryView, ModelSelectView, MetricsView, HomeView.
* `components/` - context bar progress display, status bar details, spinner, codeblock, markdown renderer, and diffview (unified diff color visualizer).

---

## Technical Debt & Deferred Items

* **Named Plan Files**: Currently, Plan mode saves plans using timestamped names (e.g., `.lmhub/plan-{timestamp}.json`). Support for custom-named plans (e.g., `.lmhub/plans/add-jwt-auth.json`) is deferred.

---

## Verification Commands

* **Compile Codebase**: `go build ./...`
* **Static Analysis**: `go vet ./...`
* **Run Unit Tests**: `go test ./... -v`
