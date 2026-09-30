#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/shared.sh"
remove_skill() {
  local dest=$1
  is_owned_dir "$dest" || return 0
  local skill_hash expected_hash
  skill_hash="$(file_hash "$dest/SKILL.md" 2>/dev/null || true)"
  expected_hash="$(awk '$1 == "SKILL.md" { print $2 }' "$(owned_marker "$dest")")"
  [[ -n "$skill_hash" && "$skill_hash" == "$expected_hash" ]] && rm -f "$dest/SKILL.md"
  while IFS= read -r item; do
    [[ "$item" == "$(owned_marker "$dest")" ]] && continue
    [[ -L "$item" ]] && rm -f "$item"
  done < <(find "$dest" -type f -o -type l | sort -r)
  rm -f "$(owned_marker "$dest")"
  find "$dest" -type d -empty -delete
  [[ -d "$dest" ]] && rmdir "$dest" 2>/dev/null || true
  echo "Removed $dest"
}

for LLM_PROVIDER in codex claude; do
  SKILLS_DEST="$(provider_skills_dest)"
  GUIDANCE_DEST="$(provider_guidance_dest)"

  if [[ -d "$SKILLS_DEST" ]]; then
    while IFS= read -r source; do
      remove_skill "$SKILLS_DEST/$(basename "$source")"
    done < <(source_skills)
  fi

  # Remove the pre-1.0 Claude layout as well. Only owned skill directories
  # are removed, so user-created content under the legacy parent is preserved.
  LEGACY_SKILLS_DEST="$(provider_legacy_skills_dest)"
  if [[ -n "$LEGACY_SKILLS_DEST" && -d "$LEGACY_SKILLS_DEST" ]]; then
    while IFS= read -r source; do
      remove_skill "$LEGACY_SKILLS_DEST/$(basename "$source")"
    done < <(source_skills)
    rmdir "$LEGACY_SKILLS_DEST" 2>/dev/null || true
  fi

  marker="$(owned_marker "$(dirname "$GUIDANCE_DEST")")"
  if [[ -f "$marker" ]] && grep -qx "$AGENT_TOOLKIT_HOME" "$marker"; then
    guidance_hash="$(file_hash "$GUIDANCE_DEST" 2>/dev/null || true)"
    expected_hash="$(awk '$1 == "GUIDANCE.md" { print $2 }' "$marker")"
    [[ -n "$guidance_hash" && "$guidance_hash" == "$expected_hash" ]] && rm -f "$GUIDANCE_DEST"
    rm -f "$marker"
    [[ ! -e "$GUIDANCE_DEST" ]] && echo "Removed $GUIDANCE_DEST"
  elif [[ -f "$GUIDANCE_DEST" ]] && grep -Fxq "$agent_toolkit_start" "$GUIDANCE_DEST"; then
    strip_guidance_blocks "$GUIDANCE_DEST"
    echo "Removed agent-toolkit guidance from $GUIDANCE_DEST"
  elif [[ -f "$GUIDANCE_DEST" ]] && has_concise_output_block "$GUIDANCE_DEST"; then
    CONCISE_OUTPUT=false
    update_concise_output_block "$GUIDANCE_DEST"
    echo "Removed agent-toolkit concise-output guidance from $GUIDANCE_DEST"
  fi
done
