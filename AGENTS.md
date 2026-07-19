# AGENTS.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Core development commands

### Setup and build
- `make deps` — download and tidy Go modules.
- `make build` — build both binaries into `bin/` (`octa-agentd`, `octa-agent`).
- `make build-daemon` / `make build-cli` — build one binary.
- `make install` — install both binaries via `go install`.

### Run locally
- `./bin/octa-agentd` — start daemon (polls and processes queued goals).
- `./bin/octa-agentd --browser-port 8765` — start daemon with browser automation enabled on a specific port.
- `./bin/octa-agent init` — create `~/.config/octaai/config.yaml`.
- `./bin/octa-agent goal "<description>"` — submit a goal.
- `./bin/octa-agent status` — list goal states and progress.
- `./bin/octa-agent logs <goal-id>` — inspect execution logs and task outcomes.
- `./bin/octa-agent approvals` / `approve <id>` / `deny <id>` — handle human approval gates.

### Test, lint, and validation
- `make test` — run all tests (`go test -v ./...`).
- `make test-coverage` — run tests with coverage report (`coverage.out`, `coverage.html`).
- `go test -v ./pkg/workflow -run TestName` — run a single test in one package.
- `make lint` — run `golangci-lint run` (requires `golangci-lint` installed).
- `make fmt` and `make vet` — baseline formatting and static checks.

## High-level architecture

### Runtime shape (daemon + CLI + state store)
- `cmd/octa-agent/main.go`: CLI for creating goals, checking status/logs, and resolving approvals.
- `cmd/octa-agentd/main.go`: long-running daemon that loads config/LLM/tools, polls goals every 5 seconds, resumes interrupted `EXECUTING`/`EVALUATING` goals, and processes goals concurrently.
- `pkg/storage/sqlite.go`: central persistence layer — tables: `goals`, `tasks`, `execution_steps`, `logs`, `checkpoints`, `approvals`. SQLite is the source of truth for resumability. Default path: `~/.config/octaai/state.db`.

### Goal state machine
Goals move through: `IDLE → PLANNING → EXECUTING → EVALUATING → COMPLETED / FAILED`, with optional states `RETRYING` (recoverable failure triggers replanning), `WAITING_FOR_APPROVAL` (permission gate), and `BLOCKED`. Valid transitions are enforced by `engine.CanTransition()` in `pkg/engine/state.go`.

### Goal execution pipeline
- `pkg/agent/agent.go`: thin wrapper around the engine.
- `pkg/engine/engine.go`: drives the state machine loop (default: 50 max loops, 500ms pause per loop):
  1. Planner creates 1–2 template tasks (project setup + LLM-driven execution task); the LLM chooses concrete tool calls per step inside `executeLLMTask`.
  2. Engine runs ready tasks respecting dependencies; parallel up to `MaxParallel=3` when enabled.
  3. Every tool call produces a persisted `ExecutionStep` record (pending → running → completed/failed) with retry count bounded by `engine.max_retries` (default 3).
  4. Evaluator chain runs after each step; can signal retry or fatal failure.
  5. On recoverable failure with `EnableReplan=true`, `replan.go` adds one corrective task.
- `pkg/engine/runner.go`: executes tool calls with per-step timeout (default 5 min) + permission checks + optional Docker isolation (argv-safe).
- `pkg/engine/checkpoint.go`: checkpoints saved during execution; daemon resumes interrupted goals on restart.

### Evaluator chain
`pkg/evaluator/evaluator.go` runs four evaluators in sequence per step:
1. **ToolResultEvaluator** — basic success/failure from `ToolResult.Success`.
2. **BuildResultEvaluator** — detects compile/build error patterns in command output.
3. **TestResultEvaluator** — detects test failure patterns.
4. **GoalCompletionEvaluator** — heuristic check based on step outcomes (not a separate LLM call).

Each returns one of: `success`, `partial_success`, `retry_required`, `fatal_failure` (defined in `pkg/execution/types.go`).

### Planner
`pkg/planner/planner.go` uses a lightweight template planner (1–2 tasks) with an LLM yes/no project-name prompt. Actual tool selection happens per-step in the engine. `pkg/workflow` provides DAG validation utilities used in tests only.

### Extensibility and tool loading
- `pkg/plugin/registry.go` + `pkg/plugin/builtin.go`: plugin system wires capability groups into tool registry at daemon startup.
- Default plugins: `coding` (filesystem, command, git), `devops` (ssh, http), `browser` (only when `--browser-port` is passed).
- Adding a new tool: implement `tools.Tool` interface (Name/Schema/Execute), register it in the relevant plugin's `Load()`, and add a `CheckTool` case in `pkg/permission/manager.go`.

### Safety, approvals, and isolation
- `pkg/permission/manager.go`: enforces policy for all six tools — `Allow`, `Deny`, or `RequireApproval`. Command `cwd`, git paths, HTTP URLs (SSRF guard), browser domains, and SSH always gated.
- `pkg/approval/service.go`: persists pending approvals; resolved interactively via `octa-agent approve/deny <id>`.
- `pkg/isolation/docker.go`: wraps tool args for Docker execution when `isolation.require_docker_for` lists the tool type.

### LLM providers
Configured via `llm.provider` in config. Supported values: `ollama` (default, no API key needed), `openai`, `claude`. The provider interface (`pkg/llm/provider.go`) exposes `Complete(ctx, messages)` — all planners and evaluators go through this single interface.

### Memory
- `pkg/memory/manager.go`: appends facts learned during execution to a per-goal store.
- `pkg/memory/semantic.go`: TF-IDF semantic retrieval of past facts for planner context (not a vector DB).

### Configuration that affects behavior
- Primary config: `~/.config/octaai/config.yaml` (created by `octa-agent init`).
- Key sections in `pkg/config/config.go`:
 - `llm`: provider / model / base_url / api_key / temperature / max_tokens
 - `safety`: allow_paths, allow_http_hosts, deny_commands, require_confirmation_for
 - `engine`: max_loops, max_retries, enable_replan, enable_parallel
 - `isolation`: enabled, docker (image/network/memory/cpu limits), require_docker_for
 - `browser`: enabled, port, token, browser_domains
 - `storage`: type, path (defaults to `~/.config/octaai/state.db`)
 - `features`: `use_htn_planner` / `use_dag_executor` / `use_capabilities` are wired; AG2/MCP/vector/reflection flags remain unimplemented.
 - Daemon health: `--health-addr` (default `127.0.0.1:8766`) serves `/healthz` and `/readyz`.

### Memory note
- `pkg/memory/semantic.go` is TF-IDF keyword retrieval, not a vector database. `features.use_vector_memory` is unimplemented.
