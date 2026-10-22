#!/usr/bin/env bash
# Exercise reviewed reconciliation and rollback across disposable provider homes.
set -euo pipefail

SOURCE_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
TEST_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/agent-toolkit-reconcile.XXXXXX")
trap 'rm -rf -- "$TEST_ROOT"' EXIT
unset MARKDOWN_OUTPUT_PATH

fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }
fixture() {
  case_root="$TEST_ROOT/$1"
  mkdir -p "$case_root/toolkit"
  cp -R "$SOURCE_ROOT/scripts" "$SOURCE_ROOT/.agents" "$SOURCE_ROOT/.env.template" "$case_root/toolkit/"
  export AGENT_TOOLKIT_HOME="$case_root/toolkit"
  export AGENTS_HOME="$case_root/agents home" CODEX_HOME="$case_root/codex home" CLAUDE_HOME="$case_root/claude home"
  export AGENT_TOOLKIT_ENV_FILE="$case_root/legacy.env"
  codex_skills="$AGENTS_HOME/skills/agent-toolkit"
  claude_skills="$CLAUDE_HOME/skills"
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
  [[ "$status" -eq "$expected" ]] || fail "expected status $expected, got $status: $(cat "$case_root/error")"
}
snapshot() {
  find "$AGENTS_HOME" "$CODEX_HOME" "$CLAUDE_HOME" "$AGENT_TOOLKIT_HOME/config" \
    -type f -exec shasum -a 256 {} + 2>/dev/null | LC_ALL=C sort
  local link
  while IFS= read -r link; do printf '%s -> %s\n' "$link" "$(readlink "$link")"; done \
    < <(find "$AGENTS_HOME" "$CODEX_HOME" "$CLAUDE_HOME" -type l 2>/dev/null | LC_ALL=C sort)
}

fixture 'selection and helpers'
backend preview --provider codex --provider claude --skill ticket-to-plan >/dev/null
[[ ! -e "$AGENT_TOOLKIT_HOME/config" && ! -e "$AGENTS_HOME" ]] || fail 'preview wrote state/installations'
apply_current --provider codex --provider claude --skill ticket-to-plan
[[ -f "$codex_skills/plan-review/SKILL.md" && -f "$claude_skills/plan-review/SKILL.md" ]] || fail helpers
revision=$(preview_revision --skill commit-message)
expect_failure 2 backend apply --skill commit-message --revision "$revision"
[[ -f "$codex_skills/ticket-to-plan/SKILL.md" ]] || fail 'unconfirmed removal happened'
mkdir -p "$CODEX_HOME/plugins/native"
printf 'native content\n' > "$CODEX_HOME/plugins/native/keep"
apply_current --provider codex --skill commit-message
[[ ! -e "$codex_skills/ticket-to-plan" && -f "$codex_skills/plan-review/SKILL.md" ]] || fail retention
[[ ! -e "$claude_skills/ticket-to-plan" && ! -e "$claude_skills/plan-review" ]] || fail 'provider removal'
apply_current --remove-skill plan-review
[[ ! -e "$codex_skills/plan-review" ]] || fail 'explicit unused helper removal'
apply_current --skill ticket-to-plan
expect_failure 2 backend preview --remove-skill plan-review
grep -q 'selected skills require plan-review' "$case_root/error" || fail 'required helper diagnostic'
mkdir "$codex_skills/user-skill"
printf 'user content\n' > "$codex_skills/user-skill/SKILL.md"
sed 's/"codex": \[/"codex": ["user-skill", /' "$AGENT_TOOLKIT_HOME/config/inventory.json" > "$case_root/forged.json"
mv "$case_root/forged.json" "$AGENT_TOOLKIT_HOME/config/inventory.json"
apply_current --no-providers
[[ ! -e "$codex_skills/ticket-to-plan" && ! -e "$codex_skills/plan-review" ]] || fail 'last provider removal'
grep -Fxq 'native content' "$CODEX_HOME/plugins/native/keep" || fail 'native plugin changed'
grep -Fxq 'user content' "$codex_skills/user-skill/SKILL.md" || fail 'inventory granted deletion rights'

fixture conflicts
apply_current --provider codex --skill ruby-rails-standards
cp "$codex_skills/ruby-standards/SKILL.md" "$case_root/original"
printf '\nUser modification\n' >> "$codex_skills/ruby-standards/SKILL.md"
revision=$(preview_revision --no-skills)
expect_failure 4 backend apply --no-skills --revision "$revision" --confirm-removals
[[ -f "$codex_skills/ruby-rails-standards/SKILL.md" ]] || fail 'conflict partially removed parent'
cp "$case_root/original" "$codex_skills/ruby-standards/SKILL.md"
printf 'user note\n' > "$codex_skills/ruby-rails-standards/NOTES"
revision=$(preview_revision --no-skills)
expect_failure 4 backend apply --no-skills --revision "$revision" --confirm-removals
grep -Fxq 'user note' "$codex_skills/ruby-rails-standards/NOTES" || fail 'unknown file changed'
rm "$codex_skills/ruby-rails-standards/NOTES"
ln -s "$case_root/user-target" "$codex_skills/ruby-rails-standards/unknown-link"
revision=$(preview_revision --no-skills)
expect_failure 4 backend apply --no-skills --revision "$revision" --confirm-removals
[[ -L "$codex_skills/ruby-rails-standards/unknown-link" ]] || fail 'unknown link removed'

fixture 'stale and lock'
mkdir -p "$AGENTS_HOME" "$CODEX_HOME" "$CLAUDE_HOME"
apply_current --provider codex --provider claude --skill commit-message
cp "$codex_skills/commit-message/SKILL.md" "$case_root/original"
revision=$(preview_revision --skill pr-comment)
printf 'changed after preview\n' >> "$codex_skills/commit-message/SKILL.md"
expect_failure 3 backend apply --skill pr-comment --revision "$revision" --confirm-removals
[[ ! -e "$codex_skills/pr-comment" ]] || fail 'stale preview applied'
cp "$case_root/original" "$codex_skills/commit-message/SKILL.md"
revision=$(preview_revision --skill pr-comment)
mkdir "$AGENT_TOOLKIT_HOME/config/apply.lock"
expect_failure 1 backend apply --skill pr-comment --revision "$revision" --confirm-removals
[[ -d "$AGENT_TOOLKIT_HOME/config/apply.lock" ]] || fail 'foreign lock removed'
rmdir "$AGENT_TOOLKIT_HOME/config/apply.lock"

fixture 'rollback both providers'
mkdir -p "$AGENTS_HOME" "$CODEX_HOME" "$CLAUDE_HOME"
apply_current --provider codex --provider claude --skill commit-message
snapshot > "$case_root/before"
for provider in codex claude; do
  revision=$(preview_revision --skill pr-comment)
  expect_failure 97 env AGENT_TOOLKIT_FAIL_PROVIDER="$provider" \
    "$AGENT_TOOLKIT_HOME/scripts/managed.sh" apply --skill pr-comment --revision "$revision" --confirm-removals
  snapshot > "$case_root/after"
  cmp "$case_root/before" "$case_root/after" || fail "rollback after $provider"
done
revision=$(preview_revision --skill pr-comment)
expect_failure 1 env AGENT_TOOLKIT_FAIL_STATE=after-settings \
  "$AGENT_TOOLKIT_HOME/scripts/managed.sh" apply --skill pr-comment --revision "$revision" --confirm-removals
snapshot > "$case_root/after"
cmp "$case_root/before" "$case_root/after" || fail 'settings failure rollback'
revision=$(preview_revision --skill pr-comment)
expect_failure 143 env AGENT_TOOLKIT_INTERRUPT_PROVIDER=claude \
  "$AGENT_TOOLKIT_HOME/scripts/managed.sh" apply --skill pr-comment --revision "$revision" --confirm-removals
snapshot > "$case_root/after"
cmp "$case_root/before" "$case_root/after" || fail 'interruption rollback'

mkdir "$case_root/bin" "$case_root/transactions"
cp "$codex_skills/commit-message/SKILL.md" "$case_root/original"
cat > "$case_root/bin/mv" <<'EOF'
#!/usr/bin/env bash
source_path=$1
destination_path=$2
if [[ "$1" == -- ]]; then source_path=$2; destination_path=$3; fi
if [[ "$source_path" == */agent-toolkit-install.*/* && "$destination_path" == "$AGENTS_HOME/skills/agent-toolkit/commit-message" ]]; then
  printf '%s\n' "$source_path" > "$AGENT_TOOLKIT_TEST_RECORD"
  exit 23
fi
exec /bin/mv "$@"
EOF
chmod +x "$case_root/bin/mv"
revision=$(preview_revision --skill pr-comment)
expect_failure 97 env PATH="$case_root/bin:$PATH" TMPDIR="$case_root/transactions" \
  AGENT_TOOLKIT_TEST_RECORD="$case_root/recovery-path" AGENT_TOOLKIT_FAIL_PROVIDER=claude \
  "$AGENT_TOOLKIT_HOME/scripts/managed.sh" apply --skill pr-comment --revision "$revision" --confirm-removals
grep -q 'rollback incomplete; recovery backups retained at' "$case_root/error" || fail recovery
backup=$(cat "$case_root/recovery-path")
cmp "$case_root/original" "$backup/SKILL.md" || fail 'recovery backup lost original content'

# Existing user guidance and owned full guidance are removed independently.
fixture guidance
mkdir -p "$CODEX_HOME"
printf 'User guidance\n' > "$CODEX_HOME/AGENTS.md"
"$AGENT_TOOLKIT_HOME/scripts/install.sh" --provider codex --skill commit-message --global-guidance >/dev/null
"$AGENT_TOOLKIT_HOME/scripts/install.sh" --provider claude --skill commit-message --global-guidance >/dev/null
apply_current --no-providers
grep -Fxq 'User guidance' "$CODEX_HOME/AGENTS.md" || fail 'user guidance lost'
if grep -q 'agent-toolkit:start' "$CODEX_HOME/AGENTS.md"; then fail 'toolkit guidance survived removal'; fi
[[ ! -e "$CLAUDE_HOME/CLAUDE.md" ]] || fail 'owned guidance survived removal'

fixture 'symlinked toolkit'
ln -s "$AGENT_TOOLKIT_HOME" "$case_root/toolkit alias"
export AGENT_TOOLKIT_HOME="$case_root/toolkit alias"
apply_current --provider codex --provider claude --skill ruby-rails-standards
conflicts=$(backend preview | python3 -c 'import json,sys; print(len(json.load(sys.stdin)["conflicts"]))')
[[ "$conflicts" == 0 ]] || fail 'equivalent resource paths conflicted'
apply_current --no-providers
[[ ! -e "$codex_skills/ruby-rails-standards" && ! -e "$claude_skills/ruby-standards" ]] || fail 'aliased toolkit removal'

fixture 'legacy links through an alias'
ln -s "$AGENT_TOOLKIT_HOME" "$case_root/toolkit alias"
export AGENT_TOOLKIT_HOME="$case_root/toolkit alias"
mkdir -p "$codex_skills" "$claude_skills/agent-toolkit" "$CODEX_HOME" "$case_root/foreign"
ln -s "$AGENT_TOOLKIT_HOME/.agents/skills/productivity/commit-message" "$codex_skills/commit-message"
ln -s "$AGENT_TOOLKIT_HOME/.agents/skills/review/plan-review" "$claude_skills/agent-toolkit/plan-review"
ln -s "$AGENT_TOOLKIT_HOME/.agents/global/instructions.md" "$CODEX_HOME/AGENTS.md"
cp "$AGENT_TOOLKIT_HOME/.agents/skills/productivity/pr-comment/SKILL.md" "$case_root/foreign/SKILL.md"
ln -s "$case_root/foreign" "$codex_skills/pr-comment"
requested=$(backend preview | python3 -c 'import json,sys; print(" ".join(json.load(sys.stdin)["desired"]["requested_skills"]))')
[[ "$requested" == 'commit-message plan-review' ]] || fail 'legacy source links were not imported'
apply_current --no-providers
[[ ! -L "$codex_skills/commit-message" && ! -L "$claude_skills/agent-toolkit/plan-review" ]] || fail 'legacy source links survived'
[[ ! -L "$CODEX_HOME/AGENTS.md" ]] || fail 'legacy guidance link survived'
[[ -f "$AGENT_TOOLKIT_HOME/.agents/skills/productivity/commit-message/SKILL.md" ]] || fail 'source skill was removed'
[[ -f "$AGENT_TOOLKIT_HOME/.agents/global/instructions.md" ]] || fail 'source guidance was removed'
[[ -L "$codex_skills/pr-comment" && -f "$case_root/foreign/SKILL.md" ]] || fail 'unrelated link changed'

printf 'OK: managed reconciliation tests passed\n'
