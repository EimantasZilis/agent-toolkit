#!/usr/bin/env bash
# Exercise shared guidance settings, migration, preservation, and rollback.
set -euo pipefail

SOURCE_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
TEST_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/agent-toolkit-guidance-test.XXXXXX")
trap 'rm -rf -- "$TEST_ROOT"' EXIT
unset MARKDOWN_OUTPUT_PATH
fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }
fixture() {
  case_root="$TEST_ROOT/$1"
  mkdir -p "$case_root/toolkit"
  cp -R "$SOURCE_ROOT/scripts" "$SOURCE_ROOT/.agents" "$SOURCE_ROOT/.env.template" "$case_root/toolkit/"
  export AGENT_TOOLKIT_HOME="$case_root/toolkit" AGENT_TOOLKIT_ENV_FILE="$case_root/legacy.env"
  export AGENTS_HOME="$case_root/agents" CODEX_HOME="$case_root/codex" CLAUDE_HOME="$case_root/claude"
  mkdir -p "$CODEX_HOME" "$CLAUDE_HOME"
  printf 'User Codex instructions\n' > "$CODEX_HOME/AGENTS.md"
  printf 'User Claude instructions\n' > "$CLAUDE_HOME/CLAUDE.md"
}
backend() { "$AGENT_TOOLKIT_HOME/scripts/managed.sh" "$@"; }
preview_revision() { backend preview "$@" | python3 -c 'import json,sys; print(json.load(sys.stdin)["revision"])'; }
apply_current() {
  local revision
  revision=$(preview_revision "$@")
  backend apply "$@" --revision "$revision" --confirm-removals >/dev/null
}
expect_failure() {
  local expected=$1 status
  shift
  if "$@" > "$case_root/error" 2>&1; then fail 'expected failure'; else status=$?; fi
  [[ "$status" -eq "$expected" ]] || fail "expected $expected, got $status: $(cat "$case_root/error")"
}
assert_contains() { grep -Fq -- "$1" "$2" || fail "missing $1 in $2"; }
assert_absent_text() { if grep -Fq -- "$1" "$2"; then fail "unexpected $1 in $2"; fi; }
snapshot() {
  find "$AGENTS_HOME" "$CODEX_HOME" "$CLAUDE_HOME" "$AGENT_TOOLKIT_HOME/config" \
    -type f -exec shasum -a 256 {} + | LC_ALL=C sort
}

fixture settings
output="$case_root/old artifacts"
mkdir "$output"
printf 'retained artifact\n' > "$output/old.md"
apply_current --provider codex --provider claude --skill ticket-to-plan --concise true --output-path "$output"
for guidance in "$CODEX_HOME/AGENTS.md" "$CLAUDE_HOME/CLAUDE.md"; do
  assert_contains "$output/<repo-slug>" "$guidance"
  assert_contains 'At the start of every session' "$guidance"
  assert_absent_text '# agent-toolkit guidance' "$guidance"
done
[[ -f "$AGENTS_HOME/skills/agent-toolkit/concise-output/SKILL.md" && -f "$CLAUDE_HOME/skills/concise-output/SKILL.md" ]] || fail 'automatic skill missing'
apply_current --output-path '' --concise false
for guidance in "$CODEX_HOME/AGENTS.md" "$CLAUDE_HOME/CLAUDE.md"; do
  assert_absent_text 'agent-toolkit:start' "$guidance"
  assert_absent_text "$output" "$guidance"
done
assert_contains 'User Codex instructions' "$CODEX_HOME/AGENTS.md"
assert_contains 'User Claude instructions' "$CLAUDE_HOME/CLAUDE.md"
[[ -f "$AGENTS_HOME/skills/agent-toolkit/concise-output/SKILL.md" ]] || fail 'disabled auto-loading uninstalled skill'
assert_contains 'No centralized output root is configured' "$AGENTS_HOME/skills/agent-toolkit/ticket-to-plan/SKILL.md"
assert_contains 'retained artifact' "$output/old.md"

apply_current --concise true
printf '\nNew user instructions\n' >> "$CODEX_HOME/AGENTS.md"
apply_current --output-path "$case_root/new output"
assert_contains 'New user instructions' "$CODEX_HOME/AGENTS.md"
source "$AGENT_TOOLKIT_HOME/scripts/shared.sh"
managed_block=$(awk '/^<!-- agent-toolkit:start -->$/ { inside = 1 } inside { print } /^<!-- agent-toolkit:end -->$/ { inside = 0 }' "$CODEX_HOME/AGENTS.md")
printf '%s\n' "$managed_block" >> "$CODEX_HOME/AGENTS.md"
apply_current
[[ $(grep -Fc '<!-- agent-toolkit:start -->' "$CODEX_HOME/AGENTS.md") -eq 1 ]] || fail duplicate
cp "$CODEX_HOME/AGENTS.md" "$case_root/clean-guidance"
sed 's/standing guidance/modified instructions/' "$case_root/clean-guidance" > "$CODEX_HOME/AGENTS.md"
revision=$(preview_revision --concise false)
expect_failure 4 backend apply --concise false --revision "$revision" --confirm-removals
assert_contains 'modified instructions' "$CODEX_HOME/AGENTS.md"
cp "$case_root/clean-guidance" "$CODEX_HOME/AGENTS.md"

snapshot > "$case_root/before"
revision=$(preview_revision --output-path "$case_root/failed output")
expect_failure 97 env AGENT_TOOLKIT_FAIL_PROVIDER=claude "$AGENT_TOOLKIT_HOME/scripts/managed.sh" \
  apply --output-path "$case_root/failed output" --revision "$revision" --confirm-removals
snapshot > "$case_root/after"
cmp "$case_root/before" "$case_root/after" || fail 'guidance rollback changed files or saved settings'

fixture legacy
"$AGENT_TOOLKIT_HOME/scripts/install.sh" --provider codex --skill commit-message --global-guidance >/dev/null
assert_contains '# agent-toolkit guidance' "$CODEX_HOME/AGENTS.md"
apply_current --full-template false --concise false --output-path "$case_root/central"
assert_absent_text '# agent-toolkit guidance' "$CODEX_HOME/AGENTS.md"
assert_contains 'User Codex instructions' "$CODEX_HOME/AGENTS.md"
assert_contains 'GUIDANCE_BLOCK ' "$CODEX_HOME/.agent-toolkit-owned"
"$AGENT_TOOLKIT_HOME/scripts/install.sh" --provider codex --skill commit-message --global-guidance >/dev/null
assert_contains '# agent-toolkit guidance' "$CODEX_HOME/AGENTS.md"
apply_current --full-template false --concise false --output-path ''
assert_absent_text 'agent-toolkit:start' "$CODEX_HOME/AGENTS.md"
assert_contains 'User Codex instructions' "$CODEX_HOME/AGENTS.md"

fixture uninstall
apply_current --provider codex --skill commit-message --concise true
cp "$CODEX_HOME/AGENTS.md" "$case_root/clean-guidance"
printf '\n<!-- agent-toolkit:concise-output:start -->\nUser-edited loading instructions\n<!-- agent-toolkit:concise-output:end -->\n' >> "$CODEX_HOME/AGENTS.md"
"$AGENT_TOOLKIT_HOME/scripts/uninstall.sh" > /dev/null 2> "$case_root/warnings"
assert_contains 'User-edited loading instructions' "$CODEX_HOME/AGENTS.md"
assert_contains 'preserving modified or foreign guidance' "$case_root/warnings"
cp "$case_root/clean-guidance" "$CODEX_HOME/AGENTS.md"
"$AGENT_TOOLKIT_HOME/scripts/uninstall.sh" >/dev/null
assert_contains 'User Codex instructions' "$CODEX_HOME/AGENTS.md"
assert_absent_text 'agent-toolkit:start' "$CODEX_HOME/AGENTS.md"

printf 'OK: managed guidance tests passed\n'
