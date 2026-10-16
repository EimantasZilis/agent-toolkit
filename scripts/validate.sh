#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/shared.sh"
errors=0
fail() { echo "ERROR: $*" >&2; errors=$((errors + 1)); }
pass() { echo "OK: $*"; }

metadata() {
  awk -v field="$2" '/^---$/ { if (++block == 1) next; if (block == 2) exit } block == 1 && $0 ~ "^" field ":" { sub("^" field ":[[:space:]]*", ""); print; exit }' "$1"
}

check_links() {
  local file=$1 ref target
  while IFS= read -r ref; do
    [[ -z "$ref" || "$ref" == http://* || "$ref" == https://* || "$ref" == \#* ]] && continue
    target="$(dirname "$file")/$ref"
    [[ -e "$target" ]] || fail "${file#"$AGENT_TOOLKIT_HOME/"}: broken relative link $ref"
  done < <(grep -oE '\[[^]]*\]\([^)]+\)' "$file" | sed -E 's/.*\]\(([^)]+)\)/\1/' || true)
}

check_adapter_key() {
  local key=$1 origin=$2
  grep -Fxq -- "$key" <<< "$skill_keys" || fail "$origin: no source skill for adapter key '$key'"
}

check_adapters() {
  local root="$AGENT_TOOLKIT_HOME/.agents/providers/codex/skills" file key
  if [[ -d "$root" ]]; then
    while IFS= read -r file; do
      key=${file#"$root/"}
      key=${key%/agents/openai.yaml}
      check_adapter_key "$key" "${file#"$AGENT_TOOLKIT_HOME/"}"
    done < <(find "$root" -name openai.yaml | sort)
  fi

  file="$AGENT_TOOLKIT_HOME/.agents/providers/claude/explicit-only.txt"
  if [[ ! -f "$file" || ! -r "$file" ]]; then
    fail "missing or unreadable Claude explicit-only list: $file"
    return
  fi
  while IFS= read -r key || [[ -n "$key" ]]; do
    [[ -z "$key" ]] && continue
    check_adapter_key "$key" "${file#"$AGENT_TOOLKIT_HOME/"}"
  done < "$file"
}

echo "Validating provider-neutral agent-toolkit skills"
[[ -d "$AGENT_TOOLKIT_HOME/.agents/skills" ]] || fail "missing .agents/skills"
[[ -f "$(source_guidance)" ]] || fail "missing .agents/global/instructions.md"
[[ ! -e "$AGENT_TOOLKIT_HOME/.cursor" ]] || fail ".cursor must not exist"
[[ ! -e "$AGENT_TOOLKIT_HOME/config.yml" ]] || fail "config.yml must not exist"
[[ ! -e "$AGENT_TOOLKIT_HOME/requirements.txt" ]] || fail "requirements.txt must not exist"
[[ -z "$(find "$AGENT_TOOLKIT_HOME/.agents/skills" -type f -path '*/agents/openai.yaml' -print -quit)" ]] || fail "provider metadata must not be stored in .agents/skills"

skill_count=0
while IFS= read -r file; do
  rel=${file#"$AGENT_TOOLKIT_HOME/"}; folder=$(basename "$(dirname "$file")")
  name=$(metadata "$file" name); description=$(metadata "$file" description)
  [[ $(sed -n '1,80p' "$file" | awk '/^---$/ { count++ } END { print count + 0 }') -ge 2 ]] || fail "$rel: unterminated YAML front matter"
  [[ -n "$name" ]] || fail "$rel: missing metadata name"
  [[ -n "$description" ]] || fail "$rel: missing metadata description"
  [[ "$name" == "$folder" ]] || fail "$rel: metadata name '$name' does not match folder '$folder'"
  check_links "$file"
  skill_count=$((skill_count + 1))
done < <(find "$AGENT_TOOLKIT_HOME/.agents/skills" -mindepth 3 -maxdepth 3 -type f -name SKILL.md | sort)

[[ $skill_count -gt 0 ]] || fail "no skills found in .agents/skills"
skill_keys="$(while IFS= read -r source; do skill_key "$source"; done < <(source_skills))"
check_adapters
[[ -f "$AGENT_TOOLKIT_HOME/.env.template" ]] || fail "missing .env.template"
grep -qx 'LLM_PROVIDER=' "$AGENT_TOOLKIT_HOME/.env.template" || fail ".env.template must define a blank LLM_PROVIDER"
grep -qx 'CONCISE_OUTPUT=True' "$AGENT_TOOLKIT_HOME/.env.template" || fail ".env.template must default CONCISE_OUTPUT to True"
grep -qx '# MARKDOWN_OUTPUT_PATH=' "$AGENT_TOOLKIT_HOME/.env.template" || fail ".env.template must document the optional MARKDOWN_OUTPUT_PATH setting"

if [[ $errors -gt 0 ]]; then echo "Validation failed with $errors error(s)."; exit 1; fi
pass "$skill_count skills, metadata, links, provider boundaries, and layout are valid"
