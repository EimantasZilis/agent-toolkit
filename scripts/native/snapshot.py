"""Build and validate self-contained Git marketplace snapshots."""

from __future__ import annotations

import hashlib
import re
import shutil
import subprocess
import tempfile
from pathlib import Path

from .catalog import Catalog
from .release import PROVIDERS
from .release import ReleaseBuilder
from .release import package_name
from .release import write_json
from .release_validation import read_json
from .release_validation import require
from .release_validation import validate_package
from .validation import tree_members
from .validation import validate_tree

CATALOG_PATHS = {
    "codex": ".agents/plugins/marketplace.json",
    "claude": ".claude-plugin/marketplace.json",
}
INPUTS = (".agents", "scripts", "LICENSE")


def git(root: Path, *arguments: str) -> str:
    """Run a read-only Git query scoped to the source repository."""
    return subprocess.check_output(
        ["git", "-C", str(root), *arguments], text=True
    ).strip()


def source_commit(root: Path) -> str:
    """Require committed inputs, including ignored skill resources we would copy."""
    require(
        Path(git(root, "rev-parse", "--show-toplevel")).resolve() == root.resolve(),
        "Snapshot source must be a Git repository root",
    )
    changed = git(
        root, "-c", "core.filemode=true", "diff", "--name-only", "HEAD", "--", *INPUTS
    )
    untracked = git(root, "ls-files", "--others", "--exclude-standard", "--", *INPUTS)
    require(
        not changed and not untracked,
        "Commit source inputs before generating a snapshot",
    )
    tracked = set(git(root, "ls-files", "-z", "--", *INPUTS).split("\0"))
    for name in INPUTS:
        base = root / name
        for path in [base, *base.rglob("*")] if base.is_dir() else [base]:
            require(not path.is_symlink(), f"Symlink source input: {path}")
            if path.is_file():
                relative = path.relative_to(root)
                if (
                    name == "scripts"
                    and "__pycache__" in relative.parts
                    and path.suffix == ".pyc"
                ):
                    continue
                require(
                    relative.as_posix() in tracked,
                    f"Untracked or ignored source input: {path}",
                )
    return git(root, "rev-parse", "HEAD")


def inventory(root: Path) -> dict[str, str]:
    """Hash payload files, excluding the inventory document and root Git metadata."""
    return {
        path.relative_to(root).as_posix(): hashlib.sha256(path.read_bytes()).hexdigest()
        for path in tree_members(root, frozenset({".git"}))
        if path.is_file() and path != root / "snapshot.json"
    }


def modes(root: Path) -> dict[str, int]:
    """Record Git executable modes independently of the caller's umask."""
    return {
        name: 0o100755 if (root / name).stat().st_mode & 0o100 else 0o100644
        for name in inventory(root)
    }


def combine_catalogs(root: Path) -> None:
    """Move both catalogs to the Git root and point at distinct provider trees."""
    for provider, relative in CATALOG_PATHS.items():
        original = root / provider / relative
        catalog = read_json(original)
        for entry in catalog["plugins"]:
            path = f"./{provider}/plugins/{entry['name']}"
            if provider == "codex":
                entry["source"]["path"] = path
            else:
                entry["source"] = path
        write_json(root / relative, catalog)
        shutil.rmtree(root / provider / relative.split("/")[0])
        shutil.rmtree(root / provider / "zips")
    release = read_json(root / "release.json")
    for record in release["packages"]:
        del record["archive_sha256"]
    write_json(root / "release.json", release)


def build_snapshot(source: Path, version: str, destination: Path) -> None:
    """Stage and validate a snapshot from clean inputs without overwriting output."""
    source = source.resolve()
    commit = source_commit(source)
    builder = ReleaseBuilder(Catalog(source), version)
    require(
        not destination.exists() and not destination.is_symlink(),
        f"Output already exists: {destination}",
    )
    destination.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(
        prefix=".snapshot-", dir=destination.parent
    ) as temporary:
        root = Path(temporary) / "snapshot"
        builder.populate(root)
        combine_catalogs(root)
        write_json(
            root / "snapshot.json",
            {
                "schema_version": 1,
                "source_commit": commit,
                "version": version,
                "files": inventory(root),
                "modes": modes(root),
            },
        )
        # Detect edits made while rendering, including a source commit change.
        require(
            source_commit(source) == commit, "Source changed during snapshot generation"
        )
        validate_snapshot(root, (source,))
        root.rename(destination)


def validate_catalogs(root: Path, names: dict[str, list[str]]) -> None:
    """Require exact catalog membership and confined, provider-specific sources."""
    for provider, relative in CATALOG_PATHS.items():
        catalog = read_json(root / relative)
        require(catalog.get("name") == "agent-toolkit", "Invalid marketplace name")
        entries = catalog["plugins"]
        require(
            sorted(entry["name"] for entry in entries) == sorted(names[provider]),
            f"Marketplace membership mismatch: {provider}",
        )
        if provider == "claude":
            require(
                bool(catalog.get("owner", {}).get("name")), "Missing marketplace owner"
            )
        for entry in entries:
            path = entry["source"]
            if provider == "codex":
                require(path["source"] == "local", "Invalid Codex source type")
                require(
                    entry["policy"]
                    == {"installation": "AVAILABLE", "authentication": "ON_INSTALL"},
                    "Invalid Codex marketplace policy",
                )
                path = path["path"]
            require(
                path == f"./{provider}/plugins/{entry['name']}",
                f"Escaping or cross-provider marketplace path: {path}",
            )


def validate_packages(root: Path, release: dict) -> dict[str, list[str]]:
    """Reuse archive-release package checks without requiring ZIP copies."""
    expected = {
        (provider, package_name(selection, list(release["bundles"])))
        for provider in PROVIDERS
        for selection in [*release["catalog"], *release["bundles"]]
    }
    actual = [(record["provider"], record["name"]) for record in release["packages"]]
    require(
        len(actual) == len(expected) and set(actual) == expected,
        "Snapshot package membership mismatch",
    )
    names = {
        provider: sorted(name for owner, name in expected if owner == provider)
        for provider in PROVIDERS
    }
    for record in release["packages"]:
        validate_package(root, record, release, archives=False)
        require(
            (
                root / record["provider"] / "plugins" / record["name"] / "LICENSE"
            ).is_file(),
            "Missing package license",
        )
    for provider in PROVIDERS:
        require(
            sorted(path.name for path in (root / provider / "plugins").iterdir())
            == names[provider],
            f"Unexpected package directories: {provider}",
        )
    return names


def validate_snapshot(root: Path, forbidden_roots: tuple[Path, ...] = ()) -> None:
    """Validate relocated catalogs, provenance, package contracts, hashes, and modes."""
    require(not root.is_symlink(), "Snapshot root must not be a symlink")
    validate_tree(root, forbidden_roots, ignored_root_names=frozenset({".git"}))
    metadata = read_json(root / "snapshot.json")
    require(metadata.get("schema_version") == 1, "Unsupported snapshot schema")
    require(
        isinstance(metadata.get("source_commit"), str)
        and bool(re.fullmatch(r"[a-f0-9]{40}|[a-f0-9]{64}", metadata["source_commit"])),
        "Invalid source commit provenance",
    )
    require(
        metadata.get("files") == inventory(root), "Snapshot file inventory mismatch"
    )
    require(
        metadata.get("modes") == modes(root), "Snapshot permission inventory mismatch"
    )
    release = read_json(root / "release.json")
    require(
        isinstance(release.get("version"), str)
        and bool(
            re.fullmatch(
                r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", release["version"]
            )
        )
        and release["version"] == metadata.get("version"),
        "Invalid snapshot version",
    )
    names = validate_packages(root, release)
    validate_catalogs(root, names)
