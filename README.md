# OctaAI - Autonomous Software Engineering Agent

OctaAI is an autonomous AI agent that can design, implement, test, deploy, and manage software projects end-to-end with minimal human intervention.

## Features

- **Autonomous Planning & Execution**: Takes high-level prompts and breaks them into actionable tasks
- **Multi-LLM Support**: Works with OpenAI, Claude, and Ollama (local models)
- **Code Generation & Self-Repair**: Generates code, runs tests, and fixes errors automatically
- **Remote Server Management**: SSH into servers, install packages, configure services
- **Deployment Automation**: Deploy applications, configure Nginx, setup SSL certificates
- **Browser Automation**: Control a live Firefox browser — navigate, fill forms, extract data, and automate web workflows via a companion extension
- **Safety First**: Configurable allowed paths and command filtering

## Architecture

```
octa-agentd  (Daemon - Agent Runtime)
     |
     +-- LLM Provider Layer (OpenAI/Claude/Ollama)
     +-- Tool Registry
          +-- Filesystem Tools
          +-- Code Execution Tools
          +-- Git Tools
          +-- SSH Tools
          +-- HTTP Tools
          +-- Browser Tools  ←── WebSocket ──► Firefox Extension
```

## Browser Automation

OctaAI can control a live Firefox browser through a companion extension, enabling web scraping, form automation, login flows, and multi-step web workflows.

### Quick setup

1. Build and start the daemon with browser support:
   ```bash
   ./bin/octa-agentd --browser-port 8765
   ```
2. Load the Firefox extension from this repo:
   ```bash
   # In Firefox: about:debugging → Load Temporary Add-on
   # → plugins/firefox-addon/src/manifest.json
   ```
3. Set a shared token in `config.yaml` and in the extension Settings (`Server URL`: `ws://localhost:8765/ws`).

See [docs/BROWSER_AUTOMATION.md](docs/BROWSER_AUTOMATION.md) for the full setup guide and [examples/browser/](examples/browser/) for usage examples.

## Quick Start

### Prerequisites

- Go 1.21+
- Ollama (optional, for local models)

### Installation

```bash
# Clone the repository
git clone https://github.com/mparvin/octaai.git
cd octaai

# Build the project
make build

# Or install directly
make install
```

### Configuration

Create a configuration file at `~/.config/octaai/config.yaml`:

```yaml
projects_root: "/home/user/Projects"

llm:
  provider: "ollama"
  model: "qwen2.5:32b"
  base_url: "http://localhost:11434"
  temperature: 0.3

safety:
  allow_paths:
    - "/home/user/Projects"
  deny_commands:
    - "rm -rf /"
```

### Usage

#### Start the Agent Daemon

```bash
octa-agentd
```

#### Submit a Goal (Example: LocalWeather Project)

```bash
octa-agent goal "Write a python script that gives weather data about all countries in Europe, store in redis, create a Flask site, name it LocalWeather"
```

#### Check Goal Status

```bash
octa-agent status
```

#### Example: Remote Server Deployment

```bash
octa-agent goal "IP: 1.2.3.4, Username: root, Password: 123456. Setup nginx, deploy github.com/mparvin/ip9.git, configure SSL for ip9.com"
```

## Project Structure

```
octaai/
├── cmd/
│   ├── octa-agentd/      # Agent daemon
│   └── octa-agent/       # CLI client
├── pkg/
│   ├── agent/            # Core agent logic
│   ├── config/           # Configuration
│   ├── llm/              # LLM provider abstraction
│   ├── storage/          # State persistence
│   └── tools/            # Tool implementations
├── examples/             # Example workflows
└── docs/                 # Documentation
```

## Development status

Core daemon + CLI, tools, permissions, Docker isolation, browser automation, and TF-IDF memory are production-usable.

Experimental v2 packages (HTN planner, capability registry, workflow DAG helpers) exist but are **not** wired into the engine until Phase 7 of [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md). Do not enable `features.*` expecting behavior changes.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) and [IMPLEMENTATION_LOG.md](IMPLEMENTATION_LOG.md).

## License

MIT License - See LICENSE file for details

## Contributing

Contributions are welcome! Please see CONTRIBUTING.md for guidelines.
