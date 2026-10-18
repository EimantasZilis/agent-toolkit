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

transaction_root="$(mktemp -d "${TMPDIR:-/tmp}/agent-toolkit-install.XXXXXX")"
transaction_active=yes
backup_count=0
backup_paths=()
backup_slots=()
backup_modes=()
backup_states=()

path_exists() { [[ -e "$1" || -L "$1" ]]; }
remove_path() {
  if [[ -d "$1" && ! -L "$1" ]]; then
    rm -rf "$1"
  else
    rm -f "$1"
  fi
}
backup_path() {
  local path=$1 mode=${2:-move} slot="$transaction_root/$backup_count"
  backup_paths[$backup_count]="$path"
  backup_slots[$backup_count]="$slot"
  backup_modes[$backup_count]="$mode"
  backup_states[$backup_count]=pending
  if path_exists "$path"; then
    if [[ "$mode" == copy ]]; then
      cp -p "$path" "$slot"
    else
      mv "$path" "$slot"
    fi
    backup_states[$backup_count]=present
  else
    backup_states[$backup_count]=absent
  fi
  backup_count=$((backup_count + 1))
}
rollback_transaction() {
  local status=$? restore_failed=no
  [[ "${transaction_active:-no}" == yes ]] || return 0
  transaction_active=no
  local i path slot mode state
  for ((i = backup_count - 1; i >= 0; i--)); do
    path="${backup_paths[$i]}"
    slot="${backup_slots[$i]}"
    mode="${backup_modes[$i]}"
    state="${backup_states[$i]}"
    [[ "$state" == pending ]] && continue
    if [[ "$state" == present ]]; then
      if ! path_exists "$slot" || ! remove_path "$path"; then
        restore_failed=yes
        continue
      fi
      if [[ "$mode" == copy ]]; then
        cp -p "$slot" "$path" || restore_failed=yes
      else
        mv "$slot" "$path" || restore_failed=yes
      fi
    else
      remove_path "$path" || restore_failed=yes
    fi
  done
  if [[ "$restore_failed" == yes ]]; then
    printf 'Error: rollback incomplete; recovery backups retained at %s\n' "$transaction_root" >&2
    [[ "$status" -ne 0 ]] || status=1
  else
    rm -rf -- "$transaction_root" || status=1
  fi
  return "$status"
}
trap rollback_transaction EXIT
trap 'exit 130' INT TERM

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

trap - EXIT INT TERM
transaction_active=no
rm -rf "$transaction_root"
printf 'Installed %s %s skills in %s.\n' "$skill_count" "$LLM_PROVIDER" "$SKILLS_DEST"
