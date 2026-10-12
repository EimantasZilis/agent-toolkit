#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/shared.sh"

AGENT_TOOLKIT_ENV_FILE="${AGENT_TOOLKIT_ENV_FILE:-$AGENT_TOOLKIT_HOME/.env}"

prompt_for_value() {
  local label=$1 default=$2 variable=$3 display_default answer
  display_default="${4:-$default}"
  printf '%s [%s]: ' "$label" "$display_default" >&2
  IFS= read -r answer || {
    echo "Error: unable to read $label." >&2
    return 1
  }
  printf -v "$variable" '%s' "${answer:-$default}"
}

configure_installation() {
  local config_file=$AGENT_TOOLKIT_ENV_FILE template="$AGENT_TOOLKIT_HOME/.env.template"
  local provider concise output

  if [[ ! -f "$config_file" ]]; then
    if [[ ! -t 0 || ! -t 1 ]]; then
      echo "Error: configuration is missing; run make install interactively to create .env." >&2
      return 1
    fi
    mkdir -p -- "$(dirname "$config_file")"
    cp -- "$template" "$config_file"
  fi

  if [[ -t 0 && -t 1 ]]; then
    provider="$(config_value "$config_file" LLM_PROVIDER 2>/dev/null || printf '%s' codex)"
    output="$(config_value "$config_file" MARKDOWN_OUTPUT_PATH 2>/dev/null || true)"
    concise="$(config_value "$config_file" CONCISE_OUTPUT 2>/dev/null || printf '%s' True)"
    [[ -n "$provider" ]] || provider=codex
    [[ -n "$concise" ]] || concise=True

    prompt_for_value "LLM provider (codex or claude)" "$provider" provider || return 1
    prompt_for_value "Markdown output path (where generated output Markdown files are stored; Enter a global path or leave blank for repo-local outputs)" "$output" output repo-local || return 1
    prompt_for_value "Concise output (whether to enable concise responses for all LLM outputs: True or False)" "$concise" concise || return 1

    case "$provider" in
      codex|claude) ;;
      *) echo "Error: invalid LLM provider '$provider'; expected codex or claude." >&2; return 1 ;;
    esac
    if [[ -n "$output" && "$output" != /* ]]; then
      echo "Error: Markdown output path must be blank or an absolute path." >&2
      return 1
    fi
    case "$(printf '%s' "$concise" | tr '[:upper:]' '[:lower:]')" in
      true|1|yes|on|false|0|no|off) ;;
      *) echo "Error: invalid concise output '$concise'; expected True or False." >&2; return 1 ;;
    esac

    persist_config_setting "$config_file" LLM_PROVIDER "$provider"
    persist_config_setting "$config_file" MARKDOWN_OUTPUT_PATH "$output"
    persist_config_setting "$config_file" CONCISE_OUTPUT "$concise"
  fi
}

configure_installation

# Installation provider and concise-output behavior are controlled by the
# repository's .env, even when the caller has exported values. The output
# location intentionally supports an environment override for CI.
unset LLM_PROVIDER
unset CONCISE_OUTPUT
load_config

SKILLS_DEST="$(provider_skills_dest)"
GUIDANCE_DEST="$(provider_guidance_dest)"
SOURCE_GUIDANCE="$(source_guidance)"
mkdir -p "$(dirname "$GUIDANCE_DEST")"

# Preflight every destination before changing any of them.
guidance_user_owned=no
if [[ -e "$GUIDANCE_DEST" || -L "$GUIDANCE_DEST" ]]; then
  if [[ -L "$GUIDANCE_DEST" && ( "$(resolved_path "$GUIDANCE_DEST")" == "$(cd "$(dirname "$SOURCE_GUIDANCE")" && pwd -P)/$(basename "$SOURCE_GUIDANCE")" || "$(resolved_path "$GUIDANCE_DEST")" == "$AGENT_TOOLKIT_HOME/.agents/global/AGENTS.md" ) ]]; then
    :
  elif [[ -f "$GUIDANCE_DEST" && ! -L "$GUIDANCE_DEST" ]]; then
    if [[ ! -f "$(owned_marker "$(dirname "$GUIDANCE_DEST")")" ]]; then
      guidance_user_owned=yes
    elif ! grep -qx "$AGENT_TOOLKIT_HOME" "$(owned_marker "$(dirname "$GUIDANCE_DEST")")"; then
      echo "Error: refusing to overwrite non-managed path: $GUIDANCE_DEST" >&2
      exit 1
    elif ! is_managed_guidance "$GUIDANCE_DEST"; then
      guidance_user_owned=yes
    fi
  elif [[ ! -f "$(owned_marker "$(dirname "$GUIDANCE_DEST")")" ]] || ! grep -qx "$AGENT_TOOLKIT_HOME" "$(owned_marker "$(dirname "$GUIDANCE_DEST")")"; then
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
done < <(source_skills)

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
  [[ "${transaction_active:-no}" == yes ]] || return 0
  transaction_active=no
  local i path slot mode state
  for ((i = backup_count - 1; i >= 0; i--)); do
    path="${backup_paths[$i]}"
    slot="${backup_slots[$i]}"
    mode="${backup_modes[$i]}"
    state="${backup_states[$i]}"
    [[ "$state" == pending ]] && continue
    remove_path "$path" 2>/dev/null || true
    if path_exists "$slot"; then
      if [[ "$mode" == copy ]]; then
        cp -p "$slot" "$path" 2>/dev/null || true
      else
        mv "$slot" "$path" 2>/dev/null || true
      fi
    fi
  done
  rm -rf "$transaction_root" 2>/dev/null || true
}
trap rollback_transaction EXIT
trap 'exit 130' INT TERM

backup_path "$GUIDANCE_DEST" "$([[ "$guidance_user_owned" == yes ]] && printf copy || printf move)"
backup_path "$(owned_marker "$(dirname "$GUIDANCE_DEST")")"
if [[ "$guidance_user_owned" == yes ]]; then
  append_guidance "$GUIDANCE_DEST"
else
  render_guidance "$GUIDANCE_DEST"
  printf '%s\nGUIDANCE.md %s\n' "$AGENT_TOOLKIT_HOME" "$(file_hash "$GUIDANCE_DEST")" > "$(owned_marker "$(dirname "$GUIDANCE_DEST")")"
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
done < <(source_skills)

trap - EXIT INT TERM
transaction_active=no
rm -rf "$transaction_root"
echo "Installed $skill_count $LLM_PROVIDER skills in $SKILLS_DEST and managed guidance at $GUIDANCE_DEST."
