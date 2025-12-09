# OctaAI Examples

This directory contains example use cases demonstrating OctaAI's capabilities.

## Example 1: LocalWeather Project

**Goal**: Create a Python project that fetches weather data for European countries, stores it in Redis, and serves it via Flask.

```bash
octa-agent goal "Write a python script that gives weather data about all countries in Europe, store in redis, create a Flask site, name it LocalWeather"
```

**What the agent will do**:
1. Create project directory structure
2. Generate `requirements.txt` with Flask, Redis, requests
3. Create weather API client
4. Implement Redis storage layer
5. Create Flask application with routes
6. Write tests
7. Run tests and fix any errors
8. Generate README with instructions

## Example 2: Remote Server Deployment

**Goal**: Deploy a GitHub project on a remote server with Nginx and SSL.

```bash
octa-agent goal "IP: 192.168.1.100, Username: root, Password: mypass123. Setup nginx, deploy github.com/mparvin/ip9.git, configure SSL for ip9.com"
```

**What the agent will do**:
1. SSH into the server
2. Install required packages (nginx, certbot, git, runtime dependencies)
3. Clone the repository
4. Install application dependencies
5. Start the application service
6. Configure Nginx reverse proxy
7. Obtain SSL certificate with certbot
8. Test HTTPS endpoint
9. Report deployment status

## Example 3: Simple CLI Tool

**Goal**: Create a Go command-line tool.

```bash
octa-agent goal "Create a Go CLI tool called 'textproc' that counts words, lines, and characters in a file"
```

**What the agent will do**:
1. Initialize Go module
2. Create main.go with flag parsing
3. Implement file reading and counting logic
4. Add tests
5. Build the binary
6. Run tests to verify functionality

## Example 4: Data Processing Pipeline

**Goal**: Build an automated data processing pipeline.

```bash
octa-agent goal "Create a Python script that downloads CSV from URL, cleans the data, calculates statistics, and saves results to JSON"
```

**What the agent will do**:
1. Create project structure
2. Implement CSV downloader
3. Add data cleaning functions
4. Create statistical analysis module
5. Implement JSON output
6. Write unit tests
7. Create sample data for testing
8. Verify end-to-end pipeline

## Tips for Writing Goals

### Be Specific
❌ "Make a web app"
✅ "Create a Flask web app with user authentication, PostgreSQL database, and REST API for blog posts"

### Include Technology Preferences
❌ "Create a website"
✅ "Create a Node.js/Express website with React frontend and MongoDB"

### Specify Deployment Details
For server tasks, include:
- IP address or hostname
- Credentials (username/password or SSH key path)
- Domain name (for SSL/proxy)
- Port numbers (if non-standard)

### Break Down Complex Goals
For very large projects, submit multiple goals:
1. "Create backend API with authentication"
2. "Create frontend React application"
3. "Setup CI/CD pipeline"
4. "Deploy to production server"

## Monitoring Progress

Check goal status:
```bash
octa-agent status
```

View detailed logs:
```bash
octa-agent logs goal_1234567890
```

## Configuration

Before running examples, ensure your configuration is set:

```bash
# Initialize default config
octa-agent init

# Edit config
vim ~/.config/octaai/config.yaml

# Start daemon
octa-agentd
```

## Troubleshooting

### Agent Not Processing Goals
- Ensure `octa-agentd` is running
- Check logs with `octa-agent logs <goal-id>`
- Verify LLM connection (Ollama running, API keys set)

### Permission Errors
- Check `safety.allow_paths` in config
- Ensure projects_root directory exists and is writable

### LLM Errors
- For Ollama: Ensure service is running (`ollama serve`)
- For OpenAI/Claude: Verify API keys are set in environment or config

### Task Failures
- Agent will retry failed tasks up to 3 times
- Check task-specific error messages in logs
- May need to adjust the goal description for clarity
