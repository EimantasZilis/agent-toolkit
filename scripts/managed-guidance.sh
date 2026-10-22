# Remove unchanged managed guidance when a provider is deselected.

managed_guidance_block() {
  awk -v start="$agent_toolkit_start" -v end="$agent_toolkit_end" '
    $0 == start { inside = 1 }
    inside { print }
    $0 == end { inside = 0 }
  ' "$1"
}

managed_plan_guidance() {
  local destination marker action
  managed_contains "$managed_providers" "$LLM_PROVIDER" && return 0
  destination=$(provider_guidance_dest)
  marker=$(owned_marker "$(dirname "$destination")")
  if link_points_to "$destination" "$(source_guidance)" ||
    link_points_to "$destination" "$AGENT_TOOLKIT_HOME/.agents/global/AGENTS.md"; then
    managed_add_operation "$LLM_PROVIDER" remove-guidance guidance "$destination"
  elif is_managed_guidance "$destination" && [[ ! -L "$destination" && ! -L "$marker" ]]; then
    managed_add_operation "$LLM_PROVIDER" remove-guidance guidance "$destination"
  elif [[ -f "$destination" && ! -L "$destination" ]] && grep -Fxq -- "$agent_toolkit_start" "$destination"; then
    action=conflict
    if path_exists "$marker" && { [[ -L "$marker" ]] || ! grep -Fxq -- "$AGENT_TOOLKIT_HOME" "$marker"; }; then
      managed_add_operation "$LLM_PROVIDER" conflict guidance "$destination"
      return 0
    fi
    # Appended legacy blocks lack a marker. Compare with their imported saved
    # settings before removing the block, preserving all surrounding content.
    local temp
    temp=$(mktemp "${TMPDIR:-/tmp}/agent-toolkit-guidance.XXXXXX")
    if (CONCISE_OUTPUT=$managed_previous_concise MARKDOWN_OUTPUT_PATH=$managed_previous_output render_guidance "$temp"); then
      if [[ "$(managed_guidance_block "$destination")" == "$(cat "$temp")" ]]; then action=strip-guidance; fi
    fi
    rm -f -- "$temp"
    managed_add_operation "$LLM_PROVIDER" "$action" guidance "$destination"
  fi
}

managed_apply_guidance() {
  local action=$1 destination=$2 marker
  marker=$(owned_marker "$(dirname "$destination")")
  if [[ "$action" == strip-guidance ]]; then
    backup_path "$destination" copy
    strip_guidance_blocks "$destination"
  else
    backup_path "$destination"
  fi
  if [[ -f "$marker" && ! -L "$marker" ]] && grep -Fxq -- "$AGENT_TOOLKIT_HOME" "$marker"; then
    backup_path "$marker"
  fi
}
