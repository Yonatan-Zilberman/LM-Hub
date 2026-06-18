# LMHub

> A terminal-based, local-first AI agent harness for **LM Studio** featuring Ask, Plan, and Build modes.

LMHub wraps LM Studio's local model APIs with a structured engineering workflow, client-side metrics tracking, persistent memory management, RAG codebase indexing, and a sandboxed filesystem execution layer with full rollback/undo support.

---

## Key Features

- **Multi-Mode Execution Layout**:
  - **Ask Mode (`Ctrl+A`)**: Stateful conversational chat with streaming Markdown rendering.
  - **Plan Mode (`Ctrl+P`)**: Structured reasoning generating explicit, step-by-step JSON plans.
  - **Build Mode (`Ctrl+B`)**: Autonomous ReAct agent loop executing sandboxed filesystem, Git, Docker, and Web tools.
- **Persistent Agent Memory (`Ctrl+E`)**: Project-scoped and global key-value facts database (`bbolt` backed) with automatic post-session fact extraction.
- **RAG Codebase Indexing**: Embedded vector database (`bbolt` backed) with sliding-window or CGO-based Tree-sitter AST chunking for precise semantic retrieval.
- **Inference Metrics Overlay (`Ctrl+I`)**: Real-time tracking of generation speed (tok/sec), TTFT (ms), RAM usage, and visual context window allocations.
- **Safety Guardrails & Tool Undo (`Ctrl+Z`)**: Active classification of tools (Safe/Warn/Dangerous). Built-in file, directory, and git staged rollback stack.
- **Prompt Template Browser (`Ctrl+T`)**: Built-in YAML-extensible prompt library supporting mode auto-switching and cursor position offsets.

---

## Installation & Build

LMHub requires **Go 1.22+**.

```bash
# Clone the repository
git clone https://github.com/yonatanzilberman/lmhub.git
cd LM-Hub

# Build the standard binary
make build

# Build with Tree-sitter AST support (requires CGO and gcc)
make build-treesitter

# Run the TUI
./lmhub
```

---

## Keyboard Shortcuts

| Shortcut | Action |
| --- | --- |
| `Ctrl+A` | Switch to **Ask** (Chat) mode |
| `Ctrl+P` | Switch to **Plan** mode |
| `Ctrl+B` | Switch to **Build** mode |
| `Ctrl+M` | Open **Model Browser** |
| `Ctrl+I` | Open **Inference Metrics** overlay |
| `Ctrl+E` | Open **Agent Memory Facts Center** |
| `Ctrl+T` | Open **Prompt Template Browser** |
| `Ctrl+Z` | Open **Undo/Rollback History** (Build mode) |
| `Ctrl+S` | Save current session history to disk |
| `Ctrl+L` | Clear active chat history |
| `Ctrl+Q` | Quit the application |

---

## Slash Commands (Chat view input)

- `/save [name]` - Save conversation history to `.lmhub/sessions/`
- `/load <id>` - Restore a saved conversation session by ID
- `/clear` - Reset chat history
- `/mem` - Toggle the Memory Facts Center
- `/context` - Edit `.lmhub/context.md` project rules in your default `$EDITOR`
- `/help` - Print available slash commands

---

## CLI Reference

LMHub can be run headlessly or in non-interactive mode.

```bash
# Run one-shot query
lmhub ask "Explain cosine similarity" --temp 0.5

# Run one-shot with prompt template and input file
lmhub ask --template "debug-go-error" --input panic.log

# Generate a structured plan in JSON
lmhub plan "Refactor user authentication to JWTs" --output plan.json

# Execute a plan autonomously in a specific workspace directory
lmhub build --plan plan.json --cwd ./src/myproject

# Manage RAG Indexing
lmhub index                  # Index current codebase
lmhub index --stats          # Show index statistics
lmhub index --clear          # Wipe vector store

# Manage Memory Facts
lmhub memory list            # List current facts
lmhub memory add "fact content"
lmhub memory global list     # Manage global memory facts
```

---

## Configuration

Default configuration file location:
- **macOS**: `~/.config/lmhub/config.yaml`
- **Linux**: `~/.config/lmhub/config.yaml` (follows XDG spec)
- **Windows**: `%APPDATA%\lmhub\config.yaml`

### Schema Overview

```yaml
lmstudio:
  base_url: "http://localhost:1234"
  timeout_seconds: 120
  embedding_model: "text-embedding-nomic-embed-text-v1.5"

mode_models:
  ask: ""
  plan: "qwen/qwen3.6-27b"
  build: "qwen/qwen3.6-35b-a3b"

rag:
  enabled: true
  top_k: 3
  max_tokens: 1200
  min_score: 0.72

memory:
  enabled: true
  auto_extract: true
  auto_extract_threshold: 0.8

safety:
  require_confirm_dangerous: true
  require_confirm_warn: false
  show_diff_before_write: true
```
