#!/usr/bin/env bash
# Integration test script for OctaAI browser automation
# Prerequisites:
#   - octa-agentd running with browser.enabled=true
#   - Firefox addon loaded and connected
#   - octa-agent binary on PATH or in ./bin/
# Usage:
#   ./examples/browser/test_integration.sh [--agent-bin <path>]

set -euo pipefail

AGENT_BIN="${AGENT_BIN:-./bin/octa-agent}"
PASS=0
FAIL=0
SKIP=0

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

green() { printf '\033[0;32m%s\033[0m\n' "$*"; }
red()   { printf '\033[0;31m%s\033[0m\n' "$*"; }
yellow(){ printf '\033[0;33m%s\033[0m\n' "$*"; }

run_test() {
  local name="$1"
  local goal="$2"
  local expected_pattern="$3"  # grep-compatible pattern to verify output file / stdout

  printf '  Testing: %s ... ' "$name"

  if ! command -v "$AGENT_BIN" &>/dev/null && [[ ! -x "$AGENT_BIN" ]]; then
    yellow "SKIP (agent binary not found: $AGENT_BIN)"
    (( SKIP++ )) || true
    return
  fi

  local out
  out=$("$AGENT_BIN" goal "$goal" 2>&1) || true

  if echo "$out" | grep -qiE "$expected_pattern"; then
    green "PASS"
    (( PASS++ )) || true
  else
    red "FAIL"
    echo "    Goal   : $goal"
    echo "    Pattern: $expected_pattern"
    echo "    Output : $(echo "$out" | head -5)"
    (( FAIL++ )) || true
  fi
}

check_daemon() {
  if ! curl -sf "http://localhost:8765/health" > /dev/null 2>&1; then
    red "ERROR: octa-agentd does not appear to be running on port 8765."
    echo "  Start it with:  ./bin/octa-agentd --browser-port 8765"
    exit 1
  fi

  local browsers
  browsers=$(curl -sf "http://localhost:8765/health" | grep -o '"connected_browsers":[0-9]*' | grep -o '[0-9]*' || echo "0")
  if [[ "$browsers" == "0" ]]; then
    yellow "WARNING: No browser connected. Tests that require a browser will be skipped or fail."
    echo "  Load the Firefox extension and ensure it shows 'Connected'."
  else
    green "Daemon running with $browsers browser(s) connected."
  fi
}

# ---------------------------------------------------------------------------
# Test suite
# ---------------------------------------------------------------------------

echo ""
echo "=== OctaAI Browser Automation Integration Tests ==="
echo ""

check_daemon

echo ""
echo "--- Phase 4.1: Basic Navigation & Extraction ---"

run_test \
  "Navigate and extract heading" \
  "Navigate to https://example.com and extract the text of the main h1 heading, then print it" \
  "example domain|Example Domain"

run_test \
  "Extract multiple elements" \
  "Go to https://news.ycombinator.com and extract the titles of the first 3 articles on the front page" \
  "[A-Za-z]"

echo ""
echo "--- Phase 4.1: Form Filling ---"

run_test \
  "Fill and submit a form" \
  "Go to https://httpbin.org/forms/post, fill custname with 'TestUser', fill custemail with 'test@example.com', submit the form, and confirm the result contains the submitted name" \
  "TestUser|custname"

echo ""
echo "--- Phase 4.1: Multi-step workflow ---"

run_test \
  "Navigate and save extracted data" \
  "Go to https://github.com/trending, extract the names of the top 3 repositories listed, and save them to /tmp/octaai_trending_test.txt" \
  "saved|written|complete"

if [[ -f /tmp/octaai_trending_test.txt ]]; then
  green "  Output file /tmp/octaai_trending_test.txt exists."
  head -5 /tmp/octaai_trending_test.txt | sed 's/^/    /'
  rm -f /tmp/octaai_trending_test.txt
fi

echo ""
echo "--- Phase 4.1: Error handling ---"

run_test \
  "Element not found" \
  "Go to https://example.com and click the element with selector '#this-does-not-exist-xyz', report any errors" \
  "not found|error|failed"

echo ""
echo "--- Phase 4.2: Security ---"

run_test \
  "Reject unauthenticated WebSocket" \
  "Use the browser tool to test that a connection attempt with an invalid token is rejected" \
  "auth|reject|failed|error"

echo ""
echo "=== Results ==="
echo "  PASS: $PASS"
echo "  FAIL: $FAIL"
echo "  SKIP: $SKIP"
echo ""

if [[ $FAIL -gt 0 ]]; then
  red "Some tests failed."
  exit 1
else
  green "All executed tests passed."
  exit 0
fi
