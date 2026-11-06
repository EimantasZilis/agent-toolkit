#!/usr/bin/env bash
# Remove reviewed owned installations and state in one managed transaction.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/shared.sh"
source "$SCRIPT_DIR/install-options.sh"
source "$SCRIPT_DIR/managed-options.sh"
source "$SCRIPT_DIR/managed-guidance.sh"
source "$SCRIPT_DIR/managed-reconcile.sh"
source "$SCRIPT_DIR/install-managed.sh"

[[ $# -eq 0 ]] || { printf 'Usage: scripts/uninstall.sh\n' >&2; exit 2; }
SKILL_CATALOG=$(source_skills)
validate_catalog
validate_dependencies
managed_load
managed_previous_concise=$managed_concise
managed_parse_options --no-providers --no-skills --concise false --full-template false --output-path ''
managed_resolve_selection
managed_plan
installation_preview
managed_expected_revision=$managed_state_revision
managed_confirm=true
managed_forget=true
managed_apply
printf 'Removed owned installation and saved settings. User files and native plugins were preserved.\n'
