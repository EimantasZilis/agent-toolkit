#!/usr/bin/env bash
# Preview and apply Bash setup through the managed installation engine.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/shared.sh"
source "$SCRIPT_DIR/install-options.sh"
source "$SCRIPT_DIR/managed-options.sh"
source "$SCRIPT_DIR/managed-guidance.sh"
source "$SCRIPT_DIR/managed-reconcile.sh"
source "$SCRIPT_DIR/install-managed.sh"

SKILL_CATALOG=$(source_skills)
parse_install_options "$@"
if [[ "$help_only" == true ]]; then installation_usage; exit 0; fi
if [[ "$list_only" == true ]]; then
  while IFS= read -r source; do printf '%s\n' "${source##*/}"; done <<< "$SKILL_CATALOG"
  exit 0
fi
validate_catalog
validate_dependencies
installation_main
