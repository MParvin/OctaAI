# OctaAI Implementation Plan

Machine-readable roadmap derived from the full repository audit (2026-07-18).
Updated 2026-07-19 — remaining phases completed.

---

## Phase 1 — Critical Security & Broken CI

### [x] Task 1.1 — Fix Docker isolation shell injection
### [x] Task 1.2 — Close filesystem path escape (symlink + prefix)
### [x] Task 1.3 — Upgrade vulnerable dependencies
### [x] Task 1.4 — Restore green CI (`make test`)
### [x] Task 1.5 — Harden config secret handling

---

## Phase 2 — Browser Automation Correctness

### [x] Task 2.1 — Align WebSocket URL path (server ↔ extension)
### [x] Task 2.2 — Align default port (8765 vs 9090)
### [x] Task 2.3 — Add missing extension icons / fix load blockers
### [x] Task 2.4 — Fix documentation contradictions for browser security
### [x] Task 2.5 — Move WS token out of query string

---

## Phase 3 — Feature-Flag Honesty & Incomplete Surface Cleanup

### [x] Task 3.1 — Document/disable unwired feature flags
### [x] Task 3.2 — Correct stale product documentation
### [x] Task 3.3 — Implement HTN planner `Replan`

---

## Phase 4 — Defense-in-Depth & Safety Hardening

### [x] Task 4.1 — SSRF defense-in-depth in HTTP tool
### [x] Task 4.2 — Reject empty `allow_paths` in production defaults
### [x] Task 4.3 — Restrict WebSocket empty Origin policy
### [x] Task 4.4 — SQLite concurrency hardening

---

## Phase 5 — Testing Foundations

### [x] Task 5.1 — Add tests for permission/path/SSRF edge cases
### [x] Task 5.2 — Add storage + approval unit tests
### [x] Task 5.3 — Add browser package tests + minimal integration smoke
### [x] Task 5.4 — Raise engine/tools coverage for critical paths
Done: engine ~19%, tools ~26%, total ~34% (was ~19% overall / engine ~4%).
### [x] Task 5.5 — Add govulncheck to CI

---

## Phase 6 — DevOps / Operability

### [x] Task 6.1 — CI completeness (lint + coverage gate)
Done: `make lint`, `make coverage-gate` (default min 20%), CI runs both.
### [x] Task 6.2 — Document Docker runtime usage (no K8s yet)
Done: README + GETTING_STARTED Docker section; Dockerfile non-root + health port.
### [x] Task 6.3 — Daemon health/readiness beyond browser `/health`
Done: `pkg/health` + `--health-addr` (default `127.0.0.1:8766`) `/healthz` `/readyz`.

---

## Phase 7 — Architecture Integration

### [x] Task 7.1 — Wire HTN planner behind `features.use_htn_planner`
Done: `pkg/planner/htn_bridge.go` + engine wiring with v1 fallback.
### [x] Task 7.2 — Implement DAG executor behind `features.use_dag_executor`
Done: `pkg/executor` Scheduler used when flag enabled.
### [x] Task 7.3 — Capability registry wiring (`use_capabilities`)
Done: builtin registry registered when HTN or capabilities flag is on.
### [x] Task 7.4 — Defer AG2 / MCP / vector memory until foundations exist
Done: flags remain false/unimplemented in example config and docs.

---

## Explicitly Out of Scope Until Approved

- Implementing AG2 / MCP / vector DB
- Kubernetes / Helm / Terraform
- Large engine rewrite without feature flags
- Any production exposure of browser WebSocket beyond localhost
