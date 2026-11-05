#!/usr/bin/env bash
# Run terminal setup from source; preserve explicit Bash installation.
set -euo pipefail

main() {
  local script_dir root status
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  root="$(cd "$script_dir/.." && pwd)"

  if [[ $# -eq 0 && -t 0 && -t 1 ]]; then
    if ! command -v go >/dev/null 2>&1; then
      printf 'Terminal setup requires Go (see %s/go.mod). Use make install-cli for Bash setup.\n' "$root" >&2
      return 1
    fi

    cd "$root"
    if go run ./cmd/agent-toolkit --root "$root"; then
      return 0
    else
      status=$?
      printf 'Source setup failed. See diagnostics above, or use make install-cli for Bash setup.\n' >&2
      return "$status"
    fi
  fi

  exec bash "$script_dir/install.sh" "$@"
}

main "$@"
