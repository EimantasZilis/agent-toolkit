# Versioned desired settings and observed inventories; never source saved data.

managed_state_paths() {
  managed_state_dir="$AGENT_TOOLKIT_HOME/config"
  managed_settings_file="$managed_state_dir/settings.env"
  managed_inventory_file="$managed_state_dir/inventory.json"
}

managed_words() {
  printf '%s\n' "$@" | tr ' ' '\n' | awk 'NF' | LC_ALL=C sort -u | paste -sd ' ' -
}

managed_contains() { [[ " $1 " == *" $2 "* ]]; }

managed_setting() {
  awk -v key="$1=" 'index($0, key) == 1 { print substr($0, length(key) + 1); exit }' "$managed_settings_file"
}

managed_validate_settings() {
  local name provider
  for provider in $managed_providers; do
    case "$provider" in codex|claude) ;; *) printf 'Error: invalid managed provider.\n' >&2; return 1 ;; esac
  done
  for name in $managed_requested $managed_retained; do
    [[ "$name" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]] || {
      printf 'Error: invalid managed skill identifier.\n' >&2
      return 1
    }
  done
  if [[ ( -n "$managed_output" && "$managed_output" != /* ) ||
    "$managed_output" == *$'\n'* || "$managed_output" == *$'\r'* ]]; then
    printf 'Error: output path must be an absolute single-line path or blank.\n' >&2
    return 1
  fi
}

managed_regular_or_absent() {
  if [[ -L "$1" || ( -e "$1" && ! -f "$1" ) ]]; then
    printf 'Error: state must be a regular file: %s\n' "$1" >&2
    return 1
  fi
}

managed_check_state_paths() {
  [[ ! -L "$managed_state_dir" ]] || {
    printf 'Error: state directory must not be a symlink: %s\n' "$managed_state_dir" >&2
    return 1
  }
  managed_regular_or_absent "$managed_settings_file"
  managed_regular_or_absent "$managed_inventory_file"
}

managed_lock() {
  managed_check_state_paths
  mkdir -p -- "$managed_state_dir"
  if ! mkdir -- "$managed_state_dir/apply.lock" 2>/dev/null; then
    printf 'Error: another apply is active; lock: %s/apply.lock\n' "$managed_state_dir" >&2
    return 1
  fi
  managed_lock_held=yes
}

managed_observed_skills() (
  local provider=$1 root path name source
  LLM_PROVIDER=$provider
  for root in "$(provider_skills_dest)" "$(provider_legacy_skills_dest)"; do
    [[ -n "$root" && -d "$root" ]] || continue
    while IFS= read -r path; do
      name=${path##*/}
      [[ "$name" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]] || continue
      if [[ -L "$path" ]]; then
        source="$(skill_source "$name" 2>/dev/null || true)"
        [[ -n "$source" && "$(resolved_path "$path" 2>/dev/null)" == "$source" ]] || continue
      else
        is_owned_dir "$path" || continue
      fi
      printf '%s\n' "$name"
    done < <(find "$root" -mindepth 1 -maxdepth 1 \( -type d -o -type l \) | LC_ALL=C sort)
  done
)

managed_read_inventory() {
  local line provider names encoded regex seen='' names_regex
  inventory_codex=''
  inventory_claude=''
  [[ -f "$managed_inventory_file" ]] || return 0
  grep -Fxq '  "schema_version": 1,' "$managed_inventory_file" || {
    printf 'Error: unsupported managed inventory version.\n' >&2
    return 1
  }
  # Only controlled provider/skill identifiers are parsed here, never paths or
  # arbitrary JSON strings. Inventory is written in this versioned row format.
  regex='^    "(codex|claude)": \[(.*)\],?$'
  names_regex='^("[a-z0-9]+(-[a-z0-9]+)*"(, "[a-z0-9]+(-[a-z0-9]+)*")*)?$'
  while IFS= read -r line; do
    [[ "$line" =~ $regex ]] || continue
    provider=${BASH_REMATCH[1]}
    encoded=${BASH_REMATCH[2]}
    if managed_contains "$seen" "$provider" || [[ ! "$encoded" =~ $names_regex ]]; then
      printf 'Error: malformed managed inventory.\n' >&2
      return 1
    fi
    seen+=" $provider"
    names=$(printf '%s' "$encoded" | tr '", ' '   ')
    local name
    for name in $names; do
      [[ "$name" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]] || {
        printf 'Error: invalid inventory skill.\n' >&2
        return 1
      }
    done
    case "$provider" in
      codex) inventory_codex=$(managed_words "$names") ;;
      claude) inventory_claude=$(managed_words "$names") ;;
    esac
  done < "$managed_inventory_file"
  [[ "$(managed_words "$seen")" == 'claude codex' ]] || {
    printf 'Error: incomplete managed inventory.\n' >&2
    return 1
  }
}

managed_load() {
  local legacy=${AGENT_TOOLKIT_ENV_FILE:-$AGENT_TOOLKIT_HOME/.env} provider selection installed
  managed_state_paths
  managed_check_state_paths
  managed_read_inventory
  managed_providers=''
  managed_requested=''
  managed_retained=''
  managed_output=''
  managed_concise=false
  managed_full_template=false

  if [[ -f "$managed_settings_file" ]]; then
    [[ "$(managed_setting STATE_VERSION)" == 1 ]] || {
      printf 'Error: unsupported managed settings version.\n' >&2
      return 1
    }
    managed_providers=$(managed_setting PROVIDERS)
    managed_requested=$(managed_setting REQUESTED_SKILLS)
    managed_retained=$(managed_setting RETAINED_HELPERS)
    managed_output=$(managed_setting MARKDOWN_OUTPUT_PATH)
    managed_concise=$(boolean_setting CONCISE_OUTPUT "$(managed_setting CONCISE_OUTPUT)")
    managed_full_template=$(boolean_setting GLOBAL_GUIDANCE "$(managed_setting GLOBAL_GUIDANCE)")
    managed_validate_settings
    return 0
  fi

  if [[ -f "$legacy" ]]; then
    provider=$(config_value "$legacy" LLM_PROVIDER || true)
    case "$provider" in codex|claude) managed_providers=$provider ;; esac
    selection=$(config_value "$legacy" SKILL_SELECTION || printf all)
    if bundle_names | grep -Fxq -- "$selection"; then
      managed_requested=$(managed_words "$(bundle_members "$selection")")
    else
      managed_requested=$(managed_words "$selection")
    fi
    managed_output=$(config_value "$legacy" MARKDOWN_OUTPUT_PATH || true)
    managed_concise=$(boolean_setting CONCISE_OUTPUT "$(config_value "$legacy" CONCISE_OUTPUT || printf True)")
    managed_full_template=$(boolean_setting GLOBAL_GUIDANCE "$(config_value "$legacy" GLOBAL_GUIDANCE || printf True)")
  fi
  for provider in codex claude; do
    installed=$(managed_words "$(managed_observed_skills "$provider")")
    case "$provider" in
      codex) inventory_codex=$installed ;;
      claude) inventory_claude=$installed ;;
    esac
    [[ -n "$installed" ]] || continue
    managed_providers=$(managed_words "$managed_providers" "$provider")
    managed_requested=$(managed_words "$managed_requested" "$installed")
  done
  managed_validate_settings
}

managed_json_array() {
  local name separator=''
  printf '['
  for name in $1; do
    printf '%s"%s"' "$separator" "$name"
    separator=', '
  done
  printf ']'
}

managed_save() {
  local settings_temp inventory_temp
  managed_validate_settings
  settings_temp=$(mktemp "$managed_state_dir/settings.XXXXXX")
  inventory_temp=$(mktemp "$managed_state_dir/inventory.XXXXXX")
  {
    printf 'STATE_VERSION=1\nPROVIDERS=%s\nREQUESTED_SKILLS=%s\nRETAINED_HELPERS=%s\n' \
      "$managed_providers" "$managed_requested" "$managed_retained"
    printf 'MARKDOWN_OUTPUT_PATH=%s\nCONCISE_OUTPUT=%s\nGLOBAL_GUIDANCE=%s\n' \
      "$managed_output" "$managed_concise" "$managed_full_template"
  } > "$settings_temp"
  {
    printf '{\n  "schema_version": 1,\n  "revision": "%s",\n  "providers": {\n' "$(file_hash "$settings_temp")"
    printf '    "codex": '; managed_json_array "$inventory_codex"; printf ',\n'
    printf '    "claude": '; managed_json_array "$inventory_claude"; printf '\n  }\n}\n'
  } > "$inventory_temp"
  if ! mv -- "$settings_temp" "$managed_settings_file" ||
    [[ "${AGENT_TOOLKIT_FAIL_STATE:-}" == after-settings ]] ||
    ! mv -- "$inventory_temp" "$managed_inventory_file"; then
    rm -f -- "$settings_temp" "$inventory_temp"
    printf 'Error: unable to persist managed settings and inventory.\n' >&2
    return 1
  fi
}

managed_save_legacy() {
  local provider=$LLM_PROVIDER
  managed_providers=$(managed_words "$managed_providers" "$provider")
  managed_requested=$(managed_words "$managed_requested" "$requested_names")
  managed_output=$configured_output
  managed_concise=$CONCISE_OUTPUT
  managed_full_template=$GLOBAL_GUIDANCE
  case "$provider" in
    codex) inventory_codex=$(managed_words "$(managed_observed_skills codex)") ;;
    claude) inventory_claude=$(managed_words "$(managed_observed_skills claude)") ;;
  esac
  managed_save
}
