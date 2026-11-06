# Provider routing for the current installation layout.

provider_skills_dest() {
  case "$LLM_PROVIDER" in
    codex) printf '%s/skills/agent-toolkit\n' "${AGENTS_HOME:-$HOME/.agents}" ;;
    claude) printf '%s/skills\n' "${CLAUDE_HOME:-$HOME/.claude}" ;;
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
