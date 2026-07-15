package engine

import (
	"fmt"
	"strings"

	"github.com/mparvin/octaai/pkg/storage"
	"github.com/mparvin/octaai/pkg/tools"
)

func (e *Engine) getSystemPrompt() string {
	return fmt.Sprintf(`You are an autonomous software engineering agent. You can:
- Create and modify files
- Run commands and programs
- Use Git operations
- SSH into servers
- Make HTTP requests
- Control a browser via the browser tool

All file paths and command working directories are relative to the projects root: %s
Do not use ~, $HOME, or absolute paths outside the projects root.
Create a subdirectory for each new project (e.g. "my-project/script.py", cwd "my-project").

When the goal asks to develop, create, or write a program/script/application, use the filesystem
tool to write complete source code files. Do NOT use the http tool to call external APIs instead
of writing code — implement the logic in source files the user can run locally.

Always respond with specific, actionable tool calls.
When executing, choose the right tool and provide complete arguments.`, e.cfg.ProjectsRoot)
}

func projectContext(goal *storage.Goal) string {
	if goal.ProjectName == "" {
		return ""
	}
	return fmt.Sprintf("\nProject directory: %s (create all files under this subdirectory)\n", goal.ProjectName)
}

func (e *Engine) buildTaskExecutionPrompt(goal *storage.Goal, task *storage.Task) string {
	toolsList := formatToolSchemas(e.tools)

	contextInfo := e.memory.SearchContext(goal.ID, task.Description+" "+goal.Description, 8)

	if contextInfo != "" {
		contextInfo = "\n\n" + contextInfo
	}

	return fmt.Sprintf(`Goal: %s

Task: %s
Projects root: %s%s
%s
Tools:%s

INSTRUCTIONS:
1. Analyze the task and choose the appropriate tool
2. For creating/writing programs or scripts, use "filesystem" with action "write_file" and provide complete source code
3. Do NOT use the "http" tool when the goal is to develop code — write the source files instead
4. All paths are relative to the projects root above (e.g. "my-project/main.py", not "~/" or absolute paths)
5. You MUST respond with ONLY a JSON object - no explanations, no markdown
6. The JSON MUST have exactly two keys: "tool" and "args"

EXAMPLE - Creating a Python script:
{"tool": "filesystem", "args": {"action": "write_file", "path": "github-list/main.py", "content": "#!/usr/bin/env python3\\nimport sys\\n..."}}

EXAMPLE - Running a command:
{"tool": "command", "args": {"command": "python main.py octocat", "cwd": "github-list"}}

Your JSON response:`, goal.Description, task.Description, e.cfg.ProjectsRoot, projectContext(goal), contextInfo, toolsList)
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
