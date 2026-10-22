# Plan safe filesystem operations and apply them under one state transaction.

managed_output_parent() {
  local path=$managed_output
  [[ -n "$path" ]] || { printf 'Error: directory creation requires a global output path.\n' >&2; return 2; }
  while ! path_exists "$path"; do path=$(dirname "$path"); done
  [[ -d "$path" && -w "$path" && -x "$path" ]] || {
    printf 'Error: output path needs a writable, searchable directory: %s\n' "$path" >&2
    return 2
  }
}

managed_create_output_directory() {
  local path=$managed_output i
  local missing=()
  managed_output_parent
  while ! path_exists "$path"; do
    missing[${#missing[@]}]=$path
    path=$(dirname "$path")
  done
  for ((i = ${#missing[@]} - 1; i >= 0; i--)); do
    mkdir -- "${missing[$i]}"
    transaction_record_directory "${missing[$i]}"
  done
}

managed_fingerprint_path() {
  local path=$1 item target
  printf 'PATH %s\n' "$path"
  if [[ -L "$path" ]]; then
    readlink "$path" || return 1
  elif [[ -d "$path" ]]; then
    find "$path" -type f -exec shasum -a 256 {} + | LC_ALL=C sort || return 1
    while IFS= read -r item; do
      target=$(readlink "$item") || return 1
      printf 'LINK %s %s\n' "$item" "$target"
    done < <(find "$path" -type l | LC_ALL=C sort)
    find "$path" -type d | LC_ALL=C sort
  elif [[ -f "$path" ]]; then
    file_hash "$path"
  elif [[ -e "$path" ]]; then
    printf 'SPECIAL\n'
  else
    printf 'ABSENT\n'
  fi
}

managed_safe_provider_root() {
  local base path=$SKILLS_DEST
  case "$LLM_PROVIDER" in
    codex) base=${AGENTS_HOME:-$HOME/.agents} ;;
    claude) base=${CLAUDE_HOME:-$HOME/.claude} ;;
  esac
  base=${base%/}
  [[ -n "$base" ]] || base=/
  [[ "$base" == /* && "$base" != *$'\n'* && "$base" != *$'\r'* ]] || return 1
  while [[ "$path" != "$base" && "$path" != / ]]; do
    [[ ! -L "$path" ]] || return 1
    path=$(dirname "$path")
  done
}

managed_revision() (
  local provider root
  {
    for root in "$managed_settings_file" "$managed_inventory_file" \
      "${AGENT_TOOLKIT_ENV_FILE:-$AGENT_TOOLKIT_HOME/.env}" \
      "$AGENT_TOOLKIT_HOME/.agents" "$AGENT_TOOLKIT_HOME/scripts"; do
      managed_fingerprint_path "$root" || exit 1
    done
    if [[ -n "${managed_native_inventory:-}" ]]; then
      managed_fingerprint_path "$managed_native_inventory" || exit 1
    fi
    for provider in codex claude; do
      LLM_PROVIDER=$provider
      for root in "$(provider_skills_dest)" "$(provider_legacy_skills_dest)" \
        "$(provider_guidance_dest)" "$(owned_marker "$(dirname "$(provider_guidance_dest)")")"; do
        [[ -n "$root" ]] || continue
        managed_fingerprint_path "$root" || exit 1
      done
    done
  } | shasum -a 256 | awk '{ print $1 }'
)

managed_safe_skill() {
  local source=$1 destination=$2 key=$3 item rel expected
  if [[ -L "$destination" ]]; then
    link_points_to "$destination" "$source"
    return
  fi
  [[ ! -L "$destination/SKILL.md" && ! -L "$(owned_marker "$destination")" ]] || return 1
  is_managed_skill "$destination" || return 1
  [[ $(wc -l < "$(owned_marker "$destination")") -eq 2 ]] || return 1
  while IFS= read -r item; do
    rel=${item#"$destination/"}
    case "$rel" in
      SKILL.md|.agent-toolkit-owned) continue ;;
      agents/openai.yaml) expected=$(provider_skill_metadata "$key") ;;
      *) expected="$source/$rel" ;;
    esac
    if [[ -d "$item" && ! -L "$item" ]]; then
      [[ -d "$source/$rel" || ( "$rel" == agents && -f "$(provider_skill_metadata "$key")" ) ]] || return 1
    else
      [[ -n "$expected" && -f "$expected" && -L "$item" ]] || return 1
      link_points_to "$item" "$expected" || return 1
    fi
  done < <(find "$destination" -mindepth 1 | LC_ALL=C sort)
}

managed_resources_complete() {
  local source=$1 destination=$2 key=$3 file rel metadata
  while IFS= read -r file; do
    rel=${file#"$source/"}
    [[ -L "$destination/$rel" && -f "$destination/$rel" ]] || return 1
  done < <(find "$source" -type f ! -name SKILL.md)
  metadata=$(provider_skill_metadata "$key")
  [[ -z "$metadata" || ! -f "$metadata" || -L "$destination/agents/openai.yaml" ]]
}

managed_expected_manifest() {
  local explicit=no
  if provider_skill_is_explicit "$2"; then explicit=yes; fi
  render_skill_manifest "$1" "$explicit"
  print_skill_output_policy "$1"
}

managed_add_operation() {
  operation_providers[${#operation_providers[@]}]=$1
  operation_actions[${#operation_actions[@]}]=$2
  operation_names[${#operation_names[@]}]=$3
  operation_paths[${#operation_paths[@]}]=$4
  [[ "$2" != conflict ]] || managed_conflicts=$((managed_conflicts + 1))
  case "$2" in remove|remove-guidance|strip-guidance) managed_removals=true ;; esac
  return 0
}

managed_plan_selected() {
  local name source destination key expected current action
  while IFS= read -r name; do
    [[ -n "$name" ]] || continue
    source=$(skill_source "$name") || return 2
    key=$(skill_key "$source")
    destination="$SKILLS_DEST/$name"
    action=install
    if path_exists "$destination"; then
      if ! managed_safe_skill "$source" "$destination" "$key"; then
        action=conflict
      else
        action=update
        expected=$(managed_expected_manifest "$source" "$key" | shasum -a 256 | awk '{print $1}')
        current=$(file_hash "$destination/SKILL.md")
        if [[ ! -L "$destination" && "$expected" == "$current" ]] &&
          managed_resources_complete "$source" "$destination" "$key"; then
          action=keep
        fi
      fi
    fi
    managed_add_operation "$LLM_PROVIDER" "$action" "$name" "$destination"
  done <<< "$selected_names"
}

managed_plan_removals() {
  local root path name source key action
  for root in "$SKILLS_DEST" "$(provider_legacy_skills_dest)"; do
    [[ -n "$root" && -d "$root" && ! -L "$root" ]] || continue
    while IFS= read -r path; do
      name=${path##*/}
      if managed_contains "$managed_providers" "$LLM_PROVIDER" && [[ "$root" == "$SKILLS_DEST" ]] &&
        grep -Fxq -- "$name" <<< "$selected_names"; then
        continue
      fi
      # Inventory never grants ownership. Unknown user paths are left alone.
      if [[ -L "$path" ]]; then
        source=$(skill_source "$name" 2>/dev/null || true)
        [[ -n "$source" ]] && link_points_to "$path" "$source" || continue
      else
        is_owned_dir "$path" || continue
      fi
      action=conflict
      source=$(skill_source "$name" 2>/dev/null || true)
      if [[ -n "$source" ]]; then
        key=$(skill_key "$source")
        if managed_safe_skill "$source" "$path" "$key"; then action=remove; fi
      fi
      managed_add_operation "$LLM_PROVIDER" "$action" "$name" "$path"
    done < <(find "$root" -mindepth 1 -maxdepth 1 \( -type d -o -type l \) | LC_ALL=C sort)
  done
}

managed_plan() {
  local provider
  operation_providers=()
  operation_actions=()
  operation_names=()
  operation_paths=()
  managed_conflicts=0
  managed_removals=false
  managed_state_revision=$(managed_revision)
  for provider in codex claude; do
    LLM_PROVIDER=$provider
    SKILLS_DEST=$(provider_skills_dest)
    if ! managed_safe_provider_root; then
      managed_add_operation "$provider" conflict skills "$SKILLS_DEST"
      continue
    fi
    if managed_contains "$managed_providers" "$provider"; then
      managed_plan_selected
    fi
    managed_plan_removals
    managed_plan_guidance
  done
}

managed_apply_operation() {
  local action=$1 name=$2 path=$3 source key
  case "$action" in
    keep) return 0 ;;
    install|update|remove)
      source=$(skill_source "$name")
      key=$(skill_key "$source")
      if path_exists "$path" && ! managed_safe_skill "$source" "$path" "$key"; then
        printf 'Error: ownership changed since preview: %s\n' "$path" >&2
        return 4
      fi
      mkdir -p -- "$(dirname "$path")"
      backup_path "$path"
      if [[ "$action" != remove ]]; then render_skill "$source" "$path" "$key"; fi
      ;;
    write-guidance|remove-guidance|strip-guidance) managed_apply_guidance "$action" "$path" ;;
  esac
}

managed_apply() {
  local provider i
  [[ -n "$managed_expected_revision" ]] || { printf 'Error: apply requires --revision from preview.\n' >&2; return 2; }
  if [[ "$managed_expected_revision" != "$managed_state_revision" ]]; then
    printf 'Error: stale preview; request a new preview.\n' >&2
    return 3
  fi
  [[ "$managed_conflicts" -eq 0 ]] || { printf 'Error: resolve ownership conflicts before apply.\n' >&2; return 4; }
  if [[ "$managed_removals" == true && "$managed_confirm" != true ]]; then
    printf 'Error: removals require --confirm-removals.\n' >&2
    return 2
  fi
  transaction_begin
  managed_lock
  if [[ "$managed_expected_revision" != "$(managed_revision)" ]]; then
    printf 'Error: stale preview; request a new preview.\n' >&2
    return 3
  fi
  backup_path "$managed_settings_file" copy
  backup_path "$managed_inventory_file" copy
  if [[ "$managed_create_output" == true ]]; then managed_create_output_directory; fi
  for provider in codex claude; do
    LLM_PROVIDER=$provider
    SKILLS_DEST=$(provider_skills_dest)
    managed_safe_provider_root || { printf 'Error: unsafe provider destination.\n' >&2; return 4; }
    for ((i = 0; i < ${#operation_actions[@]}; i++)); do
      [[ "${operation_providers[$i]}" == "$provider" ]] || continue
      managed_apply_operation "${operation_actions[$i]}" "${operation_names[$i]}" "${operation_paths[$i]}"
    done
    if [[ "${AGENT_TOOLKIT_FAIL_PROVIDER:-}" == "$provider" ]]; then
      printf 'Error: test failure injected after provider %s.\n' "$provider" >&2
      return 97
    fi
    if [[ "${AGENT_TOOLKIT_INTERRUPT_PROVIDER:-}" == "$provider" ]]; then kill -TERM "$$"; fi
  done
  inventory_codex=$(managed_words "$(managed_observed_skills codex)")
  inventory_claude=$(managed_words "$(managed_observed_skills claude)")
  managed_save
  transaction_commit
}
