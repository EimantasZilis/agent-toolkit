#!/usr/bin/env bash
# Refresh the saved installation without changing selections or invoking Go.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/shared.sh"
source "$SCRIPT_DIR/install-options.sh"
source "$SCRIPT_DIR/managed-options.sh"
source "$SCRIPT_DIR/managed-guidance.sh"
source "$SCRIPT_DIR/managed-reconcile.sh"
source "$SCRIPT_DIR/install-managed.sh"

update_selection() {
  local name
  for name in $managed_requested $managed_retained $inventory_codex $inventory_claude; do
    if ! skill_source "$name" >/dev/null; then
      printf 'Error: saved skills changed; review setup with make install-cli.\n' >&2
      return 2
    fi
  done
  selected_names=''
  for name in $managed_requested $managed_retained; do include_skill "$name"; done
  if [[ "$managed_concise" == true ]]; then include_skill concise-output; fi
}

main() {
  local dry_run=false
  case "${1:-}" in
    --dry-run) dry_run=true; shift ;;
    --help|-h) printf 'Usage: scripts/update.sh [--dry-run]\nRefresh saved providers and preferences from this checkout.\n'; return 0 ;;
  esac
  [[ $# -eq 0 ]] || { printf 'Usage: scripts/update.sh [--dry-run]\n' >&2; return 2; }

  SKILL_CATALOG=$(source_skills)
  if ! validate_catalog || ! validate_dependencies; then
    printf 'Error: catalog changed; review selections with make install-cli.\n' >&2
    return 2
  fi
  managed_load
  if [[ ! -f "$managed_settings_file" || -z "$managed_providers" ]]; then
    printf 'Error: no saved installation; run make install-cli first.\n' >&2
    return 2
  fi

  managed_previous_concise=$managed_concise
  # Refresh always uses saved preferences, never a temporary setup override.
  MARKDOWN_OUTPUT_PATH=$managed_output
  managed_parse_options
  update_selection
  managed_plan
  installation_preview

  if [[ "$managed_conflicts" -ne 0 || "$managed_removals" == true ]]; then
    printf 'Error: update cannot resolve conflicts or remove paths; review make install-cli.\n' >&2
    return 4
  fi
  if [[ "$dry_run" == true ]]; then printf 'Dry run: no files changed.\n'; return 0; fi

  managed_expected_revision=$managed_state_revision
  managed_apply
  printf 'Updated saved installation. Start new agent sessions to load refreshed skills.\n'
}

main "$@"
