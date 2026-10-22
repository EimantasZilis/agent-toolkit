#!/usr/bin/env bash
# Preview and apply managed changes across providers without an interactive UI.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/shared.sh"
source "$SCRIPT_DIR/install-options.sh"
source "$SCRIPT_DIR/managed-options.sh"
source "$SCRIPT_DIR/managed-guidance.sh"
source "$SCRIPT_DIR/managed-reconcile.sh"
source "$SCRIPT_DIR/managed-protocol.sh"
trap 'managed_protocol_failure "$?"' EXIT

main() {
  managed_protocol_options "$@"
  [[ ${#managed_arguments[@]} -gt 0 ]] || { printf 'Error: a managed command is required.\n' >&2; return 2; }
  set -- "${managed_arguments[@]}"
  local command=${1:-}
  case "$command" in catalog|status|preview|apply) shift ;; *) printf 'Usage: scripts/managed.sh catalog|status|preview|apply [options]\n' >&2; return 2 ;; esac
  SKILL_CATALOG=$(source_skills)
  validate_catalog
  validate_dependencies
  if [[ "$command" == catalog ]]; then
    [[ $# -eq 0 ]] || { printf 'Error: catalog accepts only protocol/inventory options.\n' >&2; return 2; }
    managed_emit_catalog
    return
  fi
  managed_load
  managed_previous_output=$managed_output
  managed_previous_concise=$managed_concise
  managed_previous_full_template=$managed_full_template
  if [[ "$command" == status ]]; then
    [[ $# -eq 0 ]] || { printf 'Error: status accepts only protocol/inventory options.\n' >&2; return 2; }
    selected_names=$(managed_status_selection)
    managed_emit_status
    return
  fi
  managed_parse_options "$@"
  managed_resolve_selection
  managed_plan
  if [[ "$command" == apply ]]; then
    managed_apply
    managed_state_revision=$(managed_revision)
  fi
  managed_emit_plan "$command"
}

main "$@"
