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

physical_path() (
  local path=$1
  [[ -e "$path" ]] || return 1
  if [[ -d "$path" ]]; then
    cd -- "$path" || return 1
    pwd -P
  else
    cd -- "$(dirname "$path")" || return 1
    printf '%s/%s\n' "$(pwd -P)" "$(basename "$path")"
  fi
)

link_points_to() {
  local link=$1 expected=$2 target actual
  [[ -L "$link" && -e "$link" && -e "$expected" ]] || return 1
  target=$(resolved_path "$link") || return 1
  actual=$(physical_path "$target") || return 1
  expected=$(physical_path "$expected") || return 1
  [[ "$actual" == "$expected" ]]
}

owned_marker() { printf '%s/.agent-toolkit-owned\n' "$1"; }
file_hash() { shasum -a 256 "$1" | awk '{print $1}'; }

is_owned_dir() { [[ -f "$(owned_marker "$1")" ]] && grep -Fxq -- "$AGENT_TOOLKIT_HOME" "$(owned_marker "$1")"; }
is_managed_skill() {
  local destination=$1 marker expected_hash current_hash
  marker="$(owned_marker "$destination")"
  is_owned_dir "$destination" || return 1
  expected_hash="$(awk '$1 == "SKILL.md" { print $2 }' "$marker")"
  current_hash="$(file_hash "$destination/SKILL.md" 2>/dev/null || true)"
  [[ -n "$expected_hash" && "$expected_hash" == "$current_hash" ]]
}
