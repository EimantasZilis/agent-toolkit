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

persist_output_setting() {
  local config_file=$1 value=$2 temp
  temp="$(mktemp "${config_file}.XXXXXX")"
  awk -v value="$value" '
    BEGIN { replaced = 0 }
    /^[[:space:]]*MARKDOWN_OUTPUT_PATH[[:space:]]*=/ {
      if (!replaced) {
        print "MARKDOWN_OUTPUT_PATH=" value
        replaced = 1
      }
      next
    }
    { print }
    END {
      if (!replaced) print "MARKDOWN_OUTPUT_PATH=" value
    }
  ' "$config_file" > "$temp"
  mv "$temp" "$config_file"
}

load_output_location() {
  local config_file="${AGENT_TOOLKIT_ENV_FILE:-$AGENT_TOOLKIT_HOME/.env}"
  local configured=no value=''

  if [[ -n "${MARKDOWN_OUTPUT_PATH+x}" ]]; then
    configured=yes
    value="$MARKDOWN_OUTPUT_PATH"
  elif [[ -f "$config_file" ]]; then
    while IFS= read -r line || [[ -n "$line" ]]; do
      [[ "$line" =~ ^[[:space:]]*MARKDOWN_OUTPUT_PATH[[:space:]]*=(.*)[[:space:]]*$ ]] || continue
      value="${BASH_REMATCH[1]}"
      value="${value#\"}"; value="${value%\"}"
      value="${value#\'}"; value="${value%\'}"
      configured=yes
      break
    done < "$config_file"
  fi

  if [[ "$configured" == no ]]; then
    if [[ ! -t 0 || ! -t 1 ]]; then
      echo "Error: MARKDOWN_OUTPUT_PATH is not configured; set it to an absolute path or run installation interactively and leave it blank for repo-local outputs." >&2
      return 1
    fi
    printf 'Markdown output directory (leave blank for repo-local outputs): ' >&2
    IFS= read -r value || {
      echo "Error: unable to read MARKDOWN_OUTPUT_PATH." >&2
      return 1
    }
    persist_output_setting "$config_file" "$value"
  fi

  if [[ -n "$value" && "$value" != /* ]]; then
    echo "Error: MARKDOWN_OUTPUT_PATH must be blank for repo-local outputs or an absolute path." >&2
    return 1
  fi

  MARKDOWN_OUTPUT_PATH="$value"
  export MARKDOWN_OUTPUT_PATH
}

load_config() {
  load_provider
  load_concise_output
  load_output_location
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
