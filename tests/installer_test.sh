#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/agent-toolkit-tests.XXXXXX")"
trap 'rm -rf "$TEST_ROOT"' EXIT

# State belongs to the toolkit root, so test against a disposable toolkit too.
mkdir "$TEST_ROOT/toolkit"
cp -R "$REPO_ROOT/scripts" "$REPO_ROOT/.agents" "$REPO_ROOT/.env.template" "$TEST_ROOT/toolkit/"
REPO_ROOT="$TEST_ROOT/toolkit"
export AGENT_TOOLKIT_HOME="$REPO_ROOT"

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
  [[ -L "$root/agents/skills/agent-toolkit/python-standards/references/core.md" ]] ||
    fail "expected Python standards core reference symlink"
  [[ -L "$root/agents/skills/agent-toolkit/python-standards/references/tests.md" ]] ||
    fail "expected Python standards tests reference symlink"
  [[ -L "$root/agents/skills/agent-toolkit/python-standards/scripts/check_function_size.py" ]] ||
    fail "expected Python standards helper symlink"
  assert_file "$root/codex/AGENTS.md"
  assert_file "$root/codex/.agent-toolkit-owned"
  [[ -L "$root/agents/skills/agent-toolkit/next-commit/agents/openai.yaml" ]] ||
    fail "expected Codex metadata symlink"
  [[ -L "$root/agents/skills/agent-toolkit/plan-review/agents/openai.yaml" ]] ||
    fail "expected moved plan-review metadata symlink"
  assert_contains 'allow_implicit_invocation: false' \
    "$root/agents/skills/agent-toolkit/plan-review/agents/openai.yaml"
}

test_claude_install() {
  local root="$TEST_ROOT/claude"
  run_install claude "$root" >/dev/null
  assert_directory "$root/claude/skills"
  [[ -L "$root/claude/skills/python-standards/references/core.md" ]] ||
    fail "expected Claude Python standards core reference symlink"
  assert_file "$root/claude/CLAUDE.md"
  assert_contains 'invoke the /concise-output skill' "$root/claude/CLAUDE.md"
  assert_contains 'Treat its' "$root/claude/CLAUDE.md"
  assert_contains 'standing guidance for all reader-facing responses' \
    "$root/claude/CLAUDE.md"
  assert_contains 'disable-model-invocation: true' \
    "$root/claude/skills/next-commit/SKILL.md"
  assert_contains 'disable-model-invocation: true' \
    "$root/claude/skills/plan-review/SKILL.md"
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

test_output_setting_preserves_backslashes() {
  local root="$TEST_ROOT/literal-output" value='/output/with\backslashes\n'
  mkdir -p "$root"
  printf 'MARKDOWN_OUTPUT_PATH=\n' > "$root/config.env"
  AGENT_TOOLKIT_HOME="$REPO_ROOT" bash -c \
    'source "$AGENT_TOOLKIT_HOME/scripts/config.sh"; persist_output_setting "$1" "$2"' \
    _ "$root/config.env" "$value"
  [[ "$(cat "$root/config.env")" == "MARKDOWN_OUTPUT_PATH=$value" ]] ||
    fail 'configuration update changed literal backslashes'
}

test_failed_config_update_preserves_original() {
  local root="$TEST_ROOT/config-failure" status
  mkdir -p "$root/bin"
  printf 'LLM_PROVIDER=codex\n' > "$root/config.env"
  printf '#!/usr/bin/env bash\nexit 23\n' > "$root/bin/awk"
  chmod +x "$root/bin/awk"
  if PATH="$root/bin:$PATH" AGENT_TOOLKIT_HOME="$REPO_ROOT" bash -c \
    'source "$AGENT_TOOLKIT_HOME/scripts/config.sh"; persist_output_setting "$1" /output' \
    _ "$root/config.env"; then
    fail 'expected configuration rewrite failure'
  else
    status=$?
  fi
  [[ "$status" -eq 1 ]] || fail "unexpected configuration failure status: $status"
  [[ "$(cat "$root/config.env")" == LLM_PROVIDER=codex ]] ||
    fail 'failed configuration rewrite replaced the original'
  [[ -z "$(find "$root" -name 'config.env.*' -print)" ]] ||
    fail 'failed configuration rewrite left temporary files'
}

test_ownership_requires_literal_repository_path() {
  local root="$TEST_ROOT/literal.ownership"
  mkdir -p "$root/skill" "$root/guidance"
  printf '%s\n' "${root/./X}" > "$root/skill/.agent-toolkit-owned"
  cp "$root/skill/.agent-toolkit-owned" "$root/guidance/.agent-toolkit-owned"
  printf 'guidance\n' > "$root/guidance/AGENTS.md"
  AGENT_TOOLKIT_HOME="$root" bash -c '
    source "$1/scripts/paths.sh"
    printf "GUIDANCE.md %s\n" "$(file_hash "$2/guidance/AGENTS.md")" >> "$2/guidance/.agent-toolkit-owned"
    if is_owned_dir "$2/skill"; then exit 1; fi
    if is_managed_guidance "$2/guidance/AGENTS.md"; then exit 1; fi
  ' _ "$REPO_ROOT" "$root" || fail 'ownership accepted a regex match for the repository path'
}

test_unrelated_guidance_symlink_is_preserved() {
  local root="$TEST_ROOT/guidance-symlink" output status
  mkdir -p "$root/codex"
  printf 'user guidance\n' > "$root/user.md"
  ln -s "$root/user.md" "$root/codex/AGENTS.md"
  printf '%s\n' "$REPO_ROOT" > "$root/codex/.agent-toolkit-owned"
  if output="$(run_install codex "$root" 2>&1)"; then
    fail 'expected installation to reject an unrelated guidance symlink'
  else
    status=$?
  fi
  [[ "$status" -eq 1 ]] || fail "unexpected symlink rejection status: $status"
  [[ "$output" == *'refusing to overwrite non-managed path'* ]] ||
    fail "missing symlink rejection diagnostic: $output"
  [[ -L "$root/codex/AGENTS.md" ]] || fail 'unrelated guidance symlink was replaced'
  [[ "$(cat "$root/user.md")" == 'user guidance' ]] || fail 'symlink target was changed'
  assert_absent "$root/agents"
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

test_failed_rollback_retains_recovery_backup() {
  local root="$TEST_ROOT/failed-rollback" output status
  run_install codex "$root" >/dev/null
  cp "$root/codex/AGENTS.md" "$root/original.md"
  mkdir -p "$root/bin" "$root/transactions"
  cat > "$root/bin/mv" <<'EOF'
#!/usr/bin/env bash
source_path=$1
destination_path=$2
if [[ "$1" == -- ]]; then
  source_path=$2
  destination_path=$3
fi
if [[ "$source_path" == */agent-toolkit-install.*/* && "$destination_path" == "$AGENT_TOOLKIT_TEST_GUIDANCE_PATH" ]]; then
  printf '%s\n' "$source_path" > "$AGENT_TOOLKIT_TEST_BACKUP_RECORD"
  printf 'injected restore failure\n' >&2
  exit 23
fi
exec /bin/mv "$@"
EOF
  chmod +x "$root/bin/mv"
  if output="$(PATH="$root/bin:$PATH" TMPDIR="$root/transactions" \
    AGENT_TOOLKIT_TEST_GUIDANCE_PATH="$root/codex/AGENTS.md" \
    AGENT_TOOLKIT_TEST_BACKUP_RECORD="$root/backup-path" \
    run_install codex "$root" 1 2>&1)"; then
    fail 'expected injected installation failure'
  else
    status=$?
  fi
  [[ "$status" -eq 97 ]] || fail "rollback changed installation failure status: $status"
  [[ "$output" == *'rollback incomplete; recovery backups retained at '* ]] ||
    fail "missing recovery diagnostic: $output"
  local backup
  backup="$(cat "$root/backup-path")"
  assert_file "$backup"
  cmp -s "$root/original.md" "$backup" || fail 'original guidance backup was lost'
}

test_codex_install
test_claude_install
test_repo_local_output_policy
test_external_output_policy
test_environment_output_override
test_missing_output_setting_requires_interactive_configuration
test_blank_output_setting_persists_exactly
test_output_setting_preserves_backslashes
test_failed_config_update_preserves_original
test_ownership_requires_literal_repository_path
test_unrelated_guidance_symlink_is_preserved
test_changed_managed_guidance_is_preserved
test_changed_managed_skill_is_preserved
test_user_owned_content_is_preserved
test_reinstall_and_uninstall
test_rollback
test_failed_rollback_retains_recovery_backup
echo "OK: installer integration tests passed"
