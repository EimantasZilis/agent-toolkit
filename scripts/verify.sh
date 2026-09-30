#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/shared.sh"
"$SCRIPT_DIR/validate.sh"

errors=0
fail() { echo "ERROR: $*" >&2; errors=$((errors + 1)); }

check_installed_skill() {
  local source=$1 key dest
  key="$(skill_key "$source")"; dest="$SKILLS_DEST/$(basename "$source")"
  [[ -d "$dest" ]] || { fail "missing installed skill: $dest"; return; }
  if ! is_owned_dir "$dest"; then
    echo "WARNING: user-owned skill was not replaced: $dest" >&2
    return
  fi
  [[ -f "$dest/SKILL.md" ]] || fail "missing rendered manifest: $dest/SKILL.md"
  if [[ "$LLM_PROVIDER" == codex && -f "$AGENT_TOOLKIT_HOME/.agents/providers/codex/skills/$key/agents/openai.yaml" ]]; then
    [[ -L "$dest/agents/openai.yaml" && "$(resolved_path "$dest/agents/openai.yaml")" == "$(cd "$(dirname "$AGENT_TOOLKIT_HOME/.agents/providers/codex/skills/$key/agents/openai.yaml")" && pwd -P)/openai.yaml" ]] || fail "missing Codex metadata: $dest/agents/openai.yaml"
  fi
  if [[ "$LLM_PROVIDER" == claude ]] && grep -Fxq "$key" "$AGENT_TOOLKIT_HOME/.agents/providers/claude/explicit-only.txt"; then
    grep -q '^disable-model-invocation: true$' "$dest/SKILL.md" || fail "missing Claude explicit-only metadata: $dest/SKILL.md"
  fi
}

verify_provider() {
  local provider=$1 provider_root="$VERIFY_ROOT/$1" config_file="$VERIFY_ROOT/$1/.env"
  mkdir -p "$provider_root"
  printf 'LLM_PROVIDER=%s\nCONCISE_OUTPUT=True\n' "$provider" > "$config_file"

  AGENT_TOOLKIT_ENV_FILE="$config_file" \
    AGENTS_HOME="$provider_root/agents" \
    CODEX_HOME="$provider_root/codex" \
    CLAUDE_HOME="$provider_root/claude" \
    "$SCRIPT_DIR/install.sh" >/dev/null

  LLM_PROVIDER="$provider"
  AGENTS_HOME="$provider_root/agents"
  CODEX_HOME="$provider_root/codex"
  CLAUDE_HOME="$provider_root/claude"
  export LLM_PROVIDER AGENTS_HOME CODEX_HOME CLAUDE_HOME

  SKILLS_DEST="$(provider_skills_dest)"
  GUIDANCE_DEST="$(provider_guidance_dest)"
  while IFS= read -r source; do check_installed_skill "$source"; done < <(source_skills)
  [[ -f "$GUIDANCE_DEST" ]] || fail "missing managed guidance: $GUIDANCE_DEST"
  [[ -f "$(owned_marker "$(dirname "$GUIDANCE_DEST")")" ]] || fail "missing guidance ownership marker"
  echo "OK: $provider temporary installation is complete and owned paths verify"
}

VERIFY_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/agent-toolkit-verify.XXXXXX")"
cleanup() { rm -rf "$VERIFY_ROOT"; }
trap cleanup EXIT

verify_provider codex
verify_provider claude
[[ $errors -eq 0 ]] || exit 1
