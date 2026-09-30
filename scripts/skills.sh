# Skill rendering helpers.

render_skill() {
  local source=$1 dest=$2 key=$3
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
  awk -v explicit="$explicit" '
    FNR == 1 && $0 == "---" { print; in_front = 1; next }
    in_front && $0 == "---" { if (explicit == "yes") print "disable-model-invocation: true"; print; in_front = 0; next }
    in_front && $0 ~ /^disable-model-invocation:/ { seen = 1 }
    { print }
    END { if (explicit == "yes" && seen) exit 0 }
  ' "$source/SKILL.md" > "$dest/SKILL.md"
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
