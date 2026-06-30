# LMH (LM-Hub)

[![Go Report Card](https://goreportcard.com/badge/github.com/Yonatan-Zilberman/LM-Hub)](https://goreportcard.com/report/github.com/Yonatan-Zilberman/LM-Hub)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Yonatan-Zilberman/LM-Hub)](https://golang.org)

> A terminal-based, local-first AI agent harness for **LM Studio** featuring Ask, Plan, and Build modes.

LMH wraps LM Studio's local model APIs with a structured engineering workflow, client-side metrics tracking, persistent memory management, RAG codebase indexing, and a sandboxed filesystem execution layer with full rollback/undo support.

---

## Features

### Modes
- **Ask** — Stateful chat with project context, memory, and RAG injection
- **Plan** — Structured JSON implementation plans with human review and approval
- **Build** — Autonomous ReAct agent with sandboxed tools and per-action undo

### Agent Capabilities
- Filesystem, shell, git, Docker, and web tools with safety tiers
- Interactive `ask_user` prompts during Build mode execution
- Context window management: warn, trim, summarize, and hard-stop escalation
- Persistent project and global memory with LLM fact extraction
- RAG codebase indexing with semantic retrieval

### Developer Experience
- Full-screen Bubbletea TUI with persistent sidebar and keybinding bar
- Headless CLI for `ask`, `plan`, `build`, `index`, `memory`, and `models`
- Session save/load, prompt templates, and first-run setup wizard
- Single Go binary — no external services beyond LM Studio

---

## Quick Start

Get from zero to your first session in under a minute:

```bash
# 1. Install (release download, falls back to source build)
curl -fsSL https://raw.githubusercontent.com/Yonatan-Zilberman/LM-Hub/main/install.sh | sh

# 2. Start LM Studio and load a chat model (see LM Studio Setup below)

# 3. Initialize project context in your repo
cd your-project
lmh init

# 4. Launch the TUI
lmh
```

Press `Ctrl+A` for Ask mode, type a question, and press Enter.

---

## Architecture Overview

```
                          ┌───────────────────────────┐
                          │    LM Studio API server   │
                          │   (localhost:1234 / lms)  │
                          └─────────────┬─────────────┘
                                        │ (JSON-RPC / REST)
                                        ▼
   ┌─────────────────────────────────────────────────────────────────────────┐
   │                                  LMH                                    │
   │  ┌───────────────────────────┬───────────────┬───────────────────────┐  │
   │  │   Ask Mode (Ctrl+A)       │  Plan Mode    │   Build Mode (Ctrl+B) │  │
   │  │   Interactive Chat &      │  (Ctrl+P)     │   Autonomous ReAct    │  │
   │  │   RAG Context Retrieval   │  JSON Plans   │   Sandboxed Exec Loop │  │
   │  └─────────────┬─────────────┴───────┬───────┴───────────┬───────────┘  │
   └────────────────┼─────────────────────┼───────────────────┼──────────────┘
                    ▼                     ▼                   ▼
            ┌───────────────┐     ┌───────────────┐     ┌───────────────┐
            │ Vector Store  │     │ Project Specs │     │ Sandboxed OS  │
            │ (index.db)    │     │ (context.md)  │     │ Shell/Tools   │
            └───────────────┘     └───────────────┘     └───────────────┘
```

---

## LM Studio Setup

1. **Download & Install**: Visit [lmstudio.ai](https://lmstudio.ai) and download the app for your OS.
2. **Download Models**:
   - A chat/instruction model (e.g., `Qwen/Qwen2.5-7B-Instruct-GGUF`)
   - An embedding model (e.g., `nomic-ai/nomic-embed-text-v1.5-GGUF`) for RAG
3. **Start the Local Server**:
   - Open the **Local Server** tab in LM Studio
   - Select your chat model and click **Start Server** (default port `1234`)
4. **Optional CLI**: `lms server start`

---

## Installation

### curl installer (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/Yonatan-Zilberman/LM-Hub/main/install.sh | sh
```

Options: `./install.sh --source` to build locally; `./install.sh --dir /usr/local/bin` for a custom path.

### go install

```bash
go install github.com/Yonatan-Zilberman/LM-Hub/cmd/lmh@latest
ln -sf "$(go env GOPATH)/bin/lmh" "$(go env GOPATH)/bin/lmhub"
```

### make install (from clone)

```bash
git clone https://github.com/Yonatan-Zilberman/LM-Hub.git
cd LM-Hub
make install
```

---

## Building from Source

### Prerequisites
- **Go 1.22+**
- **LM Studio** running locally for runtime use
- **gcc** (optional) — only required for Tree-sitter AST chunking (`make build-treesitter`)

```bash
git clone https://github.com/Yonatan-Zilberman/LM-Hub.git
cd LM-Hub

make build              # Standard binary → ./lmh
make build-treesitter   # With Go AST chunking (CGO)
make test               # Run unit tests
make lint               # go vet
```

---

## Mode Guide

| Mode | Shortcut | Purpose |
|------|----------|---------|
| **Ask** | `Ctrl+A` | Conversational Q&A with context and memory |
| **Plan** | `Ctrl+P` | Generate a structured JSON plan for review |
| **Build** | `Ctrl+B` | Execute tasks autonomously with tools and undo |

**Typical workflow:** Ask a question → Plan a feature → Approve the plan → Build it step by step.

```bash
# Headless examples
lmh ask "How does auth work in this repo?"
lmh plan "Add JWT authentication" --output plan.json
lmh build --plan plan.json --cwd ./myapp
```

---

## Keyboard Shortcuts

| Shortcut | Action |
| --- | --- |
| `Ctrl+A` | Switch to **Ask** (Chat) mode |
| `Ctrl+P` | Switch to **Plan** mode |
| `Ctrl+B` | Switch to **Build** mode |
| `Ctrl+M` | Open **Model Browser** |
| `Ctrl+G` | Open **Inference Metrics** overlay |
| `Ctrl+E` | Open **Agent Memory Facts Center** |
| `Ctrl+T` | Open **Prompt Template Browser** |
| `Ctrl+Z` | Open **Undo/Rollback History** (Build mode) |
| `Ctrl+S` | Save current session history to disk |
| `Ctrl+L` | Clear active chat history |
| `Ctrl+H` | Toggle **Help** overlay |
| `Ctrl+C` | Cancel active streaming / plan generation |
| `Ctrl+Q` | Quit the application (unloads all models from LM Studio) |
| `Tab` | Cycle through Ask/Plan/Build modes |

---

## Slash Commands (Chat view input)

- `/save [name]` — Save conversation to `.lmhub/sessions/` (or `sessions.save_dir`)
- `/load <id>` — Restore a saved session
- `/clear` — Reset chat history
- `/mem` — Toggle Memory Facts Center
- `/context` — Edit `.lmhub/context.md` in `$EDITOR`
- `/t` — Toggle Prompt Templates view
- `/help` — Print available slash commands

---

## CLI Reference

```bash
lmh                          # Launch interactive TUI (default)
lmh ask "question"           # One-shot Ask mode
lmh plan "task" --output plan.json
lmh build --plan plan.json --cwd ./project
lmh index [--watch|--clear|--stats]
lmh memory list|add|forget|clear|global
lmh models list|load|unload
lmh config show|edit
lmh sessions list|delete
lmh init                     # Create starter .lmhub/context.md
lmh --version
```

---

## Configuration

Default config locations:
- **macOS / Linux**: `~/.config/lmhub/config.yaml`
- **Windows**: `%APPDATA%\lmhub\config.yaml`

```yaml
lmstudio:
  base_url: "http://localhost:1234"
  timeout_seconds: 120
  embedding_model: "text-embedding-nomic-embed-text-v1.5"

mode_models:
  ask: ""
  plan: "qwen/qwen3.6-27b"
  build: "qwen/qwen3.6-35b-a3b"

mode_inference:
  ask:
    temperature: 0.7
    max_tokens: 8192
  plan:
    temperature: 0.3
    max_tokens: 4096
  build:
    temperature: 0.5
    max_tokens: 8192

rag:
  enabled: true
  reindex_on_start: false
  exclude_patterns:
    - "node_modules/**"
    - "vendor/**"

sessions:
  auto_save: true
  save_dir: ""
```

See [docs/lmhub-implementation-plan.md](docs/lmhub-implementation-plan.md) for the full schema reference.

---

## Contributing

1. Read [docs/agent.md](docs/agent.md) for coding rules and architecture decisions.
2. Fork the repo and create a feature branch.
3. Write tests alongside code changes (`_test.go` in the same package).
4. Run `go vet ./...` and `go test ./...` before opening a PR.
5. Use [Conventional Commits](https://www.conventionalcommits.org/) for commit messages.
6. Update `README.md` and `docs/agent.md` when adding user-facing features.

---

## License

This project is licensed under the **MIT License** — see [LICENSE](LICENSE).

Copyright (c) 2026 Yonatan Zilberman
