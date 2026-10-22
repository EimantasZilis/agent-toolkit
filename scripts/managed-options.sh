# Parse managed reconciliation arguments as data, never executable shell input.

managed_remove_word() {
  local word
  for word in $1; do
    [[ "$word" == "$2" ]] || printf '%s\n' "$word"
  done
}

managed_parse_options() {
  local providers_set=no skills_set=no name
  managed_expected_revision=''
  managed_confirm=false
  managed_removed=''
  managed_output_set=no
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --provider|--skill|--remove-skill|--output-path|--revision|--concise|--full-template)
        [[ $# -ge 2 ]] || { printf 'Error: %s requires a value.\n' "$1" >&2; return 2; }
        case "$1" in
          --provider)
            case "$2" in codex|claude) ;; *) printf 'Error: invalid provider.\n' >&2; return 2 ;; esac
            [[ "$providers_set" == yes ]] || managed_providers=''
            providers_set=yes
            managed_providers=$(managed_words "$managed_providers" "$2")
            ;;
          --skill)
            skill_source "$2" >/dev/null || return 2
            [[ "$skills_set" == yes ]] || managed_requested=''
            skills_set=yes
            managed_requested=$(managed_words "$managed_requested" "$2")
            ;;
          --remove-skill)
            skill_source "$2" >/dev/null || return 2
            managed_removed=$(managed_words "$managed_removed" "$2")
            ;;
          --output-path) managed_output=$2; managed_output_set=yes ;;
          --concise) managed_concise=$(boolean_setting CONCISE_OUTPUT "$2") ;;
          --full-template) managed_full_template=$(boolean_setting GLOBAL_GUIDANCE "$2") ;;
          --revision)
            [[ "$2" =~ ^[a-f0-9]{64}$ ]] || { printf 'Error: invalid revision.\n' >&2; return 2; }
            managed_expected_revision=$2
            ;;
        esac
        shift 2
        ;;
      --no-providers) managed_providers=''; providers_set=yes; shift ;;
      --no-skills) managed_requested=''; skills_set=yes; shift ;;
      --confirm-removals) managed_confirm=true; shift ;;
      *) printf 'Error: unknown managed option: %s\n' "$1" >&2; return 2 ;;
    esac
  done
  for name in $managed_removed; do
    managed_requested=$(managed_words "$(managed_remove_word "$managed_requested" "$name")")
    managed_retained=$(managed_words "$(managed_remove_word "$managed_retained" "$name")")
  done
  managed_validate_settings
}

managed_resolve_selection() {
  local helper name installed
  # Preserve installed helpers when their parent is deselected. Explicit
  # removal still fails if another selected skill requires that helper.
  installed=$(managed_words "$inventory_codex" "$inventory_claude" \
    "$(managed_observed_skills codex)" "$(managed_observed_skills claude)")
  for helper in $(awk '!/^#/ { for (i = 2; i <= NF; i++) print $i }' "$AGENT_TOOLKIT_HOME/.agents/dependencies.txt" | sort -u); do
    if managed_contains "$installed" "$helper" && ! managed_contains "$managed_removed" "$helper"; then
      managed_retained=$(managed_words "$managed_retained" "$helper")
    fi
  done
  requested_names=$(printf '%s\n' "$managed_requested" | tr ' ' '\n')
  if [[ "$managed_previous_concise" == true ]] && managed_contains "$installed" concise-output &&
    ! managed_contains "$managed_removed" concise-output; then
    managed_retained=$(managed_words "$managed_retained" concise-output)
  fi
  selected_names=''
  for name in $managed_requested $managed_retained; do
    include_skill "$name" || return 2
  done
  if [[ "$managed_concise" == true ]]; then include_skill concise-output || return 2; fi
  for name in $managed_removed; do
    if grep -Fxq -- "$name" <<< "$selected_names"; then
      printf 'Error: selected skills require %s; remove its parents first.\n' "$name" >&2
      return 2
    fi
  done
  MARKDOWN_OUTPUT_PATH=${MARKDOWN_OUTPUT_PATH-$managed_output}
  [[ "$managed_output_set" == no ]] || MARKDOWN_OUTPUT_PATH=$managed_output
  [[ ( -z "$MARKDOWN_OUTPUT_PATH" || "$MARKDOWN_OUTPUT_PATH" == /* ) &&
    "$MARKDOWN_OUTPUT_PATH" != *$'\n'* && "$MARKDOWN_OUTPUT_PATH" != *$'\r'* ]] || {
    printf 'Error: output override must be absolute and single-line, or blank.\n' >&2
    return 2
  }
}
