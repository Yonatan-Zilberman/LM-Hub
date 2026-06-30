# LMHub — Implementation Plan (v2)

> **Documentation note (2026-07):** This plan predates several shipped changes. The primary CLI binary is now **`lmh`** (not `lmhub`). LM Studio APIs use **`/api/v1/`** (not v0). Inference metrics overlay is **`Ctrl+G`** (not Ctrl+I). See [docs/agent.md](agent.md) for the authoritative architecture decisions log.

> A local-first AI agent harness for LM Studio with Ask, Plan, and Build modes.
> Updated to include: live context window telemetry, RAG/embeddings, inference metrics, project context files, per-tool undo/rollback, persistent agent memory, and prompt template library.

---

## 1. Project Overview

**LMHub** is a terminal-based AI agent harness that wraps LM Studio's local model server with a structured three-mode workflow designed for coding and IT operations. It communicates with LM Studio via its OpenAI-compatible API and management API, handles automatic model loading/unloading between modes, and provides a safe, sandboxed tool-execution layer in Build mode.

### Design Principles
- **One model loaded at a time** — switching modes with different pinned models triggers an unload/load sequence automatically
- **Fast and low-footprint** — compiled Go binary, no runtime, minimal RAM overhead outside of the model itself
- **Explicit over implicit** — destructive operations always require confirmation; the agent never acts silently
- **Offline-first** — fully functional without internet; web search/fetch is an opt-in tool
- **Context-aware** — live token usage, injection budgets, and model telemetry are visible at all times
- **Recoverable** — every Build mode action is undoable; nothing is permanently lost without intent
- **Portable** — macOS primary target; Linux and Windows supported via build flags and OS abstraction layer

---

## 2. Technology Stack

| Layer | Choice | Reason |
|---|---|---|
| Language | **Go 1.22+** | Compiled single binary, low memory, fast startup, excellent concurrency |
| TUI Framework | **Bubbletea** (Charmbracelet) | Elm-like architecture, mature, testable |
| TUI Styling | **Lipgloss** (Charmbracelet) | CSS-like terminal styling |
| TUI Components | **Bubbles** (Charmbracelet) | Spinner, viewport, textinput, list, progress bar components |
| Markdown Render | **Glamour** (Charmbracelet) | Renders model markdown output in terminal |
| HTTP Client | `net/http` + **resty/v2** | Ergonomic REST calls with retry logic |
| Config | **viper** + **YAML** | Flexible config with env var overrides |
| Git Operations | **go-git/v5** | Pure Go git, no system git dependency |
| Docker Client | **docker/client** (official SDK) | Full Docker Engine API access |
| Web Fetch | `net/http` + **goquery** | HTML parsing for web tool |
| Web Search | **DuckDuckGo Instant API** (no key) or configurable | No API key required by default |
| JSON Parsing | `encoding/json` + **gjson** | Fast path extraction for tool call parsing |
| Token Counting | **tiktoken-go** | Approximate token counts for context management |
| Vector Store | **bbolt** (embedded key/value) + cosine similarity | Local-only vector index, no external DB |
| Embeddings | LM Studio `/v1/embeddings` + any compatible embedding model | Configured via `embedding_model` in config; no external service required |
| Persistence | **bbolt** | Single-file embedded DB for memory, vector index, sessions |
| Testing | `testing` + **testify** | Standard Go + assertion library |
| Build/Release | **goreleaser** | Cross-platform binary releases |

---

## 3. Repository Structure

```
lmhub/
├── cmd/
│   └── lmhub/
│       └── main.go                    # Entrypoint, CLI flag parsing, TUI bootstrap
│
├── internal/
│   ├── api/
│   │   ├── client.go                  # Base HTTP client (retries, timeout, base URL)
│   │   ├── chat.go                    # /v1/chat/completions (streaming + non-streaming)
│   │   ├── embeddings.go              # /v1/embeddings — embedding model calls (model ID from config)
│   │   ├── models.go                  # /api/v0/models — list, load, unload, status
│   │   ├── telemetry.go               # /api/v0/models/loaded — context size, memory, speed
│   │   └── types.go                   # Shared API structs (Message, Tool, ToolCall, ModelInfo, etc.)
│   │
│   ├── agent/
│   │   ├── agent.go                   # Core agent loop (ReAct: think → act → observe → repeat)
│   │   ├── context.go                 # Context window manager (token counting, trimming, summarization)
│   │   ├── budget.go                  # Injected context budget (memory + RAG + project ctx combined cap)
│   │   ├── parser.go                  # Tool call parser (JSON-first, regex fallback, 5-layer strategy)
│   │   └── prompts.go                 # System prompt templates per mode
│   │
│   ├── modelmanager/
│   │   ├── manager.go                 # Model lifecycle: detect, load, unload, auto-switch
│   │   ├── registry.go                # In-memory registry of available/loaded models
│   │   ├── watcher.go                 # Polls LM Studio for model state changes
│   │   └── metrics.go                 # Tracks tokens/sec, TTFT, memory, context fill %
│   │
│   ├── modes/
│   │   ├── ask/
│   │   │   └── mode.go                # Ask mode: stateful chat loop, no tools
│   │   ├── plan/
│   │   │   ├── mode.go                # Plan mode: structured reasoning, approval gate
│   │   │   └── schema.go              # Plan output schema (steps, files, risks, confidence)
│   │   └── build/
│   │       ├── mode.go                # Build mode: full agentic loop with tools
│   │       ├── session.go             # Build session state (files modified, commands run, git state)
│   │       └── undo.go                # Per-tool-call undo stack (snapshots + inverse ops)
│   │
│   ├── tools/
│   │   ├── registry.go                # Tool registry — register, lookup, execute, permissions
│   │   ├── filesystem.go              # read_file, write_file, list_dir, delete_file, create_dir
│   │   ├── shell.go                   # run_command (sandboxed, with timeout and output capture)
│   │   ├── git.go                     # git_status, git_diff, git_add, git_commit, git_log, git_branch
│   │   ├── docker.go                  # docker_ps, docker_exec, docker_logs, docker_build, docker_compose
│   │   ├── web.go                     # web_search, web_fetch (HTML → markdown extraction)
│   │   └── types.go                   # Tool definition, ToolResult, ToolPermission, UndoRecord structs
│   │
│   ├── rag/
│   │   ├── indexer.go                 # Walks project dir, chunks files, generates embeddings, stores in bbolt
│   │   ├── retriever.go               # Cosine similarity search, returns top-k chunks
│   │   ├── chunker.go                 # Language-aware chunking (functions, classes, blocks)
│   │   ├── watcher.go                 # fsnotify-based file watcher, re-indexes on change
│   │   └── store.go                   # bbolt-backed vector + metadata store
│   │
│   ├── memory/
│   │   ├── memory.go                  # Persistent agent memory: store, retrieve, forget facts
│   │   ├── extractor.go               # Post-conversation fact extraction (asks model to extract key facts)
│   │   └── store.go                   # bbolt-backed memory store (keyed by project + global)
│   │
│   ├── templates/
│   │   ├── library.go                 # Template library: load, list, search, apply
│   │   ├── builtin.go                 # Built-in templates for common coding/IT tasks
│   │   └── types.go                   # Template struct (name, description, tags, prompt, mode)
│   │
│   ├── config/
│   │   ├── config.go                  # Config struct, load, validate, defaults
│   │   └── schema.go                  # Full config schema with comments
│   │
│   ├── safety/
│   │   ├── guardrails.go              # Classifies tool calls as safe/warn/dangerous
│   │   └── confirm.go                 # Interactive confirmation prompts for destructive actions
│   │
│   ├── session/
│   │   ├── session.go                 # Session state (current mode, history, loaded model)
│   │   └── history.go                 # Conversation history persistence (JSONL)
│   │
│   └── ui/
│       ├── app.go                     # Root Bubbletea model, routes between views
│       ├── views/
│       │   ├── home.go                # Home screen — mode selector, model status
│       │   ├── chat.go                # Shared chat view (used by all modes)
│       │   ├── modelselect.go         # Model browser and loader view
│       │   ├── plan.go                # Plan review and approval view
│       │   ├── confirm.go             # Destructive action confirmation overlay
│       │   ├── undohistory.go         # Build mode undo history panel (Ctrl+Z browser)
│       │   ├── metrics.go             # Inference metrics overlay panel
│       │   ├── memory.go              # Agent memory viewer/editor
│       │   └── templates.go           # Prompt template browser
│       ├── components/
│       │   ├── statusbar.go           # Bottom bar: mode, model, token fill bar, speed, LM status
│       │   ├── contextbar.go          # Live context window fill bar with token breakdown
│       │   ├── spinner.go             # Loading/thinking indicator
│       │   ├── codeblock.go           # Syntax-highlighted code renderer
│       │   └── diffview.go            # Git diff viewer (side-by-side or unified)
│       └── styles/
│           └── theme.go               # Lipgloss theme (colors, borders, padding)
│
├── pkg/
│   └── platform/
│       ├── platform.go                # OS detection interface
│       ├── darwin.go                  # macOS-specific paths, shell defaults (zsh)
│       ├── linux.go                   # Linux-specific paths, shell defaults (bash)
│       └── windows.go                 # Windows-specific paths, shell defaults (PowerShell)
│
├── config.yaml                        # Default user config (created on first run)
├── .lmhub/                            # Per-project runtime directory (gitignored)
│   ├── context.md                     # Project context file (injected into every session)
│   ├── sessions/                      # Saved conversation sessions (JSONL)
│   ├── index.db                       # bbolt: RAG vector index for this project
│   ├── memory.db                      # bbolt: persistent agent memory for this project
│   ├── logs/                          # Debug logs
│   └── cache/                         # Web fetch cache, model registry cache
│
├── templates/                         # User-defined prompt templates
│   └── *.yaml                         # One file per template
│
├── Makefile                           # build, test, lint, release targets
├── goreleaser.yaml                    # Cross-platform release config
├── go.mod
├── go.sum
└── README.md
```

---

## 4. Architecture Overview

```
┌──────────────────────────────────────────────────────────────────────┐
│                          LMHub TUI (Bubbletea)                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────┐  ┌────────┐  │
│  │Home View │  │Chat View │  │Plan View │  │Models  │  │Metrics │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────────┘  └────────┘  │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  Context Bar: [████████████░░░░░░░░░░░░] 12,400 / 32,768 tok  │  │
│  └────────────────────────────────────────────────────────────────┘  │
└─────────┼─────────────┼──────────────┼──────────────────────────────┘
          └─────────────┴──────┬────────┘
                               ▼
         ┌────────────────────────────────┐
         │         Context Budget         │
         │  system + memory + rag + hist  │  ← hard cap: never exceed model ctx
         └────────────────┬───────────────┘
                          │
          ┌───────────────▼───────────────┐     ┌─────────────────────────┐
          │          Mode Router          │     │      Model Manager       │
          │    ask  |  plan  |  build     │     │  list / load / unload    │
          └───────────────┬───────────────┘     │  metrics / telemetry     │
                          │                     └──────────┬──────────────┘
                          ▼                               │
          ┌───────────────────────────────┐               ▼
          │         Agent Loop            │    ┌─────────────────────────┐
          │   think → act → observe       │    │    LM Studio API         │
          │   (ReAct pattern)             │    │  :1234/v1/  (chat/embed) │
          └───────┬───────────────────────┘    │  :1234/api/v0/ (mgmt)   │
                  │                            └─────────────────────────┘
    ┌─────────────┼──────────────────────┐
    ▼             ▼                      ▼
┌────────┐  ┌──────────┐  ┌─────────────────────────────────┐
│  RAG   │  │  Memory  │  │         Tool Registry            │
│retriev.│  │ inject   │  │ filesystem│shell│git│docker│web  │
└────────┘  └──────────┘  └─────────────────────────────────┘
                                        │
                           ┌────────────▼────────────┐
                           │   Safety / Guardrails    │
                           │   safe / warn / danger   │
                           └────────────┬─────────────┘
                                        │
                           ┌────────────▼────────────┐
                           │    Undo Stack            │
                           │  snapshot before exec    │
                           │  inverse op on rollback  │
                           └─────────────────────────┘
```

---

## 5. LM Studio API Integration

### Endpoints Used

```
# Management API (model lifecycle)
GET  http://localhost:1234/api/v0/models           # List all installed models + metadata
POST http://localhost:1234/api/v0/models/load      # Load a model into memory
POST http://localhost:1234/api/v0/models/unload    # Unload the current model
GET  http://localhost:1234/api/v0/models/loaded    # Currently loaded model + context info

# Inference API (OpenAI-compatible)
POST http://localhost:1234/v1/chat/completions     # Chat with streaming support
POST http://localhost:1234/v1/embeddings           # Generate embeddings (nomic-embed-text)
GET  http://localhost:1234/v1/models               # List currently loaded models
```

### Live Context Window Data

The `/api/v0/models/loaded` endpoint returns real-time model state. LMHub polls this every 2 seconds while a session is active and surfaces the data in the context bar:

```go
type LoadedModelInfo struct {
    ModelID         string  `json:"model_id"`
    ContextLength   int     `json:"context_length"`     // Max tokens (e.g. 32768)
    TokensUsed      int     `json:"tokens_used"`        // Current fill
    TokensFree      int     `json:"tokens_free"`
    FillPercent     float64 `json:"fill_pct"`
    RAMUsedGB       float64 `json:"ram_used_gb"`
    TokensPerSecond float64 `json:"tokens_per_sec"`     // From last completion
    TTFT_ms         int     `json:"ttft_ms"`            // Time to first token
}
```

This is displayed as a live progress bar in the UI — color shifts green → yellow → red as the context fills.

### Model Auto-Switch Flow

```
User switches to Build mode (pinned: <build-model-id from config>)
       │
       ▼
Model Manager checks currently loaded model
       │
       ├─ Same model? → Skip, proceed immediately
       │
       └─ Different model?
              │
              ▼
       POST /api/v0/models/unload  (current model)
              │
       TUI: "Unloading <model-id>..." spinner (non-blocking)
              │
              ▼
       POST /api/v0/models/load    (target model)
              │
       TUI: "Loading <model-id>... (XX.X GB)" progress
              │
              ▼
       Poll /api/v0/models/loaded until state = "ready"
              │
              ▼
       Context bar resets → Mode activates → Session begins
```

---

## 6. Mode Specifications

### 6.1 Ask Mode

A clean, stateful conversation loop. No tools, no planning overhead. Ideal for quick questions, concept explanations, and general IT Q&A. Memory and project context are injected if available.

**Behavior:**
- Maintains full conversation history in memory for the session
- Streams model responses token by token
- Renders markdown, code blocks with syntax highlighting
- Project context file (`.lmhub/context.md`) injected into system prompt if present
- Relevant agent memory facts injected up to memory budget (default: 800 tokens)
- `/clear` resets context; `/save` exports session; `/mem` opens memory viewer
- Context window monitored live; warns at 70%, trims at 85%, summarizes at 90%

**System Prompt Template:**
```
You are an expert assistant specializing in software development and IT operations.
Be concise, accurate, and technical. Use code blocks for any code or commands.
Current working directory: {cwd}
OS: {os} | Shell: {shell}

{if project_context}
== Project Context ==
{project_context}
{endif}

{if memory_facts}
== What I know about this project ==
{memory_facts}
{endif}
```

---

### 6.2 Plan Mode

Structured reasoning with a mandatory human approval gate before any action is taken. Uses lower temperature and stricter prompting to produce reliable structured JSON output. Models with weaker instruction-following are automatically flagged via parse failure tracking (see Section 16).

**Behavior:**
1. User describes a task (or selects from prompt template library)
2. RAG retrieves relevant codebase context (top 3 chunks, ≤800 tokens) if index exists
3. Agent reasons and produces a structured `Plan` JSON object
4. TUI renders plan as an interactive checklist with risk highlights
5. User approves (`y`), edits inline, or rejects (`n`)
6. Approved plan is saved as `.lmhub/plan-{timestamp}.json` and can be handed to Build mode

**Plan Schema (JSON):**
```json
{
  "title": "string",
  "summary": "string",
  "confidence": 0.0-1.0,
  "estimated_steps": 5,
  "risks": ["string"],
  "steps": [
    {
      "id": 1,
      "description": "string",
      "type": "file_edit | shell | git | docker | info",
      "target": "path/or/command",
      "reversible": true,
      "requires_confirm": false
    }
  ],
  "files_affected": ["string"],
  "rollback_strategy": "string"
}
```

**System Prompt Template:**
```
You are a senior systems architect planning technical tasks.
Respond ONLY with a valid JSON plan matching the provided schema. No prose outside the JSON.
Be explicit about risks and irreversible operations. Confidence must reflect genuine uncertainty.
Schema: {plan_schema}

{if rag_chunks}
== Relevant codebase context ==
{rag_chunks}
{endif}
```

---

### 6.3 Build Mode

Full agentic loop with tool access, undo/rollback, and RAG context injection. The agent reasons, selects tools, executes them (with safety checks and undo snapshots), observes results, and iterates.

**Key UX principle for local models:** Build mode is designed for **short, targeted iterations**, not monolithic "do everything" tasks. The UI nudges users toward scoping tasks to 5–7 tool calls per run. Longer chains can be chained manually via Plan → Build handoff.

**Agent Loop (ReAct Pattern):**
```
SYSTEM: {build_system_prompt}  ← includes project ctx, memory, RAG, tool schemas
USER:   {task}
        ↓
ASSISTANT: <thought>reasoning here</thought>
           <tool_call>{"name": "read_file", "args": {"path": "main.go"}}</tool_call>
        ↓
[safety check → undo snapshot → execute]
        ↓
TOOL RESULT: <tool_result>{"content": "...file contents..."}</tool_result>
        ↓
ASSISTANT: <thought>I see the issue in line 42...</thought>
           <tool_call>{"name": "write_file", "args": {...}}</tool_call>
        ↓
[loop until ASSISTANT produces final answer with no tool_call]
```

**Max iterations:** Configurable (default: 15, reflecting local model limits). At 10 iterations, a non-blocking warning appears. At 15, the loop pauses and asks the user to continue or stop.

**System Prompt Template:**
```
You are an expert software engineer and systems administrator executing tasks autonomously.
Work in small, focused steps. Prefer reading before writing. One tool call per response.
You have access to the following tools: {tool_list_with_schemas}

Rules:
- Reason inside <thought> tags before every action
- Use exactly one tool per response
- Always read a file before writing it
- Never delete without explicit confirmation from the user
- If uncertain about scope, use ask_user before proceeding
- Prefer targeted edits over full file rewrites

Current working directory: {cwd} (you may not write outside this directory)
Git status: {git_status}
OS: {os} | Shell: {shell}

{if project_context}
== Project Context ==
{project_context}
{endif}

{if memory_facts}
== Known project facts ==
{memory_facts}
{endif}

{if rag_chunks}
== Relevant codebase context ==
{rag_chunks}
{endif}
```

---

## 7. Tool System

### Tool Interface (Go)

```go
type Tool interface {
    Name()        string
    Description() string
    Schema()      ToolSchema       // JSON schema for args
    Permission()  PermissionLevel  // Safe | Warn | Dangerous
    Undoable()    bool             // Whether this tool supports undo snapshots
    Execute(ctx context.Context, args map[string]any) (ToolResult, error)
    Snapshot(ctx context.Context, args map[string]any) (UndoRecord, error)  // Called before Execute if Undoable
}

type PermissionLevel int
const (
    Safe      PermissionLevel = iota  // Execute silently
    Warn                               // Show what will happen, auto-proceed (or block if config)
    Dangerous                          // Block — require explicit y/N confirmation
)

type UndoRecord struct {
    ToolName    string
    Args        map[string]any
    InverseOp   string          // "write_file", "delete_file", "git_stash_pop", etc.
    InverseArgs map[string]any  // Args to pass to inverse op
    Snapshot    []byte          // Raw file bytes if applicable
    Timestamp   time.Time
    Description string          // Human-readable: "Overwrote auth.go (312 bytes)"
}
```

### Tool Inventory

#### Filesystem Tools
| Tool | Args | Permission | Undoable | Notes |
|---|---|---|---|---|
| `read_file` | `path`, `start_line?`, `end_line?` | Safe | No | Returns content, optionally line-ranged |
| `write_file` | `path`, `content`, `mode: overwrite\|append` | Warn | **Yes** | Snapshots original before write, shows diff |
| `create_dir` | `path`, `recursive?` | Safe | **Yes** | Inverse: delete the dir if empty |
| `list_dir` | `path`, `recursive?`, `filter?` | Safe | No | Returns file tree as JSON |
| `delete_file` | `path` | Dangerous | **Yes** | Snapshots file to undo store before deletion |
| `move_file` | `src`, `dst` | Warn | **Yes** | Inverse: move back |
| `search_files` | `pattern`, `path`, `type: content\|name` | Safe | No | Grep-like search |

#### Shell Tools
| Tool | Args | Permission | Undoable | Notes |
|---|---|---|---|---|
| `run_command` | `cmd`, `cwd?`, `timeout?` | Warn/Dangerous | No | Shell cmds are not undoable; warn prominently |
| `ask_user` | `question` | Safe | No | Agent-initiated clarification |

**Shell Blocklist (always Dangerous, extra confirmation):**
```
rm -rf /, mkfs, dd if=, format, DROP TABLE, truncate, shutdown, reboot, :(){ :|:& };:
```

#### Git Tools
| Tool | Args | Permission | Undoable | Notes |
|---|---|---|---|---|
| `git_status` | `cwd?` | Safe | No | Porcelain status |
| `git_diff` | `cwd?`, `staged?`, `file?` | Safe | No | Unified diff |
| `git_add` | `paths[]` | Warn | **Yes** | Inverse: git_restore --staged |
| `git_commit` | `message`, `amend?` | Warn | **Yes** | Inverse: git reset --soft HEAD~1 |
| `git_log` | `n?`, `oneline?` | Safe | No | Recent commits |
| `git_branch` | `action: list\|create\|switch\|delete`, `name?` | Warn/Dangerous | **Yes** (create) | Delete is Dangerous and not undoable |
| `git_stash` | `action: push\|pop\|list` | Warn | No | |

#### Docker Tools
| Tool | Args | Permission | Undoable | Notes |
|---|---|---|---|---|
| `docker_ps` | `all?` | Safe | No | Lists containers |
| `docker_logs` | `container`, `tail?` | Safe | No | Streams logs |
| `docker_exec` | `container`, `cmd` | Warn | No | Executes in running container |
| `docker_build` | `context`, `tag`, `dockerfile?` | Warn | No | Builds image |
| `docker_compose` | `action: up\|down\|restart\|logs`, `file?`, `services?` | Warn/Dangerous | No | down is Dangerous |
| `docker_pull` | `image` | Warn | No | Pulls image |

#### Web Tools
| Tool | Args | Permission | Undoable | Notes |
|---|---|---|---|---|
| `web_search` | `query`, `n?` | Safe | No | DuckDuckGo, returns n results |
| `web_fetch` | `url`, `format: markdown\|raw` | Safe | No | Fetches + converts to markdown |

---

## 8. Per-Tool-Call Undo / Rollback System

Every undoable tool call in Build mode creates an `UndoRecord` before execution. These are stored in a session-scoped undo stack and browsable via `Ctrl+Z`.

### Undo Stack Behavior

```
Build session starts
       │
       ▼
Tool: write_file(auth.go, ...)
       ├─ Snapshot: read current auth.go → store bytes in UndoRecord
       ├─ Execute: write new content
       └─ Push UndoRecord to stack

Tool: git_add([auth.go])
       ├─ Snapshot: record staged state
       ├─ Execute: stage file
       └─ Push UndoRecord to stack

User presses Ctrl+Z
       │
       ▼
Pop top UndoRecord (git_add)
       ├─ Run inverse: git restore --staged auth.go
       └─ Remove from stack

User presses Ctrl+Z again
       │
       ▼
Pop next UndoRecord (write_file)
       ├─ Run inverse: write_file(auth.go, snapshot_bytes)
       └─ Remove from stack
```

### Undo UI (Ctrl+Z panel)

```
┌─ Undo History ──────────────────────────────────────────────┐
│  [5]  git_add          staged auth.go, middleware.go         │ ← most recent
│  [4]  write_file       overwrote auth.go (312 → 489 bytes)  │
│  [3]  write_file       overwrote middleware.go (198 bytes)   │
│  [2]  create_dir       created internal/auth/               │
│  [1]  write_file       created internal/auth/types.go       │
│                                                              │
│  [Enter] Undo selected   [U] Undo all   [Esc] Close         │
└──────────────────────────────────────────────────────────────┘
```

**Non-undoable actions** (shell commands, docker ops) are shown in the undo panel with a ⚠ icon and the note "cannot be undone — consider reverting manually."

---

## 9. RAG / Codebase Indexing

LMHub uses whichever embedding model is configured in `embedding_model` (defaulting to any LM Studio-compatible embedding model available on the server) to build a local vector index of any project. This allows Build and Plan modes to retrieve relevant code context rather than stuffing entire files into the prompt. Any embedding model served by LM Studio at `/v1/embeddings` is supported — the vector dimension is read from the model's first response and stored in the index metadata.

### How It Works

```
lmhub index          ← user runs once per project (or automatically on first Build entry)
       │
       ▼
Indexer walks .lmhub/../ (project root)
       │
       ├─ Skips: .git/, node_modules/, vendor/, build/, dist/, *.bin, *.png, etc.
       │
       ▼
Chunker splits each file into semantic chunks:
       ├─ Go/Python/JS/TS: function and class boundaries (tree-sitter)
       ├─ Other text: sliding window (512 tokens, 64-token overlap)
       └─ Max chunk size: 512 tokens
       │
       ▼
For each chunk: POST /v1/embeddings (configured embedding model) → N-dim vector
       │
       ▼
Store: chunk text + vector + metadata (file, line range, language) → .lmhub/index.db
       │
       ▼
fsnotify watcher re-indexes changed files incrementally
```

### Retrieval at Query Time

```go
func (r *Retriever) Query(ctx context.Context, query string, topK int) ([]Chunk, error) {
    // 1. Embed the query using the configured embedding model
    queryVec, err := r.api.Embed(ctx, query)

    // 2. Cosine similarity search over all stored vectors
    results := r.store.Search(queryVec, topK)

    // 3. Return top-k chunks sorted by similarity score
    return results, nil
}
```

### Context Budget for RAG

RAG chunks are injected into the prompt but count against a strict budget:

```yaml
rag:
  enabled: true
  top_k: 3                  # Retrieve top 3 chunks per query
  max_tokens: 1200          # Hard cap: never inject more than 1200 tokens of RAG
  min_score: 0.72           # Ignore chunks below this cosine similarity
  reindex_on_start: false   # Set true to re-index on every Build mode entry
  exclude_patterns:         # Gitignore-style patterns
    - "*.lock"
    - "*.sum"
    - "testdata/**"
```

### CLI Commands

```bash
lmhub index              # Index current project
lmhub index --watch      # Index + watch for changes
lmhub index --clear      # Wipe and re-index from scratch
lmhub index --stats      # Show: files indexed, chunks, index size, last updated
```

---

## 10. Inference Metrics & Live Context Window

### What Is Tracked

Every streaming completion updates the metrics store with:

| Metric | Source | Display |
|---|---|---|
| Tokens/sec (generation speed) | Chunk timestamps during stream | Status bar |
| Time to first token (TTFT) | Time from request → first chunk | Metrics panel |
| Total tokens generated | Count of streamed tokens | Metrics panel |
| Context fill % | `/api/v0/models/loaded` poll | Context bar (live) |
| Context tokens used | `/api/v0/models/loaded` | Context bar |
| Context max (model limit) | `/api/v0/models/loaded` | Context bar |
| RAM used by model | `/api/v0/models/loaded` | Metrics panel |

### Context Bar (always visible, below chat area)

```
Context  [████████████████████░░░░░░░░░░░░░░░░░]  12,814 / 32,768  (39%)
          ██ system  ███ history  █ memory  ██ rag           39 tok/s  TTFT 1.2s
```

Color states:
- **Green** (0–69%): Normal
- **Yellow** (70–84%): Approaching limit, auto-trim armed
- **Orange** (85–89%): Trim in progress
- **Red** (90%+): Summarizing or hard stop

The bar also shows a breakdown of how injected context is allocated: system prompt, conversation history, memory, and RAG chunks — so you can see at a glance what's consuming your context budget.

### Metrics Overlay (Ctrl+I)

```
┌─ Inference Metrics ────────────────────────────────────────┐
│  Model:           <currently loaded model id>               │
│  Architecture:    <arch> (e.g. MoE models show active params)│
│  RAM Used:        XX.X GB                                   │
│                                                            │
│  Last Completion                                           │
│  ─────────────────────────────────────────────────────     │
│  Time to first token:    1,240 ms                          │
│  Generation speed:       41.3 tok/s                        │
│  Total tokens generated: 512                               │
│  Total time:             13.6 s                            │
│                                                            │
│  Context Window                                            │
│  ─────────────────────────────────────────────────────     │
│  Capacity:               32,768 tokens                     │
│  Used:                   12,814 tokens (39%)               │
│  System prompt:          1,204 tok                         │
│  Conversation history:   9,812 tok                         │
│  Memory injection:       612 tok                           │
│  RAG injection:          1,186 tok                         │
│  Remaining:              19,954 tokens                     │
│                                                            │
│  [Esc] Close                                               │
└────────────────────────────────────────────────────────────┘
```

---

## 11. Project Context Files

A `.lmhub/context.md` file in any project root is automatically injected into the system prompt of all three modes. This replaces the need to re-explain your project every session.

### File Format

```markdown
# Project: MyApp

## What this is
A Go REST API for managing inventory. Uses PostgreSQL, Redis, Docker Compose for local dev.

## Stack
- Go 1.22, chi router, sqlx, pgx
- PostgreSQL 16, Redis 7
- Docker Compose for local dev

## Key conventions
- All handlers in internal/handlers/, one file per resource
- Database migrations in migrations/, use goose
- Errors always wrapped with fmt.Errorf("context: %w", err)
- Never use global state

## What NOT to do
- Do not modify the generated protobuf files in gen/
- Do not use the logger package, use slog
- Never commit .env files
```

### Behavior

- File is loaded on every session start if present
- Truncated to `project_context.max_tokens` (default 800) if too long
- `/context` command opens it in `$EDITOR` from within LMHub
- `lmhub init` generates a starter `context.md` by asking the model to summarize the project structure
- If no `context.md` exists, LMHub silently skips injection with no error

### Context Budget

```
Total injected context budget (all sources combined):
  project_context:  800 tokens max
  memory_facts:     800 tokens max
  rag_chunks:      1200 tokens max
  ─────────────────────────────────
  Total injection:  2800 tokens max  (hardcoded ceiling, regardless of individual settings)
```

This prevents injection features from crowding out actual conversation history on smaller context windows.

---

## 12. Persistent Agent Memory

LMHub maintains a project-scoped and a global memory store. After each session, the agent optionally extracts key facts and stores them for future sessions.

### Memory Types

```go
type MemoryFact struct {
    ID          string     // UUID
    Scope       string     // "project:{path}" or "global"
    Content     string     // The fact itself: "Auth module uses JWT, secret in .env as JWT_SECRET"
    Source      string     // "user" (explicitly told) | "extracted" (post-session extraction)
    Confidence  float64    // 0.0-1.0 (extracted facts may be wrong)
    CreatedAt   time.Time
    LastUsed    time.Time
    UseCount    int
}
```

### Memory Storage

Facts are stored in `.lmhub/memory.db` (bbolt, project-scoped) and `~/.config/lmhub/global-memory.db` (global facts). Both are plain local files, no cloud sync, no external service.

### Post-Session Extraction

At session end (on `/save` or graceful exit), LMHub optionally runs a lightweight extraction pass:

```
System: Extract key technical facts from this conversation as a JSON array.
        Each fact: {"content": "...", "confidence": 0.0-1.0}
        Only extract concrete, reusable facts. Max 5 facts. No opinions.
        Ignore anything conversational or ephemeral.

[conversation history appended]
```

Extracted facts with confidence ≥ 0.8 are saved automatically. Lower-confidence facts are presented for user review before saving.

### Memory Commands

```bash
/mem list          # Show all memory facts for current project
/mem add "fact"    # Manually add a fact
/mem forget <id>   # Delete a specific fact
/mem clear         # Wipe all project memory
/mem global        # Show/manage global memory facts
```

### Memory UI (Ctrl+E)

```
┌─ Agent Memory — myapp ─────────────────────────────────────┐
│  Project Facts (6)                                         │
│  ────────────────────────────────────────────────────────  │
│  [user]  Auth uses JWT; secret stored in .env JWT_SECRET   │
│  [user]  PostgreSQL runs on port 5433 locally (not 5432)   │
│  [auto]  Migrations managed by goose (confidence: 94%)     │
│  [auto]  Redis used for session caching (confidence: 91%)  │
│  [auto]  chi router, not gorilla/mux (confidence: 97%)     │
│  [auto]  Test DB: TEST_DATABASE_URL env var (conf: 88%)    │
│                                                            │
│  Global Facts (2)                                          │
│  ────────────────────────────────────────────────────────  │
│  [user]  Prefer explicit error handling over panic()       │
│  [user]  Always use snake_case for DB column names         │
│                                                            │
│  [A] Add  [D] Delete  [C] Clear project  [Esc] Close       │
└────────────────────────────────────────────────────────────┘
```

---

## 13. Prompt Template Library

A built-in and user-extensible library of prompt templates for common coding and IT tasks. Activated with `Ctrl+T` or the `/t` command.

### Template Format (YAML)

```yaml
# templates/debug-go-error.yaml
name: "Debug Go error"
description: "Paste a Go error/panic and get a root-cause analysis with fix"
tags: ["go", "debugging", "errors"]
mode: ask                   # ask | plan | build — auto-switches mode on apply
prompt: |
  I'm getting the following error in my Go application:

  ```
  {cursor}
  ```

  Please:
  1. Explain what this error means
  2. Identify the most likely root cause
  3. Show me the fix with a corrected code snippet
  4. Explain how to prevent this class of error in future
```

`{cursor}` marks where the user's typed text is inserted when they apply the template.

### Built-in Template Categories

**Debugging & Errors**
- Debug Go error / panic
- Explain shell command exit code
- Parse and fix Docker build error
- Interpret git conflict markers

**Code Generation**
- Write a Go HTTP handler (REST)
- Generate a Dockerfile for a {language} app
- Write a systemd unit file
- Write a docker-compose.yml for {stack}

**Code Review**
- Review this function for correctness
- Check this SQL query for N+1 problems
- Audit this shell script for safety issues
- Review this Dockerfile for best practices

**IT Operations**
- Diagnose this nginx error log
- Write a cron job for {task}
- Explain this kernel log message
- Generate an SSH config for {scenario}

**Refactoring**
- Refactor this function to be testable
- Extract this logic into a reusable package
- Add proper error handling to this function
- Convert this callback to use context.Context

### Template Browser (Ctrl+T)

```
┌─ Prompt Templates ──────────────────────────────────────────┐
│  Search: go error_____                                      │
│  ─────────────────────────────────────────────────────────  │
│  > Debug Go error / panic                     [ask]   ↵     │
│    Review Go function for correctness         [ask]   ↵     │
│    Add error handling to Go function          [build] ↵     │
│    Refactor Go function to be testable        [build] ↵     │
│                                                             │
│  [/] Search  [↑↓] Navigate  [Enter] Apply  [E] Edit  [Esc] │
└─────────────────────────────────────────────────────────────┘
```

Applying a template:
1. Pre-fills the input with the template text
2. Places cursor at `{cursor}` position
3. Auto-switches to the template's target mode (with model swap if pinned)
4. User edits and submits normally

---

## 14. Context Window Management

Context is the scarcest resource with local models. LMHub manages it through a layered strategy.

### Context Budget (Injection Cap)

Before any prompt is sent, the budget manager calculates how much space is available and allocates it strictly:

```
Model context limit (e.g. 32,768 tokens)
  - System prompt base:    ~500 tokens (reserved)
  - Injection budget:     2,800 tokens (hard cap, all injected sources combined)
  - Conversation history: remainder (dynamically sized)
  - Response budget:      max_tokens from config (default 8,192)
```

Injection sources are filled in priority order:
1. Project context file (highest priority, most stable)
2. Memory facts (high priority, small)
3. RAG chunks (lower priority, variable)

If budget is exceeded, RAG chunks are trimmed first, then memory facts (least recently used dropped), then project context is truncated.

### Escalating Response Strategy

```
70% full → Status bar turns yellow. No action yet.
85% full → Auto-trim: drop oldest N turns (keep system prompt + last 4 turns). Log trim event.
90% full → Summarize: ask model to summarize conversation, replace history with summary + marker.
98% full → Hard pause: agent stops, user must /clear or /save and start fresh.
```

Token counting uses `tiktoken-go` with cl100k_base as a universal approximation (typically within ~5% for models in the 27B–35B range; exact tokenizers vary by model family and are not publicly available in Go).

---

## 15. Safety & Guardrails

### Three-Tier Permission System

| Tier | Behavior | Examples |
|---|---|---|
| **Safe** | Execute immediately, show result | read_file, list_dir, git_status, web_search |
| **Warn** | Show intent + short pause, auto-proceed (or block if `require_confirm_warn: true`) | write_file (shows diff), git_commit, docker_exec |
| **Dangerous** | Full block: "This cannot be undone. Proceed? [y/N]" | delete_file, git_branch delete, compose down, shell blocklist |

### Pre-Execution Checklist

```
1. Tool is registered and enabled?
2. Args pass schema validation?
3. Path args within scope (no traversal outside project dir)?
4. Shell command matches blocklist? → Escalate to Dangerous
5. File size guard (write > 10MB → Warn)?
6. Undoable? → Take snapshot before executing
7. Permission level → route to confirmation handler
8. Execute → push UndoRecord to stack
```

### Scope Pinning

On Build mode entry, the user sets a working directory. All file and shell operations are validated against this scope. Writes outside scope are blocked (not confirmed — always blocked).

---

## 16. Tool Call Parsing Strategy

Local models don't always produce clean JSON. The parser uses a 5-layer fallback:

```
Layer 1: Native function calling
         → tool_calls field in completion response (if model/LM Studio supports it)

Layer 2: XML-style tags
         → <tool_call>{...}</tool_call> blocks in response text

Layer 3: Markdown code blocks
         → ```json ... ``` blocks parsed as tool call

Layer 4: Regex extraction
         → {"name": "...", "args": {...}} pattern anywhere in output

Layer 5: Failure
         → Log raw output, return ParseError
         → Feed back to model: "Your last response could not be parsed as a tool call.
            Raw output: [output]. Please respond with a valid tool call."
```

Parse failures are tracked per model. If a model fails >3 times consecutively, LMHub surfaces a warning: "This model is producing unreliable tool call output. Consider switching to a different model or adjusting the system prompt via `lmhub config edit`."

---

## 17. TUI Layout

### Standard View (Ask / Plan mode)

```
┌──────────────────────────────────────────────────────────────────┐
│ LMHub  [ASK]  [PLAN]  [BUILD]          Ctrl+T: Templates        │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                                                            │  │
│  │ > How do I write a systemd service that restarts on fail?  │  │
│  │                                                            │  │
│  │ Create a unit file at /etc/systemd/system/myapp.service:   │  │
│  │                                                            │  │
│  │ ```ini                                                     │  │
│  │ [Unit]                                                     │  │
│  │ Description=MyApp Service                                  │  │
│  │ After=network.target                                       │  │
│  │                                                            │  │
│  │ [Service]                                                  │  │
│  │ ExecStart=/usr/local/bin/myapp                             │  │
│  │ Restart=on-failure                                         │  │
│  │ RestartSec=5                                               │  │
│  │ ```                                                        │  │
│  │                                                            │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  > ________________________________________________  [Enter]     │
│                                                                  │
├──────────────────────────────────────────────────────────────────┤
│  Context  [████████░░░░░░░░░░░░░░░░░░░░░░]  8,204 / 32,768      │
│           ██ sys  ████ hist  █ mem  ░ rag        41 tok/s  ●     │
├──────────────────────────────────────────────────────────────────┤
│  ASK │ <loaded-model-id> │ XX.X GB RAM │ Ready                  │
└──────────────────────────────────────────────────────────────────┘
```

### Build Mode View (with undo panel)

```
┌──────────────────────────────────────────────────────────────────┐
│ LMHub  [ASK]  [PLAN]  [BUILD ●]        Ctrl+Z: Undo  Iter: 3/15 │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────────────────────┐  ┌────────────────────┐   │
│  │                                  │  │  Tool Activity     │   │
│  │ > Add JWT auth to the API        │  │  ─────────────────  │   │
│  │                                  │  │  ✓ read_file       │   │
│  │  <thought> I'll start by         │  │    internal/auth   │   │
│  │  reading the current auth...     │  │  ✓ read_file       │   │
│  │  </thought>                      │  │    main.go         │   │
│  │                                  │  │  ✓ write_file      │   │
│  │  Reading internal/auth.go...     │  │    auth.go  [undo] │   │
│  │  ✓ Done (312 bytes)              │  │  ⟳ write_file      │   │
│  │                                  │  │    middleware.go   │   │
│  │  Writing updated auth.go...      │  │                    │   │
│  │  ✓ Done — diff:                  │  │  Scope: ./myapp/   │   │
│  │  + func ValidateJWT(...)         │  │  Changed: 2 files  │   │
│  │  + func GenerateJWT(...)         │  └────────────────────┘   │
│  │                                  │                           │
│  └──────────────────────────────────┘                           │
│                                                                  │
│  > ________________________________________________  [Enter]     │
│                                                                  │
├──────────────────────────────────────────────────────────────────┤
│  Context  [████████████░░░░░░░░░░░░░░░░░░]  14,100 / 32,768     │
│           ██ sys  ██████ hist  █ mem  ███ rag      38 tok/s  ●  │
├──────────────────────────────────────────────────────────────────┤
│  BUILD │ qwen3.6-35b-a3b │ MoE ~3B active │ 21.4 GB │ Running   │
└──────────────────────────────────────────────────────────────────┘
```

### Keyboard Shortcuts

| Shortcut | Action |
|---|---|
| `Ctrl+A` | Switch to Ask mode |
| `Ctrl+P` | Switch to Plan mode |
| `Ctrl+B` | Switch to Build mode |
| `Ctrl+M` | Open model browser |
| `Ctrl+T` | Open prompt template browser |
| `Ctrl+Z` | Open undo history panel (Build mode) |
| `Ctrl+I` | Open inference metrics overlay |
| `Ctrl+E` | Open agent memory viewer |
| `Ctrl+S` | Save current session |
| `Ctrl+L` | Clear context |
| `Ctrl+C` | Cancel streaming / interrupt agent |
| `Ctrl+Q` | Quit |
| `Ctrl+H` | Show help overlay |
| `/context` | Edit `.lmhub/context.md` in $EDITOR |
| `Tab` | Cycle through mode tabs |

---

## 18. CLI Interface

```bash
# Launch TUI (default)
lmhub

# One-shot ask
lmhub ask "What ports does PostgreSQL use?"

# Apply a template non-interactively
lmhub ask --template "debug-go-error" --input error.txt

# Plan a task
lmhub plan "Refactor auth module to use JWTs" --output plan.json

# Execute a plan
lmhub build --plan plan.json --cwd ./myproject

# Model management
lmhub models list
lmhub models load qwen/qwen3.6-35b-a3b
lmhub models unload

# RAG index management
lmhub index                   # Index current directory
lmhub index --watch           # Index + watch for changes
lmhub index --clear           # Wipe and re-index
lmhub index --stats           # Index stats

# Memory management
lmhub memory list             # Show all memory facts
lmhub memory add "fact"       # Add a fact manually
lmhub memory forget <id>      # Remove a fact
lmhub memory clear            # Wipe project memory

# Project setup
lmhub init                    # Generate starter .lmhub/context.md

# Config
lmhub config show
lmhub config edit
```

---

## 19. Configuration Schema

```yaml
# ~/.config/lmhub/config.yaml

lmstudio:
  base_url: "http://localhost:1234"
  timeout_seconds: 120
  stream: true
  metrics_poll_interval_ms: 2000    # How often to poll /api/v0/models/loaded

mode_models:
  ask:   ""                          # Empty = use whatever is loaded
  plan:  "qwen/qwen3.6-27b"
  build: "qwen/qwen3.6-35b-a3b"

inference:
  temperature: 0.7
  max_tokens: 8192
  top_p: 0.95
  repeat_penalty: 1.1

mode_inference:
  plan:
    temperature: 0.3
    max_tokens: 4096
  build:
    temperature: 0.5
    max_tokens: 8192

agent:
  max_iterations: 15
  context_warn_pct: 70
  context_trim_pct: 85
  context_summarize_pct: 90

# Injected context budget (all sources combined hard cap)
context_budget:
  project_context_max_tokens: 800
  memory_max_tokens: 800
  rag_max_tokens: 1200
  total_max_tokens: 2800            # Never exceed this regardless of individual settings

rag:
  enabled: true
  top_k: 3
  max_tokens: 1200
  min_score: 0.72
  reindex_on_start: false
  exclude_patterns:
    - "*.lock"
    - "*.sum"
    - "testdata/**"
    - "vendor/**"
    - "node_modules/**"
    - "*.pb.go"

memory:
  enabled: true
  auto_extract: true                # Extract facts at session end
  auto_extract_threshold: 0.8      # Min confidence to auto-save
  max_facts_per_project: 50
  max_facts_global: 20

project_context:
  enabled: true
  max_tokens: 800
  file_name: "context.md"          # Looked up in .lmhub/ relative to cwd

templates:
  builtin_enabled: true
  user_dir: "~/.config/lmhub/templates/"

tools:
  shell:
    timeout_seconds: 30
    allowed_shells: ["zsh", "bash"]
    blocklist:
      - "rm -rf /"
      - "mkfs"
      - "dd if="
      - ":(){:|:&};:"
  web:
    search_provider: "duckduckgo"
    serper_api_key: ""
    fetch_timeout_seconds: 10
    cache_ttl_minutes: 60
  docker:
    socket: "/var/run/docker.sock"

safety:
  require_confirm_dangerous: true
  require_confirm_warn: false
  show_diff_before_write: true
  max_file_write_bytes: 10485760

ui:
  theme: "dark"
  markdown_style: "dracula"
  show_token_count: true
  show_thinking_tags: false
  show_context_bar: true
  show_metrics_in_statusbar: true

sessions:
  save_dir: "~/.local/share/lmhub/sessions"
  auto_save: true
  max_history: 50

log:
  level: "warn"
  file: "~/.local/share/lmhub/logs/lmhub.log"
```

---

## 20. Platform Support

### macOS (Primary)
- Default shell: `zsh`
- Config dir: `~/.config/lmhub/`
- Data dir: `~/.local/share/lmhub/`
- Docker socket: `/var/run/docker.sock`
- LM Studio API: `http://localhost:1234`

### Linux
- Default shell: `bash`
- Config/data dirs: XDG base dir spec
- Docker socket: `/var/run/docker.sock`
- Build tag: `//go:build linux`

### Windows (Future)
- Default shell: `PowerShell`
- Config dir: `%APPDATA%\lmhub\`
- Docker socket: `//./pipe/docker_engine`
- Build tag: `//go:build windows`

---

## 21. Development Phases

### Phase 1 — Foundation (3–5 days)
- [ ] Repo scaffold, go.mod, Makefile
- [ ] LM Studio API client (chat + management endpoints)
- [ ] Telemetry client (`/api/v0/models/loaded` poller)
- [ ] Model manager (list, load, unload, auto-switch)
- [ ] Basic config loading (viper + YAML)
- [ ] Minimal TUI: Bubbletea app shell, status bar, context bar (live token fill)
- [ ] Ask mode (chat loop, streaming, markdown render)
- [ ] Inference metrics overlay (Ctrl+I)

**Milestone:** Ask mode works end-to-end. Live context bar and metrics are visible.

---

### Phase 2 — Plan Mode + Context Infrastructure (3–4 days)
- [ ] Plan schema and JSON parser
- [ ] Plan mode agent loop (structured output, no tools)
- [ ] Plan review TUI (checklist, approve/reject)
- [ ] Context window manager (token counting, warn/trim/summarize)
- [ ] Context budget allocator (project ctx + memory + RAG combined cap)
- [ ] Project context file loading (`.lmhub/context.md`)
- [ ] System prompt tuning for Qwen3 and Gemma4

**Milestone:** Plan mode works. Project context is injected. Context bar shows budget breakdown.

---

### Phase 3 — Build Mode Core + Undo (5–7 days)
- [ ] Tool registry and interface (with Undoable() and Snapshot())
- [ ] Filesystem tools (read, write, list, search, delete, move)
- [ ] Shell tool (blocklist, timeout, scope enforcement)
- [ ] Safety layer (permission tiers, confirmation prompts)
- [ ] Undo stack and UndoRecord store
- [ ] Undo history panel (Ctrl+Z)
- [ ] ReAct agent loop with iteration counter
- [ ] Tool call parser (all 5 layers)
- [ ] Build session state (files modified, commands run)
- [ ] Scope pinning (working directory enforcement)

**Milestone:** Build mode works with file/shell tools. Every write is undoable. Undo panel functional.

---

### Phase 4 — Build Mode Extended (4–5 days)
- [ ] Git tools (go-git integration)
- [ ] Docker tools (Docker SDK)
- [ ] Web tools (search + fetch)
- [ ] Diff viewer in TUI
- [ ] Plan → Build handoff (load `.lmhub/plan-*.json` and execute)
- [ ] Tool activity panel (right-side Build mode UI)
- [ ] Parse failure tracking and model reliability warning

**Milestone:** All tools operational. Plan → Build handoff works.

---

### Phase 5 — RAG & Embeddings (3–4 days)
- [ ] Embeddings API client (`/v1/embeddings` with nomic-embed-text)
- [ ] File chunker (tree-sitter for Go/Python/JS, sliding window fallback)
- [ ] bbolt vector store (chunk storage + cosine similarity search)
- [ ] Indexer (walk project, chunk, embed, store)
- [ ] fsnotify file watcher (incremental re-indexing)
- [ ] Retriever (query → embed → search → top-k chunks)
- [ ] RAG injection into Plan and Build system prompts
- [ ] CLI commands: `lmhub index`, `lmhub index --stats`, `lmhub index --clear`

**Milestone:** Project can be indexed. Relevant code chunks are injected automatically into Plan and Build prompts.

---

### Phase 6 — Memory + Templates (3–4 days)
- [ ] bbolt memory store (project-scoped + global)
- [ ] Post-session fact extraction (calls model after /save)
- [ ] Memory injection into all mode system prompts
- [ ] Memory viewer/editor (Ctrl+E)
- [ ] CLI memory commands
- [ ] Built-in template library (20+ templates)
- [ ] Template YAML loader (user-defined templates)
- [ ] Template browser TUI (Ctrl+T)
- [ ] Template application (pre-fill input, auto-switch mode)

**Milestone:** Memory persists across sessions. Templates accelerate common tasks.

---

### Phase 7 — Polish & Platform (3–5 days)
- [ ] Linux support and testing
- [ ] Windows build tags and PowerShell adapter
- [ ] Session persistence and `/save` `/load`
- [ ] `lmhub init` (generates starter context.md)
- [ ] CLI mode (non-interactive `lmhub ask/plan/build`)
- [ ] `goreleaser` cross-platform binaries
- [ ] First-run wizard (detects LM Studio, writes default config)
- [ ] README, example configs, full help text

**Milestone:** Distributable release-quality binary. Works on macOS and Linux.

---

## 22. Key Go Dependencies

```go
require (
    // TUI
    github.com/charmbracelet/bubbletea     v0.27.x
    github.com/charmbracelet/lipgloss      v0.13.x
    github.com/charmbracelet/bubbles       v0.20.x
    github.com/charmbracelet/glamour       v0.8.x

    // HTTP
    github.com/go-resty/resty/v2           v2.16.x

    // Config
    github.com/spf13/viper                 v1.19.x
    gopkg.in/yaml.v3                       v3.0.x

    // Git
    github.com/go-git/go-git/v5           v5.13.x

    // Docker
    github.com/docker/docker               v27.x.x
    github.com/docker/distribution         v2.8.x

    // Web
    github.com/PuerkitoBio/goquery        v1.10.x

    // JSON
    github.com/tidwall/gjson              v1.18.x

    // Token counting
    github.com/tiktoken-go/tokenizer      v0.2.x

    // Storage (RAG + memory)
    go.etcd.io/bbolt                       v1.3.x

    // File watching (RAG incremental re-index)
    github.com/fsnotify/fsnotify          v1.7.x

    // Code chunking (RAG)
    github.com/smacker/go-tree-sitter     v0.0.x

    // Testing
    github.com/stretchr/testify           v1.10.x

    // Release
    // goreleaser installed as a dev tool, not a Go dep
)
```

---

## 23. Testing Strategy

- **Unit tests:** Tool call parser (fixture outputs from each model), context trimmer, budget allocator, undo stack, cosine similarity search, fact extractor, template loader
- **Integration tests:** LM Studio API client (against real or mock server), each tool with temp dir/repo, RAG indexer with sample project
- **End-to-end tests:** CLI `lmhub ask` with mock LM Studio server returning fixture SSE streams
- **Manual testing matrix:** Each mode × each model × macOS before any phase milestone is marked complete

---

## 24. Important Implementation Notes for the Coding Agent

1. **LM Studio management API is partially undocumented.** Probe `/api/v0/models/loaded` on a live LM Studio instance before coding against it. The response shape may differ from what's assumed here. Use `lms` CLI as a fallback if needed.

2. **Streaming responses:** Use `bufio.Scanner` on the SSE stream from `/v1/chat/completions`. Split on `data: ` prefix, skip `[DONE]`, unmarshal each chunk independently. Track timestamps between chunks to compute tokens/sec.

3. **nomic-embed-text via LM Studio:** The embeddings endpoint requires nomic-embed-text to be the loaded model. Since only one model can be loaded at a time, indexing must happen with nomic loaded, and chat inference requires switching to a chat model. Handle this gracefully: indexing is a separate CLI command (`lmhub index`), not done inline during a chat session.

4. **bbolt is single-writer.** Open the database once at startup and pass the handle around. Do not open multiple connections from goroutines.

5. **Cosine similarity in Go:** Implement as a pure function over `[]float32` slices. nomic-embed-text produces 768-dimensional vectors. For a local project index (typically <10,000 chunks), brute-force cosine search is fast enough — no need for an ANN index.

6. **Model load time is variable.** A 35B model can take 15–60 seconds. The TUI must remain responsive during load. Use goroutines + Bubbletea `Cmd` messages — never block the `Update` loop.

7. **Gemma4 structured output.** Gemma4 models produce less reliable JSON than Qwen3. For Plan mode, if Gemma4 is selected, inject an extra instruction: "IMPORTANT: Your response must be raw JSON only. No markdown, no explanation, no code fences." Track parse failure rate and surface a model recommendation if it exceeds the threshold.

8. **Undo snapshots for large files.** Capped at `max_file_write_bytes` (default 10MB). If a file exceeds this, skip the snapshot and mark the UndoRecord as "snapshot unavailable — manual recovery required."

9. **go-git limitations.** Complex rebases, worktrees, and shallow clones may require shelling out to system git. Detect the git binary at startup; if missing, disable git tools and warn the user.

10. **Windows line endings.** All `write_file` operations normalize to LF unless the existing file uses CRLF (detect on read). Use `bytes.ReplaceAll` to normalize before writing.

11. **Context bar polling.** The 2-second poll on `/api/v0/models/loaded` should be a background goroutine sending `Msg` updates to Bubbletea. If LM Studio is unreachable, display "●" as red in the status bar and retry silently — never crash or block.

12. **Tree-sitter bindings.** `go-tree-sitter` requires C bindings. Ensure CGO is enabled in the build and that grammar files for Go, Python, JavaScript, TypeScript, and Bash are vendored or fetched at build time. Add a `//go:build cgo` gate and fall back to sliding-window chunking if CGO is unavailable.
