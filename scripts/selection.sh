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
  # Mark before descending: conditional Ruby/Rails dependencies form a cycle.
  selected_names+="$name"$'\n'
  dependencies="$(awk -v name="$name" '$1 == name { $1 = ""; print }' "$AGENT_TOOLKIT_HOME/.agents/dependencies.txt")"
  [[ -n "$dependencies" ]] || return 0
  read -r -a items <<< "$dependencies"
  for dependency in "${items[@]}"; do
    include_skill "$dependency" || return 1
  done
}

resolve_selection() {
  local source name
  local -a names
  requested_names=''
  selected_names=''
  case "$SKILL_SELECTION" in
    starter) requested_names=$'concise-output\ncommit-message\n' ;;
    all)
      while IFS= read -r source; do
        requested_names+="${source##*/}"$'\n'
      done <<< "$SKILL_CATALOG"
      ;;
    *)
      read -r -a names <<< "$SKILL_SELECTION"
      [[ "${#names[@]}" -gt 0 ]] || { printf 'Error: select at least one skill.\n' >&2; return 1; }
      for name in "${names[@]}"; do
        skill_source "$name" >/dev/null || return 1
        requested_names+="$name"$'\n'
      done
      ;;
  esac
  while IFS= read -r name; do
    [[ -n "$name" ]] || continue
    include_skill "$name" || return 1
  done <<< "$requested_names"
  if [[ "$GLOBAL_GUIDANCE" == true && "$CONCISE_OUTPUT" == true ]]; then
    include_skill concise-output || return 1
  fi
}

selected_sources() {
  local name
  while IFS= read -r name; do
    [[ -n "$name" ]] || continue
    skill_source "$name" || return 1
  done <<< "$selected_names"
}

missing_supporting_files() {
  local source=$1 destination=$2 file rel missing=''
  while IFS= read -r file; do
    rel=${file#"$source/"}
    [[ -e "$destination/$rel" ]] || missing+="${missing:+, }$rel"
  done < <(find "$source" -type f ! -name SKILL.md | sort)
  printf '%s\n' "$missing"
}

preview_installation() {
  local source name role action missing
  printf 'Provider: %s\nSelection: %s\n' "$LLM_PROVIDER" "$SKILL_SELECTION"
  while IFS= read -r source; do
    name=${source##*/}
    role=dependency
    grep -Fxq -- "$name" <<< "$requested_names" && role=requested
    action=install/update
    if is_user_owned_skill "$source" "$SKILLS_DEST/$name"; then
      action='skip: user-owned'
      missing="$(missing_supporting_files "$source" "$SKILLS_DEST/$name")"
      if [[ -n "$missing" ]]; then
        printf '    Warning: missing supporting files: %s. Move the existing directory aside to install the complete skill.\n' "$missing"
      fi
    fi
    printf '  %s (%s; %s) -> %s/%s\n' "$name" "$role" "$action" "$SKILLS_DEST" "$name"
  done < <(selected_sources)
  if [[ "$GLOBAL_GUIDANCE" == true ]]; then
    printf 'Global guidance: update %s\n' "$GUIDANCE_DEST"
  else
    printf 'Global guidance: unchanged (%s)\n' "$GUIDANCE_DEST"
  fi
  printf 'Configuration: save successful choices to %s\n' "$AGENT_TOOLKIT_ENV_FILE"
  printf 'Previously installed, unselected skills are kept.\n'
}
