package engine

import (
	"fmt"
	"strings"

	"github.com/mparvin/octaai/pkg/storage"
	"github.com/mparvin/octaai/pkg/tools"
)

func (e *Engine) getSystemPrompt() string {
	return `You are an autonomous software engineering agent. You can:
- Create and modify files
- Run commands and programs
- Use Git operations
- SSH into servers
- Make HTTP requests
- Control a browser via the browser tool

Always respond with specific, actionable tool calls.
When executing, choose the right tool and provide complete arguments.`
}

func (e *Engine) buildTaskExecutionPrompt(goal *storage.Goal, task *storage.Task) string {
	toolsList := formatToolSchemas(e.tools)

	contextInfo := e.memory.SearchContext(goal.ID, task.Description+" "+goal.Description, 8)

	if contextInfo != "" {
		contextInfo = "\n\n" + contextInfo
	}

	return fmt.Sprintf(`Goal: %s

Task: %s
%s
Tools:%s

INSTRUCTIONS:
1. Analyze the task and choose the appropriate tool
2. For creating files, use "filesystem" with action "write_file", providing the full file content
3. You MUST respond with ONLY a JSON object - no explanations, no markdown
4. The JSON MUST have exactly two keys: "tool" and "args"

EXAMPLE - Creating a file:
{"tool": "filesystem", "args": {"action": "write_file", "path": "project/app.py", "content": "print('hello')"}}

EXAMPLE - Running a command:
{"tool": "command", "args": {"command": "ls -la", "cwd": "project"}}

Your JSON response:`, goal.Description, task.Description, contextInfo, toolsList)
}

func formatToolSchemas(registry *tools.Registry) string {
	var b strings.Builder
	for _, tool := range registry.List() {
		schema := tool.Schema()
		b.WriteString(fmt.Sprintf("\n\n%s: %s", schema.Name, schema.Description))
		b.WriteString("\nParameters:")
		for paramName, paramSchema := range schema.Parameters {
			required := ""
			if paramSchema.Required {
				required = " (required)"
			}
			b.WriteString(fmt.Sprintf("\n  - %s%s: %s", paramName, required, paramSchema.Description))
		}
	}
	return b.String()
}
