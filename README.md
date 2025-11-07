# Cursor Session Export CLI

A Go CLI tool to extract and export chat sessions from Cursor Editor's modern globalStorage format.

## Features

- ✅ Extract sessions from modern globalStorage format (cursorDiskKV table)
- ✅ Reconstruct conversations from bubbles, composers, and context data
- ✅ Export to multiple formats: JSONL, Markdown, YAML, JSON
- ✅ List available sessions with improved formatting
- ✅ Show messages for a specific session with filtering options
- ✅ Intelligent caching system (YAML index + JSON session files)
- ✅ Rich text extraction including thinking and tool calls
- ✅ Uses real Cursor session IDs (composerId)
- ✅ Async processing for performance
- ✅ Works on macOS and Linux

## Installation

### Quick Install (Recommended)

```bash
# Clone the repository
git clone <repository-url>
cd cursor-chat-cli

# Run the install script (fully automatic - no manual steps!)
./install.sh
```

The script automatically:

- ✅ Builds the binary
- ✅ Installs to `~/.local/bin`
- ✅ Detects your shell (zsh on macOS, bash/zsh on Ubuntu)
- ✅ Adds `~/.local/bin` to PATH in your shell config (`~/.zshrc` or `~/.bashrc`)
- ✅ Sources the config so it works immediately
- ✅ Verifies the installation

**No manual configuration needed!** Works on macOS (zsh) and Ubuntu (bash/zsh).

### Option 2: Using Go install

```bash
# Install directly from source
go install github.com/k/cursor-session@latest

# Or install from local directory
cd cursor-chat-cli
go install .
```

This installs to `$GOPATH/bin` or `$HOME/go/bin` (default). Make sure this directory is in your PATH.

### Alternative: Using Make

```bash
# Build and install
make install

# Or just build
make build
```

### Alternative: Manual installation

```bash
# Build the binary
go build -buildvcs=false -o cursor-session .

# Copy to a directory in your PATH
sudo cp cursor-session /usr/local/bin/  # System-wide
# OR
cp cursor-session ~/.local/bin/  # User-specific
```

### Verify Installation

After installation, verify it works:

```bash
# Check if it's in PATH
which cursor-session

# Test the command
cursor-session list

# Or if not in PATH yet
~/.local/bin/cursor-session list
```

## Usage

### List Sessions

```bash
cursor-session list
```

Lists all available chat sessions with their IDs, names, message counts, and creation dates. The list shows short IDs (first 8 characters) for readability, but you can use the full session ID with other commands.

```bash
# Clear cache and rebuild
cursor-session list --clear-cache
```

### Show Session Messages

```bash
cursor-session show <session-id>
```

Display messages from a specific session:

```bash
cursor-session show <session-id> --limit 10
cursor-session show <session-id> --since "2025-01-01T00:00:00Z"
```

### Export Sessions

```bash
# Export all sessions as JSONL (default)
cursor-session export

# Export as Markdown
cursor-session export --format md

# Export as YAML
cursor-session export --format yaml

# Export as JSON
cursor-session export --format json

# Specify output directory
cursor-session export --out ./my-exports

# Filter by workspace
cursor-session export --workspace <workspace-hash>
```

### Reconstruct Intermediary Format

```bash
cursor-session reconstruct
```

Reconstructs conversations and saves to intermediary JSON format for debugging.

### Upgrade

```bash
cursor-session upgrade
```

Upgrades cursor-session to the latest version by:
1. Pulling latest changes from the repository
2. Rebuilding the binary
3. Reinstalling to the current installation location

**Note**: This command works if you installed from a cloned repository. If you installed via `go install`, use:
```bash
go install github.com/k/cursor-session@latest
```

## Supported Formats

- **JSONL**: One message per line, machine-readable
- **Markdown**: Human-readable with code blocks preserved
- **YAML**: Structured data format
- **JSON**: Pretty-printed JSON

## Architecture

The tool uses a multi-step process:

1. **Extract**: Load bubbles, composers, and contexts from cursorDiskKV
2. **Reconstruct**: Match bubbles to conversation headers
3. **Normalize**: Convert to unified Session format
4. **Cache**: Store reconstructed sessions for fast access
5. **Export**: Write to desired format

### Caching

The tool uses an intelligent caching system:

- **Cache Location**: `~/.cursor-session-cache/`
- **Index File**: `sessions.yaml` - Contains session metadata
- **Session Files**: `session_<id>.json` - Individual session data
- Cache is automatically validated against database modification time
- Individual sessions are cached after reconstruction for faster future access

### Text Extraction

The tool uses a three-tier text extraction strategy:

1. **Primary**: Use `bubble.text` if available
2. **Fallback**: Parse `bubble.richText` JSON structure (including thinking/tool calls)
3. **Enhancement**: Append code blocks from `bubble.codeBlocks[]`

## Requirements

- Go 1.22+
- Cursor Editor with globalStorage (modern format)
- macOS or Linux

## License

MIT
