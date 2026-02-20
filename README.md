# Idony

A highly opinionated, extensible multi-agent AI system written in Go.

## Features
- **Dual-Mode Architecture**: Run as a background daemon (`idony-server`) with a rich TUI client (`idony`).
- **Team Management**: Define specialized sub-agents with unique personalities, models, and toolsets.
- **Collaborative Reasoning**: Run "Councils" where multiple agents discuss and solve problems together.
- **Hierarchical Planning**: Interactive project and task management system.
- **Rich Toolset**:
    - **Web Surfing**: Search and scrape content via headless browser.
    - **Media**: Transcribe YouTube videos and audio files locally via Whisper.
    - **Communication**: Send/Receive emails and interact via Telegram (voice & text).
    - **Automation**: Schedule tasks using cron or one-shot timers.
    - **System**: Direct access to shell, files, and local image generation (SwarmUI).

## Installation

### 1. Prerequisites (Debian 13)
```bash
sudo apt update && sudo apt install yt-dlp ffmpeg build-essential cmake libopenblas-dev flite
```

### 2. Setup Configuration
```bash
cp config.example.txt config.txt
# Edit config.txt with your API keys and preferences
```

### 3. Install as System Service
```bash
./scripts/install_service.sh
```

### 4. Run the Client
```bash
./idony
```

## Hotkeys
- `Ctrl+P`: Toggle Project Planner
- `Ctrl+H`: Toggle History/Agents Side Panel
- `/exit`: Quit

## License
MIT
