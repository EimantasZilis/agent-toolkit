#!/usr/bin/env bash
# Route terminal setup through a pinned TUI; preserve explicit Bash installation.
set -euo pipefail

main() {
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

  if [[ $# -eq 0 && -t 0 && -t 1 && -f "$script_dir/tui-release.json" ]]; then
    exec python3 "$script_dir/launch_tui.py"
  fi

  exec bash "$script_dir/install.sh" "$@"
}

main "$@"
