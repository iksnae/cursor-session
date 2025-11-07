![cursor session](./images/simple-icon.png)
[![codecov](https://codecov.io/gh/iksnae/cursor-session/branch/main/graph/badge.svg)](https://codecov.io/gh/iksnae/cursor-session)

# Cursor Session

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

## Documentation

- [Usage Guide](docs/USAGE.md) - Complete command reference
- [Implementation Details](IMPLEMENTATION.md) - Technical implementation summary
- [Technical Design](docs/TDD.md) - Architecture and design decisions

## License

MIT
