#!/usr/bin/env bash
# Preview and apply managed changes across providers without an interactive UI.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/shared.sh"
source "$SCRIPT_DIR/install-options.sh"
source "$SCRIPT_DIR/managed-options.sh"
source "$SCRIPT_DIR/managed-guidance.sh"
source "$SCRIPT_DIR/managed-reconcile.sh"

main() {
  local command=${1:-}
  case "$command" in preview|apply) shift ;; *) printf 'Usage: scripts/managed.sh preview|apply [options]\n' >&2; return 2 ;; esac
  SKILL_CATALOG=$(source_skills)
  validate_catalog
  validate_dependencies
  managed_load
  managed_previous_output=$managed_output
  managed_previous_concise=$managed_concise
  managed_previous_full_template=$managed_full_template
  managed_parse_options "$@"
  managed_resolve_selection
  managed_plan
  if [[ "$command" == apply ]]; then managed_apply; else managed_print_preview; fi
}

main "$@"
