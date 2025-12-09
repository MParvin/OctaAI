# OctaAI - Autonomous Software Engineering Agent

OctaAI is an autonomous AI agent that can design, implement, test, deploy, and manage software projects end-to-end with minimal human intervention.

## Features

- **Autonomous Planning & Execution**: Takes high-level prompts and breaks them into actionable tasks
- **Multi-LLM Support**: Works with OpenAI, Claude, and Ollama (local models)
- **Code Generation & Self-Repair**: Generates code, runs tests, and fixes errors automatically
- **Remote Server Management**: SSH into servers, install packages, configure services
- **Deployment Automation**: Deploy applications, configure Nginx, setup SSL certificates
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
```

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

## Development Phases

- [x] Phase 1: Skeleton & LLM Provider
- [x] Phase 2: Filesystem & Code Runner Tools
- [ ] Phase 3: Error Loop & Self-Repair
- [ ] Phase 4: SSH Tool
- [ ] Phase 5: Workflow Integration
- [ ] Phase 6: Memory & Vector Store

## License

MIT License - See LICENSE file for details

## Contributing

Contributions are welcome! Please see CONTRIBUTING.md for guidelines.
