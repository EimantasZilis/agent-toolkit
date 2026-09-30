#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/shared.sh"
load_config
"$SCRIPT_DIR/validate.sh"

errors=0
fail() { echo "ERROR: $*" >&2; errors=$((errors + 1)); }
SKILLS_DEST="$(provider_skills_dest)"
GUIDANCE_DEST="$(provider_guidance_dest)"

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

while IFS= read -r source; do check_installed_skill "$source"; done < <(source_skills)
[[ -f "$GUIDANCE_DEST" ]] || fail "missing managed guidance: $GUIDANCE_DEST"
[[ -f "$(owned_marker "$(dirname "$GUIDANCE_DEST")")" ]] || fail "missing guidance ownership marker"
[[ $errors -eq 0 ]] || exit 1
echo "OK: $LLM_PROVIDER installation is complete and owned paths verify"
