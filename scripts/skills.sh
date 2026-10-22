# Skill rendering helpers.

render_skill_manifest() {
  local source=$1 explicit=$2
  # Environment values preserve literal backslashes in paths (awk -v does not).
  AGENT_TOOLKIT_RENDER_CATALOG="${SKILL_CATALOG:-$(source_skills)}" \
    AGENT_TOOLKIT_RENDER_SELECTED="${selected_names:-}" \
    AGENT_TOOLKIT_RENDER_DEST="$SKILLS_DEST" \
    AGENT_TOOLKIT_RENDER_PROVIDER="$LLM_PROVIDER" \
    awk -v explicit="$explicit" '
    BEGIN {
      count = split(ENVIRON["AGENT_TOOLKIT_RENDER_CATALOG"], paths, "\n")
      for (i = 1; i <= count; i++) {
        name = paths[i]; sub(/^.*\//, "", name)
        sources[name] = paths[i] "/SKILL.md"
      }
      count = split(ENVIRON["AGENT_TOOLKIT_RENDER_SELECTED"], names, "\n")
      for (i = 1; i <= count; i++) selected[names[i]] = 1
    }
    FNR == 1 && $0 == "---" { print; in_front = 1; next }
    in_front && $0 == "---" {
      if (explicit == "yes" && !seen) print "disable-model-invocation: true"
      print; in_front = 0; next
    }
    in_front && /^disable-model-invocation:/ { seen = 1 }
    !in_front {
      rest = $0; result = ""
      while (match(rest, /\]\(\.\.\/[^)]*\/SKILL[.]md\)/)) {
        result = result substr(rest, 1, RSTART - 1)
        link = substr(rest, RSTART, RLENGTH)
        name = link; sub(/\/SKILL[.]md\)$/, "", name); sub(/^.*\//, "", name)
        target = sources[name]
        if (selected[name]) target = ENVIRON["AGENT_TOOLKIT_RENDER_DEST"] "/" name "/SKILL.md"
        result = result (target != "" ? "](<" target ">)" : link)
        rest = substr(rest, RSTART + RLENGTH)
      }
      $0 = result rest
      if (ENVIRON["AGENT_TOOLKIT_RENDER_PROVIDER"] == "claude") {
        rest = $0; result = ""
        while (match(rest, /\$[a-z0-9]+(-[a-z0-9]+)*/)) {
          result = result substr(rest, 1, RSTART - 1)
          token = substr(rest, RSTART, RLENGTH); name = substr(token, 2)
          result = result (name in sources ? "/" name : token)
          rest = substr(rest, RSTART + RLENGTH)
        }
        $0 = result rest
      }
    }
    { print }
  ' "$source/SKILL.md"
}

print_skill_output_policy() {
  local source=$1
  # Only workflows that produce or discover Markdown need these task-local rules.
  case "${source##*/}" in
    developer-highlights|next-commit|next-phase|plan-writing-conventions|plan-execution-conventions|plan-review|requirements-to-tickets|review-branch|test-cases|ticket-context|ticket-to-plan|to-markdown) ;;
    *) return 0 ;;
  esac
  {
    printf '\n'
    awk '/^## Markdown artifact outputs/ { emit = 1 } emit { print }' "$(source_guidance)"
    printf '\n### Output location for this skill\n\n'
    if [[ -n "${MARKDOWN_OUTPUT_PATH:-}" ]]; then
      printf 'The configured centralized output root is `%s`.\n' "$MARKDOWN_OUTPUT_PATH"
      printf 'Use `%s/<repo-slug>/<branch-slug>/<skill-name>/...` unless the user supplies a path.\n' "$MARKDOWN_OUTPUT_PATH"
    else
      printf 'Use `<repo-root>/<branch-slug>/<skill-name>/...` unless the user supplies a path.\n'
      printf 'No centralized output root is configured for this installation.\n'
    fi
    printf 'These rules apply only while performing this skill.\n'
  }
}

append_skill_output_policy() {
  print_skill_output_policy "$1" >> "$2"
}

render_skill() {
  local source=$1 dest=$2 key=$3 file rel
  local explicit=no
  if provider_skill_is_explicit "$key"; then
    explicit=yes
  fi
  mkdir -p "$dest"
  if is_owned_dir "$dest"; then
    local old_hash current_hash
    old_hash="$(awk '$1 == "SKILL.md" { print $2 }' "$(owned_marker "$dest")")"
    current_hash="$(file_hash "$dest/SKILL.md" 2>/dev/null || true)"
    [[ -n "$old_hash" && "$old_hash" == "$current_hash" ]] && rm -f "$dest/SKILL.md"
    find "$dest" -type l -delete
    rm -f "$(owned_marker "$dest")"
  fi
  render_skill_manifest "$source" "$explicit" > "$dest/SKILL.md"
  append_skill_output_policy "$source" "$dest/SKILL.md"
  printf '%s\nSKILL.md %s\n' "$AGENT_TOOLKIT_HOME" "$(file_hash "$dest/SKILL.md")" > "$(owned_marker "$dest")"
  while IFS= read -r file; do
    rel=${file#"$source/"}; [[ "$rel" == SKILL.md ]] && continue
    mkdir -p "$dest/$(dirname "$rel")"
    ln -s "$file" "$dest/$rel"
  done < <(find "$source" -type f ! -name SKILL.md | sort)
  local metadata
  metadata="$(provider_skill_metadata "$key")"
  if [[ -n "$metadata" && -f "$metadata" ]]; then
    mkdir -p "$dest/agents"; ln -s "$metadata" "$dest/agents/openai.yaml"
  fi
}
