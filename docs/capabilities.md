# Idony Capabilities

Idony is a multi-agent AI system with a wide range of integrated tools.

## 1. Agent Management
- **Specialized Agents**: Create bots with unique names, personalities, and toolsets.
- **Councils**: Group multiple agents to solve complex problems through discussion.
- **Interactive Creation**: Idony can help you define and configure new agents through chat.

## 2. Web & Research
- **Web Browser**: Search Google and scrape clean Markdown content from any URL.
- **RSS Reader**: Subscribe to feeds and get automated summaries of new items.

## 3. Media & Audio
- **YouTube Transcriber**: Grab transcripts from videos (official subs or auto-generated).
- **Audio Transcriber**: Local speech-to-text using OpenAI Whisper (static binary).
- **Vision & Multimodal**: Analyze images sent via Telegram, the TUI, or the PWA.
- **Sub-Agent Vision**: Main agents can delegate visual tasks to sub-agents, who automatically inherit or receive image data.
- **Text-to-Speech**: Reply with audio messages using CMU Flite.

## 4. Communication & Interfaces
- **Telegram**: Full bot integration with support for text, voice, and photos.
- **Email**: Send and receive emails via SMTP/IMAP (supports SSL/TLS and trusted senders).
- **Go-Powered PWA**: A native-feeling web interface written in Go (WebAssembly).
- **Server-Driven UI (SDUI)**: Tools define their own UI schemas, allowing the mobile/web clients to render complex forms dynamically without code changes.

## 5. System & Automation
- **Scheduling**: Run any prompt on a recurring (cron) or one-shot (timestamp) basis.
- **Project Planner**: Manage hierarchical projects and tasks with agent assignments.
- **Shell Access**: Execute commands, read/write files, and list directories.
- **Image Generation**: Local image creation via SwarmUI integration.
- **Live Configuration**: Reload `config.txt` settings on-the-fly without restarting.
