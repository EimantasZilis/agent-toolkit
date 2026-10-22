#!/usr/bin/env bash
# Query the shared installer catalog for native packaging.

set -euo pipefail

main() {
  local operation="$2"

  source "$1/scripts/shared.sh"
  SKILL_CATALOG=$(source_skills)
  validate_catalog
  validate_dependencies

  # These globals configure the sourced selection helpers.
  # shellcheck disable=SC2034
  GLOBAL_GUIDANCE=false
  # shellcheck disable=SC2034
  CONCISE_OUTPUT=false
  # shellcheck disable=SC2034
  SKILL_SELECTION=$3

  if [[ "$operation" == resolve_selection ]]; then
    resolve_selection
    printf '%s' "$selected_names"
  else
    "$operation" "$3"
  fi
}

main "$@"
