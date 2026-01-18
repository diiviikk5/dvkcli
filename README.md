# dvkcli - Master Divik's AI Slave

A local-first AI terminal assistant running on Ollama. Your personal pocket AI slave for conversations, coding, and work tasks.

## Features

- 🍷 **Wine & Gold Theme** - Royal aesthetic
- 🧠 **Memory** - Remembers past conversations with vector search
- 💬 **Chat** - Natural conversation with your AI slave
- 🔍 **Search** - Semantic search over chat history
- 📤 **Export** - Save conversations to markdown
- ⚡ **Fast** - Local inference with Ollama

## Installation

### Prerequisites

1. Install [Ollama](https://ollama.ai/download)
2. Pull a model:
   ```bash
   ollama pull qwen2.5:3b
   ```

### Install dvkcli

```bash
go install github.com/diiviikk5/dvkcli/cmd/dvkcli@latest
```

Or clone and build:
```bash
git clone https://github.com/diiviikk5/dvkcli.git
cd dvkcli
go install ./cmd/dvkcli
```

## Usage

Just run:
```bash
dvkcli
```

### Commands

| Command | Description |
|---------|-------------|
| `/help` | Show all commands |
| `/models` | List available Ollama models |
| `/search <query>` | Search past conversations |
| `/clear` | Clear current conversation |
| `/export` | Export chat to markdown |

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Ctrl+N` | New conversation |
| `Ctrl+L` | Load last conversation |
| `Ctrl+E` | Export conversation |
| `↑/↓` or `j/k` | Scroll |
| `PgUp/PgDown` | Page scroll |
| `Ctrl+C` | Quit |

## Configuration

Config is stored in `~/.dvkcli/config.json`:

```json
{
  "ollama_url": "http://localhost:11434",
  "model": "qwen2.5:3b",
  "embed_model": "nomic-embed-text",
  "memory_enabled": true,
  "theme": "cyberpunk"
}
```

## Tech Stack

- **Go** - Fast, single binary
- **Bubbletea** - Terminal UI framework
- **Lipgloss** - Styling
- **SQLite** - Local memory storage
- **Ollama** - Local LLM inference

## License

MIT
