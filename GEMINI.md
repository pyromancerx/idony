# GEMINI.md - Project Context: Idony

This file provides architectural and operational context for **Idony**, a highly opinionated, extensible AI agent system.

## Project Overview

**Idony** is a Go-based multi-agent system featuring a TUI interface, SQLite persistence, and a modular tool architecture. It is designed to manage a team of specialized sub-agents and facilitate collaborative problem-solving through a "Council" system.

- **Status:** v1.3.0 (Multi-Target Scheduling & Specialized Agents)
- **Language:** Go (Golang)
- **TUI Framework:** `github.com/rivo/tview`
- **Database:** SQLite (CGO-free via `modernc.org/sqlite`)
- **LLM Backend:** Ollama (Local)
- **Coding Engine:** `gemini-cli`

## Core Architecture

### 1. Agentic Loop
Idony operates on a structured **Think -> Plan -> Act -> Observe** cycle.
- **Thought:** LLM reasoning in JSON format.
- **Action:** Tool execution or final response.
- **Observation:** Result of tool execution fed back to the LLM.

### 2. Specialized Sub-Agents
- **Definitions:** unique personas with specific personalities, restricted toolsets, and model overrides.
- **Persistence:** Definitions and tasks are stored in SQLite.
- **Monitoring:** Active sub-agents are tracked in real-time in the UI.

### 3. Council System
- **Collaboration:** Groups of sub-agents can be tasked with solving a single problem together.
- **Multi-turn:** Discussion occurs in rounds where agents see each other's contributions.

### 4. Modular Tools
- **Extensible:** New tools implement a simple `Tool` interface.
- **Direct Access:** All tools are available as `/command` in the terminal.
- **Configurable:** Uses a modular `config.txt` for all tool settings.

## Directory Structure
- `cmd/idony/`: Main entry point and TUI logic.
- `internal/agent/`: Core agent, sub-agent manager, scheduler, and council logic.
- `internal/llm/`: Ollama API client.
- `internal/tools/`: Built-in capabilities (Gemini, SwarmUI, Browser, etc.).
- `internal/db/`: Persistence layer.
- `internal/config/`: Modular configuration management.

## Current Goals
- **Web Browser Integration:** Building a standalone `idony-browser` utility to allow agents to surf and scrape the web.
- **Mobile/Social Expansion:** (Planned) Integration with Telegram and Discord.

## Development Conventions
- **JSON for Agent IO:** Always use structured JSON for agent thought processes and tool parameters.
- **Local-First:** Prioritize local tools and models.
- **Non-blocking:** Background tasks (sub-agents, councils) must not block the main UI thread.

---
*Last Updated: 2026-02-19*
