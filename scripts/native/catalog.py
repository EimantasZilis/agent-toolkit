"""Read the same catalog and selection contract used by managed installs."""

from __future__ import annotations

import os
import subprocess
from pathlib import Path


BACKEND_QUERY = """
source "$1/scripts/shared.sh"
SKILL_CATALOG=$(source_skills)
validate_catalog
validate_dependencies
GLOBAL_GUIDANCE=false
CONCISE_OUTPUT=false
SKILL_SELECTION=$3
if [[ "$2" == resolve_selection ]]; then
  resolve_selection
  printf '%s' "$selected_names"
else
  "$2" "$3"
fi
"""


class Catalog:
    """Resolve source identities and helper closures through the Bash backend."""

    def __init__(self, root: Path) -> None:
        self.root = root.resolve()
        records = self.query("catalog_records").splitlines()
        self.sources = {
            name: self.root / ".agents/skills" / key
            for name, _category, key in (record.split("\t") for record in records)
        }
        self.bundles = self.query("bundle_names").splitlines()

    def query(self, operation: str, selection: str = "") -> str:
        """Query a fixed backend operation with data passed as arguments."""
        if operation not in {
            "catalog_records",
            "bundle_names",
            "bundle_members",
            "resolve_selection",
        }:
            raise ValueError(f"Unknown catalog operation: {operation}")

        result = subprocess.run(
            [
                "bash",
                "-euo",
                "pipefail",
                "-c",
                BACKEND_QUERY,
                "catalog",
                str(self.root),
                operation,
                selection,
            ],
            env={**os.environ, "AGENT_TOOLKIT_HOME": str(self.root)},
            check=True,
            capture_output=True,
            text=True,
        )
        return result.stdout

    def select(self, selection: str) -> list[str]:
        """Return a sorted, deduplicated transitive skill closure."""
        return sorted(set(self.query("resolve_selection", selection).splitlines()))
