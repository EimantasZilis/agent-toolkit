#!/usr/bin/env bash
# Exercise catalog selections and helper closure without provider mutations.
set -euo pipefail
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../scripts" && pwd)/shared.sh"

TEST_ROOT=$(mktemp -d)
trap 'rm -rf -- "$TEST_ROOT"' EXIT
cp -R "$AGENT_TOOLKIT_HOME/.agents" "$TEST_ROOT/"
AGENT_TOOLKIT_HOME=$TEST_ROOT
SKILL_CATALOG=$(source_skills)
GLOBAL_GUIDANCE=false
CONCISE_OUTPUT=false
fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }

validate_catalog
validate_dependencies
[[ $(bundle_members starter) == $'concise-output\ncommit-message' ]] || fail starter
[[ $(bundle_members all | wc -l) -eq $(catalog_records | wc -l) ]] || fail all
for category in $(catalog_records | cut -f2 | sort -u); do
  expected=$(catalog_records | awk -F '\t' -v category="$category" '$2 == category { print $1 }')
  [[ $(bundle_members "$category") == "$expected" ]] || fail "$category"
done

# A transitive diamond must include a helper just once.
printf '\nplan-review ruby-rails-standards\ncommit-message ruby-standards\n' >> "$TEST_ROOT/.agents/dependencies.txt"
SKILL_SELECTION='ticket-to-plan commit-message'
resolve_selection
[[ $(printf '%s' "$selected_names" | sort -u | wc -l) -eq 5 ]] || fail closure
[[ $(printf '%s' "$selected_names" | grep -c '^ruby-standards$') -eq 1 ]] || fail deduplication

printf '\ninvalid missing-skill\n' >> "$TEST_ROOT/.agents/bundles.txt"
if validate_catalog >/dev/null 2>&1; then fail 'accepted unknown bundle member'; fi
printf 'starter concise-output\nstarter commit-message\n' > "$TEST_ROOT/.agents/bundles.txt"
if validate_catalog >/dev/null 2>&1; then fail 'accepted duplicate bundle'; fi
printf 'coding concise-output\nstarter commit-message\n' > "$TEST_ROOT/.agents/bundles.txt"
if validate_catalog >/dev/null 2>&1; then fail 'accepted category collision'; fi
printf 'starter\n' > "$TEST_ROOT/.agents/bundles.txt"
if validate_catalog >/dev/null 2>&1; then fail 'accepted empty bundle'; fi
printf 'OK: catalog and bundle tests passed\n'
