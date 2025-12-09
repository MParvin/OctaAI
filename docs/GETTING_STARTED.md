# Getting Started with OctaAI

This guide will help you set up and start using OctaAI.

## Prerequisites

- Go 1.21 or later
- Ollama (for local LLM) or API keys for OpenAI/Claude
- Git
- Linux/macOS (Windows via WSL)

## Installation

### 1. Clone and Build

```bash
git clone https://github.com/mparvin/octaai.git
cd octaai
make build
```

This creates two binaries in `./bin/`:
- `octa-agentd` - The agent daemon
- `octa-agent` - The CLI client

### 2. Install System-Wide (Optional)

```bash
make install
```

This installs the binaries to your `$GOPATH/bin`.

## Configuration

### Initialize Configuration

```bash
./bin/octa-agent init
```

This creates `~/.config/octaai/config.yaml` with default settings.

### Configure LLM Provider

#### Option A: Ollama (Local, Recommended for Privacy)

1. Install Ollama: https://ollama.ai/download
2. Pull a model:
   ```bash
   ollama pull qwen2.5:32b
   ```
3. Your config is already set for Ollama by default!

#### Option B: OpenAI

Edit `~/.config/octaai/config.yaml`:

```yaml
llm:
  provider: "openai"
  model: "gpt-4"
  api_key: "sk-..." # Or set OPENAI_API_KEY env var
```

#### Option C: Claude

Edit `~/.config/octaai/config.yaml`:

```yaml
llm:
  provider: "claude"
  model: "claude-3-sonnet-20240229"
  api_key: "sk-ant-..." # Or set ANTHROPIC_API_KEY env var
```

### Configure Projects Directory

Edit the `projects_root` in your config to where you want projects created:

```yaml
projects_root: "/home/yourusername/Projects"
```

## Running OctaAI

### 1. Start the Agent Daemon

In one terminal:

```bash
./bin/octa-agentd
```

You should see:
```
OctaAI Agent Daemon - Starting...
Using LLM: ollama (qwen2.5:32b)
Registered 5 tools
Agent daemon is running. Waiting for goals...
```

Leave this running.

### 2. Submit a Goal

In another terminal:

```bash
./bin/octa-agent goal "Create a simple Python calculator CLI tool with add, subtract, multiply, and divide functions"
```

You should see:
```
✓ Goal created: goal_1234567890
  Description: Create a simple Python calculator CLI tool...
```

### 3. Monitor Progress

Check status:
```bash
./bin/octa-agent status
```

View detailed logs:
```bash
./bin/octa-agent logs goal_1234567890
```

### 4. View the Result

Once completed, the project will be in your `projects_root` directory:

```bash
ls ~/Projects/
```

## Your First Real Project

Let's create a weather application:

```bash
./bin/octa-agent goal "Create a Python Flask application that shows weather for major cities using OpenWeatherMap API. Include a simple HTML frontend. Name it WeatherDash"
```

The agent will:
1. Create project structure
2. Generate Flask application code
3. Create HTML templates
4. Write a README with setup instructions
5. Create tests
6. Verify everything works

When complete, navigate to the project:

```bash
cd ~/Projects/WeatherDash
cat README.md
```

Follow the instructions in the generated README to run your new application!

## Advanced Usage

### Remote Server Deployment

Deploy an application to a remote server:

```bash
./bin/octa-agent goal "IP: 192.168.1.100, Username: deploy, Password: secret123. Install nginx, deploy my-app from ~/Projects/my-app, configure reverse proxy on port 80"
```

### Multiple Goals

Submit multiple goals - they'll be processed in order:

```bash
./bin/octa-agent goal "Create backend API with FastAPI and PostgreSQL"
./bin/octa-agent goal "Create React frontend that consumes the API"
./bin/octa-agent goal "Write integration tests"
```

## Troubleshooting

### Daemon Won't Start

- **Check Ollama**: `curl http://localhost:11434/api/version`
- **Check config**: `cat ~/.config/octaai/config.yaml`
- **Check storage dir**: Ensure `~/.config/octaai/` is writable

### Goals Stay in "IDLE" State

- Ensure `octa-agentd` is running
- Check daemon output for errors
- Verify LLM connection

### Permission Errors

- Check `safety.allow_paths` in config
- Ensure your projects_root exists: `mkdir -p ~/Projects`

### LLM Not Responding

**For Ollama:**
```bash
# Check if Ollama is running
ollama list

# Start Ollama if needed
ollama serve
```

**For OpenAI/Claude:**
- Verify API key is correct
- Check internet connection
- Check API quota/billing

## Next Steps

- Read the [Examples](examples/README.md) for more use cases
- See [DESIGN.md](DESIGN.md) for architecture details
- See [PROMPT.md](PROMPT.md) for the agent's system prompt
- Check the [Makefile](Makefile) for development commands

## Getting Help

- Check logs: `./bin/octa-agent logs <goal-id>`
- Review configuration: `cat ~/.config/octaai/config.yaml`
- Submit issues: https://github.com/mparvin/octaai/issues

## Safety Notes

- The agent runs with your user permissions
- `safety.deny_commands` prevents dangerous operations
- `safety.allow_paths` restricts file operations
- Review generated code before deploying to production
- Use SSH with caution - credentials are stored in goals

Enjoy autonomous software development! 🚀
