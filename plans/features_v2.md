# Idony Features V2 Plan

This plan outlines the implementation of advanced Memory and Agent Architecture features.

## 🧠 Memory & Context

### 1. SQLite Memory (Enhanced)
**Goal:** Persist facts, preferences, and retrieve relevant context for LLM calls.
*   **Database:** Create `memories` table in SQLite (`id`, `content`, `type` [fact/preference/observation], `tags`, `created_at`).
*   **Tools:** Implement `remember` tool to explicitly save information.
*   **Integration:**
    *   Update `Agent` to fetch "relevant" memories (keyword match or recent) and inject them into the System Prompt.
    *   Ensure `save_memory` tool is registered.

### 2. Knowledge Graph
**Goal:** Structured memory as interconnected entities.
*   **Database:** Create `graph_nodes` (`id`, `label`, `type`) and `graph_edges` (`source_id`, `target_id`, `relation`).
*   **Tools:**
    *   `/graph_add`: Add a triple (Subject -> Predicate -> Object).
    *   `/graph_query`: Traverse the graph (e.g., "What is connected to ProjectX?").
*   **Visualization:** (Future) Add a graph view to the PWA.

### 3. Context Pruning
**Goal:** Manage token limits by summarizing old context.
*   **Command:** `/compact`
*   **Logic:**
    *   Select the oldest N messages from the active session.
    *   Send to LLM: "Summarize this conversation chunk."
    *   Delete old messages from DB and insert a new "system" message with the summary.

### 4. Multimodal Memory
**Goal:** Index information from non-text sources.
*   **Database:** Add `metadata` column to `messages` or a new `media_index` table.
*   **Logic:** When `RunVision` or `Transcribe` is used, store the *result* (text description/transcript) alongside the file path.
*   **Retrieval:** Allow searching for "that image about X" which queries the stored descriptions.

### 5. Self-Evolving Memory
**Goal:** Maintenance of the memory bank.
*   **Mechanism:** Background Scheduled Task (Cron).
*   **Logic:**
    *   Fetch all memories.
    *   Ask LLM: "Identify duplicates, contradictions, or outdated facts in this list. Return a JSON of operations (delete, merge, update)."
    *   Apply operations.

### 6. Markdown Memory
**Goal:** Human-readable backing.
*   **Logic:** Two-way sync or Write-Only log.
*   **Implementation:** Whenever a memory is saved to SQLite, append it to `memories/facts.md` or `memories/journal.md`.

---

## 🏗️ Agent Architecture

### 1. Agentic Tool Loop
*   **Status:** Partially exists in `internal/agent/agent.go`.
*   **Improvement:** Ensure the loop is robust (max iterations config) and explicitly exposed as a capability.

### 2. Agent Swarms
**Goal:** Specialized collaboration.
*   **Implementation:** Define a "Swarm" as a pre-configured group of Sub-Agents (e.g., "DevTeam" = Coder + Reviewer).
*   **Action:** `/swarm <goal>`. Spawns the group and assigns sub-tasks.

### 3. Agent-to-Agent Comms
**Goal:** Inter-agent messaging.
*   **Database:** `agent_messages` table.
*   **Logic:** Allow an agent to "send" a message to another agent ID. The recipient processes it in their next cycle.

### 4. Mesh Workflows
**Goal:** Autonomous goal decomposition and execution.
*   **Command:** `/mesh <goal>`
*   **Logic:**
    1.  **Plan:** Agent decomposes goal into steps (JSON).
    2.  **Execute:** Loop through steps. For each step, determine if it needs a sub-agent or a tool.
    3.  **Refine:** Use results of Step N to inform Step N+1.
