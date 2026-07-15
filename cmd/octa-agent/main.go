package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/mparvin/octaai/pkg/approval"
	"github.com/mparvin/octaai/pkg/config"
	"github.com/mparvin/octaai/pkg/planner"
	"github.com/mparvin/octaai/pkg/storage"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	cfg, err := config.LoadConfig(config.ConfigPath())
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	store, err := storage.NewSQLiteStorage(cfg.Storage.Path)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	switch command {
	case "goal":
		handleGoal(store, os.Args[2:])
	case "status":
		handleStatus(store)
	case "list":
		handleList(store)
	case "logs":
		handleLogs(store, os.Args[2:])
	case "approvals":
		handleApprovals(store)
	case "approve":
		handleApprove(store, os.Args[2:])
	case "deny":
		handleDeny(store, os.Args[2:])
	case "init":
		handleInit(cfg)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("OctaAI Agent CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  octa-agent goal [flags] <description>   Submit a new goal")
	fmt.Println("  octa-agent list                      List all goal IDs")
	fmt.Println("  octa-agent status                    Show goals status")
	fmt.Println("  octa-agent logs <goal-id>            Show logs for a goal")
	fmt.Println("  octa-agent approvals                 List pending approvals")
	fmt.Println("  octa-agent approve <approval-id>     Approve a pending action")
	fmt.Println("  octa-agent deny <approval-id>        Deny a pending action")
	fmt.Println("  octa-agent init                      Initialize configuration")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  octa-agent goal --project github-list \"Create a Python CLI for GitHub repos\"")
	fmt.Println("  octa-agent approvals")
	fmt.Println("  octa-agent approve approval_1234567890")
}

func handleGoal(store storage.Storage, args []string) {
	fs := flag.NewFlagSet("goal", flag.ExitOnError)
	projectName := fs.String("project", "", "Project directory name under projects_root")
	fs.StringVar(projectName, "p", "", "Project directory name (shorthand)")
	if err := fs.Parse(args); err != nil {
		log.Fatalf("Failed to parse goal flags: %v", err)
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		fmt.Println("Error: Goal description is required")
		fmt.Println("Usage: octa-agent goal [--project NAME] <description>")
		os.Exit(1)
	}

	description := strings.Join(remaining, " ")
	sanitizedProject := planner.SanitizeProjectName(*projectName)
	if *projectName != "" && sanitizedProject == "" {
		fmt.Println("Error: Invalid project name")
		os.Exit(1)
	}

	goal := &storage.Goal{
		ID:          fmt.Sprintf("goal_%d", time.Now().Unix()),
		Description: description,
		ProjectName: sanitizedProject,
		State:       storage.StateIdle,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := store.CreateGoal(goal); err != nil {
		log.Fatalf("Failed to create goal: %v", err)
	}

	fmt.Printf("✓ Goal created: %s\n", goal.ID)
	fmt.Printf("  Description: %s\n", goal.Description)
	if goal.ProjectName != "" {
		fmt.Printf("  Project:     %s\n", goal.ProjectName)
	}
	fmt.Println()
	fmt.Println("The goal will be processed by the agent daemon.")
	fmt.Println("Run 'octa-agent status' to check progress.")
}

func handleList(store storage.Storage) {
	goals, err := store.ListGoals()
	if err != nil {
		log.Fatalf("Failed to list goals: %v", err)
	}

	if len(goals) == 0 {
		fmt.Println("No goals found.")
		return
	}

	for _, goal := range goals {
		fmt.Println(goal.ID)
	}
}

func handleStatus(store storage.Storage) {
	goals, err := store.ListGoals()
	if err != nil {
		log.Fatalf("Failed to list goals: %v", err)
	}

	if len(goals) == 0 {
		fmt.Println("No goals found.")
		fmt.Println("Create a goal with: octa-agent goal \"<description>\"")
		return
	}

	fmt.Println("Goals Status:")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATE\tDESCRIPTION\tCREATED")
	fmt.Fprintln(w, "──\t─────\t───────────\t───────")

	for _, goal := range goals {
		desc := goal.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			goal.ID,
			goal.State,
			desc,
			goal.CreatedAt.Format("2006-01-02 15:04"),
		)
	}

	w.Flush()
	for _, goal := range goals {
		if goal.ProjectName != "" {
			fmt.Printf("   %s → project: %s\n", goal.ID, goal.ProjectName)
		}
	}
	fmt.Println()

	for _, goal := range goals {
		if goal.State == storage.StatePlanning || goal.State == storage.StateExecuting ||
			goal.State == storage.StateEvaluating || goal.State == storage.StateRetrying {
			fmt.Printf("\n📍 Active: %s\n", goal.ID)
			fmt.Printf("   State: %s\n", goal.State)

			tasks, err := store.GetTasksByGoal(goal.ID)
			if err == nil && len(tasks) > 0 {
				completed := 0
				for _, task := range tasks {
					if task.Status == "completed" {
						completed++
					}
				}
				fmt.Printf("   Progress: %d/%d tasks completed\n", completed, len(tasks))
			}
		}
		if goal.State == storage.StateWaitingForApproval {
			fmt.Printf("\n⚠ Awaiting approval: %s\n", goal.ID)
			fmt.Println("   Run 'octa-agent approvals' to review pending actions.")
		}
	}

	for _, goal := range goals {
		if goal.State == storage.StateCompleted {
			fmt.Printf("\n✓ Completed: %s\n", goal.ID)
			fmt.Printf("   %s\n", goal.Result)
		} else if goal.State == storage.StateFailed {
			fmt.Printf("\n✗ Failed: %s\n", goal.ID)
			fmt.Printf("   Error: %s\n", goal.Error)
		}
	}
}

func handleApprovals(store storage.Storage) {
	svc := approval.NewService(store)
	reqs, err := svc.ListPending()
	if err != nil {
		log.Fatalf("Failed to list approvals: %v", err)
	}

	if len(reqs) == 0 {
		fmt.Println("No pending approvals.")
		return
	}

	fmt.Println("Pending Approvals:")
	fmt.Println()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tGOAL\tTOOL\tREASON\tCREATED")
	fmt.Fprintln(w, "──\t────\t────\t──────\t───────")

	for _, req := range reqs {
		reason := req.Reason
		if len(reason) > 40 {
			reason = reason[:37] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			req.ID, req.GoalID, req.ToolName, reason,
			req.CreatedAt.Format("2006-01-02 15:04"),
		)
	}
	w.Flush()
	fmt.Println()
	fmt.Println("Use: octa-agent approve <id>  or  octa-agent deny <id>")
}

func handleApprove(store storage.Storage, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: octa-agent approve <approval-id>")
		os.Exit(1)
	}
	svc := approval.NewService(store)
	req, err := svc.Approve(args[0])
	if err != nil {
		log.Fatalf("Failed to approve: %v", err)
	}
	fmt.Printf("✓ Approved: %s\n", req.ID)
	fmt.Printf("  Goal %s will resume on next daemon cycle.\n", req.GoalID)
}

func handleDeny(store storage.Storage, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: octa-agent deny <approval-id>")
		os.Exit(1)
	}
	svc := approval.NewService(store)
	req, err := svc.Deny(args[0])
	if err != nil {
		log.Fatalf("Failed to deny: %v", err)
	}
	fmt.Printf("✗ Denied: %s\n", req.ID)
	fmt.Printf("  Goal %s marked as failed.\n", req.GoalID)
}

func handleLogs(store storage.Storage, args []string) {
	if len(args) == 0 {
		fmt.Println("Error: Goal ID is required")
		fmt.Println("Usage: octa-agent logs <goal-id>")
		os.Exit(1)
	}

	goalID := args[0]

	goal, err := store.GetGoal(goalID)
	if err != nil {
		log.Fatalf("Failed to get goal: %v", err)
	}

	fmt.Printf("Logs for Goal: %s\n", goal.ID)
	fmt.Printf("Description: %s\n", goal.Description)
	fmt.Printf("State: %s\n", goal.State)
	fmt.Println()

	logs, err := store.GetLogsByGoal(goalID)
	if err != nil {
		log.Fatalf("Failed to get logs: %v", err)
	}

	if len(logs) == 0 {
		fmt.Println("No logs available for this goal.")
		return
	}

	for _, l := range logs {
		timestamp := l.CreatedAt.Format("15:04:05")
		fmt.Printf("[%s] %s: %s\n", timestamp, l.Level, l.Message)
		if l.Data != "" {
			fmt.Printf("          Data: %s\n", l.Data)
		}
	}

	fmt.Println()
	fmt.Println("Tasks:")
	tasks, err := store.GetTasksByGoal(goalID)
	if err == nil {
		for i, task := range tasks {
			status := "⏳"
			if task.Status == "completed" {
				status = "✓"
			} else if task.Status == "failed" {
				status = "✗"
			}
			fmt.Printf("  %d. %s %s - %s\n", i+1, status, task.Status, task.Description)
			if task.Error != "" {
				fmt.Printf("     Error: %s\n", task.Error)
			}
		}
	}
}

func handleInit(cfg *config.Config) {
	configPath := config.ConfigPath()

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Configuration file already exists at: %s\n", configPath)
		fmt.Print("Overwrite? (y/N): ")
		var response string
	_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Initialization cancelled.")
			return
		}
	}

	if err := config.SaveConfig(cfg, configPath); err != nil {
		log.Fatalf("Failed to save config: %v", err)
	}

	fmt.Printf("✓ Configuration initialized at: %s\n", configPath)
	fmt.Println()
	fmt.Println("Default settings:")
	fmt.Printf("  Projects Root: %s\n", cfg.ProjectsRoot)
	fmt.Printf("  LLM Provider: %s\n", cfg.LLM.Provider)
	fmt.Printf("  LLM Model: %s\n", cfg.LLM.Model)
	fmt.Println()
	fmt.Println("Edit the configuration file to customize settings.")
	fmt.Println("Then start the agent daemon with: octa-agentd")
}
