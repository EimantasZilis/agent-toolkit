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

test_standalone_pr_skills_and_saved_choices() {
  local provider=$1 root="$TEST_ROOT/custom $1" skills output
  output="$(install_at "$root" --provider "$provider" --skill pr-description --skill pr-comment --skill pr-description)"
  skills="$(skills_at "$root" "$provider")"
  assert_skill_count "$skills" 2
  assert_absent "$skills/to-clipboard"
  [[ "$output" != *'(dependency;'* ]] || fail 'PR skills installed an unexpected helper'
  install_at "$root" >/dev/null
  assert_skill_count "$skills" 2
  install_at "$root" --skill commit-message >/dev/null
  assert_skill_count "$skills" 3
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

test_standalone_optional_integrations() {
  local provider=$1 name root skills output
  for name in developer-highlights next-commit next-phase review-branch security-check ruby-standards \
    plan-writing-conventions plan-execution-conventions; do
    root="$TEST_ROOT/standalone $provider $name"
    output="$(install_at "$root" --provider "$provider" --skill "$name")"
    skills="$(skills_at "$root" "$provider")"
    assert_skill_count "$skills" 1
    assert_file "$skills/$name/SKILL.md"
    [[ "$output" != *'(dependency;'* ]] || fail "$name installed an optional dependency"
  done
}

test_required_ruby_dependency() {
  local provider=$1 root="$TEST_ROOT/rails $1" skills output
  output="$(install_at "$root" --provider "$provider" --skill ruby-rails-standards)"
  skills="$(skills_at "$root" "$provider")"
  assert_skill_count "$skills" 2
  assert_file "$skills/ruby-standards/SKILL.md"
  assert_contains "](<$skills/ruby-standards/SKILL.md>)" "$skills/ruby-rails-standards/SKILL.md"
  [[ "$output" == *'ruby-standards (dependency; install/update)'* ]] || fail 'missing Ruby baseline'
  install_at "$root" --skill ruby-standards --skill ruby-rails-standards >/dev/null
  assert_skill_count "$skills" 2
}

test_optional_coinstallation_and_legacy_preservation() {
  local provider=$1 root="$TEST_ROOT/optional $1" skills output name
  skills="$(skills_at "$root" "$provider")"
  mkdir -p "$skills/plan-conventions"
  printf 'previously installed conventions\n' > "$skills/plan-conventions/SKILL.md"

  output="$(install_at "$root" --provider "$provider" --skill next-phase \
    --skill plan-execution-conventions --skill plan-writing-conventions --skill python-standards)"
  assert_skill_count "$skills" 5
  [[ "$output" != *'(dependency;'* ]] || fail 'optional integration became required'
  [[ "$(cat "$skills/plan-conventions/SKILL.md")" == 'previously installed conventions' ]] ||
    fail 'renaming conventions removed or changed the old installation'

  for name in plan-writing-conventions plan-execution-conventions; do
    assert_contains 'Markdown artifact discovery' "$skills/$name/SKILL.md"
    assert_contains 'No centralized output root is configured' "$skills/$name/SKILL.md"
  done
  if [[ "$provider" == claude ]]; then
    assert_contains 'disable-model-invocation: true' "$skills/plan-writing-conventions/SKILL.md"
    if grep -q '^disable-model-invocation:' "$skills/plan-execution-conventions/SKILL.md"; then
      fail 'new execution conventions unexpectedly became explicit-only'
    fi
  else
    assert_contains 'allow_implicit_invocation: false' "$skills/plan-writing-conventions/agents/openai.yaml"
  fi
  install_at "$root" --skill next-phase >/dev/null
  assert_skill_count "$skills" 5
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
  mkdir -p "$skills/ruby-standards"
  printf 'user baseline\n' > "$skills/ruby-standards/SKILL.md"
  output="$(install_at "$root" --provider claude --skill ruby-rails-standards 2>/dev/null)"
  [[ "$output" == *'ruby-standards (dependency; skip: user-owned)'* ]] || fail 'preview missed ownership conflict'
  [[ "$(cat "$skills/ruby-standards/SKILL.md")" == 'user baseline' ]] || fail 'user baseline overwritten'
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
    "$REPO_ROOT/scripts/install.sh" --skill ticket-to-plan > "$root/error" 2>&1; then
    fail 'expected settings save failure'
  fi
  [[ "$(cat "$root/settings.env")" == "$original" ]] || fail 'settings save failure lost original settings'
  assert_absent "$skills/ticket-to-plan"
  assert_absent "$skills/plan-review"
  assert_file "$skills/commit-message/SKILL.md"
}

for provider in codex claude; do
  test_fresh_starter "$provider"
  test_standalone_pr_skills_and_saved_choices "$provider"
  test_standalone_planning_and_metadata "$provider"
  test_guidance_opt_in_and_preservation "$provider"
  test_standalone_optional_integrations "$provider"
  test_required_ruby_dependency "$provider"
  test_optional_coinstallation_and_legacy_preservation "$provider"
done
test_read_only_options_and_validation
test_selective_rollback
test_user_owned_dependency
test_failed_settings_save_rolls_back_installation
printf 'OK: selective installation tests passed\n'
