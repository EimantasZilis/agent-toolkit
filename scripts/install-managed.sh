# Map the human CLI onto the same managed planner and transaction as the TUI.

installation_defaults() {
  local legacy=${AGENT_TOOLKIT_ENV_FILE:-$AGENT_TOOLKIT_HOME/.env}
  if [[ ! -f "$managed_settings_file" && ! -f "$legacy" && -z "$managed_providers" ]]; then
    managed_providers=codex
    managed_requested=$(managed_words "$(bundle_members starter)")
  fi
}

installation_prompt() {
  local selection=$managed_requested providers=$managed_providers output=$managed_output
  local full=$managed_full_template concise=$managed_concise
  prompt_for_value 'LLM providers (codex, claude, or both space-separated)' "$providers" providers
  prompt_for_value 'Skills (starter, all, or space-separated names)' "$selection" selection
  prompt_for_value 'Update global guidance (True or False)' "$full" full
  prompt_for_value 'Markdown output path (blank for repo-local)' "$output" output repo-local
  prompt_for_value 'Concise output loading (True or False)' "$concise" concise
  provider_option=$providers
  selection_option=$selection
  guidance_option=$(boolean_setting GLOBAL_GUIDANCE "$full")
  frontend_options[${#frontend_options[@]}]=--output-path
  frontend_options[${#frontend_options[@]}]=$output
  frontend_options[${#frontend_options[@]}]=--concise
  frontend_options[${#frontend_options[@]}]=$concise
}

installation_arguments() {
  local provider name
  installation_options=()
  for provider in $provider_option; do
    installation_options[${#installation_options[@]}]=--provider
    installation_options[${#installation_options[@]}]=$provider
  done
  if [[ -n "$selection_option" ]]; then
    if bundle_names | grep -Fxq -- "$selection_option"; then
      selection_option=$(bundle_members "$selection_option")
    fi
    for name in $selection_option; do
      installation_options[${#installation_options[@]}]=--skill
      installation_options[${#installation_options[@]}]=$name
    done
  fi
  if [[ -n "$guidance_option" ]]; then
    installation_options[${#installation_options[@]}]=--full-template
    installation_options[${#installation_options[@]}]=$guidance_option
    installation_options[${#installation_options[@]}]=--concise
    installation_options[${#installation_options[@]}]=$guidance_option
  fi
  if [[ ${#frontend_options[@]} -gt 0 ]]; then
    installation_options+=("${frontend_options[@]}")
  fi
}

installation_preview() {
  local i name role
  printf 'Providers: %s\n' "${managed_providers:-none}"
  printf 'Requested skills: %s\n' "${managed_requested:-none}"
  printf 'Full guidance: %s; concise loading: %s\n' "$managed_full_template" "$managed_concise"
  printf 'Saved output root: %s\n' "${managed_output:-repo-local}"
  for ((i = 0; i < ${#operation_actions[@]}; i++)); do
    name=${operation_names[$i]}
    role=previously_installed
    if [[ "$name" == guidance ]]; then
      role=guidance
    elif grep -Fxq -- "$name" <<< "$selected_names"; then
      role=$(managed_selection_role "$name")
    fi
    printf '  %s %s (%s; %s) -> %s\n' "${operation_providers[$i]}" "$name" "$role" \
      "${operation_actions[$i]}" "${operation_paths[$i]}"
  done
  printf 'Configuration: %s\n' "$managed_settings_file"
  printf 'Revision: %s\n' "$managed_state_revision"
}

installation_confirm() {
  local answer
  [[ "$managed_conflicts" -eq 0 ]] || {
    printf 'Error: resolve ownership conflicts before apply.\n' >&2
    return 4
  }
  [[ "$managed_removals" == true && "$confirm_removals" != true ]] || return 0
  if [[ -t 0 && -t 1 ]]; then
    printf 'Apply the removals listed above? [y/N]: ' >&2
    IFS= read -r answer || answer=''
    case "$answer" in y|Y|yes|YES) confirm_removals=true; return 0 ;; esac
    printf 'Canceled; no files changed.\n' >&2
    return 2
  fi
  printf 'Error: removals require --confirm-removals.\n' >&2
  return 2
}

installation_main() {
  local legacy=${AGENT_TOOLKIT_ENV_FILE:-$AGENT_TOOLKIT_HOME/.env}
  managed_load
  if [[ ! -f "$managed_settings_file" && -f "$legacy" && "$interactive_install" == false &&
    "$dry_run" == false && -z "${MARKDOWN_OUTPUT_PATH+x}" ]] &&
    ! config_value "$legacy" MARKDOWN_OUTPUT_PATH >/dev/null; then
    printf 'Error: MARKDOWN_OUTPUT_PATH is not configured; run setup interactively.\n' >&2
    return 1
  fi
  installation_defaults
  managed_previous_output=$managed_output
  managed_previous_concise=$managed_concise
  managed_previous_full_template=$managed_full_template
  if [[ "$interactive_install" == true ]]; then installation_prompt; fi
  installation_arguments
  if [[ ${#installation_options[@]} -gt 0 ]]; then
    managed_parse_options "${installation_options[@]}"
  else
    managed_parse_options
  fi
  managed_resolve_selection
  managed_plan
  installation_preview
  if [[ "$dry_run" == true ]]; then
    printf 'Dry run: no files changed.\n'
    return 0
  fi
  installation_confirm
  managed_confirm=$confirm_removals
  managed_expected_revision=$managed_state_revision
  managed_apply
  printf 'Installation refreshed. Start a new agent session.\n'
}
