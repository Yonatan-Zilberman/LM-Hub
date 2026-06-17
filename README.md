# LM Hub

> A local-first AI agent harness for LM Studio with Ask, Plan, and Build modes.

LM Hub is a terminal-based tool written in Go that integrates directly with your local LM Studio instance to provide isolation modes for software development and IT operational workflows.

## Key Features (Phase 1)
- **Ask Mode**: A stateful conversation loop with streaming completions and dynamic markdown rendering.
- **Model Browser**: Browse available models from your LM Studio server, load/unload them interactively, and view their specs.
- **Inference Metrics Overlay**: Track time to first token (TTFT), token generation speed (tokens/sec), total tokens, and context window metrics.
- **Live Context Bar**: Real-time visual feedback of your token fill capacity using color-coded alerts (Green, Yellow, Orange, Red).
- **Graceful Offline Integration**: If LM Studio is not active, the application degrades gracefully showing disconnected status states instead of crashing.

---

## Installation & Build

Ensure you have **Go 1.22+** installed on your system.

```bash
# Clone the repository
git clone https://github.com/yonatanzilberman/lmhub.git
cd LM-Hub

# Resolve dependencies
go mod tidy

# Build the binary
make build

# Run the TUI
make run
```

---

## Keyboard Shortcuts

| Shortcut | Action |
| --- | --- |
| `Ctrl+A` | Switch to Ask (Chat) mode |
| `Ctrl+M` | Open Model Browser |
| `Ctrl+I` | Open Inference Metrics overlay |
| `Ctrl+L` | Clear active chat history |
| `Ctrl+Q` | Quit the application |

---

## Configuration

On first run, LM Hub creates a default configuration file in:
- **macOS**: `~/.config/lmhub/config.yaml`

You can customize the base URL, timeout thresholds, streaming, default model overrides, and TUI theme settings inside this file.
