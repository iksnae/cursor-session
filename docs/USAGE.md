# Usage Guide

Complete guide to using the Cursor Session Export CLI for extracting chat sessions from Cursor IDE.

## Installation

For installation instructions, including downloading pre-built binaries, see the [main README](../README.md#installation).

Quick options:
- **Download pre-built binary** (recommended, no Go required) - See [Releases](https://github.com/iksnae/cursor-session/releases)
- **Build from source** - Use `./install.sh` or `go install github.com/iksnae/cursor-session@latest`

## Commands

### List Sessions

```bash
cursor-session list [--clear-cache]
```

Lists all available chat sessions with their IDs, names, message counts, and creation dates. The list shows short IDs (first 8 characters) for readability, but you can use the full session ID with other commands.

**Options:**
- `--clear-cache` - Clear the cache and rebuild the session index

**Global flags:**
- `--verbose, -v` - Enable verbose logging
- `--storage <path>` - Custom storage location
- `--copy` - Copy database files to temporary location to avoid locking issues

### Show Session Messages

```bash
cursor-session show <session-id> [--limit <number>] [--since <timestamp>]
```

Display messages from a specific session with formatted output showing user and assistant messages.

**Options:**
- `--limit <number>`, `-n <number>` - Limit the number of messages shown
- `--since <timestamp>` - Only show messages after this timestamp (ISO 8601 / RFC3339 format)

**Examples:**
```bash
cursor-session show abc123def456 --limit 10
cursor-session show abc123def456 --since "2025-01-01T00:00:00Z"
cursor-session show abc123def456 -n 5
```

**Global flags: `--verbose`, `--storage`, `--copy`**

### Export Sessions

```bash
cursor-session export [options]
```

Export sessions to various formats. Supports exporting all sessions, filtering by workspace, or exporting a specific session by ID.

**Options:**
- `--format <format>`, `-f <format>` - Export format: `jsonl` (default), `md`, `yaml`, or `json`
- `--out <directory>`, `-o <directory>` - Output directory (default: `./exports`)
- `--workspace <hash>` - Filter by workspace hash
- `--session-id <id>` - Export a specific session by ID
- `--clear-cache` - Clear the cache before running
- `--intermediary` - Save intermediary format (for debugging)

**Examples:**
```bash
# Export all sessions as JSONL (default)
cursor-session export

# Export as Markdown
cursor-session export --format md

# Export to a specific directory
cursor-session export --out ./my-exports

# Export only sessions from a specific workspace
cursor-session export --workspace abc123

# Export a specific session
cursor-session export --session-id abc123def456 --format md

# Export with cache cleared
cursor-session export --format yaml --clear-cache
```

**Global flags: `--verbose`, `--storage`, `--copy`**

### Health Check

```bash
cursor-session healthcheck [--verbose]
```

Check the health of cursor-session by verifying:
- Storage path detection
- Storage format availability (desktop app or agent CLI)
- Session data accessibility
- Session count

This command is useful for debugging storage issues, especially in CI/CD environments.

**Options:**
- `--verbose`, `-v` - Show detailed diagnostic information

**Examples:**
```bash
cursor-session healthcheck
cursor-session healthcheck --verbose
```

**Global flags: `--storage`, `--copy`**

### Snoop (Path Detection)

```bash
cursor-session snoop [--hello]
```

Attempt to find the correct path to Cursor database files across different operating systems. This command will:
- Check standard storage paths for your OS
- Verify if database files exist at those locations
- Display detailed information about what was found
- Optionally seed the database with `--hello` flag

**Options:**
- `--hello` - Invoke cursor-agent with a simple prompt to seed the database

**Examples:**
```bash
cursor-session snoop
cursor-session snoop --hello
```

**Global flags: `--verbose`, `--storage`, `--copy`**

### Upgrade

```bash
cursor-session upgrade
```

Upgrades cursor-session to the latest released version from GitHub. The command will:
1. Check your current installed version
2. Fetch the latest release from GitHub
3. Download and install the latest binary if a newer version is available

**Alternative**: If you installed via `go install`, you can also upgrade by running:
```bash
go install github.com/iksnae/cursor-session@latest
```

### Reconstruct (Debug)

```bash
cursor-session reconstruct
```

Reconstructs conversations and saves to intermediary JSON format. This is primarily useful for debugging or understanding the raw data structure.

**Global flags: `--verbose`, `--storage`, `--copy`**

## Export Formats

- **JSONL** (default): One message per line, machine-readable format
- **Markdown**: Human-readable format with code blocks preserved
- **YAML**: Structured data format
- **JSON**: Pretty-printed JSON format

## Session IDs

Session IDs are shown in shortened form (first 8 characters) in the list command for readability. You can use either the short ID or the full ID with other commands - the tool will match either format.

## Storage Backends

cursor-session supports two storage backends:

1. **Desktop App Storage** (macOS/Linux)
   - Extracts from Cursor IDE's globalStorage database
   - Location: `~/Library/Application Support/Cursor/User/globalStorage/state.vscdb` (macOS)
   - Location: `~/.config/Cursor/User/globalStorage/state.vscdb` (Linux)

2. **Agent CLI Storage** (Linux only)
   - Extracts from cursor-agent CLI session databases
   - Location: `~/.config/cursor/chats/` or `~/.cursor/chats/`
   - Automatically detected when cursor-agent is installed

The tool automatically detects and uses the available storage backend. Desktop app storage takes priority if both are available.

## Caching

Sessions are cached in `~/.cursor-session-cache/` for faster access. The cache is automatically validated and updated when Cursor's data changes. Use `--clear-cache` if you need to force a refresh.

The cache includes:
- Session index for fast listing
- Individual session files for quick access
- Automatic invalidation when source data changes

## Workspace Association

Sessions are automatically associated with workspaces based on where they were created. You can filter exports by workspace using the `--workspace` flag with the workspace hash shown in the list command.

## Global Flags

These flags are available for all commands:

- `--verbose, -v` - Enable verbose logging for debugging
- `--storage <path>` - Custom storage location (path to database file or storage directory)
- `--copy` - Copy database files to temporary location to avoid locking issues (useful when Cursor is running)

## Troubleshooting

### No sessions found

1. Run `cursor-session healthcheck` to verify storage detection
2. Run `cursor-session snoop` to check database file locations
3. Ensure Cursor IDE or cursor-agent has been used to create sessions
4. Check that you have read permissions for the storage directories

### Database locked errors

Use the `--copy` flag to copy database files to a temporary location:
```bash
cursor-session list --copy
cursor-session export --copy
```

### Agent storage not detected

On Linux, ensure cursor-agent is installed and has created sessions:
```bash
cursor-session snoop --hello
```

This will attempt to trigger cursor-agent to create a session if it's installed.
