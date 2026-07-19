#!/usr/bin/env bash
# End-to-end smoke: start daemon, submit a simple goal via CLI, wait for COMPLETED,
# assert a filesystem artifact was produced.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DAEMON="${ROOT}/bin/octa-agentd"
CLI="${ROOT}/bin/octa-agent"

OLLAMA_HOST="${OLLAMA_HOST:-http://127.0.0.1:11434}"
# Ollama often exports host:port without a scheme; normalize for Go url.Parse.
case "${OLLAMA_HOST}" in
  http://*|https://*) ;;
  *) OLLAMA_HOST="http://${OLLAMA_HOST}" ;;
esac
LIVE_HEALTH_ADDR="${LIVE_HEALTH_ADDR:-127.0.0.1:18766}"
LIVE_TEST_TIMEOUT_SEC="${LIVE_TEST_TIMEOUT_SEC:-600}"
LIVE_TEST_KEEP="${LIVE_TEST_KEEP:-0}"
POLL_INTERVAL_SEC="${POLL_INTERVAL_SEC:-5}"

MARKER_FILE="LIVE_OK.txt"
MARKER_CONTENT="LIVE_TEST_OK"

die() {
  echo "live-test: ERROR: $*" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

pick_model() {
  local tags preferred m
  tags="$(curl -sf "${OLLAMA_HOST}/api/tags")" || return 1
  if [[ -n "${LIVE_TEST_MODEL:-}" ]]; then
    if printf '%s' "${tags}" | grep -q "\"name\":\"${LIVE_TEST_MODEL}\""; then
      echo "${LIVE_TEST_MODEL}"
      return 0
    fi
    die "LIVE_TEST_MODEL=${LIVE_TEST_MODEL} is not installed in Ollama"
  fi
  for preferred in qwen2.5:32b qwen3:8b deepseek-coder-v2:16b mistral-nemo:12b dolphin-mistral:7b; do
    if printf '%s' "${tags}" | grep -q "\"name\":\"${preferred}\""; then
      echo "${preferred}"
      return 0
    fi
  done
  # Fall back to first non-embed local model name from tags.
  m="$(printf '%s' "${tags}" | sed -n 's/.*"name":"\([^"]*\)".*/\1/p' | grep -v embed | grep -v cloud | head -1 || true)"
  [[ -n "${m}" ]] || return 1
  echo "${m}"
}

need curl
need sqlite3

[[ -x "$DAEMON" ]] || die "missing binary $DAEMON (run: make build)"
[[ -x "$CLI" ]] || die "missing binary $CLI (run: make build)"

echo "live-test: checking Ollama at ${OLLAMA_HOST}..."
if ! curl -sf "${OLLAMA_HOST}/api/tags" >/dev/null; then
  die "Ollama not reachable at ${OLLAMA_HOST} (start with: ollama serve)"
fi

LIVE_TEST_MODEL="$(pick_model)" || die "no suitable Ollama model found (pull one, e.g. ollama pull qwen3:8b)"
echo "live-test: using model ${LIVE_TEST_MODEL}"

LIVE_HOME="$(mktemp -d "${TMPDIR:-/tmp}/octaai-live-XXXXXX")"
PROJECTS="${LIVE_HOME}/Projects"
CONFIG_DIR="${LIVE_HOME}/.config/octaai"
STATE_DB="${CONFIG_DIR}/state.db"
DAEMON_PID=""
DAEMON_LOG="${LIVE_HOME}/daemon.log"
PROJECT_NAME="live_smoke_$(date +%s)"

cleanup() {
  local code=$?
  if [[ -n "${DAEMON_PID}" ]] && kill -0 "${DAEMON_PID}" 2>/dev/null; then
    kill "${DAEMON_PID}" 2>/dev/null || true
    wait "${DAEMON_PID}" 2>/dev/null || true
  fi
  if [[ "${LIVE_TEST_KEEP}" == "1" ]]; then
    echo "live-test: keeping workspace at ${LIVE_HOME} (LIVE_TEST_KEEP=1)"
  else
    rm -rf "${LIVE_HOME}"
  fi
  exit "${code}"
}
trap cleanup EXIT INT TERM

mkdir -p "${PROJECTS}" "${CONFIG_DIR}"

cat >"${CONFIG_DIR}/config.yaml" <<EOF
projects_root: "${PROJECTS}"

llm:
  provider: "ollama"
  model: "${LIVE_TEST_MODEL}"
  base_url: "${OLLAMA_HOST}"
  temperature: 0.2
  max_tokens: 2048

safety:
  allow_paths:
    - "${PROJECTS}"
  deny_commands:
    - "rm -rf /"
  require_confirmation_for: []

storage:
  type: "sqlite"
  path: "${STATE_DB}"

engine:
  max_loops: 30
  max_retries: 3
  enable_replan: true
  enable_parallel: true

isolation:
  enabled: false

browser:
  enabled: false

features:
  use_htn_planner: false
  use_dag_executor: false
  use_capabilities: false
  enable_ag2: false
  use_vector_memory: false
  enable_mcp: false
  enable_adaptive_replan: false
  enable_reflection: false
EOF

export HOME="${LIVE_HOME}"

echo "live-test: starting daemon (HOME=${LIVE_HOME}, health=${LIVE_HEALTH_ADDR})..."
"${DAEMON}" --health-addr "${LIVE_HEALTH_ADDR}" >"${DAEMON_LOG}" 2>&1 &
DAEMON_PID=$!

ready=0
for _ in $(seq 1 60); do
  if curl -sf "http://${LIVE_HEALTH_ADDR}/readyz" >/dev/null 2>&1; then
    ready=1
    break
  fi
  if ! kill -0 "${DAEMON_PID}" 2>/dev/null; then
    echo "----- daemon log -----" >&2
    cat "${DAEMON_LOG}" >&2 || true
    die "daemon exited before becoming ready"
  fi
  sleep 0.5
done
[[ "${ready}" == "1" ]] || die "daemon not ready at http://${LIVE_HEALTH_ADDR}/readyz"

GOAL_DESC="Create a file named ${MARKER_FILE} in the project root containing exactly the text ${MARKER_CONTENT} and nothing else."
echo "live-test: submitting goal (project=${PROJECT_NAME})..."
GOAL_OUT="$("${CLI}" goal --project "${PROJECT_NAME}" "${GOAL_DESC}")"
echo "${GOAL_OUT}"
GOAL_ID="$(printf '%s\n' "${GOAL_OUT}" | sed -n 's/.*Goal created: \(goal_[0-9][0-9]*\).*/\1/p' | head -1)"
[[ -n "${GOAL_ID}" ]] || die "could not parse goal id from CLI output"

goal_state() {
  sqlite3 "${STATE_DB}" "SELECT state FROM goals WHERE id='${GOAL_ID}';"
}

goal_error() {
  sqlite3 "${STATE_DB}" "SELECT IFNULL(error,'') FROM goals WHERE id='${GOAL_ID}';"
}

goal_result() {
  sqlite3 "${STATE_DB}" "SELECT IFNULL(result,'') FROM goals WHERE id='${GOAL_ID}';"
}

echo "live-test: waiting for ${GOAL_ID} (timeout ${LIVE_TEST_TIMEOUT_SEC}s)..."
deadline=$((SECONDS + LIVE_TEST_TIMEOUT_SEC))
saw_activity=0
final_state=""

while (( SECONDS < deadline )); do
  if ! kill -0 "${DAEMON_PID}" 2>/dev/null; then
    echo "----- daemon log -----" >&2
    cat "${DAEMON_LOG}" >&2 || true
    die "daemon died while processing goal"
  fi

  state="$(goal_state || true)"
  echo "live-test: status poll → ${GOAL_ID} state=${state:-unknown}"
  "${CLI}" status || true
  LOGS_OUT="$("${CLI}" logs "${GOAL_ID}" 2>/dev/null || true)"
  if [[ -n "${LOGS_OUT}" && "${LOGS_OUT}" != *"No logs found"* ]]; then
    saw_activity=1
    echo "live-test: logs (tail):"
    printf '%s\n' "${LOGS_OUT}" | tail -n 20
  fi

  case "${state}" in
    COMPLETED|FAILED)
      final_state="${state}"
      break
      ;;
  esac

  sleep "${POLL_INTERVAL_SEC}"
done

if [[ -z "${final_state}" ]]; then
  echo "----- daemon log (tail) -----" >&2
  tail -n 80 "${DAEMON_LOG}" >&2 || true
  die "timed out after ${LIVE_TEST_TIMEOUT_SEC}s waiting for ${GOAL_ID} (last state=$(goal_state || echo none))"
fi

if [[ "${final_state}" != "COMPLETED" ]]; then
  echo "----- daemon log (tail) -----" >&2
  tail -n 80 "${DAEMON_LOG}" >&2 || true
  die "goal ${GOAL_ID} ended in ${final_state}: $(goal_error)"
fi

[[ "${saw_activity}" == "1" ]] || die "goal completed but no logs were observed via octa-agent logs"

ARTIFACT="${PROJECTS}/${PROJECT_NAME}/${MARKER_FILE}"
[[ -f "${ARTIFACT}" ]] || die "expected artifact missing: ${ARTIFACT}"

actual="$(tr -d '\r' <"${ARTIFACT}" | sed -e 's/[[:space:]]*$//')"
[[ "${actual}" == "${MARKER_CONTENT}" ]] || die "artifact content mismatch: got $(printf %q "${actual}"), want ${MARKER_CONTENT}"

echo "live-test: OK — ${GOAL_ID} COMPLETED"
echo "live-test: result=$(goal_result)"
echo "live-test: artifact ${ARTIFACT} contains ${MARKER_CONTENT}"
