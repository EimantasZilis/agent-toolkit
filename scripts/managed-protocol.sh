# Protocol v1 JSON output and optional supported provider-inventory exports.

managed_json_string() {
  local value=$1 character replacement code
  value=${value//\\/\\\\}
  value=${value//\"/\\\"}
  for ((code = 1; code < 32; code++)); do
    printf -v character '\\%03o' "$code"
    printf -v character '%b' "$character"
    printf -v replacement '\\u%04x' "$code"
    value=${value//"$character"/"$replacement"}
  done
  printf '"%s"' "$value"
}

managed_protocol_options() {
  managed_arguments=()
  managed_native_inventory=''
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --protocol)
        [[ $# -ge 2 ]] || { printf 'Error: --protocol requires a version.\n' >&2; return 2; }
        [[ "$2" == 1 ]] || { printf 'Error: unsupported protocol version.\n' >&2; return 5; }
        shift 2
        ;;
      --native-inventory)
        [[ $# -ge 2 && -f "$2" && ! -L "$2" ]] || {
          printf 'Error: --native-inventory requires a regular exported file.\n' >&2
          return 2
        }
        managed_native_inventory=$2
        shift 2
        ;;
      *) managed_arguments[${#managed_arguments[@]}]=$1; shift ;;
    esac
  done
}

managed_protocol_failure() {
  local status=$1 code message
  [[ "$status" -ne 0 ]] || return 0
  case "$status" in
    2) code=invalid_arguments; message='Check arguments and removal confirmation.' ;;
    3) code=stale_revision; message='Request a new preview before applying.' ;;
    4) code=ownership_conflict; message='Resolve reported ownership conflicts.' ;;
    5) code=unsupported_protocol; message='Use protocol version 1.' ;;
    130|143) code=interrupted; message='Apply was interrupted; inspect rollback diagnostics.' ;;
    *) code=operation_failed; message='Operation failed; inspect stderr and any recovery backups.' ;;
  esac
  printf '{"protocol_version":1,"ok":false,"error":{"code":"%s","exit_code":%s,"message":' "$code" "$status"
  managed_json_string "$message"
  printf '}}\n'
}

managed_emit_desired() {
  printf '"desired":{"providers":'; managed_json_array "$managed_providers"
  printf ',"requested_skills":'; managed_json_array "$managed_requested"
  printf ',"retained_helpers":'; managed_json_array "$managed_retained"
  printf ',"output_path":'; managed_json_string "$managed_output"
  printf ',"concise_output":%s,"full_template":%s}' "$managed_concise" "$managed_full_template"
  printf ',"effective_output_path":'; managed_json_string "${MARKDOWN_OUTPUT_PATH-$managed_output}"
}

managed_emit_inventory() {
  printf '"applied":{"codex":'; managed_json_array "$inventory_codex"
  printf ',"claude":'; managed_json_array "$inventory_claude"
  printf '},"observed":{"codex":'; managed_json_array "$(managed_observed_skills codex)"
  printf ',"claude":'; managed_json_array "$(managed_observed_skills claude)"
  printf '}'
}

managed_native_skills() (
  local line package member inside=false regex
  selected_names=''
  regex='^[[:space:]]*"pluginId":[[:space:]]*"([a-z0-9]+(-[a-z0-9]+)*)@agent-toolkit",?[[:space:]]*$'
  while IFS= read -r line; do
    [[ ! "$line" =~ ^[[:space:]]*\"available\": ]] || break
    if [[ "$line" =~ ^[[:space:]]*\"installed\":[[:space:]]*\[$ ]]; then inside=true; fi
    [[ "$inside" == true && "$line" =~ $regex ]] || continue
    package=${BASH_REMATCH[1]}
    if [[ "$package" == bundle-* ]]; then
      package=${package#bundle-}
      bundle_names | grep -Fxq -- "$package" || continue
      for member in $(bundle_members "$package"); do include_skill "$member" || return 1; done
    elif skill_source "$package" >/dev/null 2>&1; then
      include_skill "$package" || return 1
    fi
  done < "$managed_native_inventory"
  managed_words "$selected_names"
)

managed_emit_native_notice() {
  local names='' overlaps='' name status=unknown
  if [[ -n "$managed_native_inventory" ]] &&
    grep -Eq '^[[:space:]]*"installed":[[:space:]]*\[$' "$managed_native_inventory"; then
    status=reported
    names=$(managed_native_skills)
    for name in $names; do
      if grep -Fxq -- "$name" <<< "${selected_names:-}"; then overlaps=$(managed_words "$overlaps" "$name"); fi
    done
  fi
  printf '"native_overlap":{"status":"%s","skills":' "$status"
  managed_json_array "$names"
  printf ',"overlaps":'; managed_json_array "$overlaps"
  printf ',"message":'
  if [[ -n "$overlaps" ]]; then
    managed_json_string 'Reported native packages overlap with this selection; manage them in the provider.'
  else
    managed_json_string 'Native plugins are managed separately; overlapping skills may appear more than once.'
  fi
  printf '}'
}

managed_selection_role() {
  if managed_contains "$managed_requested" "$1"; then printf requested
  elif [[ "$1" == concise-output && "$managed_concise" == true ]]; then printf automatic
  elif managed_contains "$managed_retained" "$1"; then printf retained
  else printf required; fi
}

managed_emit_operations() {
  local filter=${1:-all} i separator='' role
  printf '['
  for ((i = 0; i < ${#operation_actions[@]}; i++)); do
    [[ "$filter" == all || "${operation_actions[$i]}" == conflict ]] || continue
    role=previously_installed
    if [[ "${operation_names[$i]}" == guidance ]]; then role=guidance
    elif grep -Fxq -- "${operation_names[$i]}" <<< "$selected_names"; then role=$(managed_selection_role "${operation_names[$i]}"); fi
    printf '%s{"provider":"%s","action":"%s","name":"%s","role":"%s","path":' \
      "$separator" "${operation_providers[$i]}" "${operation_actions[$i]}" "${operation_names[$i]}" "$role"
    managed_json_string "${operation_paths[$i]}"
    printf '}'
    separator=,
  done
  printf ']'
}

managed_emit_plan() {
  local kind=$1 name separator=''
  printf '{"protocol_version":1,"ok":true,"kind":"%s","revision":"%s",' "$kind" "$managed_state_revision"
  managed_emit_desired
  printf ',"selection":['
  while IFS= read -r name; do
    [[ -n "$name" ]] || continue
    printf '%s{"name":"%s","role":"%s"}' "$separator" "$name" "$(managed_selection_role "$name")"
    separator=,
  done <<< "$selected_names"
  printf '],"operations":'; managed_emit_operations
  printf ',"conflicts":'; managed_emit_operations conflicts
  printf ',"confirmation_required":%s,' "$managed_removals"
  if [[ "$kind" == apply ]]; then managed_emit_inventory; printf ','; fi
  managed_emit_native_notice
  printf '}\n'
}

managed_emit_status() {
  printf '{"protocol_version":1,"ok":true,"kind":"status","revision":"%s",' "$(managed_revision)"
  managed_emit_desired
  printf ','; managed_emit_inventory
  printf ','; managed_emit_native_notice
  printf '}\n'
}

managed_status_selection() (
  local name
  selected_names=''
  for name in $managed_requested $managed_retained; do
    if skill_source "$name" >/dev/null 2>&1; then include_skill "$name" || return 1; fi
  done
  if [[ "$managed_concise" == true ]]; then include_skill concise-output || return 1; fi
  printf '%s' "$selected_names"
)

managed_emit_catalog() {
  local name category key helpers separator='' codex claude
  printf '{"protocol_version":1,"ok":true,"kind":"catalog","skills":['
  while IFS=$'\t' read -r name category key; do
    helpers=$(awk -v name="$name" '$1 == name { for (i = 2; i <= NF; i++) print $i }' "$AGENT_TOOLKIT_HOME/.agents/dependencies.txt")
    codex=false
    claude=false
    if grep -Eq 'allow_implicit_invocation:[[:space:]]*false' "$AGENT_TOOLKIT_HOME/.agents/providers/codex/skills/$key/agents/openai.yaml" 2>/dev/null; then codex=true; fi
    if grep -Fxq -- "$key" "$AGENT_TOOLKIT_HOME/.agents/providers/claude/explicit-only.txt"; then claude=true; fi
    printf '%s{"name":"%s","category":"%s","key":"%s","required_helpers":' "$separator" "$name" "$category" "$key"
    managed_json_array "$helpers"
    printf ',"explicit_only":{"codex":%s,"claude":%s}}' "$codex" "$claude"
    separator=,
  done < <(catalog_records)
  printf '],"bundles":['
  separator=''
  while IFS= read -r name; do
    printf '%s{"name":"%s","skills":' "$separator" "$name"
    managed_json_array "$(bundle_members "$name")"
    printf '}'
    separator=,
  done < <(bundle_names)
  printf ']}\n'
}
