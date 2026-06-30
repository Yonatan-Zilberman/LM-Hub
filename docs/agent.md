# Agent Documentation & Rules

This is a living document tracking the current status, rules, architecture decisions, and known issues for the LM Hub TUI project.

---

## Agent Rules

Any AI coding agent working on the LM Hub codebase **MUST** strictly adhere to the following rules:

1. **Always update `agent.md`** after completing any unit of work, updating the Progress Log, Architecture Decisions, and Known Issues/Tech Debt sections.
2. **Read `agent.md` first** before starting any work session to understand current status and blockers.
3. **Follow the implementation plan phases in order** to prevent skipping structural dependencies.
4. **Write tests alongside code**, not as an afterthought. Place `_test.go` files in the same directory as the code they test.
5. **Every exported package, type, variable, and function must have a doc comment** in accordance with Go conventions.
6. **Run `go vet` and `go build`** before marking any work complete. The codebase must compile cleanly.
7. **One package per concern** — keep directories clean and align with the planned repository structure.
8. **Wrap errors with context** using `fmt.Errorf("operation name: %w", err)` to preserve stacks and provide clear logging.
9. **Never block the Bubbletea `Update` loop**. Perform all I/O, network, and intensive CPU tasks in goroutines sending `tea.Msg`s to the app.
10. **Enforce workspace scope pinning** on all filesystem and shell operations to prevent directory traversal or system damage.
11. **Commit atomically** with descriptive commit messages (e.g., following Conventional Commits format).
12. **Keep functions small** (aim for under 60 lines where possible). Extract helper functions.
13. **Use `context.Context`** for all API, HTTP, and background operations to support clean cancellations.
14. **Keep `config.go` and `schema.go` in sync**. Any added configuration fields must be reflected in the schema defaults.
15. **Do not add unapproved dependencies**. Any library outside of the implementation plan must be discussed and documented.
16. **Handle LM Studio being offline gracefully**. Show a disconnected state or warnings rather than crashing.
17. **Make UI/TUI components testable** using standard Bubbletea testing patterns where possible.
18. **Keep `README.md` updated** as user-facing CLI flags or features are added or changed.
19. **Log decisions and trade-offs** in the Architecture Decisions Log below when deviating from the implementation plan.
20. **Never hardcode paths, URLs, ports, or model names** — always read from configuration.

---

## Current Status

- **Current Phase**: Phase 8 — Production Polish (Complete)
- **Milestone**: `lmh` CLI rename, TUI redesign, distribution pipeline, bug fixes, and documentation sync.
- **Status**: All Phase 8 deliverables complete. Build and tests pass. Changes are uncommitted pending user review.

---

## Progress Log

### 2026-07-01 (Phase 8 Production Polish)
- Renamed primary CLI binary and entry point from `lmhub` to `lmh` (`cmd/lmh/`).
- Fixed plan-mode streaming cancellation, SSE read busy-loop, ask-mode context summarization, and plan retry inference params.
- Added `mode_inference.ask` config schema field.
- Fixed `goreleaser.yaml` (`main` key, dual `lmh`/`lmhub` binaries), rewrote `install.sh` for GitHub Releases with `--source` fallback.
- Extracted `internal/ui/layout.go` (LayoutManager) and `internal/ui/overlay.go` (OverlayManager).
- Added theme style tokens: `PanelHeaderStyle`, `KeybindBarStyle`, `DimmedOverlayStyle`, `FloatingModalStyle`.
- Split `cmd/lmh/commands.go` into per-command files (`cmd_ask.go`, `cmd_plan.go`, etc.).
- Revamped README with Features, Quick Start, Contributing, and License sections.
- TUI now unloads all LM Studio models automatically on exit via `modelmanager.UnloadAll`.

### 2026-06-18 (Phase 7 Polish & Platform Complete)
- Implemented TUI session persistence with JSON file saving/loading in `internal/session/session.go` and directory listing/cleanup in `internal/session/history.go`.
- Structured and wired session save, load, and auto-save hotkeys/slash commands (`/save`, `/load`, `Ctrl+S`, auto-save on `Ctrl+Q`) in `internal/ui/app.go` and `internal/ui/views/chat.go`.
- Restructured all application subcommands (`ask`, `plan`, `build`, `memory`, `index`, `config`, `sessions`, `init`) using Cobra CLI library in `cmd/lmh/cli.go` and `cmd/lmh/tui.go`.
- Added headless/non-interactive ask, plan, and build execution modes to the CLI.
- Extracted and implemented setup wizard in `cmd/lmh/tui.go` that runs on first startup to initialize `config.yaml`.
- Created platform support files for Linux (`pkg/platform/linux.go`) and Windows (`pkg/platform/windows.go`), and updated command execution in `internal/tools/shell.go` to be platform-independent.
- Implemented CGO-gated (`//go:build treesitter`) AST-based chunking option in `internal/rag/chunker_treesitter.go` with sliding window fallbacks.
- Created `goreleaser.yaml` config, updated `Makefile` build targets, and expanded `README.md` into a detailed quick-start manual.

### 2026-06-18 (Phase 6 Memory + Templates Complete)
- Implemented persistent project and global scopes bbolt stores in `internal/memory/store.go`.
- Implemented `MemoryManager` coordination engine in `internal/memory/memory.go`.
- Implemented LLM-based post-session and post-build fact extractor in `internal/memory/extractor.go`.
- Implemented interactive Memory Fact Center view overlay (`Ctrl+E`) in `internal/ui/views/memory.go`.
- Implemented prompt template browser overlay (`Ctrl+T`) in `internal/ui/views/templates.go` with 20 general-based templates.
- Configured template fuzzy searching, applying templates, and auto-switching modes and loading pinned models in `internal/ui/app.go`.
- Wired memory subcommands (`list`, `add`, `forget`, `clear`) in `cmd/lmh/`.
- Verified build compiles cleanly and 100% of unit tests pass.

### 2026-06-18 (Phase 5 RAG & Embeddings Complete)
- Implemented `/v1/embeddings` API client in `internal/api/embeddings.go`.
- Created token-based sliding-window `Chunker` and `Store` (bbolt vector store) in `internal/rag/`.
- Created `Indexer` supporting codebase walks, binary/ignore skipping, and batch embeddings requests.
- Created `Retriever` executing semantic queries, cosine similarity ranking, and token budget constraints.
- Wired RAG context injection into Plan and Build modes dynamically.
- Implemented `Watcher` (fsnotify-based recursive directory file watcher).
- Integrated `index` CLI subcommands (`--watch`, `--stats`, `--clear`) in `cmd/lmh/` with auto-load sequence for embedding model.
- Addressed static analysis issues: promoted `github.com/fsnotify/fsnotify` and `go.etcd.io/bbolt` to direct dependencies in `go.mod`, and refactored if-else chains to tagged switch statements on `activeView` in `internal/ui/app.go`.
- Verified build compiles cleanly and 100% of unit tests pass.

### 2026-06-17 (Phase 4 Build Mode Extended Complete)
- Implemented 7 core git tools with go-git backend and undo integration (`git_add` undo via staged restore, `git_commit` reset last commit).
- Implemented 6 docker tools via official Docker SDK, compose wrapper, and offline sockets safety.
- Implemented 2 web tools (DuckDuckGo instant search fallback scraper, goquery fetch cleanups, caching TTL).
- Integrated `DiffView` scrollable viewport inside confirmation modals (`write_file` unified diff generation via `go-difflib`) and build session panels (`Ctrl+D` toggle key).
- Completed Plan→Build sequential step loader and requires-confirm gates.
- Implemented parse failure consecutive threshold tracker (3+ errors) and non-blocking dismissible warning banner.
- Verified build compiles cleanly and all unit tests pass (100%).

### 2026-06-17 (Phase 3 Build Mode Core + Undo Complete)
- Defined core `Tool`, `ToolResult`, and `UndoRecord` models (`internal/tools/types.go`).
- Created thread-safe `Registry` mapping names to instances with schema validations and scope check bindings (`internal/tools/registry.go`).
- Implemented 7 filesystem tools with strict directory traversal prevention and rollback inverses (`internal/tools/filesystem.go`, `path.go`).
- Implemented `run_command` tool running zsh/bash, capturing outputs, enforcing timeouts, and checking blocklist keywords (`internal/tools/shell.go`).
- Built Safety Layer (`Classifier`, `FileSizeGuard`) and user-confirm message types (`internal/safety/guardrails.go`, `confirm.go`).
- Built thread-safe `UndoStack` with pops, peeks, and batch rollbacks (`internal/tools/undo.go`).
- Implemented 5-layer tool call parser (XML/JSON fallback) and unclosed tag thought extractor (`internal/agent/parser.go`).
- Wired `BuildSession` tracking commands log, touched files, and iteration checks (`internal/modes/build/session.go`).
- Implemented background `BuildMode` ReAct loops with interactive confirmations and streaming indicators (`internal/modes/build/mode.go`).
- Built split-screen `BuildView`, centering `ConfirmView` alerts, and interactive `UndoHistoryView` rollback list (`internal/ui/views/build.go`, `confirm.go`, `undohistory.go`).
- Fully wired app routing keys (`Ctrl+B`), tab selectors, auto-loading, and metrics bars (`internal/ui/app.go`).
- Verified build compiles cleanly and all unit tests pass.

### 2026-06-17 (Phase 2 Plan Mode & Context Infrastructure Complete)
- Implemented structured Plan and PlanStep models with JSON validation, defaults injection, and correction retry loop (`internal/modes/plan/schema.go`, `mode.go`).
- Built context `BudgetManager` to coordinate Project Context, Memory, and RAG token boundaries (`internal/agent/budget.go`).
- Created Project Context Loader supporting `.lmhub/context.md` file auto-injection (`internal/agent/projectctx.go`).
- Implemented 4-stage Context escalations (Warn/Trim/NeedsSummarize/HardStop) in `ContextManager` (`internal/agent/context.go`).
- Built interactive Plan Review view (icons, reversible/non-reversible flags, confidence coloring, save controls) and Plan Chat view (`internal/ui/views/plan.go`, `planchat.go`).
- Added Plan tab to TUI header, wired model auto-swap when entering Plan mode, and integrated budget allocator context bar breakdown (`internal/ui/app.go`).
- Resolved linter warnings for string concatenation in WriteString calls (`metrics.go`, `plan.go`) and unused parameter warnings (`modelselect.go`).
- Created `project-status.md` in the project root to map component paths and status for efficient context loading.
- Verified build compiles cleanly and all 4 new test packages pass.

### 2026-06-17 (Phase 1 Foundation Complete)
- Created `agent.md` and `task.md` tracking list.
- Configured Go project dependencies (Bubbletea, Resty, Viper, Glamour, Tiktoken).
- Created config load system (`internal/config/config.go`, `schema.go`) and test.
- Implemented HTTP REST v1 API client (`internal/api/client.go`, `chat.go`, `models.go`, `telemetry.go`, `types.go`).
- Implemented Model Manager coordination flow (`internal/modelmanager/manager.go`, `registry.go`, `watcher.go`, `metrics.go`).
- Built minimal Agent prompts rendering (`internal/agent/prompts.go`) and context token trimming manager (`internal/agent/context.go`).
- Implemented Ask Mode chat controller (`internal/modes/ask/mode.go`).
- Built complete Bubbletea TUI frontend layout (`internal/ui/app.go`, `views/home.go`, `views/chat.go`, `views/modelselect.go`, `views/metrics.go`, `components/statusbar.go`, `components/contextbar.go`, `components/spinner.go`, `components/codeblock.go`, `styles/theme.go`).
- Documented project architecture and commands in `README.md`.
- Verified clean build and passing unit tests.

---

## Architecture Decisions Log

### 1. LM Studio API v1 Integration
LM Studio has transitioned model load/unload APIs from `/api/v0` to `/api/v1` in version 0.4.0+. We integrated directly with v1 endpoints (`POST /api/v1/models/load` and `POST /api/v1/models/unload`) and mapped telemetry by querying the loaded model instance configuration returned by `GET /api/v1/models`.

### 2. Client-side Metrics Estimation
Because the legacy `/api/v0/models/loaded` telemetry is no longer present in LM Studio 0.4.0+, speed (tokens/sec) and TTFT (ms) are calculated client-side during response stream processing in the API client, and total context token usage is tracked locally using `tiktoken-go`.

### 3. Local-First Vector Storage with bbolt
For the RAG indexing system, we utilized `bbolt` to store both chunk metadata and pre-computed `[]float32` embeddings. Since project codebases are typically under 10,000 chunks, a brute-force cosine similarity ranking over all chunks is extremely fast (well under 50ms in Go) and avoids the complexity/overhead of dedicated vector databases or CGO-dependent ANN libraries.

---

## Known Issues & Tech Debt

* **Deferred Named Plan Files**: In Phase 2, Plan mode saves plans using timestamped filenames (e.g., `.lmhub/plan-{timestamp}.json`). Supporting custom-named plans (e.g., `.lmhub/plans/add-jwt-auth.json`) is deferred to a future polish phase.
* **Tree-sitter Chunking (Partial)**: Go-only AST chunking is available behind the `treesitter` build tag (`make build-treesitter`). Multi-language tree-sitter support (Python/JS/TS) remains deferred.
