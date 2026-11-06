# Read installer options and settings without mutating configuration.

installation_usage() {
  cat <<'EOF'
Usage: scripts/install.sh [options]
  --provider codex|claude        Replace providers (repeat for both)
  --bundle NAME                  Select starter, all, or a category bundle
  --skill NAME                   Select one skill (repeatable; excludes --bundle)
  --confirm-removals             Authorize the previewed removals
  --remove-skill NAME             Remove an unused skill or helper
  --output-path PATH              Save output root (blank for repo-local)
  --concise BOOL                  Control concise loading
  --full-template BOOL            Control the full guidance template
  --no-providers                  Remove owned provider installations
  --no-skills                     Clear requested skills
  --create-output-directory       Create missing output root during apply
  --dry-run                      Preview only; do not prompt or write files
  --list                         List available skills without installing
  --help                         Show this help
With no options, prompt in a terminal; otherwise use saved settings and defaults.
EOF
}

parse_install_options() {
  provider_option=''
  frontend_options=()
  confirm_removals=false
  selection_option=''
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
          --provider)
            case "$2" in codex|claude) ;; *) printf 'Error: invalid provider.\n' >&2; return 2 ;; esac
            provider_option=$(managed_words "$provider_option" "$2")
            ;;
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
      --remove-skill|--output-path|--concise|--full-template)
        [[ $# -ge 2 ]] || { printf 'Error: %s requires a value.\n' "$1" >&2; return 2; }
        frontend_options[${#frontend_options[@]}]=$1
        frontend_options[${#frontend_options[@]}]=$2
        shift 2
        ;;
      --no-providers|--no-skills|--create-output-directory)
        frontend_options[${#frontend_options[@]}]=$1
        shift
        ;;
      --confirm-removals) confirm_removals=true; shift ;;
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
