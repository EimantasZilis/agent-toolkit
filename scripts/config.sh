# Configuration and provider routing helpers.

load_provider() {
  local config_file="${AGENT_TOOLKIT_ENV_FILE:-$AGENT_TOOLKIT_HOME/.env}"
  if [[ -z "${LLM_PROVIDER+x}" && -f "$config_file" ]]; then
    while IFS= read -r line || [[ -n "$line" ]]; do
      [[ "$line" =~ ^[[:space:]]*LLM_PROVIDER[[:space:]]*=[[:space:]]*(.*)[[:space:]]*$ ]] || continue
      LLM_PROVIDER="${BASH_REMATCH[1]}"
      LLM_PROVIDER="${LLM_PROVIDER#\"}"; LLM_PROVIDER="${LLM_PROVIDER%\"}"
      LLM_PROVIDER="${LLM_PROVIDER#\'}"; LLM_PROVIDER="${LLM_PROVIDER%\'}"
      export LLM_PROVIDER
      break
    done < "$config_file"
  fi
  if [[ -z "${LLM_PROVIDER:-}" ]]; then
    echo "Error: LLM_PROVIDER is required; set it in .env or the environment (codex or claude)." >&2
    return 1
  fi
  case "$LLM_PROVIDER" in
    codex|claude) ;;
    *) echo "Error: invalid LLM_PROVIDER '$LLM_PROVIDER'; expected codex or claude." >&2; return 1 ;;
  esac
}

load_concise_output() {
  local config_file="${AGENT_TOOLKIT_ENV_FILE:-$AGENT_TOOLKIT_HOME/.env}"
  if [[ -z "${CONCISE_OUTPUT+x}" && -f "$config_file" ]]; then
    while IFS= read -r line || [[ -n "$line" ]]; do
      [[ "$line" =~ ^[[:space:]]*CONCISE_OUTPUT[[:space:]]*=[[:space:]]*(.*)[[:space:]]*$ ]] || continue
      CONCISE_OUTPUT="${BASH_REMATCH[1]}"
      CONCISE_OUTPUT="${CONCISE_OUTPUT#\"}"; CONCISE_OUTPUT="${CONCISE_OUTPUT%\"}"
      CONCISE_OUTPUT="${CONCISE_OUTPUT#\'}"; CONCISE_OUTPUT="${CONCISE_OUTPUT%\'}"
      export CONCISE_OUTPUT
      break
    done < "$config_file"
  fi
  CONCISE_OUTPUT="${CONCISE_OUTPUT:-True}"
  case "$(printf '%s' "$CONCISE_OUTPUT" | tr '[:upper:]' '[:lower:]')" in
    true|1|yes|on) CONCISE_OUTPUT=true ;;
    false|0|no|off) CONCISE_OUTPUT=false ;;
    *) echo "Error: invalid CONCISE_OUTPUT '$CONCISE_OUTPUT'; expected true or false." >&2; return 1 ;;
  esac
  export CONCISE_OUTPUT
}

load_config() {
  load_provider
  load_concise_output
}

provider_skills_dest() {
  case "$LLM_PROVIDER" in
    codex) printf '%s/skills/agent-toolkit\n' "${AGENTS_HOME:-$HOME/.agents}" ;;
    claude) printf '%s/skills\n' "${CLAUDE_HOME:-$HOME/.claude}" ;;
  esac
}

provider_legacy_skills_dest() {
  case "$LLM_PROVIDER" in
    claude) printf '%s/skills/agent-toolkit\n' "${CLAUDE_HOME:-$HOME/.claude}" ;;
    *) return 0 ;;
  esac
}

provider_guidance_dest() {
  case "$LLM_PROVIDER" in
    codex) printf '%s/AGENTS.md\n' "${CODEX_HOME:-$HOME/.codex}" ;;
    claude) printf '%s/CLAUDE.md\n' "${CLAUDE_HOME:-$HOME/.claude}" ;;
  esac
}

provider_skill_is_explicit() {
  [[ "$LLM_PROVIDER" == claude ]] || return 1
  grep -Fxq "$1" "$AGENT_TOOLKIT_HOME/.agents/providers/claude/explicit-only.txt" 2>/dev/null
}

provider_skill_metadata() {
  [[ "$LLM_PROVIDER" == codex ]] || return 0
  printf '%s/.agents/providers/codex/skills/%s/agents/openai.yaml\n' "$AGENT_TOOLKIT_HOME" "$1"
}
