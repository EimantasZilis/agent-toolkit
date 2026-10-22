"""Read the same catalog and selection contract used by managed installs."""

from __future__ import annotations

import os
import subprocess
from pathlib import Path


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
                str(Path(__file__).with_suffix(".sh")),
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
