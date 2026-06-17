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

- **Current Phase**: Phase 1 — Foundation
- **Milestone**: Bootstrap & Ask Mode Complete
- **Status**: Completed scaffolding, client interfaces, telemetry, ask mode loop, TUI views, styles, and verified compilation and test status.

---

## Progress Log

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

*None yet.*
