![cursor session](./images/simple-icon.png)
[![codecov](https://codecov.io/gh/iksnae/cursor-session/branch/main/graph/badge.svg)](https://codecov.io/gh/iksnae/cursor-session)

# Cursor Session

A command-line tool to extract and export chat sessions from **Cursor IDE** and **cursor-agent CLI**. Extract your conversation history from Cursor's Composer and chat interface, export it in multiple formats, and keep your AI-assisted coding sessions organized.

Works with both Cursor IDE's desktop app storage (globalStorage) and cursor-agent CLI storage to extract conversations, code blocks, tool calls, and context from your chat sessions.

## Features

- 📋 **List all sessions** - See all your Cursor IDE chat sessions at a glance
- 💬 **View conversations** - Browse messages from Composer and chat sessions with filtering options
- 📤 **Export in multiple formats** - JSONL, Markdown, YAML, or JSON
- 🔍 **Rich content extraction** - Captures full conversations including code blocks, tool calls, and context
- ⚡ **Fast and efficient** - Intelligent caching for quick access to your sessions
- 🎯 **Workspace-aware** - Automatically associates sessions with your workspaces
- 🖥️ **Cross-platform** - Works on macOS and Linux
- 🔌 **Multiple storage backends** - Supports both desktop app (globalStorage) and cursor-agent CLI storage
- 🛠️ **Diagnostic tools** - Built-in healthcheck and path detection commands
- 🔄 **Auto-upgrade** - Built-in upgrade command to get the latest version

## Installation

### Quick Install (Recommended - One Command)

Install cursor-session with a single command. The installer automatically detects your platform and downloads the correct binary:

```bash
curl -fsSL https://raw.githubusercontent.com/iksnae/cursor-session/main/install-binary.sh | bash
```

That's it! The script will:

- Detect your OS and architecture (macOS Intel/ARM, Linux amd64/arm64)
- Download the latest pre-built binary
- Install to `~/.local/bin`
- Add to your PATH automatically

**Verify installation:**

```bash
cursor-session --version
```

### Manual Binary Download

If you prefer to download manually, visit the [Releases](https://github.com/iksnae/cursor-session/releases) page and download the appropriate binary for your platform:

- **macOS (Intel)**: `cursor-session-*-darwin-amd64.tar.gz`
- **macOS (Apple Silicon)**: `cursor-session-*-darwin-arm64.tar.gz`
- **Linux (x86_64)**: `cursor-session-*-linux-amd64.tar.gz`
- **Linux (ARM64)**: `cursor-session-*-linux-arm64.tar.gz`

Extract and move to your PATH:

```bash
tar -xzf cursor-session-*.tar.gz
mv cursor-session ~/.local/bin/
chmod +x ~/.local/bin/cursor-session
```

### Quick Install from Source (For Developers)

If you have Go installed, you can build from source:

```bash
# Clone the repository
git clone https://github.com/iksnae/cursor-session.git
cd cursor-session

# Run the install script (fully automatic - no manual steps!)
./install.sh
```

The script automatically builds, installs, and configures the tool. **No manual configuration needed!** Works on macOS (zsh) and Linux (bash/zsh).

### Using Go Install

**For stable releases:**

```bash
go install github.com/iksnae/cursor-session@latest
```

**For latest development version:**

```bash
go install github.com/iksnae/cursor-session@main
```

### Using Make

```bash
git clone https://github.com/iksnae/cursor-session.git
cd cursor-session
make install
```

### Manual Build

```bash
git clone https://github.com/iksnae/cursor-session.git
cd cursor-session
go build -buildvcs=false -o cursor-session .
sudo cp cursor-session /usr/local/bin/
```

### Verify Installation

```bash
cursor-session --version
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

# Check if everything is working correctly
cursor-session healthcheck

# Upgrade to the latest version
cursor-session upgrade
```

## Docker Development

For isolated development that matches the GitHub Actions environment, use Docker:

```bash
# Set CURSOR_API_KEY (optional but recommended for cursor-agent)
export CURSOR_API_KEY=your-api-key-here

# Start development container with cursor-agent
make docker-dev

# Access interactive shell
make docker-shell

# Inside container: build and test
make build
make test
./cursor-session healthcheck

# Seed database with cursor-agent
./cursor-session snoop --hello
```

See [Docker Development Guide](docs/DOCKER.md) for detailed instructions on:

- Setting up CURSOR_API_KEY for cursor-agent authentication
- Setting up cursor-agent in the container
- Testing storage behavior in Ubuntu environment
- Development workflow and troubleshooting

## Commands

### List Sessions

```bash
cursor-session list [--clear-cache]
```

Lists all available chat sessions with IDs, names, message counts, and creation dates.

### Show Session

```bash
cursor-session show <session-id> [--limit <number>] [--since <timestamp>]
```

Display messages from a specific session with optional filtering.

### Export Sessions

```bash
cursor-session export [--format <format>] [--out <directory>] [--workspace <hash>] [--session-id <id>] [--clear-cache]
```

Export sessions to various formats (jsonl, md, yaml, json). Filter by workspace or export a specific session.

### Health Check

```bash
cursor-session healthcheck [--verbose]
```

Verify that cursor-session can locate and access session data. Useful for debugging storage issues.

### Snoop (Path Detection)

```bash
cursor-session snoop [--hello]
```

Attempt to find the correct path to Cursor database files. Use `--hello` to seed the database with cursor-agent.

### Upgrade

```bash
cursor-session upgrade
```

Upgrade cursor-session to the latest released version from GitHub.

### Reconstruct (Debug)

```bash
cursor-session reconstruct
```

Reconstructs conversations and saves to intermediary JSON format. Primarily useful for debugging.

For detailed usage information, see the [Usage Guide](docs/USAGE.md).

## Requirements

- **Cursor IDE** or **cursor-agent CLI** installed
  - Desktop app: Extracts from globalStorage format (macOS/Linux)
  - Agent CLI: Extracts from cursor-agent storage (Linux only)
- macOS or Linux

## Releases

Releases are automatically created when git tags matching `v*` (e.g., `v1.0.0`, `v1.2.3`) are pushed to the repository. Each release includes:

- Pre-built binaries for macOS (Intel + ARM) and Linux (amd64 + arm64)
- SHA256 checksums for verification
- Release notes

**Version Numbering:**

- Follows [Semantic Versioning](https://semver.org/) (MAJOR.MINOR.PATCH)
- Use `@latest` with `go install` for stable releases
- Use `@main` for the latest development version

**Creating a Release:**

```bash
# Tag a new version
git tag v1.0.0
git push origin v1.0.0
```

The GitHub Actions workflow will automatically build binaries and create a release.

## Storage Backends

cursor-session supports two storage backends:

1. **Desktop App Storage** (macOS/Linux)

   - Location: `~/Library/Application Support/Cursor/User/globalStorage/state.vscdb` (macOS)
   - Location: `~/.config/Cursor/User/globalStorage/state.vscdb` (Linux)
   - Extracts from Cursor IDE's globalStorage database

2. **Agent CLI Storage** (Linux only)
   - Location: `~/.config/cursor/chats/` or `~/.cursor/chats/`
   - Extracts from cursor-agent CLI session databases
   - Automatically detected when cursor-agent is installed

The tool automatically detects and uses the available storage backend. Desktop app storage takes priority if both are available.

## Global Flags

- `--verbose, -v` - Enable verbose logging
- `--storage <path>` - Custom storage location (path to database file or storage directory)
- `--copy` - Copy database files to temporary location to avoid locking issues

## Documentation

- [Usage Guide](docs/USAGE.md) - Complete command reference
- [Docker Development Guide](docs/DOCKER.md) - Docker setup and development workflow
- [Implementation Details](docs/IMPLEMENTATION.md) - Technical implementation summary
- [Technical Design](docs/TDD.md) - Architecture and design decisions
- [Testing Guide](docs/TESTING.md) - Testing strategy and coverage

## License

MIT
