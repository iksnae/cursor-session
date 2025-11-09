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
cursor-session list
```

Lists all available chat sessions with their IDs, names, message counts, and creation dates. The list shows short IDs (first 8 characters) for readability, but you can use the full session ID with other commands.

**Options:**
- `--clear-cache` - Clear the cache and rebuild the session index

### Show Session Messages

```bash
cursor-session show <session-id>
```

Display messages from a specific session.

**Options:**
- `--limit <number>` - Limit the number of messages shown
- `--since <timestamp>` - Only show messages after this timestamp (ISO 8601 format)

**Examples:**
```bash
cursor-session show abc123def456 --limit 10
cursor-session show abc123def456 --since "2025-01-01T00:00:00Z"
```

### Export Sessions

```bash
cursor-session export [options]
```

Export sessions to various formats.

**Options:**
- `--format <format>` - Export format: `jsonl` (default), `md`, `yaml`, or `json`
- `--out <directory>` - Output directory (default: `./exports`)
- `--workspace <hash>` - Filter by workspace hash

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
```

### Reconstruct (Debug)

```bash
cursor-session reconstruct
```

Reconstructs conversations and saves to intermediary JSON format. This is primarily useful for debugging or understanding the raw data structure.

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

## Export Formats

- **JSONL** (default): One message per line, machine-readable format
- **Markdown**: Human-readable format with code blocks preserved
- **YAML**: Structured data format
- **JSON**: Pretty-printed JSON format

## Session IDs

Session IDs are shown in shortened form (first 8 characters) in the list command for readability. You can use either the short ID or the full ID with other commands - the tool will match either format.

## Caching

Sessions are cached in `~/.cursor-session-cache/` for faster access. The cache is automatically validated and updated when Cursor's data changes. Use `--clear-cache` if you need to force a refresh.

## Workspace Association

Sessions are automatically associated with workspaces based on where they were created. You can filter exports by workspace using the `--workspace` flag with the workspace hash shown in the list command.

