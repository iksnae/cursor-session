# Proof of Concept (POC) Report

### Project: Cursor Chat Export CLI

### Language: Go (v1.22+)

### Author: K

### Date: November 2025

---

## 1. Overview

This Proof of Concept (POC) explores how to **collect, parse, and export chat session data** from the **Cursor Editor**, which stores user interactions and AI chat data in both **SQLite** and **CacheStorage** formats.

The goal is to develop a **cross-platform Go CLI** capable of:

- Discovering and parsing Cursor chat history on **macOS** and **Linux**.
- Exporting structured session logs to **JSONL** or **Markdown** formats.
- Supporting both legacy (`workspaceStorage/state.vscdb`) and modern (`WebStorage/CacheStorage`) storage schemas.

This lays the foundation for integrating Cursor’s local session data into **Khaos Machine’s agent-session capture pipeline**, enabling unified historical records of messages, tool calls, and reasoning traces.

---

## 2. Objectives

| #   | Objective                   | Description                                                                   |
| --- | --------------------------- | ----------------------------------------------------------------------------- |
| 1   | Identify storage locations  | Confirm where Cursor stores chat logs on macOS and Linux.                     |
| 2   | Parse multiple formats      | Handle both SQLite (workspaceStorage) and Chromium CacheStorage (WebStorage). |
| 3   | Export chat sessions        | Output sessions as JSONL and Markdown for review or ingestion.                |
| 4   | Enable future extensibility | Support merging with external logs from Khaos Agent runtime.                  |

---

## 3. Findings

### 3.1 Cursor Storage Architecture

| Component            | Path (macOS)                                                 | Path (Linux)                             | Format                                     | Notes                                            |
| -------------------- | ------------------------------------------------------------ | ---------------------------------------- | ------------------------------------------ | ------------------------------------------------ |
| **WorkspaceStorage** | `~/Library/Application Support/Cursor/User/workspaceStorage` | `~/.config/Cursor/User/workspaceStorage` | SQLite (`state.vscdb`)                     | Stores historical chats in `ItemTable`.          |
| **WebStorage**       | `~/Library/Application Support/Cursor/User/WebStorage`       | `~/.config/Cursor/User/WebStorage`       | IndexedDB/CacheStorage (binary `_0` files) | Newer chat UIs and webviews store messages here. |

The **SQLite** database contains key-value records with keys like:

workbench.panel.aichat.view.aichat.chatdata
aiService.prompts
composerData

Values are typically JSON strings containing message arrays and metadata.

The **CacheStorage** folders include Chromium-managed cache blobs and indexes. Message data is often embedded as JSON fragments within binary files named `*_0`.

---

### 3.2 Confirmed Example Paths

**macOS Example:**

~/Library/Application Support/Cursor/User/
├── workspaceStorage/
│ └── /state.vscdb
└── WebStorage/
├── 1/CacheStorage//
│ ├── index
│ ├── index-dir/
│ ├── the-real-index
│ └── \*\_0

**Linux Example:**

~/.config/Cursor/User/
├── workspaceStorage//state.vscdb
└── WebStorage/1/CacheStorage//

---

## 4. Technical Design

### 4.1 CLI Structure

cursor-session/
├── main.go
├── cmd/
│ ├── root.go
│ ├── list.go
│ ├── export.go
│ └── scan.go
├── internal/
│ ├── detect.go # OS path detection
│ ├── sqlite_reader.go # Workspace parser
│ ├── cache_reader.go # WebStorage parser
│ ├── export.go # JSONL/Markdown writer
│ └── model.go # Session and message structs
└── go.mod

### 4.2 Key Dependencies

| Package                    | Purpose                        |
| -------------------------- | ------------------------------ |
| `modernc.org/sqlite`       | Pure-Go SQLite driver (no CGO) |
| `github.com/spf13/cobra`   | CLI framework                  |
| `github.com/tidwall/gjson` | Fast JSON extraction           |
| `encoding/json`            | Export serialization           |

---

## 5. Implementation Summary

### 5.1 Cross-Platform Path Detection

```go
switch runtime.GOOS {
case "darwin":
  base = "~/Library/Application Support/Cursor/User"
case "linux":
  base = "~/.config/Cursor/User"
default:
  base = ""
}
workspacePath := filepath.Join(base, "workspaceStorage")
webPath := filepath.Join(base, "WebStorage")


⸻

5.2 SQLite Session Extraction

SELECT key, value
FROM ItemTable
WHERE key IN (
  'aiService.prompts',
  'workbench.panel.aichat.view.aichat.chatdata',
  'composerData'
);

Each value is parsed into message arrays:

{
  "messages": [
    {"role": "user", "content": "Explain Go routines."},
    {"role": "assistant", "content": "Go routines are lightweight threads..."}
  ]
}


⸻

5.3 CacheStorage Parsing (POC level)

For each file ending with _0 inside WebStorage/*/CacheStorage/:
	•	Read file contents as binary.
	•	Locate the first { and last }.
	•	Attempt JSON unmarshal of that substring.
	•	Extract messages, role, content keys if present.

This heuristic allows parsing message arrays without needing full Chromium IndexedDB APIs.

⸻

5.4 Export Formats

JSONL

{"timestamp": "2025-11-07T14:23:01Z", "actor": "user", "content": "What is Cursor?"}
{"timestamp": "2025-11-07T14:23:03Z", "actor": "assistant", "content": "Cursor is an AI coding editor..."}

Markdown

# Session workbench.panel.aichat.view.aichat.chatdata

**user:** What is Cursor?
**assistant:** Cursor is an AI coding editor...


⸻

6. Validation Results

Test	macOS	Linux	Result
Path Detection	✅	✅	Correctly resolves both paths
SQLite Extraction	✅	✅	Sessions exported successfully
CacheStorage Scan	⚠️	⚠️	Partial success — JSON fragments found in some _0 files
JSONL Export	✅	✅	Valid structured output
Markdown Export	✅	✅	Readable text transcripts


⸻

7. Limitations
	•	CacheStorage parsing is heuristic — some files are binary blobs without readable JSON.
	•	No tool-call metadata — Cursor doesn’t record reasoning/tool invocations internally.
	•	Session metadata (timestamps, ids) may be incomplete for merged exports.
	•	GlobalStorage variants not yet covered (some Cursor builds use it).

⸻

8. Next Steps

Priority	Task	Description
High	Extend CacheStorage parser	Build proper IndexedDB parser to replace substring extraction.
High	Merge external Khaos Agent logs	Add merge subcommand to unify chat + tool logs chronologically.
Medium	Add web viewer	Serve local HTML exports via cursor-session serve.
Medium	Add filters	Support --since, --workspace, --contains "query".
Low	Add Windows support	Path detection + binary build cross-compilation.


⸻

9. Future Integration with Khaos

Once the cursor-session CLI stabilizes:
	1.	Integrate with Khaos Forge — attach as a background agent command (/export-session).
	2.	Stream session data into Story-Data-Realtime — for visualization of developer/agent interactions.
	3.	Support “thinking traces” — extend CLI to merge synthetic reasoning logs emitted by custom agents.

⸻

10. Conclusion

This POC demonstrates that Cursor’s local chat history can be successfully parsed and exported from both workspaceStorage and WebStorage directories on macOS and Linux using a standalone Go CLI.

The approach offers a clear foundation for extending toward full session capture, agent trace integration, and cross-tool analytics in the Khaos ecosystem.

⸻

Status: ✅ POC Complete
Next Milestone: Implement full CacheStorage parser and merge tool-call event streams.

⸻


```
