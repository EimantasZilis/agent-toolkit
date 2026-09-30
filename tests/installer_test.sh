#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/agent-toolkit-tests.XXXXXX")"
trap 'rm -rf "$TEST_ROOT"' EXIT

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

assert_file() { [[ -f "$1" ]] || fail "expected file: $1"; }
assert_directory() { [[ -d "$1" ]] || fail "expected directory: $1"; }
assert_absent() { [[ ! -e "$1" && ! -L "$1" ]] || fail "expected path to be absent: $1"; }
assert_contains() { grep -Fq -- "$1" "$2" || fail "expected '$1' in $2"; }
assert_not_contains() { ! grep -Fq -- "$1" "$2" || fail "did not expect '$1' in $2"; }

run_install() {
  local provider=$1 root=$2 fail_after=${3:-}
  local config_file="$root/config.env"
  mkdir -p "$root"
  printf 'LLM_PROVIDER=%s\nCONCISE_OUTPUT=True\n' "$provider" > "$config_file"
  AGENT_TOOLKIT_ENV_FILE="$config_file" \
    AGENT_TOOLKIT_FAIL_AFTER="$fail_after" \
    AGENTS_HOME="$root/agents" \
    CODEX_HOME="$root/codex" \
    CLAUDE_HOME="$root/claude" \
    "$REPO_ROOT/scripts/install.sh"
}

run_uninstall() {
  local root=$1
  AGENTS_HOME="$root/agents" \
    CODEX_HOME="$root/codex" \
    CLAUDE_HOME="$root/claude" \
    "$REPO_ROOT/scripts/uninstall.sh"
}

test_codex_install() {
  local root="$TEST_ROOT/codex root"
  run_install codex "$root" >/dev/null
  assert_directory "$root/agents/skills/agent-toolkit"
  assert_file "$root/codex/AGENTS.md"
  assert_file "$root/codex/.agent-toolkit-owned"
  [[ -L "$root/agents/skills/agent-toolkit/next-commit/agents/openai.yaml" ]] ||
    fail "expected Codex metadata symlink"
}

test_claude_install() {
  local root="$TEST_ROOT/claude"
  run_install claude "$root" >/dev/null
  assert_directory "$root/claude/skills"
  assert_file "$root/claude/CLAUDE.md"
  assert_contains 'disable-model-invocation: true' \
    "$root/claude/skills/next-commit/SKILL.md"
}

test_user_owned_content_is_preserved() {
  local root="$TEST_ROOT/user-owned"
  mkdir -p "$root/codex/skills/agent-toolkit/python-standards"
  printf 'user guidance\n' > "$root/codex/AGENTS.md"
  printf 'user skill\n' > "$root/codex/skills/agent-toolkit/python-standards/SKILL.md"
  run_install codex "$root" >/dev/null
  assert_contains 'user guidance' "$root/codex/AGENTS.md"
  assert_contains 'user skill' \
    "$root/codex/skills/agent-toolkit/python-standards/SKILL.md"
  assert_not_contains 'agent-toolkit:start' "$root/codex/skills/agent-toolkit/python-standards/SKILL.md"
}

test_reinstall_and_uninstall() {
  local root="$TEST_ROOT/reinstall"
  run_install codex "$root" >/dev/null
  run_install codex "$root" >/dev/null
  [[ "$(grep -Fc '<!-- agent-toolkit:start -->' "$root/codex/AGENTS.md")" -eq 1 ]] ||
    fail "reinstall duplicated managed guidance"
  run_uninstall "$root" >/dev/null
  assert_absent "$root/codex/AGENTS.md"
  assert_absent "$root/agents/skills/agent-toolkit/next-commit"
}

test_rollback() {
  local root="$TEST_ROOT/rollback" output status
  mkdir -p "$root/codex"
  printf 'user guidance\n' > "$root/codex/AGENTS.md"
  set +e
  output="$(run_install codex "$root" 1 2>&1)"
  status=$?
  set -e
  [[ "$status" -eq 97 ]] || fail "expected injected failure status 97, got $status"
  assert_contains 'user guidance' "$root/codex/AGENTS.md"
  assert_not_contains 'agent-toolkit:start' "$root/codex/AGENTS.md"
  assert_absent "$root/agents/skills/agent-toolkit/next-commit"
  assert_contains 'test failure injected' <(printf '%s\n' "$output")
}

test_codex_install
test_claude_install
test_user_owned_content_is_preserved
test_reinstall_and_uninstall
test_rollback
echo "OK: installer integration tests passed"
