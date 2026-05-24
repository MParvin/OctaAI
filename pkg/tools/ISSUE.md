# OctaAI agent Issues

## Issue: Agent creates empty directory but doesn't write files (FIXED)

### Symptoms
```
(base) ➜  OctaAI git:(feature/browser-automation) ✗ ./bin/octa-agentd
OctaAI Agent Daemon - Starting...
Using LLM: ollama (dolphin-mistral:7b)
Browser automation enabled on port 8765
Registered 6 tools
2025/12/25 03:14:29 Starting browser WebSocket server on localhost:8765
Agent daemon is running. Waiting for goals...
Press Ctrl+C to stop

=== Processing Goal: goal_1766619913 ===
Description: Write a docker compose that setups traefik on port 80 and 443
=== Goal goal_1766619913 completed ===
```

```
(base) ➜  traefik-docker-compose ls -lha
total 8.0K
drwxr-xr-x 2 mparvin mparvin 4.0K Dec 25 03:15 .
drwxr-xr-x 9 mparvin mparvin 4.0K Dec 25 03:15 ..
```

### Root Causes Identified

1. **No retry on parse/tool failures**: When `parseToolCall` failed or tool was not found, the task was immediately marked as "failed" without any retry attempts.

2. **allDone logic bug**: A task with status "failed" but `attempts < maxAttempts` was not being counted as pending work. This caused the main loop to exit prematurely thinking all tasks were done.

3. **Prompt clarity**: The task execution prompt wasn't clear enough for smaller LLM models like dolphin-mistral:7b, leading to malformed JSON responses.

### Fixes Applied

1. **Added retry logic for parse failures** (`agent.go` lines ~356-375):
   - When parsing fails or tool is not found, task is now set to "pending" if retries remain
   - Task is only marked "failed" after exhausting all attempts

2. **Fixed allDone detection** (`agent.go` lines ~83-105):
   - Failed tasks with remaining retries are now reset to "pending" 
   - These tasks correctly prevent premature goal completion

3. **Improved parseToolCall** (`agent.go` lines ~760-815):
   - Now accepts alternative key names: `tool_name`, `toolName`, `name` for tool
   - Accepts `arguments`, `params`, `parameters`, `tool_args` for args
   - Added better debug logging to see what LLM returns

4. **Simplified task execution prompt** (`agent.go` lines ~700-730):
   - Made prompt more concise and direct
   - Added clearer examples for common operations
   - Emphasized JSON-only response requirement