# OctaAI Architecture

Production-oriented autonomous execution engine. This document describes the refactored architecture introduced per the engineering roadmap in `prompt.md`.

## Component Overview

```
                    ┌─────────────┐
                    │  octa-agent │  CLI — submit goals, inspect status
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │ octa-agentd │  Daemon — polls goals, runs engine
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         │                 │                 │
  ┌──────▼──────┐   ┌──────▼──────┐   ┌──────▼──────┐
  │   Planner   │   │   Engine    │   │  Plugin     │
  │             │   │ (state      │   │  Registry   │
  │             │   │  machine)   │   │             │
  └─────────────┘   └──────┬──────┘   └─────────────┘
                           │
      ┌────────────────────┼────────────────────┐
      │                    │                    │
┌─────▼─────┐      ┌───────▼───────┐    ┌──────▼──────┐
│ Tool      │      │  Evaluator    │    │  Memory     │
│ Runner    │      │               │    │  Manager    │
└─────┬─────┘      └───────────────┘    └─────────────┘
      │
┌─────▼─────┐      ┌───────────────┐    ┌─────────────┐
│ Permission│      │ Observability │    │  Workflow   │
│ Manager   │      │ (logs/traces) │    │  Validator  │
└───────────┘      └───────────────┘    └─────────────┘
```

## Goal State Machine

| State | Description |
|-------|-------------|
| `IDLE` | Goal submitted, not yet picked up |
| `PLANNING` | Planner decomposing goal into tasks |
| `EXECUTING` | Tool runner executing tasks |
| `EVALUATING` | Evaluator checking results |
| `RETRYING` | Recoverable failure, will re-execute |
| `WAITING_FOR_APPROVAL` | High-risk action blocked pending human approval |
| `COMPLETED` | Goal satisfied |
| `FAILED` | Unrecoverable failure |

Valid transitions are enforced by `engine.CanTransition()`.

## Execution Step Model

Every tool invocation creates a deterministic `ExecutionStep` with:

- **Input** — tool name, arguments, metadata
- **Output** — success flag, stdout/stderr, structured data
- **Status** — pending → running → completed/failed
- **Validation** — evaluator verdict (success, retry_required, fatal_failure)
- **Retry count** — bounded retries with max limit
- **Timestamps** — started_at, completed_at

Steps are persisted in SQLite (`execution_steps` table).

## Checkpoints

Checkpoints capture resumable snapshots:

- Goal state at checkpoint time
- Memory export (key facts learned during execution)
- Step index for replay

Use `engine.Engine.SaveCheckpoint()` and `ResumeFromCheckpoint()`.

## Plugin Architecture

Tools are grouped into capability plugins:

| Plugin | Capabilities | Tools |
|--------|-------------|-------|
| `coding` | coding | filesystem, command, git |
| `devops` | devops, ssh | ssh, http |
| `browser` | browser | browser (when enabled) |

Register plugins in `plugin.DefaultPlugins()` and load via `plugin.Registry.LoadAll()`.

## Permission Manager

All tool calls pass through `permission.Manager` before execution:

- Path allow-list for filesystem operations
- Command deny-list and require-approval patterns
- SSH operations require approval by default

## Evaluator Chain

Default evaluators run in sequence:

1. **ToolResultEvaluator** — basic success/failure
2. **BuildResultEvaluator** — detect compile/build errors in command output
3. **TestResultEvaluator** — detect test failures
4. **GoalCompletionEvaluator** — verify all steps completed

## Workflow Validation

LLM-generated workflow JSON is validated before execution:

- Required node IDs and descriptions
- Valid dependency references
- No dependency cycles

See `pkg/workflow/workflow.go`.

## Directory Layout

```
pkg/
├── agent/          # Thin wrapper over engine (backward compat)
├── engine/         # State machine + execution loop
├── execution/      # Shared step/validation types
├── planner/        # Goal → task decomposition
├── evaluator/      # Result validation
├── memory/         # Task memory + TF-IDF semantic search
├── permission/     # Safety policy + approval integration
├── approval/       # Human approval service
├── isolation/      # Docker sandbox for command execution
├── observability/  # Structured logs, spans, token usage
├── workflow/       # Execution graph validation
├── plugin/         # Capability plugins
├── llm/            # Model provider layer
├── storage/        # SQLite persistence
├── tools/          # Tool implementations
└── browser/        # Firefox WebSocket server
```

## Roadmap (from prompt.md)

| Phase | Status | Scope |
|-------|--------|-------|
| A | Done | State machine, execution steps, engine refactor |
| B | Done | Planner, evaluator, permission, observability |
| C | Done | Checkpoints, memory, workflow validation, plugins |
| D | Done | Docker isolation, approval CLI, semantic memory |
| E | Done | Parallel execution graph, dynamic replanning |

## Failure Protection

- Max loop count (default 50) prevents infinite execution
- Max retries per step (default 3)
- Stall detection (no progress after N iterations)
- State transition validation
- Command deny-list
