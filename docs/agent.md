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

- **Current Phase**: Phase 3 — Build Mode Core + Undo
- **Milestone**: Autonomous Build Mode loop, Safety Classifier, and Rollback engine fully functional
- **Status**: All Phase 3 items completed. Full ReAct agent loop execution with streaming context updates and user confirmation gates is operational. Per-tool undo snapshots, sequential UndoHistory rollback list (`Ctrl+Z`), safety classifier (escalating blocklisted shell calls), 7 filesystem tools, and run_command tool are complete and passing all package unit tests.

---

## Progress Log

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

---

## Known Issues & Tech Debt

* **Deferred Named Plan Files**: In Phase 2, Plan mode saves plans using timestamped filenames (e.g., `.lmhub/plan-{timestamp}.json`). Supporting custom-named plans (e.g., `.lmhub/plans/add-jwt-auth.json`) is deferred to a future polish phase.
