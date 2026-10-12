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
  printf 'LLM_PROVIDER=%s\nCONCISE_OUTPUT=True\nMARKDOWN_OUTPUT_PATH=\n' "$provider" > "$config_file"
  AGENT_TOOLKIT_ENV_FILE="$config_file" \
    AGENT_TOOLKIT_FAIL_AFTER="$fail_after" \
    AGENTS_HOME="$root/agents" \
    CODEX_HOME="$root/codex" \
    CLAUDE_HOME="$root/claude" \
    "$REPO_ROOT/scripts/install.sh"
}

run_install_with_output_root() {
  local provider=$1 root=$2 output_root=$3
  local config_file="$root/config.env"
  mkdir -p "$root"
  printf 'LLM_PROVIDER=%s\nCONCISE_OUTPUT=True\nMARKDOWN_OUTPUT_PATH=%s\n' \
    "$provider" "$output_root" > "$config_file"
  AGENT_TOOLKIT_ENV_FILE="$config_file" \
    AGENTS_HOME="$root/agents" \
    CODEX_HOME="$root/codex" \
    CLAUDE_HOME="$root/claude" \
    "$REPO_ROOT/scripts/install.sh"
}

run_uninstall() {
  local root=$1
  AGENT_TOOLKIT_ENV_FILE="$root/config.env" \
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
  assert_contains 'invoke the /concise-output skill' "$root/claude/CLAUDE.md"
  assert_contains 'Treat its' "$root/claude/CLAUDE.md"
  assert_contains 'standing guidance for all reader-facing responses' \
    "$root/claude/CLAUDE.md"
  assert_contains 'disable-model-invocation: true' \
    "$root/claude/skills/next-commit/SKILL.md"
  assert_contains 'Apply concise, direct prose to every reader-facing response' \
    "$root/claude/skills/concise-output/SKILL.md"
}

test_repo_local_output_policy() {
  local root="$TEST_ROOT/repo-local"
  run_install codex "$root" >/dev/null
  assert_contains '<repo-root>/<branch-slug>/<skill-name>/...' "$root/codex/AGENTS.md"
  assert_contains 'When consuming Markdown from another skill' "$root/codex/AGENTS.md"
  assert_contains 'target repository' "$root/codex/AGENTS.md"
}

test_external_output_policy() {
  local root="$TEST_ROOT/external"
  local output_root="$TEST_ROOT/centralized outputs"
  run_install_with_output_root codex "$root" "$output_root" >/dev/null
  assert_contains "$output_root/<repo-slug>/<branch-slug>/<skill-name>/..." \
    "$root/codex/AGENTS.md"
  assert_contains 'configured centralized repository/branch/producer-skill directory' \
    "$root/codex/AGENTS.md"
  assert_contains "MARKDOWN_OUTPUT_PATH=$output_root" "$root/config.env"
}

test_environment_output_override() {
  local root="$TEST_ROOT/environment-output"
  local output_root="$TEST_ROOT/environment outputs"
  mkdir -p "$root"
  printf 'LLM_PROVIDER=codex\nCONCISE_OUTPUT=True\nMARKDOWN_OUTPUT_PATH=\n' \
    > "$root/config.env"
  MARKDOWN_OUTPUT_PATH="$output_root" \
    AGENT_TOOLKIT_ENV_FILE="$root/config.env" \
    AGENTS_HOME="$root/agents" \
    CODEX_HOME="$root/codex" \
    CLAUDE_HOME="$root/claude" \
    "$REPO_ROOT/scripts/install.sh" >/dev/null
  assert_contains "$output_root/<repo-slug>/<branch-slug>/<skill-name>/..." \
    "$root/codex/AGENTS.md"
  assert_contains 'MARKDOWN_OUTPUT_PATH=' "$root/config.env"
}

test_missing_output_setting_requires_interactive_configuration() {
  local root="$TEST_ROOT/missing-output" output status
  mkdir -p "$root"
  printf 'LLM_PROVIDER=codex\nCONCISE_OUTPUT=True\n' > "$root/config.env"
  set +e
  output="$(
    unset MARKDOWN_OUTPUT_PATH
    AGENT_TOOLKIT_ENV_FILE="$root/config.env" \
      AGENTS_HOME="$root/agents" \
      CODEX_HOME="$root/codex" \
      CLAUDE_HOME="$root/claude" \
      "$REPO_ROOT/scripts/install.sh" 2>&1
  )"
  status=$?
  set -e
  [[ "$status" -eq 1 ]] || fail "expected missing output configuration to fail, got $status"
  assert_contains 'MARKDOWN_OUTPUT_PATH is not configured' <(printf '%s\n' "$output")
}

test_blank_output_setting_persists_exactly() {
  local root="$TEST_ROOT/blank-output"
  mkdir -p "$root"
  printf 'LLM_PROVIDER=codex\nCONCISE_OUTPUT=True\n' > "$root/config.env"
  AGENT_TOOLKIT_HOME="$REPO_ROOT" bash -c \
    'source "$AGENT_TOOLKIT_HOME/scripts/config.sh"; persist_output_setting "$1" ""' \
    _ "$root/config.env"
  assert_contains 'MARKDOWN_OUTPUT_PATH=' "$root/config.env"
}

test_changed_managed_guidance_is_preserved() {
  local root="$TEST_ROOT/changed-guidance"
  run_install claude "$root" >/dev/null
  printf '\nuser CLAUDE.md content\n' >> "$root/claude/CLAUDE.md"
  run_install claude "$root" >/dev/null
  assert_contains 'user CLAUDE.md content' "$root/claude/CLAUDE.md"
  assert_contains 'agent-toolkit:start' "$root/claude/CLAUDE.md"
  assert_absent "$root/claude/.agent-toolkit-owned"
}

test_changed_managed_skill_is_preserved() {
  local root="$TEST_ROOT/changed-skill"
  run_install codex "$root" >/dev/null
  printf '\nuser SKILL.md content\n' >> \
    "$root/agents/skills/agent-toolkit/python-standards/SKILL.md"
  run_install codex "$root" >/dev/null
  assert_contains 'user SKILL.md content' \
    "$root/agents/skills/agent-toolkit/python-standards/SKILL.md"
}

test_user_owned_content_is_preserved() {
  local root="$TEST_ROOT/user-owned"
  mkdir -p "$root/agents/skills/agent-toolkit/python-standards" "$root/codex"
  printf 'user guidance\n' > "$root/codex/AGENTS.md"
  printf 'user skill\n' > \
    "$root/agents/skills/agent-toolkit/python-standards/SKILL.md"
  run_install codex "$root" >/dev/null
  assert_contains 'user guidance' "$root/codex/AGENTS.md"
  assert_contains 'user skill' \
    "$root/agents/skills/agent-toolkit/python-standards/SKILL.md"
  assert_not_contains 'agent-toolkit:start' \
    "$root/agents/skills/agent-toolkit/python-standards/SKILL.md"
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
  assert_absent "$root/config.env"
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
test_repo_local_output_policy
test_external_output_policy
test_environment_output_override
test_missing_output_setting_requires_interactive_configuration
test_blank_output_setting_persists_exactly
test_changed_managed_guidance_is_preserved
test_changed_managed_skill_is_preserved
test_user_owned_content_is_preserved
test_reinstall_and_uninstall
test_rollback
echo "OK: installer integration tests passed"
