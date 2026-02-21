# Idony Features V3 Plan: Tools & Automation

## ⚡ Tools & Automation

### 1. Shell Commands (Enhanced)
**Goal:** Run shell commands with better safety controls.
*   **Existing:** `ShellExecTool` in `internal/tools/system.go`.
*   **Enhancements:**
    *   Add **Timeout** support (default 30s).
    *   Add **Allowlist/Blocklist** via config (e.g., block `rm -rf /`).
    *   Ensure output capturing is robust.

### 2. File Operations (Enhanced)
**Goal:** Comprehensive file system control with safety.
*   **Existing:** `ListFilesTool`, `ReadFileTool`, `WriteFileTool`.
*   **New Tools:**
    *   `DeleteFileTool`: Remove files.
    *   `SearchFileTool`: Find files by name pattern.
*   **Safety:**
    *   **Path Restriction:** Enforce operations strictly within the project root or allowed paths.
    *   **Size Limit:** Prevent reading huge files (truncate output).

### 3. Browser Automation
**Goal:** Advanced web interaction (click, type, screenshot).
*   **Current:** Wraps `idony-browser` CLI.
*   **Implementation:**
    *   Integrate `go-rod` directly into `internal/tools/browser_native.go` (replacing or augmenting the CLI wrapper).
    *   This allows staying in Go without needing Node/Puppeteer.
    *   Add actions: `click`, `type`, `screenshot`, `navigate`.

### 4. Web Search
**Goal:** dedicated search API integration.
*   **Implementation:**
    *   Create `WebSearchTool`.
    *   Support **DuckDuckGo** (HTML scraping for privacy/free) or **Google Custom Search** (API Key).
    *   Return structured results (Title, Link, Snippet).

### 5. Scheduled Tasks (Enhanced)
**Goal:** Manage the scheduler.
*   **Existing:** `ScheduleTool` allows adding.
*   **Enhancements:**
    *   Add `list_tasks` action to `ScheduleTool`.
    *   Add `delete_task` action.
    *   Add `pause_task` (if DB supports it).

### 6. Webhook Triggers
**Goal:** Trigger agents from external events.
*   **Implementation:**
    *   **Server:** Add `POST /webhooks/{id}` endpoint.
    *   **Database:** `webhooks` table (`id`, `name`, `target_agent`, `prompt_template`).
    *   **Tool:** `WebhookTool` to create/manage these listeners.
    *   **Logic:** When webhook hits, substitute payload into template and spawn agent.

### 7. MCP Tool Bridge
**Goal:** Connect to Model Context Protocol servers.
*   **Implementation:**
    *   Create `internal/mcp/client.go`.
    *   Read MCP config (JSON file with server commands).
    *   Start MCP servers as subprocesses (stdio).
    *   Dynamically register their tools into the Agent.

---

**Implementation Order:**
1.  System Tools (Shell & File safety/expansion).
2.  Browser Native (go-rod).
3.  Web Search.
4.  Scheduler Management.
5.  Webhooks.
6.  MCP Bridge.
