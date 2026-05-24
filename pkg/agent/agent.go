package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mparvin/octaai/pkg/config"
	"github.com/mparvin/octaai/pkg/llm"
	"github.com/mparvin/octaai/pkg/storage"
	"github.com/mparvin/octaai/pkg/tools"
)

// Agent is the core autonomous agent
type Agent struct {
	cfg      *config.Config
	llm      llm.Provider
	tools    *tools.Registry
	storage  storage.Storage
	goalID   string
	maxLoops int
}

// NewAgent creates a new agent instance
func NewAgent(cfg *config.Config, llmProvider llm.Provider, toolRegistry *tools.Registry, store storage.Storage) *Agent {
	return &Agent{
		cfg:      cfg,
		llm:      llmProvider,
		tools:    toolRegistry,
		storage:  store,
		maxLoops: 50, // Safety limit
	}
}

// ProcessGoal processes a goal from start to completion
func (a *Agent) ProcessGoal(ctx context.Context, goalID string) error {
	a.goalID = goalID

	// Get goal
	goal, err := a.storage.GetGoal(goalID)
	if err != nil {
		return fmt.Errorf("failed to get goal: %w", err)
	}

	a.log(ctx, "info", fmt.Sprintf("Processing goal: %s", goal.Description))

	// Update state to planning
	goal.State = storage.StatePlanning
	goal.UpdatedAt = time.Now()
	if err := a.storage.UpdateGoal(goal); err != nil {
		return err
	}

	// Plan tasks
	if err := a.planTasks(ctx, goal); err != nil {
		return a.failGoal(goal, err)
	}

	// Update state to executing
	goal.State = storage.StateExecuting
	goal.UpdatedAt = time.Now()
	if err := a.storage.UpdateGoal(goal); err != nil {
		return err
	}

	// Execute tasks in a loop
	loopCount := 0
	lastTaskCount := 0
	noProgressCount := 0
	for loopCount < a.maxLoops {
		loopCount++

		// Check if all tasks are done
		tasks, err := a.storage.GetTasksByGoal(goalID)
		if err != nil {
			return err
		}

		a.log(ctx, "info", fmt.Sprintf("Loop %d: %d tasks total", loopCount, len(tasks)))

		allDone := true
		hasFailures := false
		completedCount := 0
		for _, task := range tasks {
			if task.Status == "completed" {
				completedCount++
			}
			if task.Status == "pending" || task.Status == "running" {
				allDone = false
			}
			if task.Status == "failed" {
				if task.Attempts >= task.MaxAttempts {
					hasFailures = true
				} else {
					// Task failed but has retries remaining - reset to pending
					a.log(ctx, "info", fmt.Sprintf("Retrying failed task: %s (attempt %d/%d)", task.Description, task.Attempts+1, task.MaxAttempts))
					task.Status = "pending"
					task.UpdatedAt = time.Now()
					a.storage.UpdateTask(&task)
					allDone = false
				}
			}
		}

		a.log(ctx, "info", fmt.Sprintf("Progress: %d/%d tasks completed", completedCount, len(tasks)))

		// Check for no progress
		if len(tasks) == lastTaskCount && completedCount == 0 && loopCount > 5 {
			noProgressCount++
			if noProgressCount > 3 {
				return a.failGoal(goal, fmt.Errorf("no progress after %d iterations", loopCount))
			}
		} else {
			noProgressCount = 0
		}
		lastTaskCount = len(tasks)

		if hasFailures {
			return a.failGoal(goal, fmt.Errorf("tasks failed after max attempts"))
		}

		if allDone {
			break
		}

		// Execute next task
		if err := a.executeNextTask(ctx, goal); err != nil {
			a.log(ctx, "error", fmt.Sprintf("Task execution error: %v", err))
			// Continue trying other tasks
		}

		// Brief pause between iterations
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}

	if loopCount >= a.maxLoops {
		return a.failGoal(goal, fmt.Errorf("exceeded maximum loop count"))
	}

	// Verify completion
	if err := a.verifyGoalCompletion(ctx, goal); err != nil {
		return a.failGoal(goal, err)
	}

	// Mark goal as completed
	now := time.Now()
	goal.State = storage.StateCompleted
	goal.UpdatedAt = now
	goal.CompletedAt = &now
	goal.Result = "Goal completed successfully"

	return a.storage.UpdateGoal(goal)
}

// planTasks asks LLM to create initial task plan
func (a *Agent) planTasks(ctx context.Context, goal *storage.Goal) error {
	a.log(ctx, "info", "Planning tasks...")

	// First, ask LLM to extract project name from goal
	projectName, needsProject := a.extractProjectInfo(ctx, goal)

	now := time.Now()
	tasks := []*storage.Task{}

	// If this needs a project directory, create that task first
	if needsProject && projectName != "" {
		tasks = append(tasks, &storage.Task{
			ID:           fmt.Sprintf("task_%s_1", goal.ID),
			GoalID:       goal.ID,
			Description:  fmt.Sprintf("Create project directory: %s", projectName),
			Status:       "pending",
			ToolName:     "filesystem",
			ToolArgs:     map[string]interface{}{"action": "create_directory", "path": projectName},
			CreatedAt:    now,
			UpdatedAt:    now,
			MaxAttempts:  3,
			Dependencies: []string{},
		})

		// Main task depends on directory creation
		tasks = append(tasks, &storage.Task{
			ID:           fmt.Sprintf("task_%s_2", goal.ID),
			GoalID:       goal.ID,
			Description:  fmt.Sprintf("Develop project '%s': %s", projectName, goal.Description),
			Status:       "pending",
			CreatedAt:    now,
			UpdatedAt:    now,
			MaxAttempts:  3,
			Dependencies: []string{fmt.Sprintf("task_%s_1", goal.ID)},
		})
	} else {
		// Simple task without project structure
		tasks = append(tasks, &storage.Task{
			ID:           fmt.Sprintf("task_%s_1", goal.ID),
			GoalID:       goal.ID,
			Description:  fmt.Sprintf("Complete goal: %s", goal.Description),
			Status:       "pending",
			CreatedAt:    now,
			UpdatedAt:    now,
			MaxAttempts:  3,
			Dependencies: []string{},
		})
	}

	// Create all tasks
	for _, task := range tasks {
		if err := a.storage.CreateTask(task); err != nil {
			return err
		}
	}

	a.log(ctx, "info", fmt.Sprintf("Created %d initial task(s)", len(tasks)))
	return nil
}

// extractProjectInfo determines if goal needs a project directory and what to name it
func (a *Agent) extractProjectInfo(ctx context.Context, goal *storage.Goal) (projectName string, needsProject bool) {
	prompt := fmt.Sprintf(`Analyze this development goal:
"%s"

Does this goal require creating a new project/application (not just a single file)?
If YES, suggest a short project directory name (lowercase, no spaces, alphanumeric with hyphens/underscores only).
If NO (just a simple file creation), respond with "NONE".

Respond in this format:
PROJECT_NAME: <name or NONE>

Examples:
- "Create a Python Flask REST API for a todo list" -> PROJECT_NAME: todo-api
- "Write a weather app using Flask" -> PROJECT_NAME: weather-app
- "Create file test.txt with content Hello" -> PROJECT_NAME: NONE
- "Build a Redis client in Go" -> PROJECT_NAME: redis-client`, goal.Description)

	resp, err := a.llm.Complete(ctx, prompt, &llm.Options{
		Temperature: 0.1,
		MaxTokens:   100,
	})
	if err != nil {
		a.log(ctx, "warn", fmt.Sprintf("Failed to extract project name: %v", err))
		return "", false
	}

	// Parse response
	content := strings.TrimSpace(resp.Content)
	if strings.Contains(strings.ToUpper(content), "PROJECT_NAME:") {
		parts := strings.Split(content, ":")
		if len(parts) >= 2 {
			name := strings.TrimSpace(parts[1])
			if name != "NONE" && name != "" {
				// Clean the name
				name = strings.ToLower(name)
				name = strings.ReplaceAll(name, " ", "-")
				return name, true
			}
		}
	}

	return "", false
}

// executeNextTask finds and executes the next pending task
func (a *Agent) executeNextTask(ctx context.Context, goal *storage.Goal) error {
	tasks, err := a.storage.GetTasksByGoal(goal.ID)
	if err != nil {
		return err
	}

	// Find next executable task (pending, dependencies met)
	var nextTask *storage.Task
	for i := range tasks {
		task := &tasks[i]
		if task.Status != "pending" {
			continue
		}

		// Check dependencies
		depsReady := true
		for _, depID := range task.Dependencies {
			depTask, err := a.storage.GetTask(depID)
			if err != nil || depTask.Status != "completed" {
				depsReady = false
				break
			}
		}

		if depsReady {
			nextTask = task
			break
		}
	}

	if nextTask == nil {
		return nil // No task ready to execute
	}

	// Execute the task
	a.log(ctx, "info", fmt.Sprintf("Executing task: %s", nextTask.Description))

	nextTask.Status = "running"
	nextTask.UpdatedAt = time.Now()
	nextTask.Attempts++
	if err := a.storage.UpdateTask(nextTask); err != nil {
		return err
	}

	// If tool is specified, execute it directly
	if nextTask.ToolName != "" {
		return a.executeToolTask(ctx, nextTask)
	}

	// Otherwise, ask LLM what to do
	return a.executeLLMTask(ctx, goal, nextTask)
}

// executeToolTask executes a task with a specific tool
func (a *Agent) executeToolTask(ctx context.Context, task *storage.Task) error {
	tool, ok := a.tools.Get(task.ToolName)
	if !ok {
		return a.failTask(task, fmt.Errorf("tool not found: %s", task.ToolName))
	}

	result, err := tool.Execute(ctx, task.ToolArgs)
	if err != nil {
		return a.failTask(task, err)
	}

	if !result.Success {
		return a.failTask(task, fmt.Errorf("tool execution failed: %s", result.Error))
	}

	// Mark task as completed
	task.Status = "completed"
	task.Result = result.Output
	task.UpdatedAt = time.Now()

	return a.storage.UpdateTask(task)
}

// executeLLMTask asks LLM to determine what action to take
func (a *Agent) executeLLMTask(ctx context.Context, goal *storage.Goal, task *storage.Task) error {
	// Build prompt for this task
	prompt := a.buildTaskExecutionPrompt(goal, task)

	a.log(ctx, "debug", fmt.Sprintf("Asking LLM for action (attempt %d/%d)", task.Attempts, task.MaxAttempts))

	resp, err := a.llm.Complete(ctx, prompt, &llm.Options{
		Temperature:  0.3,
		MaxTokens:    2048,
		SystemPrompt: a.getSystemPrompt(),
	})
	if err != nil {
		a.log(ctx, "error", fmt.Sprintf("LLM call failed: %v", err))
		return a.failTask(task, fmt.Errorf("LLM execution failed: %w", err))
	}

	a.log(ctx, "debug", fmt.Sprintf("LLM response length: %d chars", len(resp.Content)))

	// Parse tool call from response
	toolCall, err := a.parseToolCall(resp.Content)
	if err != nil {
		a.log(ctx, "error", fmt.Sprintf("Failed to parse tool call: %v. Response: %s", err, resp.Content[:min(200, len(resp.Content))]))
		// If we can retry, set to pending instead of failed
		if task.Attempts < task.MaxAttempts {
			task.Status = "pending"
			task.Error = fmt.Sprintf("Parse error: %v", err)
			task.UpdatedAt = time.Now()
			return a.storage.UpdateTask(task)
		}
		return a.failTask(task, err)
	}

	// Execute the tool
	tool, ok := a.tools.Get(toolCall.ToolName)
	if !ok {
		// If tool not found, retry with pending status if attempts remain
		if task.Attempts < task.MaxAttempts {
			a.log(ctx, "warn", fmt.Sprintf("Tool not found: %s, will retry", toolCall.ToolName))
			task.Status = "pending"
			task.Error = fmt.Sprintf("Tool not found: %s", toolCall.ToolName)
			task.UpdatedAt = time.Now()
			return a.storage.UpdateTask(task)
		}
		return a.failTask(task, fmt.Errorf("tool not found: %s", toolCall.ToolName))
	}

	result, err := tool.Execute(ctx, toolCall.Args)
	if err != nil {
		return a.failTask(task, err)
	}

	if !result.Success {
		// If task failed but can retry, keep it pending
		if task.Attempts < task.MaxAttempts {
			task.Status = "pending"
			task.Error = result.Error
			task.UpdatedAt = time.Now()
			return a.storage.UpdateTask(task)
		}
		return a.failTask(task, fmt.Errorf("tool execution failed: %s", result.Error))
	}

	// Store the result
	task.Result = result.Output
	task.UpdatedAt = time.Now()

	// Check if this is a "develop project" task - if so, ask LLM if more work is needed
	if strings.Contains(task.Description, "Develop project") {
		// Limit the number of follow-up tasks to prevent infinite loops
		tasks, _ := a.storage.GetTasksByGoal(goal.ID)
		developTaskCount := 0
		for _, t := range tasks {
			if strings.Contains(t.Description, "Develop project") {
				developTaskCount++
			}
		}

		// Maximum 10 development iterations
		if developTaskCount < 10 {
			needsMore, nextAction := a.checkIfMoreWorkNeeded(ctx, goal, task, result.Output)
			if needsMore {
				// Create a new sub-task for the next action
				now := time.Now()
				newTask := &storage.Task{
					ID:           fmt.Sprintf("task_%s_%d", goal.ID, time.Now().Unix()),
					GoalID:       goal.ID,
					Description:  nextAction,
					Status:       "pending",
					CreatedAt:    now,
					UpdatedAt:    now,
					MaxAttempts:  3,
					Dependencies: []string{task.ID},
				}
				if err := a.storage.CreateTask(newTask); err != nil {
					a.log(ctx, "warn", fmt.Sprintf("Failed to create follow-up task: %v", err))
				} else {
					a.log(ctx, "info", fmt.Sprintf("Created follow-up task: %s", nextAction))
				}
			} else {
				a.log(ctx, "info", "LLM indicates project development is complete")
			}
		} else {
			a.log(ctx, "warn", "Reached maximum development task limit (10)")
		}
	}

	// Mark task as completed
	task.Status = "completed"

	return a.storage.UpdateTask(task)
}

// checkIfMoreWorkNeeded determines if a development task needs more actions
func (a *Agent) checkIfMoreWorkNeeded(ctx context.Context, goal *storage.Goal, task *storage.Task, lastResult string) (bool, string) {
	// Get list of files in project directory
	projectFiles := a.getProjectFiles(goal.ID)

	prompt := fmt.Sprintf(`You are developing a project to achieve this goal:
Goal: %s

Current state:
- Last action: Created/modified a file
- Files in project: %s

For a basic Flask REST API, the MINIMUM required files are:
1. A main Python file (app.py, main.py, or similar) with Flask routes
2. A requirements.txt file with Flask dependency

Question: Do you need to create MORE files to complete this basic project?

Respond with DONE if:
- At least one .py file exists AND
- requirements.txt exists

Respond with:
CONTINUE: <specific filename to create next>

Or respond with:
DONE

Be SPECIFIC about which file to create next. Do NOT repeat files that already exist.`, goal.Description, projectFiles)

	resp, err := a.llm.Complete(ctx, prompt, &llm.Options{
		Temperature: 0.2,
		MaxTokens:   200,
	})
	if err != nil {
		a.log(ctx, "warn", fmt.Sprintf("Failed to check if more work needed: %v", err))
		return false, ""
	}

	content := strings.TrimSpace(resp.Content)
	if strings.HasPrefix(strings.ToUpper(content), "DONE") {
		return false, ""
	}

	if strings.Contains(strings.ToUpper(content), "CONTINUE:") {
		parts := strings.SplitN(content, ":", 2)
		if len(parts) == 2 {
			nextAction := strings.TrimSpace(parts[1])
			return true, fmt.Sprintf("Develop project '%s': %s", a.getProjectName(goal.ID), nextAction)
		}
	}

	return false, ""
}

// getProjectFiles lists files in the project directory
func (a *Agent) getProjectFiles(goalID string) string {
	projectName := a.getProjectName(goalID)
	if projectName == "" {
		return "No project directory"
	}

	tool, ok := a.tools.Get("filesystem")
	if !ok {
		return "Unknown"
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"action": "list_files",
		"path":   projectName,
	})
	if err != nil || !result.Success {
		return "No files yet"
	}

	return result.Output
}

// getProjectName extracts project name from completed tasks
func (a *Agent) getProjectName(goalID string) string {
	tasks, err := a.storage.GetTasksByGoal(goalID)
	if err != nil {
		return ""
	}

	for _, task := range tasks {
		if task.Status == "completed" && task.ToolName == "filesystem" {
			if action, ok := task.ToolArgs["action"].(string); ok && action == "create_directory" {
				if path, ok := task.ToolArgs["path"].(string); ok {
					return path
				}
			}
		}
	}
	return ""
}

func (a *Agent) verifyGoalCompletion(ctx context.Context, goal *storage.Goal) error {
	a.log(ctx, "info", "Verifying goal completion...")

	// Get all tasks
	tasks, err := a.storage.GetTasksByGoal(goal.ID)
	if err != nil {
		return err
	}

	// Build verification prompt
	var taskSummary strings.Builder
	for _, task := range tasks {
		taskSummary.WriteString(fmt.Sprintf("- %s: %s\n", task.Status, task.Description))
	}

	prompt := fmt.Sprintf(`Goal: %s

Tasks completed:
%s

Has the goal been fully satisfied? Answer ONLY with YES or NO (first word).`, goal.Description, taskSummary.String())

	resp, err := a.llm.Complete(ctx, prompt, &llm.Options{
		Temperature: 0.1,
		MaxTokens:   500,
	})
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	// Trim whitespace and convert to uppercase for checking
	response := strings.TrimSpace(strings.ToUpper(resp.Content))

	// Check if response starts with YES or contains YES as first word
	if strings.HasPrefix(response, "YES") || strings.HasPrefix(response, "YES,") || strings.HasPrefix(response, "YES.") {
		return nil
	}

	// Also accept if first word is YES
	firstWord := strings.Fields(response)
	if len(firstWord) > 0 && firstWord[0] == "YES" {
		return nil
	}

	return fmt.Errorf("goal not fully satisfied: %s", resp.Content)
}

func (a *Agent) failGoal(goal *storage.Goal, err error) error {
	now := time.Now()
	goal.State = storage.StateFailed
	goal.UpdatedAt = now
	goal.CompletedAt = &now
	goal.Error = err.Error()

	a.storage.UpdateGoal(goal)
	return err
}

func (a *Agent) failTask(task *storage.Task, err error) error {
	task.Status = "failed"
	task.Error = err.Error()
	task.UpdatedAt = time.Now()

	return a.storage.UpdateTask(task)
}

func (a *Agent) log(ctx context.Context, level, message string) {
	log := &storage.ExecutionLog{
		ID:        fmt.Sprintf("log_%d", time.Now().UnixNano()),
		GoalID:    a.goalID,
		Level:     level,
		Message:   message,
		CreatedAt: time.Now(),
	}
	a.storage.CreateLog(log)
}

// Helper methods for prompt building
func (a *Agent) getSystemPrompt() string {
	return `You are an autonomous software engineering agent. You can:
- Create and modify files
- Run commands and programs
- Use Git operations
- SSH into servers
- Make HTTP requests

Always respond with specific, actionable tool calls.
When planning tasks, break them into small, independent steps.
When executing, choose the right tool and provide complete arguments.`
}

func (a *Agent) buildPlanningPrompt(goal *storage.Goal) string {
	toolsList := ""
	for _, tool := range a.tools.List() {
		schema := tool.Schema()
		toolsList += fmt.Sprintf("- %s: %s\n", schema.Name, schema.Description)
	}

	return fmt.Sprintf(`You are planning how to achieve this goal:

Goal: %s

Available tools:
%s

Break this goal into 5-10 specific, actionable tasks. Each task should be independent where possible.
Respond in JSON format as an array of tasks:
[
  {
    "id": "task_1",
    "description": "Create project directory",
    "tool_name": "filesystem",
    "tool_args": {"action": "create_directory", "path": "ProjectName"},
    "dependencies": []
  },
  ...
]`, goal.Description, toolsList)
}

func (a *Agent) buildTaskExecutionPrompt(goal *storage.Goal, task *storage.Task) string {
	toolsList := ""
	for _, tool := range a.tools.List() {
		schema := tool.Schema()
		toolsList += fmt.Sprintf("\n\n%s: %s", schema.Name, schema.Description)
		toolsList += "\nParameters:"
		for paramName, paramSchema := range schema.Parameters {
			required := ""
			if paramSchema.Required {
				required = " (required)"
			}
			toolsList += fmt.Sprintf("\n  - %s%s: %s", paramName, required, paramSchema.Description)
			if len(paramSchema.Enum) > 0 {
				toolsList += fmt.Sprintf("\n    Allowed values: %v", paramSchema.Enum)
			}
		}
	}

	// Get project context from completed tasks
	projectContext := a.getProjectContext(goal.ID)

	contextInfo := ""
	if projectContext != "" {
		contextInfo = fmt.Sprintf("\n\nProject context:\n%s\n", projectContext)
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

EXAMPLE - Creating a docker-compose file:
{"tool": "filesystem", "args": {"action": "write_file", "path": "project-name/docker-compose.yml", "content": "version: '3'\nservices:\n  web:\n    image: nginx\n    ports:\n      - '80:80'"}}

EXAMPLE - Running a command:
{"tool": "command", "args": {"command": "ls -la", "cwd": "project-name"}}

Your JSON response:`, goal.Description, task.Description, contextInfo, toolsList)
}

// getProjectContext retrieves context about the project from completed tasks
func (a *Agent) getProjectContext(goalID string) string {
	tasks, err := a.storage.GetTasksByGoal(goalID)
	if err != nil {
		return ""
	}

	var context strings.Builder
	for _, task := range tasks {
		if task.Status == "completed" && task.ToolName == "filesystem" {
			if action, ok := task.ToolArgs["action"].(string); ok {
				if action == "create_directory" {
					if path, ok := task.ToolArgs["path"].(string); ok {
						context.WriteString(fmt.Sprintf("- Project directory: %s\n", path))
					}
				}
			}
		}
	}

	return context.String()
}

type toolCall struct {
	ToolName string
	Args     map[string]interface{}
}

func (a *Agent) parseToolCall(content string) (*toolCall, error) {
	// Try to extract JSON from response
	// Handle cases where LLM adds markdown code blocks
	content = strings.ReplaceAll(content, "```json", "")
	content = strings.ReplaceAll(content, "```", "")
	content = strings.TrimSpace(content)

	// Log the raw content for debugging
	a.log(context.Background(), "debug", fmt.Sprintf("Parsing tool call from: %s", content[:min(500, len(content))]))

	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")

	if start == -1 || end == -1 {
		return nil, fmt.Errorf("no JSON object found in response (length: %d): %s", len(content), content[:min(100, len(content))])
	}

	jsonStr := content[start : end+1]

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w. JSON: %s", err, jsonStr[:min(200, len(jsonStr))])
	}

	// Try multiple possible key names for the tool
	toolName := ""
	for _, key := range []string{"tool", "tool_name", "toolName", "name"} {
		if name, ok := parsed[key].(string); ok && name != "" {
			toolName = name
			break
		}
	}
	if toolName == "" {
		return nil, fmt.Errorf("tool name not found in response. Keys found: %v", getMapKeys(parsed))
	}

	// Try multiple possible key names for the arguments
	var args map[string]interface{}
	for _, key := range []string{"args", "arguments", "params", "parameters", "tool_args"} {
		if a, ok := parsed[key].(map[string]interface{}); ok {
			args = a
			break
		}
	}
	if args == nil {
		return nil, fmt.Errorf("args not found or not an object in response. Keys found: %v", getMapKeys(parsed))
	}

	a.log(context.Background(), "debug", fmt.Sprintf("Parsed tool call: %s with args: %v", toolName, args))

	return &toolCall{
		ToolName: toolName,
		Args:     args,
	}, nil
}

// getMapKeys returns the keys of a map for debugging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (a *Agent) parseTasksFromLLM(content string, goalID string) ([]*storage.Task, error) {
	// Extract JSON array
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")

	if start == -1 || end == -1 {
		return nil, fmt.Errorf("no JSON array found in response")
	}

	jsonStr := content[start : end+1]

	var taskData []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &taskData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	tasks := make([]*storage.Task, 0, len(taskData))
	now := time.Now()

	for i, data := range taskData {
		task := &storage.Task{
			ID:          fmt.Sprintf("task_%s_%d", goalID, i+1),
			GoalID:      goalID,
			Description: data["description"].(string),
			Status:      "pending",
			CreatedAt:   now,
			UpdatedAt:   now,
			MaxAttempts: 3,
		}

		if toolName, ok := data["tool_name"].(string); ok {
			task.ToolName = toolName
		}

		if toolArgs, ok := data["tool_args"].(map[string]interface{}); ok {
			task.ToolArgs = toolArgs
		}

		if deps, ok := data["dependencies"].([]interface{}); ok {
			task.Dependencies = make([]string, len(deps))
			for j, dep := range deps {
				task.Dependencies[j] = dep.(string)
			}
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}
