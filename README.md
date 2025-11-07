# Cursor Session Export CLI

A command-line tool to extract and export chat sessions from **Cursor IDE**. Extract your conversation history from Cursor's Composer and chat interface, export it in multiple formats, and keep your AI-assisted coding sessions organized.

Works with Cursor IDE's modern storage format (globalStorage) to extract conversations, code blocks, tool calls, and context from your chat sessions.

## Features

- 📋 **List all sessions** - See all your Cursor IDE chat sessions at a glance
- 💬 **View conversations** - Browse messages from Composer and chat sessions with filtering options
- 📤 **Export in multiple formats** - JSONL, Markdown, YAML, or JSON
- 🔍 **Rich content extraction** - Captures full conversations including code blocks, tool calls, and context
- ⚡ **Fast and efficient** - Intelligent caching for quick access to your sessions
- 🎯 **Workspace-aware** - Automatically associates sessions with your workspaces
- 🖥️ **Cross-platform** - Works on macOS and Linux

## Installation

### Quick Install (Recommended)

```bash
# Clone the repository
git clone https://github.com/iksnae/cursor-session.git
cd cursor-session

# Run the install script (fully automatic - no manual steps!)
./install.sh
```

The script automatically builds, installs, and configures the tool. **No manual configuration needed!** Works on macOS (zsh) and Linux (bash/zsh).

### Alternative Installation Methods

**Using Go:**

```bash
go install github.com/iksnae/cursor-session@latest
```

**Using Make:**

```bash
make install
```

**Manual Build:**

```bash
go build -buildvcs=false -o cursor-session .
sudo cp cursor-session /usr/local/bin/
```

### Verify Installation

```bash
cursor-session list
```

## Quick Start

```bash
# List all your Cursor IDE chat sessions
cursor-session list

# View messages from a specific session
cursor-session show <session-id>

# Export all sessions as Markdown
cursor-session export --format md
```

## Basic Usage

**List sessions:**

```bash
cursor-session list
```

**View a session:**

```bash
cursor-session show <session-id>
```

**Export sessions:**

```bash
cursor-session export --format md
```

For detailed usage information, see the [Usage Guide](docs/USAGE.md).

## Requirements

- **Cursor IDE** installed (extracts from globalStorage format)
- macOS or Linux

## Documentation

- [Usage Guide](docs/USAGE.md) - Complete command reference
- [Implementation Details](IMPLEMENTATION.md) - Technical implementation summary
- [Technical Design](docs/TDD.md) - Architecture and design decisions

## License

MIT
