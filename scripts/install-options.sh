# Read installer options and settings without mutating configuration.

installation_usage() {
  cat <<'EOF'
Usage: scripts/install.sh [options]
  --provider codex|claude        Select the target provider
  --bundle NAME                  Select starter, all, or a category bundle
  --skill NAME                   Select one skill (repeatable; excludes --bundle)
  --global-guidance              Update global guidance
  --no-global-guidance           Leave global guidance untouched
  --dry-run                      Preview only; do not prompt or write files
  --list                         List available skills without installing
  --help                         Show this help
With no options, prompt in a terminal; otherwise use saved settings and defaults.
EOF
}

parse_install_options() {
  provider_option=''
  selection_option=''
  guidance_option=''
  bundle_option=''
  dry_run=false
  list_only=false
  help_only=false
  interactive_install=false
  [[ "$#" -eq 0 && -t 0 && -t 1 ]] && interactive_install=true
  while [[ "$#" -gt 0 ]]; do
    case "$1" in
      --provider|--bundle|--skill)
        [[ "$#" -ge 2 && -n "$2" && "$2" != --* ]] || {
          printf 'Error: %s requires a value.\n' "$1" >&2; return 1;
        }
        case "$1" in
          --provider) provider_option=$2 ;;
          --bundle) bundle_option=$2 ;;
          --skill)
            [[ "$2" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]] || {
              printf 'Error: invalid skill name: %s\n' "$2" >&2; return 1;
            }
            skill_source "$2" >/dev/null || return 1
            selection_option+="${selection_option:+ }$2"
            ;;
        esac
        shift 2
        ;;
      --global-guidance|--no-global-guidance)
        [[ -z "$guidance_option" ]] || { printf 'Error: specify guidance once.\n' >&2; return 1; }
        guidance_option=true
        [[ "$1" == --no-global-guidance ]] && guidance_option=false
        shift
        ;;
      --dry-run) dry_run=true; shift ;;
      --list) list_only=true; shift ;;
      --help|-h) help_only=true; shift ;;
      *) printf 'Error: unknown option: %s\n' "$1" >&2; return 1 ;;
    esac
  done
  if [[ -n "$bundle_option" ]]; then
    [[ -z "$selection_option" ]] || { printf 'Error: --skill and --bundle cannot be combined.\n' >&2; return 1; }
    bundle_names | grep -Fxq -- "$bundle_option" || {
      printf 'Error: unknown bundle: %s\n' "$bundle_option" >&2; return 1;
    }
    selection_option=$bundle_option
  fi
}

boolean_setting() {
  case "$(printf '%s' "$2" | tr '[:upper:]' '[:lower:]')" in
    true|1|yes|on) printf 'true\n' ;;
    false|0|no|off) printf 'false\n' ;;
    *) printf 'Error: invalid %s: %s; expected True or False.\n' "$1" "$2" >&2; return 1 ;;
  esac
}

prompt_for_value() {
  local label=$1 default=$2 variable=$3 answer
  printf '%s [%s]: ' "$label" "${4:-$default}" >&2
  IFS= read -r answer || { printf 'Error: unable to read %s.\n' "$label" >&2; return 1; }
  printf -v "$variable" '%s' "${answer:-$default}"
}

read_installation_settings() {
  local config_file=$AGENT_TOOLKIT_ENV_FILE selection_default=starter guidance_default=False
  if [[ -f "$config_file" ]]; then
    selection_default=all
    guidance_default=True
  fi
  # Provider and concise preferences retain their existing .env precedence.
  LLM_PROVIDER="${provider_option:-$(config_value "$config_file" LLM_PROVIDER || true)}"
  LLM_PROVIDER=${LLM_PROVIDER:-codex}
  SKILL_SELECTION="${selection_option:-$(config_value "$config_file" SKILL_SELECTION || printf '%s' "$selection_default")}"
  GLOBAL_GUIDANCE="${guidance_option:-$(config_value "$config_file" GLOBAL_GUIDANCE || printf '%s' "$guidance_default")}"
  CONCISE_OUTPUT="$(config_value "$config_file" CONCISE_OUTPUT || printf True)"
  configured_output="$(config_value "$config_file" MARKDOWN_OUTPUT_PATH || true)"

  if [[ "$interactive_install" == true ]]; then
    prompt_for_value 'LLM provider (codex or claude)' "$LLM_PROVIDER" LLM_PROVIDER
    prompt_for_value 'Skills (starter, all, or space-separated names)' "$SKILL_SELECTION" SKILL_SELECTION
    prompt_for_value 'Update global guidance (True or False)' "$GLOBAL_GUIDANCE" GLOBAL_GUIDANCE
    prompt_for_value 'Markdown output path (blank for repo-local)' "$configured_output" configured_output repo-local
    if [[ "$(boolean_setting GLOBAL_GUIDANCE "$GLOBAL_GUIDANCE")" == true ]]; then
      prompt_for_value 'Concise output in global guidance (True or False)' "$CONCISE_OUTPUT" CONCISE_OUTPUT
    fi
  elif [[ "$dry_run" == false && -f "$config_file" && -z "${MARKDOWN_OUTPUT_PATH+x}" ]] &&
    ! config_value "$config_file" MARKDOWN_OUTPUT_PATH >/dev/null; then
    printf 'Error: MARKDOWN_OUTPUT_PATH is not configured; set it to an absolute path or run installation interactively and leave it blank for repo-local outputs.\n' >&2
    return 1
  fi

  case "$LLM_PROVIDER" in
    codex|claude) ;;
    *) printf 'Error: invalid LLM provider: %s; expected codex or claude.\n' "$LLM_PROVIDER" >&2; return 1 ;;
  esac
  GLOBAL_GUIDANCE="$(boolean_setting GLOBAL_GUIDANCE "$GLOBAL_GUIDANCE")"
  CONCISE_OUTPUT="$(boolean_setting CONCISE_OUTPUT "$CONCISE_OUTPUT")"
  [[ "$SKILL_SELECTION" =~ [^[:space:]] ]] || {
    printf 'Error: select at least one skill.\n' >&2; return 1;
  }
  MARKDOWN_OUTPUT_PATH=${MARKDOWN_OUTPUT_PATH-$configured_output}
  if [[ -n "$MARKDOWN_OUTPUT_PATH" && "$MARKDOWN_OUTPUT_PATH" != /* ]]; then
    printf 'Error: MARKDOWN_OUTPUT_PATH must be blank or an absolute path.\n' >&2
    return 1
  fi
  if [[ "$AGENT_TOOLKIT_ENV_FILE" == *$'\n'* || "$MARKDOWN_OUTPUT_PATH" == *$'\n'* || "$SKILL_SELECTION" == *$'\n'* ]]; then
    printf 'Error: configuration values must fit on one line.\n' >&2
    return 1
  fi
}

save_installation_settings() {
  local config_file=$AGENT_TOOLKIT_ENV_FILE
  if [[ ! -f "$config_file" ]]; then
    cp -- "$AGENT_TOOLKIT_HOME/.env.template" "$config_file"
  fi
  persist_config_setting "$config_file" LLM_PROVIDER "$LLM_PROVIDER"
  persist_config_setting "$config_file" SKILL_SELECTION "$SKILL_SELECTION"
  persist_config_setting "$config_file" GLOBAL_GUIDANCE "$GLOBAL_GUIDANCE"
  persist_config_setting "$config_file" CONCISE_OUTPUT "$CONCISE_OUTPUT"
  persist_config_setting "$config_file" MARKDOWN_OUTPUT_PATH "$configured_output"
}
