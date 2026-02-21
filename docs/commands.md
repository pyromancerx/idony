# Idony Command Reference

All tools can be accessed directly using the `/` prefix in the TUI client.

## Core Commands
- `/exit` or `/quit`: Close the application.
- `/model <name>`: Switch the LLM model for the current session.
- `/image <path> [prompt]`: Send a local image for analysis.

## Tool Commands
- `/ls [path]`: List files.
- `/cat <path>`: Read file content.
- `/exec <cmd>`: Run shell command.
- `/browser {"action": "search|scrape", ...}`: Web interaction.
- `/transcribe {"action": "youtube|file", ...}`: Media processing.
- `/email {"action": "send|check", ...}`: Manage mail.
- `/rss {"action": "add|list|fetch"}`: News aggregation.
- `/planner {"action": "create_project|add_task", ...}`: Project management.
- `/subagent {"action": "spawn|spawn_named|result|list|define", ...}`: Manage specialized agents. Inherits images from context.
- `/council {"action": "define|run", ...}`: Group collaboration.
- `/update_config <KEY=VALUE>`: Update a setting in memory and save to `config.txt`.
- `/reload_config`: Reload all settings from `config.txt` and refresh the agent.
- `/update_personality <text>`: Update the main bot persona.

## TUI Hotkeys
- `Tab`: Cycle focus between windows.
- `Arrows & Enter`: Navigate the status bar menu or project tree.
- `Ctrl+C`: Force quit.
