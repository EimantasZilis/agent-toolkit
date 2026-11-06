# Provider-neutral skill selection and dependency resolution.

skill_source() {
  local name=$1 source found=''
  while IFS= read -r source; do
    [[ "${source##*/}" == "$name" ]] || continue
    if [[ -n "$found" ]]; then
      printf 'Error: ambiguous skill name: %s\n' "$name" >&2
      return 1
    fi
    found=$source
  done <<< "$SKILL_CATALOG"
  if [[ -z "$found" ]]; then
    printf 'Error: unknown skill: %s\n' "$name" >&2
    return 1
  fi
  printf '%s\n' "$found"
}

validate_dependencies() {
  local file="$AGENT_TOOLKIT_HOME/.agents/dependencies.txt" name dependencies dependency seen=''
  local -a items
  [[ -r "$file" ]] || { printf 'Error: missing dependency registry: %s\n' "$file" >&2; return 1; }
  while read -r name dependencies; do
    [[ -z "$name" || "$name" == \#* ]] && continue
    skill_source "$name" >/dev/null || return 1
    if grep -Fxq -- "$name" <<< "$seen"; then
      printf 'Error: duplicate dependency entry: %s\n' "$name" >&2
      return 1
    fi
    seen+="$name"$'\n'
    [[ -n "$dependencies" ]] || { printf 'Error: empty dependency entry: %s\n' "$name" >&2; return 1; }
    read -r -a items <<< "$dependencies"
    for dependency in "${items[@]}"; do
      skill_source "$dependency" >/dev/null || return 1
    done
  done < "$file"
}

include_skill() {
  local name=$1 dependency dependencies
  local -a items
  grep -Fxq -- "$name" <<< "$selected_names" && return 0
  skill_source "$name" >/dev/null || return 1
  # Mark before descending to deduplicate dependencies and guard against cycles.
  selected_names+="$name"$'\n'
  dependencies="$(awk -v name="$name" '$1 == name { $1 = ""; print }' "$AGENT_TOOLKIT_HOME/.agents/dependencies.txt")"
  [[ -n "$dependencies" ]] || return 0
  read -r -a items <<< "$dependencies"
  for dependency in "${items[@]}"; do
    include_skill "$dependency" || return 1
  done
}

resolve_selection() {
  local name
  local -a names
  requested_names=''
  selected_names=''
  if bundle_names | grep -Fxq -- "$SKILL_SELECTION"; then
    requested_names="$(bundle_members "$SKILL_SELECTION")"
  else
    read -r -a names <<< "$SKILL_SELECTION"
    [[ "${#names[@]}" -gt 0 ]] || { printf 'Error: select at least one skill.\n' >&2; return 1; }
    for name in "${names[@]}"; do
      skill_source "$name" >/dev/null || return 1
      requested_names+="$name"$'\n'
    done
  fi
  while IFS= read -r name; do
    [[ -n "$name" ]] || continue
    include_skill "$name" || return 1
  done <<< "$requested_names"
  if [[ "$GLOBAL_GUIDANCE" == true && "$CONCISE_OUTPUT" == true ]]; then
    include_skill concise-output || return 1
  fi
}
