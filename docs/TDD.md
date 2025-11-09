# Technical Design Document

### Project: Cursor Session Export CLI

### Author: K

### Date: November 2025

---

## 1. Purpose

The purpose of this document is to define the **technical architecture, design decisions, and implementation strategy** for the `cursor-session` CLI.
This tool will enable cross-platform (macOS + Linux) discovery, extraction, and export of **Cursor Editor chat sessions**, unifying legacy (SQLite-based) and modern (CacheStorage-based) data into structured session logs.

The system forms part of the **Khaos Machine developer observability layer**, allowing Cursor Agent and chat histories to be indexed, analyzed, and merged with other agent session data (tool calls, reasoning traces, etc.).

---

## 2. Goals & Non-Goals

### 2.1 Goals

- ✅ Discover Cursor chat session data across macOS and Linux.
- ✅ Parse modern storage backends:
  - Desktop app: SQLite (`state.vscdb` in globalStorage)
  - Agent CLI: SQLite (`store.db` files in cursor-agent storage) - Linux only
- ✅ Export structured logs in **JSONL**, **Markdown**, **YAML**, and **JSON**.
- ✅ Support both desktop app and cursor-agent CLI storage formats.
- ✅ Intelligent caching for fast access.
- ✅ Diagnostic tools for troubleshooting.

### 2.2 Non-Goals

- ❌ Direct access to Cursor's internal APIs.
- ❌ Real-time monitoring of Cursor processes.
- ❌ Full IndexedDB schema reconstruction.
- ❌ Chain-of-thought or reasoning reconstruction (beyond exposed data).
- ❌ Windows support for agent storage (initially).

---

## 3. System Overview

### 3.1 High-Level Diagram

+––––––––––––––––––––––––––––+
| Cursor Session Export CLI |
+––––––––––––––––––––––––––––+
| |
| [OS Path Detector] → Detects base storage paths |
| [Storage Scanner] → Finds .vscdb / CacheStorage |
| [SQLite Parser] → Extracts chat messages |
| [Cache Parser] → Heuristic JSON extraction |
| [Normalizer] → Converts to unified schema |
| [Exporter] → JSONL / Markdown / HTML |
| [Ext. Log Integrator] → Merges Khaos Agent data |
| |
+––––––––––––––––––––––––––––+
↑ ↓
Local FS paths Exported session archives

---

## 4. Environment & Platform Support

| OS      | Desktop App Path                             | Agent CLI Path              | Format(s)        | Supported |
| ------- | --------------------------------------------- | -------------------------- | ---------------- | --------- |
| macOS   | `~/Library/Application Support/Cursor/User/` | N/A                        | SQLite (globalStorage) | ✅        |
| Linux   | `~/.config/Cursor/User/`                      | `~/.config/cursor/chats/`  | SQLite (both)    | ✅        |
| Windows | `%APPDATA%/Cursor/User/`                      | N/A                        | SQLite           | Planned   |

---

## 5. Components

### 5.1 CLI Layer (`cmd/`)

Implements user-facing commands via **Cobra**.
Each subcommand maps to an operation in the core package.

| Command      | Description                                    |
| ------------ | ---------------------------------------------- |
| `list`       | Lists available sessions from both storages.   |
| `show`       | Displays messages from a specific session.     |
| `export`     | Exports sessions in desired format.            |
| `healthcheck`| Verifies storage access and session availability. |
| `snoop`      | Finds database file paths and verifies access.  |
| `upgrade`    | Upgrades to latest version from GitHub.         |
| `reconstruct`| Reconstructs conversations (debug).             |

---

### 5.2 Core Modules (`internal/`)

#### 5.2.1 detect.go

- Determines base path dynamically by OS.
- Provides a struct:

  ```go
  type StoragePaths struct {
      Workspace string
      Web       string
  }

  5.2.2 storage.go (Desktop App Storage)
  	•	Opens state.vscdb using modernc.org/sqlite.
  	•	Queries cursorDiskKV table for bubble and composer data.
  	•	Parses JSON values with gjson.
  	•	Extracts messages, contexts, and code blocks.
  	•	Returns unified Session structs.

  5.2.3 agent_storage.go (Agent CLI Storage)
  	•	Scans cursor-agent storage directory for store.db files.
  	•	Opens multiple SQLite databases (one per session).
  	•	Queries session tables for messages and metadata.
  	•	Returns normalized Session objects.

  5.2.4 model.go
  Defines base data structures:
  ```

type Message struct {
Timestamp string `json:"timestamp,omitempty"`
Actor string `json:"actor"`
Content string `json:"content"`
}

type Session struct {
ID string `json:"id"`
Workspace string `json:"workspace,omitempty"`
Source string `json:"source"` // sqlite | cache
Messages []Message `json:"messages"`
}

5.2.5 export.go
Handles export logic for multiple formats:
• JSONL: one message per line.
• Markdown: easy human-readable transcript.
• HTML: optional for local viewing.

⸻

6. Data Flow

6.1 Extraction Sequence

User runs: cursor-session export --format jsonl

1. Detect OS → resolve Cursor storage paths (desktop app and/or agent CLI).
2. Check for desktop app storage (globalStorage/state.vscdb).
3. If not found, check for agent CLI storage (~/.config/cursor/chats/).
4. Load data from selected backend (bubbles, composers, contexts).
5. Reconstruct conversations from raw data.
6. Normalize all sessions → assign UUIDs, associate workspaces.
7. Cache sessions for future fast access.
8. Export to /exports/<session-id>.<format>

⸻

7. Error Handling & Resilience

Condition Handling
Missing paths Warn user; skip and continue.
Corrupt DB file Log warning; continue.
Non-JSON content Skip; continue to next file.
Permission denied Prompt with hint to use sudo or adjust permissions.
Output directory missing Auto-create exports/.

⸻

8. Logging & Telemetry
   • Use structured logs (timestamp + event).
   • Optional --verbose flag for debug output.
   • Example:

[2025-11-07T14:10:00Z] INFO Found 3 SQLite DBs
[2025-11-07T14:10:03Z] WARN Cache file parse failed: index_4_0
[2025-11-07T14:10:05Z] INFO Exported 5 sessions → exports/

⸻

9. Export Specification

9.1 JSONL Schema

Field Type Description
timestamp string Optional ISO8601 timestamp
actor string user, assistant, or tool
content string Message text

9.2 Markdown Schema

# Session <ID>

**user:** What is Cursor?
**assistant:** Cursor is an AI coding editor...

⸻

10. Extensibility

Future Feature Description
Agent Log Merge Integrate Khaos Agent tool-call logs to unify story of conversation + reasoning.
IndexedDB Parser Replace binary JSON heuristic with structured index parsing.
Web UI Viewer Serve local viewer via cursor-session serve.
Query Interface Filter sessions (--since, --workspace, --contains "search term").
Cloud Sync Option to push exports to Khaos backend via REST/GraphQL.

⸻

11. Example Workflow

# List sessions

cursor-session list

# Show messages from a session

cursor-session show <session-id> --limit 10

# Export all sessions as JSONL

cursor-session export --format jsonl --out ./exports

# Export as Markdown for readability

cursor-session export --format md

# Export specific session

cursor-session export --session-id <id> --format md

# Check storage health

cursor-session healthcheck

# Find database paths

cursor-session snoop

# Upgrade to latest version

cursor-session upgrade

Output structure:

exports/
├── session_2025-11-07T14-00-00Z.jsonl
├── session_2025-11-07T14-00-00Z.md
└── logs/

⸻

12. Performance Considerations

Operation Est. Time Notes
SQLite scan O(N) per workspace Typically < 100ms per DB
CacheStorage scan O(N) per cache file JSON detection heuristic; may require parallelism
Export Linear with message count Stream-based writer avoids memory pressure

Parallel scanning via Go routines is safe; bounded concurrency recommended (runtime.NumCPU()).

⸻

13. Security & Privacy
    • All operations are local — no remote API calls.
    • Sensitive data (e.g., chat content) stays on the user’s machine.
    • No telemetry or analytics by default.
    • Future remote sync (Khaos integration) will require explicit opt-in with encryption.

⸻

14. Testing Strategy

Test Type Description
Unit Tests Functions for path detection, SQLite parsing, export formatting.
Integration Tests Run CLI against mock Cursor directories.
Cross-Platform Tests Validate on macOS and Linux under CI.
Regression Tests Re-run after schema changes to verify backward compatibility.

⸻

15. Deliverables
    • ✅ cursor-session binary (macOS/Linux)
    • ✅ Source code in Go with modular packages
    • ✅ Example dataset for testing (fixtures/)
    • ✅ Documentation:
    • README.md
    • USAGE.md
    • IMPLEMENTATION.md
    • TDD.md
    • TESTING.md
    • Research documentation in docs/research/
    • ✅ Multiple storage backend support
    • ✅ Caching system
    • ✅ Diagnostic tools
    • ✅ Auto-upgrade functionality
    • 🚧 Future: Dockerized version for CI automation

⸻

16. Conclusion

The cursor-session CLI bridges the gap between local AI-assisted coding sessions and structured analytical pipelines.
It provides a solid technical foundation for future session intelligence, enabling Khaos Machine to ingest developer–agent interactions for insight, replay, and automation.

This design supports both backward compatibility and forward evolution toward full agent trace unification.

⸻

Status: ✅ Implementation Complete

**Current Status:**
- ✅ Desktop app storage (globalStorage) support - macOS/Linux
- ✅ Agent CLI storage support - Linux
- ✅ Multiple export formats (JSONL, Markdown, YAML, JSON)
- ✅ Caching system for fast access
- ✅ Diagnostic tools (healthcheck, snoop)
- ✅ Auto-upgrade functionality
- ✅ Workspace association
- ✅ Progress indicators

**Future Enhancements:**
- Windows support for agent storage
- Search functionality across sessions
- Date range filtering
- Integration with Khaos Machine runtime agents
