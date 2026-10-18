#!/usr/bin/env bash
# Exercise adapter validation against isolated source trees.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/agent-toolkit-validation.XXXXXX")"
trap 'rm -rf "$TEST_ROOT"' EXIT

fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }

fixture() {
  local root="$TEST_ROOT/$1"
  mkdir -p "$root"
  cp -R "$REPO_ROOT/.agents" "$REPO_ROOT/scripts" "$REPO_ROOT/.env.template" "$root/"
  printf '%s\n' "$root"
}

assert_invalid() {
  local root=$1 expected=$2 output status
  if output="$(AGENT_TOOLKIT_HOME="$root" "$root/scripts/validate.sh" 2>&1)"; then
    fail "expected validation to reject $expected"
  else
    status=$?
  fi
  [[ "$status" -eq 1 ]] || fail "expected validation status 1, got $status: $output"
  [[ "$output" == *"$expected"* ]] || fail "missing diagnostic '$expected': $output"
}

test_optional_codex_adapter() {
  local root
  root="$(fixture optional)"
  mv "$root/.agents/providers/codex/skills/review/plan-review/agents/openai.yaml" "$root/optional.yaml"
  AGENT_TOOLKIT_HOME="$root" "$root/scripts/validate.sh" >/dev/null
}

test_moved_codex_adapter() {
  local root
  root="$(fixture moved)"
  mkdir -p "$root/.agents/providers/codex/skills/scoping/plan-review/agents"
  mv "$root/.agents/providers/codex/skills/review/plan-review/agents/openai.yaml" \
    "$root/.agents/providers/codex/skills/scoping/plan-review/agents/openai.yaml"
  assert_invalid "$root" "no source skill for adapter key 'scoping/plan-review'"
}

test_removed_codex_skill() {
  local root
  root="$(fixture removed)"
  mv "$root/.agents/skills/review/plan-review" "$root/removed-skill"
  assert_invalid "$root" "no source skill for adapter key 'review/plan-review'"
}

test_stale_claude_entry() {
  local root
  root="$(fixture claude-stale)"
  printf '\nimplementation/removed-skill' >> "$root/.agents/providers/claude/explicit-only.txt"
  assert_invalid "$root" "no source skill for adapter key 'implementation/removed-skill'"
}

test_missing_claude_list() {
  local root
  root="$(fixture claude-missing)"
  mv "$root/.agents/providers/claude/explicit-only.txt" "$root/missing-list.txt"
  assert_invalid "$root" 'missing or unreadable Claude explicit-only list'
}

test_unknown_dependency() {
  local root
  root="$(fixture dependency)"
  printf '\ncommit-message missing-helper\n' >> "$root/.agents/dependencies.txt"
  assert_invalid "$root" 'unknown skill: missing-helper'
}

test_duplicate_dependency_entry() {
  local root
  root="$(fixture duplicate-dependency)"
  printf '\npr-description to-clipboard\n' >> "$root/.agents/dependencies.txt"
  assert_invalid "$root" 'duplicate dependency entry: pr-description'
}

test_optional_codex_adapter
test_moved_codex_adapter
test_removed_codex_skill
test_stale_claude_entry
test_missing_claude_list
test_unknown_dependency
test_duplicate_dependency_entry
printf 'OK: adapter validation tests passed\n'
