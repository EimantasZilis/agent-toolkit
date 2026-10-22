#!/usr/bin/env bash
# Install provider skills and guidance with rollback on failure.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/shared.sh"
source "$SCRIPT_DIR/install-options.sh"

AGENT_TOOLKIT_ENV_FILE="${AGENT_TOOLKIT_ENV_FILE:-$AGENT_TOOLKIT_HOME/.env}"

SKILL_CATALOG="$(source_skills)"
parse_install_options "$@"
if [[ "$help_only" == true ]]; then
  installation_usage
  exit 0
fi
if [[ "$list_only" == true ]]; then
  while IFS= read -r source; do printf '%s\n' "${source##*/}"; done <<< "$SKILL_CATALOG"
  exit 0
fi
read_installation_settings
validate_dependencies
resolve_selection

SKILLS_DEST="$(provider_skills_dest)"
GUIDANCE_DEST="$(provider_guidance_dest)"
SOURCE_GUIDANCE="$(source_guidance)"
preview_installation
if [[ "$dry_run" == true ]]; then
  printf 'Dry run: no files changed.\n'
  exit 0
fi
if [[ -L "$AGENT_TOOLKIT_ENV_FILE" || ( -e "$AGENT_TOOLKIT_ENV_FILE" && ! -f "$AGENT_TOOLKIT_ENV_FILE" ) ]]; then
  printf 'Error: configuration must be a regular file: %s\n' "$AGENT_TOOLKIT_ENV_FILE" >&2
  exit 1
fi

# Preflight every destination before changing any of them.
guidance_user_owned=no
if [[ "$GLOBAL_GUIDANCE" == true && ( -e "$GUIDANCE_DEST" || -L "$GUIDANCE_DEST" ) ]]; then
  if [[ -L "$GUIDANCE_DEST" && ( "$(resolved_path "$GUIDANCE_DEST")" == "$(cd "$(dirname "$SOURCE_GUIDANCE")" && pwd -P)/$(basename "$SOURCE_GUIDANCE")" || "$(resolved_path "$GUIDANCE_DEST")" == "$AGENT_TOOLKIT_HOME/.agents/global/AGENTS.md" ) ]]; then
    :
  elif [[ -f "$GUIDANCE_DEST" && ! -L "$GUIDANCE_DEST" ]]; then
    if [[ ! -f "$(owned_marker "$(dirname "$GUIDANCE_DEST")")" ]]; then
      guidance_user_owned=yes
    elif ! grep -Fxq -- "$AGENT_TOOLKIT_HOME" "$(owned_marker "$(dirname "$GUIDANCE_DEST")")"; then
      echo "Error: refusing to overwrite non-managed path: $GUIDANCE_DEST" >&2
      exit 1
    elif ! is_managed_guidance "$GUIDANCE_DEST"; then
      guidance_user_owned=yes
    fi
  else
    echo "Error: refusing to overwrite non-managed path: $GUIDANCE_DEST" >&2
    exit 1
  fi
fi

while IFS= read -r source; do
  key="$(skill_key "$source")"
  dest="$SKILLS_DEST/$(basename "$source")"
  if is_user_owned_skill "$source" "$dest"; then
    echo "Warning: skipping user-owned skill with the same name: $dest" >&2
  fi
done < <(selected_sources)

managed_state_paths
transaction_begin
managed_lock
managed_load
backup_path "$managed_settings_file" copy
backup_path "$managed_inventory_file" copy

if [[ "$GLOBAL_GUIDANCE" == true ]]; then
  mkdir -p "$(dirname "$GUIDANCE_DEST")"
  backup_path "$GUIDANCE_DEST" "$([[ "$guidance_user_owned" == yes ]] && printf copy || printf move)"
  backup_path "$(owned_marker "$(dirname "$GUIDANCE_DEST")")"
  if [[ "$guidance_user_owned" == yes ]]; then
    append_guidance "$GUIDANCE_DEST"
  else
    render_guidance "$GUIDANCE_DEST"
    printf '%s\nGUIDANCE.md %s\n' "$AGENT_TOOLKIT_HOME" "$(file_hash "$GUIDANCE_DEST")" > "$(owned_marker "$(dirname "$GUIDANCE_DEST")")"
  fi
fi

skill_count=0
while IFS= read -r source; do
  key="$(skill_key "$source")"
  dest="$SKILLS_DEST/$(basename "$source")"
  is_user_owned_skill "$source" "$dest" && continue
  backup_path "$dest"
  render_skill "$source" "$dest" "$key"
  skill_count=$((skill_count + 1))
  if [[ "${AGENT_TOOLKIT_FAIL_AFTER:-}" == "$skill_count" ]]; then
    echo "Error: test failure injected after skill $skill_count." >&2
    exit 97
  fi
done < <(selected_sources)

mkdir -p -- "$(dirname "$AGENT_TOOLKIT_ENV_FILE")"
backup_path "$AGENT_TOOLKIT_ENV_FILE" copy
save_installation_settings
managed_save_legacy

transaction_commit
printf 'Installed %s %s skills in %s.\n' "$skill_count" "$LLM_PROVIDER" "$SKILLS_DEST"
