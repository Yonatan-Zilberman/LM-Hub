# Project Status: LM Hub TUI

This file tracks the active status of the LM Hub codebase, detailing completed packages, directory structure, and immediate upcoming work.

---

## Active Status

* **Current Phase**: Phase 3 Complete (Build Mode Core + Undo)
* **Status**: Compilation is clean, all unit tests are passing.

---

## Codebase Directory & Component Map

### `cmd/lmhub/`
* [main.go](file:///Users/yonatanzilberman/Documents/LM-Hub/cmd/lmhub/main.go): App initialization. Loads config, spawns API client, model manager watcher, tokenizer context/budget structures, instantiates Modes, tools registry, and launches the Bubbletea program.

### `internal/api/`
* [client.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/api/client.go) / [chat.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/api/chat.go) / [models.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/api/models.go): LM Studio API v1 integration. SSE streaming completions, loaded model configuration queries, and client-side metrics calculation (TTFT, tokens/sec).

### `internal/config/`
* [schema.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/config/schema.go) / [config.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/config/config.go): Application schema defaults and Viper configuration parser.

### `internal/modelmanager/`
* [manager.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modelmanager/manager.go) / [watcher.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modelmanager/watcher.go): Manages active model unloading/loading with status channels. Telemetry Metrics tracker.

### `internal/agent/`
* [context.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/agent/context.go): Token counting (`tiktoken-go`) and 4-stage context escalation strategy (`ContextOK`, `ContextWarn`, `ContextTrimmed`, `ContextNeedsSummarize`, `ContextHardStop`).
* [budget.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/agent/budget.go): Budget allocator that maps and limits context inputs dynamically based on priority (Project Context -> Memory -> RAG).
* [projectctx.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/agent/projectctx.go): Loads `.lmhub/context.md` from the project root and truncates to budget limits.
* [prompts.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/agent/prompts.go): Prompt templates for Ask, Plan, and Build modes.
* [parser.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/agent/parser.go): Extracts tool calls from model output using a 5-layer fallback parsing strategy.

### `internal/modes/`
* `ask/` - [mode.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modes/ask/mode.go): Coordinates Ask stateful conversation chat loop.
* `plan/` - [schema.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modes/plan/schema.go) / [mode.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modes/plan/mode.go): Coordinates Plan structured reasoning, JSON schema parser and validation, retry correction loop, and Gemma instruction safety injection.
* `build/` - [mode.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modes/build/mode.go) / [session.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/modes/build/session.go): Coordinates Build autonomous loop, updates updates callback, and tracks session state.

### `internal/tools/`
* [types.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/types.go) / [registry.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/registry.go) / [path.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/path.go): Defines core tool interfaces, manages tool execution validation, scope checking.
* [filesystem.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/filesystem.go): Implements files tools (read_file, write_file, list_dir, search_files, delete_file, move_file, create_dir).
* [shell.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/shell.go): Implements run_command tool with blocklist protection and process execution.
* [undo.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/tools/undo.go): Thread-safe undo operations stack.

### `internal/safety/`
* [guardrails.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/safety/guardrails.go) / [confirm.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/safety/confirm.go): Classifies execution safety tiers and handles confirmation queries structure.

### `internal/ui/`
* [app.go](file:///Users/yonatanzilberman/Documents/LM-Hub/internal/ui/app.go): Bubbletea program root coordinate, tab layout selection (`Ctrl+A` -> Ask, `Ctrl+P` -> Plan, `Ctrl+B` -> Build), model auto-swapping, and overlays routing.
* `views/` - ChatView, PlanChatView, PlanView, BuildView, ConfirmView, UndoHistoryView, ModelSelectView, MetricsView, HomeView.
* `components/` - context bar progress display, status bar details, spinner, codeblock, and markdown renderer.

---

## Technical Debt & Deferred Items

* **Named Plan Files**: Currently, Plan mode saves plans using timestamped names (e.g., `.lmhub/plan-{timestamp}.json`). Support for custom-named plans (e.g., `.lmhub/plans/add-jwt-auth.json`) is deferred.

---

## Verification Commands

* **Compile Codebase**: `go build ./...`
* **Static Analysis**: `go vet ./...`
* **Run Unit Tests**: `go test ./... -v`
