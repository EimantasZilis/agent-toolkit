# Manage verified guidance blocks without claiming surrounding user content.

managed_print_guidance() (
  local temp
  if [[ "$managed_full_template" == true ]]; then
    temp=$(mktemp "${TMPDIR:-/tmp}/agent-toolkit-guidance.XXXXXX")
    trap 'rm -f -- "$temp"' EXIT
    CONCISE_OUTPUT=$managed_concise render_guidance "$temp" || return 1
    cat "$temp"
    return
  fi
  [[ -n "$MARKDOWN_OUTPUT_PATH" || "$managed_concise" == true ]] || return 0
  printf '%s\n%s\n' "$agent_toolkit_start" "$agent_toolkit_notice"
  if [[ -n "$MARKDOWN_OUTPUT_PATH" ]]; then
    printf '\n## Markdown output\n\n'
    printf 'Use `%s/<repo-slug>/<branch-slug>/<skill-name>/...` for Markdown artifacts unless the user supplies a path.\n' "$MARKDOWN_OUTPUT_PATH"
    printf 'Keep each invocation under its skill directory and do not overwrite existing artifacts without explicit direction.\n'
  fi
  if [[ "$managed_concise" == true ]]; then
    printf '\n'
    print_concise_output_block
  fi
  printf '\n%s\n' "$agent_toolkit_end"
)

managed_validate_guidance() {
  local destination=$1 marker line block='' ending='' trusted_hash='' hash
  marker=$(owned_marker "$(dirname "$destination")")
  guidance_block_count=0
  guidance_block_hash=''
  if path_exists "$marker"; then
    [[ -f "$marker" && ! -L "$marker" ]] || return 1
    grep -Fxq -- "$AGENT_TOOLKIT_HOME" "$marker" || return 1
    [[ $(wc -l < "$marker") -eq 2 ]] || return 1
    trusted_hash=$(awk '$1 == "GUIDANCE_BLOCK" { print $2 }' "$marker")
    [[ "$trusted_hash" =~ ^[a-f0-9]{64}$ ]] || return 1
  fi
  while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ -z "$ending" ]]; then
      case "$line" in
        "$agent_toolkit_start") ending=$agent_toolkit_end ;;
        "$concise_output_start") ending=$concise_output_end ;;
        "$agent_toolkit_end"|"$concise_output_end") return 1 ;;
        *) continue ;;
      esac
    elif [[ "$line" == "$agent_toolkit_start" ||
      ( "$ending" == "$concise_output_end" && "$line" == "$concise_output_start" ) ]]; then
      return 1
    fi
    block+="$line"$'\n'
    [[ "$line" == "$ending" ]] || continue
    hash=$(printf '%s' "$block" | shasum -a 256 | awk '{print $1}')
    [[ -n "$trusted_hash" && "$hash" == "$trusted_hash" ]] || return 1
    guidance_block_count=$((guidance_block_count + 1))
    guidance_block_hash=$hash
    block=''
    ending=''
  done < "$destination"
  [[ -z "$ending" ]] || return 1
  [[ ! -e "$marker" || "$guidance_block_count" -gt 0 ]]
}

managed_plan_guidance() {
  local destination marker desired='' desired_hash action
  destination=$(provider_guidance_dest)
  marker=$(owned_marker "$(dirname "$destination")")
  if managed_contains "$managed_providers" "$LLM_PROVIDER"; then desired=$(managed_print_guidance) || return 1; fi
  if ! path_exists "$destination"; then
    if path_exists "$marker"; then
      action=conflict
    elif [[ -n "$desired" ]]; then
      action=write-guidance
    else
      return 0
    fi
  elif [[ ! -f "$destination" || -L "$destination" ]]; then
    [[ -n "$desired" ]] || return 0
    action=conflict
  elif ! managed_validate_guidance "$destination"; then
    if [[ -z "$desired" ]] && ! grep -Eq '^<!-- agent-toolkit:(start|concise-output:start) -->$' "$destination" &&
      ! { [[ -f "$marker" ]] && grep -Fxq -- "$AGENT_TOOLKIT_HOME" "$marker"; }; then
      return 0
    fi
    action=conflict
  elif [[ -z "$desired" ]]; then
    [[ "$guidance_block_count" -gt 0 ]] || return 0
    action=remove-guidance
    if print_guidance_without_blocks "$destination" | awk 'NF { found = 1 } END { exit !found }'; then action=strip-guidance; fi
  else
    desired_hash=$(printf '%s\n' "$desired" | shasum -a 256 | awk '{print $1}')
    if [[ "$guidance_block_count" -eq 1 && "$guidance_block_hash" == "$desired_hash" ]]; then
      return 0
    fi
    action=write-guidance
  fi
  managed_add_operation "$LLM_PROVIDER" "$action" guidance "$destination"
}

managed_apply_guidance() {
  local action=$1 destination=$2 marker block
  marker=$(owned_marker "$(dirname "$destination")")
  mkdir -p -- "$(dirname "$destination")"
  if [[ -f "$destination" && ! -L "$destination" && "$action" != remove-guidance ]]; then
    backup_path "$destination" copy
    strip_guidance_blocks "$destination"
  else
    backup_path "$destination"
  fi
  backup_path "$marker"
  [[ "$action" == write-guidance ]] || return 0
  block=$(managed_print_guidance)
  if [[ -s "$destination" ]]; then printf '\n' >> "$destination"; fi
  printf '%s\n' "$block" >> "$destination"
  printf '%s\nGUIDANCE_BLOCK %s\n' "$AGENT_TOOLKIT_HOME" \
    "$(printf '%s\n' "$block" | shasum -a 256 | awk '{print $1}')" > "$marker"
}
