#!/usr/bin/env bash
# Exercise state import and persistence in an isolated toolkit and provider homes.
set -euo pipefail

SOURCE_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
TEST_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/agent-toolkit-state.XXXXXX")
trap 'rm -rf -- "$TEST_ROOT"' EXIT
mkdir "$TEST_ROOT/toolkit"
cp -R "$SOURCE_ROOT/scripts" "$SOURCE_ROOT/.agents" "$SOURCE_ROOT/.env.template" \
  "$SOURCE_ROOT/.gitignore" "$TEST_ROOT/toolkit/"
export AGENT_TOOLKIT_HOME="$TEST_ROOT/toolkit"
export AGENTS_HOME="$TEST_ROOT/agents home" CODEX_HOME="$TEST_ROOT/codex home"
export CLAUDE_HOME="$TEST_ROOT/claude home" AGENT_TOOLKIT_ENV_FILE="$TEST_ROOT/custom.env"
unset MARKDOWN_OUTPUT_PATH

source "$AGENT_TOOLKIT_HOME/scripts/shared.sh"
source "$AGENT_TOOLKIT_HOME/scripts/install-options.sh"
SKILL_CATALOG=$(source_skills)
fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }
install_fixture() { "$AGENT_TOOLKIT_HOME/scripts/install.sh" "$@" >/dev/null; }

managed_load
[[ -z "$managed_providers$managed_requested$inventory_codex$inventory_claude" ]] || fail fresh
[[ ! -e "$AGENT_TOOLKIT_HOME/config" ]] || fail 'read created state directory'

install_fixture --provider codex --skill ticket-to-plan
install_fixture --provider claude --skill commit-message
managed_load
[[ "$managed_providers" == 'claude codex' ]] || fail providers
[[ "$inventory_codex" == 'plan-review ticket-to-plan' ]] || fail 'Codex actual inventory'
[[ "$inventory_claude" == commit-message ]] || fail 'Claude actual inventory'
[[ "$managed_requested" == 'commit-message ticket-to-plan' ]] || fail 'shared requested selection'
python3 -m json.tool "$managed_inventory_file" >/dev/null

# Import differing legacy installations without writes or silent pruning.
rm -rf -- "$AGENT_TOOLKIT_HOME/config"
printf 'LLM_PROVIDER=claude\nSKILL_SELECTION=commit-message\nMARKDOWN_OUTPUT_PATH=\nCONCISE_OUTPUT=False\n' > "$AGENT_TOOLKIT_ENV_FILE"
managed_load
[[ "$managed_providers" == 'claude codex' ]] || fail 'import dropped provider'
[[ "$managed_requested" == 'commit-message plan-review ticket-to-plan' ]] || fail 'import dropped skill/helper'
[[ ! -e "$managed_state_dir" ]] || fail 'import mutated state'

# Literal path round-trip: managed files are data, never sourced shell code.
mkdir "$managed_state_dir"
managed_output="$TEST_ROOT/output with spaces/\"quoted\"/\$(touch NEVER_EXECUTE)"
managed_retained=plan-review
managed_save
expected_output=$managed_output
managed_load
[[ "$managed_output" == "$expected_output" && "$managed_retained" == plan-review ]] || fail roundtrip
[[ ! -e NEVER_EXECUTE && ! -e "$TEST_ROOT/NEVER_EXECUTE" ]] || fail evaluation

cp "$managed_settings_file" "$TEST_ROOT/settings.before"
cp "$managed_inventory_file" "$TEST_ROOT/inventory.before"
cp "$AGENT_TOOLKIT_ENV_FILE" "$TEST_ROOT/legacy.before"
if AGENT_TOOLKIT_FAIL_STATE=after-settings install_fixture --provider codex --skill pr-comment 2> "$TEST_ROOT/error"; then
  fail 'expected partial persistence failure'
fi
cmp "$managed_settings_file" "$TEST_ROOT/settings.before" || fail 'settings not restored'
cmp "$managed_inventory_file" "$TEST_ROOT/inventory.before" || fail 'inventory not restored'
cmp "$AGENT_TOOLKIT_ENV_FILE" "$TEST_ROOT/legacy.before" || fail 'legacy settings not restored'
[[ ! -e "$AGENTS_HOME/skills/agent-toolkit/pr-comment" ]] || fail 'failed install survived'
[[ ! -e "$managed_state_dir/apply.lock" ]] || fail 'lock survived failure'

# Environment output overrides affect this invocation, not saved preferences.
MARKDOWN_OUTPUT_PATH="$TEST_ROOT/transient" install_fixture --provider codex --skill ticket-to-plan
managed_load
[[ -z "$managed_output" ]] || fail 'invocation override persisted'
grep -Fq "$TEST_ROOT/transient" "$AGENTS_HOME/skills/agent-toolkit/ticket-to-plan/SKILL.md" || fail override

(cd /private/tmp && install_fixture --provider claude --skill pr-description)
[[ -f "$AGENT_TOOLKIT_HOME/config/settings.env" ]] || fail 'state not anchored to toolkit'
git -C "$SOURCE_ROOT" check-ignore "$SOURCE_ROOT/config/settings.env" >/dev/null || fail 'state not ignored'

# A forged inventory entry cannot establish ownership or authorize deletion.
mkdir -p "$AGENTS_HOME/skills/agent-toolkit/user-skill"
printf 'user content\n' > "$AGENTS_HOME/skills/agent-toolkit/user-skill/SKILL.md"
inventory_codex=$(managed_words "$inventory_codex" user-skill)
managed_save
managed_load
[[ "$(managed_observed_skills codex)" != *user-skill* ]] || fail 'inventory conferred ownership'
grep -Fxq 'user content' "$AGENTS_HOME/skills/agent-toolkit/user-skill/SKILL.md" || fail 'user content changed'

printf 'OK: managed state tests passed\n'
