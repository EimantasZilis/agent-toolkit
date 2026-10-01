# Skill discovery, ownership, and filesystem helpers.

source_skills() { find "$AGENT_TOOLKIT_HOME/.agents/skills" -mindepth 3 -maxdepth 3 -type f -name SKILL.md -exec dirname {} \; | sort; }
skill_key() {
  local path=$1 prefix="$AGENT_TOOLKIT_HOME/.agents/skills/"
  path=${path#"$prefix"}
  printf '%s\n' "$path"
}

resolved_path() {
  local target=$1 link
  link="$(readlink "$target")"
  [[ "$link" == /* ]] && printf '%s/%s\n' "$(cd "$(dirname "$link")" && pwd -P)" "$(basename "$link")" || printf '%s/%s\n' "$(cd "$(dirname "$target")/$(dirname "$link")" && pwd -P)" "$(basename "$link")"
}
owned_marker() { printf '%s/.agent-toolkit-owned\n' "$1"; }
file_hash() { shasum -a 256 "$1" | awk '{print $1}'; }

is_owned_dir() { [[ -f "$(owned_marker "$1")" ]] && grep -qx "$AGENT_TOOLKIT_HOME" "$(owned_marker "$1")"; }
is_managed_skill() {
  local destination=$1 marker expected_hash current_hash
  marker="$(owned_marker "$destination")"
  is_owned_dir "$destination" || return 1
  expected_hash="$(awk '$1 == "SKILL.md" { print $2 }' "$marker")"
  current_hash="$(file_hash "$destination/SKILL.md" 2>/dev/null || true)"
  [[ -n "$expected_hash" && "$expected_hash" == "$current_hash" ]]
}
is_managed_guidance() {
  local destination=$1 marker expected_hash current_hash
  marker="$(owned_marker "$(dirname "$destination")")"
  [[ -f "$marker" ]] || return 1
  grep -qx "$AGENT_TOOLKIT_HOME" "$marker" || return 1
  expected_hash="$(awk '$1 == "GUIDANCE.md" { print $2 }' "$marker")"
  current_hash="$(file_hash "$destination" 2>/dev/null || true)"
  [[ -n "$expected_hash" && "$expected_hash" == "$current_hash" ]]
}
is_user_owned_skill() {
  local source=$1 dest=$2
  [[ -e "$dest" || -L "$dest" ]] || return 1
  if [[ -L "$dest" && "$(resolved_path "$dest")" == "$(cd "$(dirname "$source")" && pwd -P)/$(basename "$source")" ]]; then
    return 1
  fi
  is_managed_skill "$dest" && return 1
  return 0
}
ensure_destination() {
  local target=$1
  if [[ -e "$target" || -L "$target" ]]; then
    if [[ -L "$target" && "$(resolved_path "$target")" == "$2" ]]; then rm "$target"; return; fi
    is_owned_dir "$target" || { echo "Error: refusing to overwrite non-managed path: $target" >&2; return 1; }
  fi
  mkdir -p "$target"
}
