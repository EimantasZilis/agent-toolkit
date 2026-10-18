#!/usr/bin/env bash
# Exercise selective installation in isolated provider homes.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/agent-toolkit-selection.XXXXXX")"
trap 'rm -rf -- "$TEST_ROOT"' EXIT
unset MARKDOWN_OUTPUT_PATH

fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }
assert_file() { [[ -f "$1" ]] || fail "missing file: $1"; }
assert_absent() { [[ ! -e "$1" && ! -L "$1" ]] || fail "unexpected path: $1"; }
assert_contains() { grep -Fq -- "$1" "$2" || fail "missing '$1' in $2"; }

install_at() {
  local root=$1
  shift
  AGENT_TOOLKIT_ENV_FILE="$root/settings.env" \
    AGENTS_HOME="$root/agents" CODEX_HOME="$root/codex" CLAUDE_HOME="$root/claude" \
    "$REPO_ROOT/scripts/install.sh" "$@"
}

skills_at() {
  case "$2" in
    codex) printf '%s/agents/skills/agent-toolkit\n' "$1" ;;
    claude) printf '%s/claude/skills\n' "$1" ;;
  esac
}

assert_skill_count() {
  local count
  count="$(find "$1" -name SKILL.md -type f | wc -l | tr -d ' ')"
  [[ "$count" == "$2" ]] || fail "expected $2 installed skills, got $count"
}

test_fresh_starter() {
  local provider=$1 root="$TEST_ROOT/starter $1" skills
  install_at "$root" --provider "$provider" > "$TEST_ROOT/preview-$provider"
  skills="$(skills_at "$root" "$provider")"
  assert_file "$skills/concise-output/SKILL.md"
  assert_file "$skills/commit-message/SKILL.md"
  assert_skill_count "$skills" 2
  assert_absent "$root/codex/AGENTS.md"
  assert_absent "$root/claude/CLAUDE.md"
  assert_absent "$root/codex/.agent-toolkit-owned"
  assert_absent "$root/claude/.agent-toolkit-owned"
  assert_contains 'SKILL_SELECTION=starter' "$root/settings.env"
  assert_contains 'GLOBAL_GUIDANCE=false' "$root/settings.env"
  assert_contains 'Global guidance: unchanged' "$TEST_ROOT/preview-$provider"
  [[ "$(cat "$skills/concise-output/SKILL.md")" == "$(cat "$REPO_ROOT/.agents/skills/productivity/concise-output/SKILL.md")" ]] ||
    fail 'starter gained unrelated standing instructions'
}

test_custom_dependencies_and_saved_choices() {
  local provider=$1 root="$TEST_ROOT/custom $1" skills output
  output="$(install_at "$root" --provider "$provider" --skill pr-description --skill pr-comment --skill pr-description)"
  skills="$(skills_at "$root" "$provider")"
  assert_skill_count "$skills" 3
  assert_file "$skills/to-clipboard/SKILL.md"
  [[ "$output" == *'to-clipboard (dependency; install/update)'* ]] || fail 'preview omitted helper'
  if [[ "$provider" == claude ]]; then
    assert_contains 'invoke `/to-clipboard`' "$skills/pr-description/SKILL.md"
  else
    assert_contains 'invoke `$to-clipboard`' "$skills/pr-description/SKILL.md"
  fi
  install_at "$root" >/dev/null
  assert_skill_count "$skills" 3
  install_at "$root" --skill commit-message >/dev/null
  assert_skill_count "$skills" 4
  assert_file "$skills/pr-description/SKILL.md"
  assert_contains 'SKILL_SELECTION=commit-message' "$root/settings.env"
}

test_standalone_planning_and_metadata() {
  local provider=$1 root="$TEST_ROOT/planning $1" skills file
  MARKDOWN_OUTPUT_PATH="$TEST_ROOT/output with spaces" \
    install_at "$root" --provider "$provider" --skill ticket-to-plan >/dev/null
  skills="$(skills_at "$root" "$provider")"
  assert_skill_count "$skills" 2
  assert_file "$skills/plan-review/SKILL.md"
  assert_contains "](<$skills/plan-review/SKILL.md>)" "$skills/ticket-to-plan/SKILL.md"
  assert_contains "](<$REPO_ROOT/.agents/skills/scoping/quick-plan/SKILL.md>)" "$skills/ticket-to-plan/SKILL.md"
  for file in "$skills/ticket-to-plan/SKILL.md" "$skills/plan-review/SKILL.md"; do
    assert_contains "$TEST_ROOT/output with spaces/<repo-slug>/<branch-slug>/<skill-name>/..." "$file"
    assert_contains 'explicit path' "$file"
    assert_contains 'detached-<short-sha>' "$file"
    assert_contains 'Markdown artifact discovery' "$file"
    if grep -Fq 'At the start of every session' "$file"; then fail 'global rule leaked into skill'; fi
  done
  if [[ "$provider" == claude ]]; then
    assert_contains 'disable-model-invocation: true' "$skills/plan-review/SKILL.md"
  else
    assert_contains 'allow_implicit_invocation: false' "$skills/plan-review/agents/openai.yaml"
  fi
  install_at "$root" >/dev/null
  assert_contains 'No centralized output root is configured' "$skills/plan-review/SKILL.md"
  [[ "$(grep -Fc '## Markdown artifact outputs' "$skills/plan-review/SKILL.md")" == 1 ]] ||
    fail 'reinstall duplicated output rules'
}

test_guidance_opt_in_and_preservation() {
  local provider=$1 root="$TEST_ROOT/guidance $1" skills guidance
  install_at "$root" --provider "$provider" --skill commit-message --global-guidance >/dev/null
  skills="$(skills_at "$root" "$provider")"
  case "$provider" in
    codex) guidance="$root/codex/AGENTS.md" ;;
    claude) guidance="$root/claude/CLAUDE.md" ;;
  esac
  assert_file "$skills/concise-output/SKILL.md"
  assert_file "$guidance"
  cp "$guidance" "$root/original-guidance"
  cp "$(dirname "$guidance")/.agent-toolkit-owned" "$root/original-marker"
  install_at "$root" --no-global-guidance >/dev/null
  cmp -s "$root/original-guidance" "$guidance" || fail 'disabled guidance was rewritten'
  cmp -s "$root/original-marker" "$(dirname "$guidance")/.agent-toolkit-owned" || fail 'guidance marker changed'
  # No guidance preflight should run when the path belongs to the user.
  mkdir -p "$root/symlink-home"
  printf 'user instructions\n' > "$root/user.md"
  ln -s "$root/user.md" "$root/symlink-home/AGENTS.md"
  AGENT_TOOLKIT_ENV_FILE="$root/settings.env" AGENTS_HOME="$root/agents" \
    CODEX_HOME="$root/symlink-home" CLAUDE_HOME="$root/claude" \
    "$REPO_ROOT/scripts/install.sh" --provider codex --no-global-guidance >/dev/null
  [[ -L "$root/symlink-home/AGENTS.md" ]] || fail 'disabled guidance symlink changed'
  [[ "$(cat "$root/user.md")" == 'user instructions' ]] || fail 'disabled guidance target changed'
}

test_dependency_closure() {
  local root="$TEST_ROOT/closure" output
  output="$(install_at "$root" --provider codex --skill next-phase --dry-run)"
  [[ "$output" == *'plan-conventions (dependency;'* && "$output" == *'security-review (dependency;'* &&
    "$output" == *'ruby-rails-standards (dependency;'* ]] || fail 'missing transitive or conditional dependency'
  [[ "$output" != *'review-branch ('* && "$output" != *'pr-description ('* ]] || fail 'installed a suggested next step'
  assert_absent "$root"
}

test_read_only_options_and_validation() {
  local root="$TEST_ROOT/read-only" output original
  output="$(install_at "$root" --list)"
  [[ "$output" == *'commit-message'* && "$output" == *'ticket-to-plan'* ]] || fail 'list omitted skills'
  install_at "$root" --provider claude --bundle all --dry-run >/dev/null
  assert_absent "$root"
  mkdir -p "$root"
  printf 'LLM_PROVIDER=claude\nSKILL_SELECTION=starter\nGLOBAL_GUIDANCE=False\nMARKDOWN_OUTPUT_PATH=\n' > "$root/settings.env"
  original="$(cat "$root/settings.env")"
  output="$(install_at "$root" --provider codex --skill pr-description --dry-run)"
  [[ "$output" == *'Provider: codex'* ]] || fail 'CLI provider did not override saved provider'
  [[ "$(cat "$root/settings.env")" == "$original" ]] || fail 'dry-run changed settings'
  assert_absent "$root/agents"
  if install_at "$root" --skill missing-skill > "$root/error" 2>&1; then fail 'accepted unknown skill'; fi
  if install_at "$root" --skill all > "$root/error" 2>&1; then fail 'accepted bundle as a skill name'; fi
  if install_at "$root" --bundle starter --skill commit-message > "$root/error" 2>&1; then fail 'accepted mixed selection'; fi
  if install_at "$root" --provider invalid > "$root/error" 2>&1; then fail 'accepted invalid provider'; fi
  if install_at "$root" --bundle missing > "$root/error" 2>&1; then fail 'accepted invalid bundle'; fi
  if install_at "$root" --skill > "$root/error" 2>&1; then fail 'accepted missing argument'; fi
  if install_at "$root" --unknown > "$root/error" 2>&1; then fail 'accepted unknown option'; fi
  [[ "$(cat "$root/settings.env")" == "$original" ]] || fail 'invalid input changed settings'
  assert_absent "$root/claude"
}

test_selective_rollback() {
  local root="$TEST_ROOT/rollback" skills output status original
  install_at "$root" --provider claude >/dev/null
  skills="$(skills_at "$root" claude)"
  original="$(cat "$root/settings.env")"
  cp "$skills/commit-message/SKILL.md" "$root/original-skill"
  if output="$(AGENT_TOOLKIT_FAIL_AFTER=1 install_at "$root" --skill pr-description 2>&1)"; then
    fail 'expected injected failure'
  else
    status=$?
  fi
  [[ "$status" == 97 && "$output" == *'test failure injected'* ]] || fail 'unexpected failure result'
  [[ "$(cat "$root/settings.env")" == "$original" ]] || fail 'failed install saved choices'
  cmp -s "$root/original-skill" "$skills/commit-message/SKILL.md" || fail 'existing skill changed'
  assert_absent "$skills/pr-description"
  AGENT_TOOLKIT_ENV_FILE="$root/settings.env" AGENTS_HOME="$root/agents" \
    CODEX_HOME="$root/codex" CLAUDE_HOME="$root/claude" \
    "$REPO_ROOT/scripts/uninstall.sh" >/dev/null
  assert_absent "$skills/commit-message"
  assert_absent "$root/settings.env"
}

test_user_owned_dependency() {
  local root="$TEST_ROOT/user-owned" skills output
  skills="$(skills_at "$root" claude)"
  mkdir -p "$skills/to-clipboard"
  printf 'user helper\n' > "$skills/to-clipboard/SKILL.md"
  output="$(install_at "$root" --provider claude --skill pr-description 2>/dev/null)"
  [[ "$output" == *'to-clipboard (dependency; skip: user-owned)'* ]] || fail 'preview missed ownership conflict'
  [[ "$(cat "$skills/to-clipboard/SKILL.md")" == 'user helper' ]] || fail 'user helper overwritten'
}

test_failed_settings_save_rolls_back_installation() {
  local root="$TEST_ROOT/settings failure" skills original
  install_at "$root" --provider claude >/dev/null
  skills="$(skills_at "$root" claude)"
  original="$(cat "$root/settings.env")"
  mkdir -p "$root/bin"
  cat > "$root/bin/mv" <<'EOF'
#!/usr/bin/env bash
printf 'injected settings rename failure\n' >&2
exit 23
EOF
  chmod +x "$root/bin/mv"
  if PATH="$root/bin:$PATH" \
    AGENT_TOOLKIT_ENV_FILE="$root/settings.env" AGENTS_HOME="$root/agents" \
    CODEX_HOME="$root/codex" CLAUDE_HOME="$root/claude" \
    "$REPO_ROOT/scripts/install.sh" --skill pr-description > "$root/error" 2>&1; then
    fail 'expected settings save failure'
  fi
  [[ "$(cat "$root/settings.env")" == "$original" ]] || fail 'settings save failure lost original settings'
  assert_absent "$skills/pr-description"
  assert_absent "$skills/to-clipboard"
  assert_file "$skills/commit-message/SKILL.md"
}

for provider in codex claude; do
  test_fresh_starter "$provider"
  test_custom_dependencies_and_saved_choices "$provider"
  test_standalone_planning_and_metadata "$provider"
  test_guidance_opt_in_and_preservation "$provider"
done
test_dependency_closure
test_read_only_options_and_validation
test_selective_rollback
test_user_owned_dependency
test_failed_settings_save_rolls_back_installation
printf 'OK: selective installation tests passed\n'
