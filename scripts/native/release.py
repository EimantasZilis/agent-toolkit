"""Generate deterministic native releases from the shared catalog."""

from __future__ import annotations

import hashlib
import json
import re
import shutil
import tempfile
import zipfile
from pathlib import Path

from .catalog import Catalog
from .render import Renderer
from .validation import validate_tree

PROVIDERS = ("codex", "claude")
PUBLISHER = "agent-toolkit contributors"


def write_json(path: Path, value: object) -> None:
    """Write stable JSON without timestamps or machine-specific paths."""
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n")


def file_inventory(root: Path) -> dict[str, str]:
    """Return relative names and SHA-256 hashes for all package files."""
    return {
        path.relative_to(root).as_posix(): hashlib.sha256(path.read_bytes()).hexdigest()
        for path in sorted(root.rglob("*"))
        if path.is_file()
    }


def package_name(selection: str, bundles: list[str]) -> str:
    """Return the stable plugin identifier for a skill or named bundle."""
    infix = "bundle-" if selection in bundles else ""
    return f"agent-toolkit-{infix}{selection}"


def plugin_manifest(name: str, version: str, provider: str) -> dict[str, object]:
    """Build provider metadata without claiming a verified publishing identity."""
    description = (
        f"Portable agent-toolkit skills: {name.removeprefix('agent-toolkit-')}."
    )
    manifest: dict[str, object] = {
        "name": name,
        "version": version,
        "description": description,
        "author": {"name": PUBLISHER},
    }
    if provider == "codex":
        manifest["skills"] = "./skills/"
        manifest["interface"] = {
            "displayName": name.removeprefix("agent-toolkit-")[:30],
            "shortDescription": "Portable engineering skills",
            "longDescription": description,
            "developerName": PUBLISHER,
            "category": "Developer Tools",
            "capabilities": [],
        }
    return manifest


def write_archive(package: Path, archive: Path) -> None:
    """Create a ZIP with stable ordering, timestamps, permissions, and root."""
    archive.parent.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(archive, "w", compression=zipfile.ZIP_DEFLATED) as output:
        for path in sorted(package.rglob("*")):
            if not path.is_file():
                continue
            info = zipfile.ZipInfo(
                path.relative_to(package).as_posix(), (1980, 1, 1, 0, 0, 0)
            )
            info.create_system = 3
            mode = 0o100755 if path.stat().st_mode & 0o111 else 0o100644
            info.external_attr = mode << 16
            info.compress_type = zipfile.ZIP_DEFLATED
            output.writestr(info, path.read_bytes())


def marketplace_entries(names: list[str], provider: str) -> list[dict[str, object]]:
    """Point each listing at a self-contained root in this release."""
    entries: list[dict[str, object]] = []
    for name in names:
        source = f"./plugins/{name}"
        entry: dict[str, object] = {"name": name, "source": source}
        if provider == "codex":
            entry["source"] = {"source": "local", "path": source}
            entry["policy"] = {
                "installation": "AVAILABLE",
                "authentication": "ON_INSTALL",
            }
            entry["category"] = "Developer Tools"
        entries.append(entry)
    return entries


class ReleaseBuilder:
    """Build a release with the same selection contract as managed installation."""

    def __init__(self, catalog: Catalog, version: str) -> None:
        if not re.fullmatch(
            r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", version
        ):
            raise ValueError("Release version must be X.Y.Z without leading zeros")
        self.catalog = catalog
        self.version = version

    def build(self, destination: Path) -> None:
        """Build and validate in a temporary directory; refuse existing output."""
        from .release_validation import validate_release  # Avoid a module cycle.

        if destination.exists() or destination.is_symlink():
            raise ValueError(f"Output already exists: {destination}")
        destination.parent.mkdir(parents=True, exist_ok=True)
        with tempfile.TemporaryDirectory(
            prefix=".native-", dir=destination.parent
        ) as temporary:
            staged = Path(temporary) / "release"
            self.populate(staged)
            validate_release(staged, (self.catalog.root,))
            staged.rename(destination)

    def populate(self, root: Path) -> None:
        """Generate each selection once per provider and save a catalog snapshot."""
        selections = sorted(self.catalog.sources) + self.catalog.bundles
        closures = {
            selection: self.catalog.select(selection) for selection in selections
        }
        packages = []
        for provider in PROVIDERS:
            for selection in selections:
                packages.append(
                    self.build_package(root, provider, selection, closures[selection])
                )
            self.write_marketplace(root / provider, provider, selections)
        write_json(
            root / "release.json",
            {
                "version": self.version,
                "catalog": self.catalog_snapshot(),
                "bundles": {
                    name: self.bundle_members(name) for name in self.catalog.bundles
                },
                "packages": packages,
            },
        )

    def build_package(
        self, root: Path, provider: str, selection: str, selected: list[str]
    ) -> dict[str, object]:
        """Render a package, attach its manifest, and hash its archive members."""
        name = package_name(selection, self.catalog.bundles)
        package = root / provider / "plugins" / name
        Renderer(self.catalog, provider, name).render(package, selected)
        manifest = plugin_manifest(name, self.version, provider)
        write_json(package / f".{provider}-plugin/plugin.json", manifest)
        shutil.copyfile(self.catalog.root / "LICENSE", package / "LICENSE")
        validate_tree(package, (self.catalog.root,))
        archive = root / provider / "zips" / f"{name}.zip"
        write_archive(package, archive)
        return {
            "provider": provider,
            "selection": selection,
            "name": name,
            "skills": selected,
            "files": file_inventory(package),
            "archive_sha256": hashlib.sha256(archive.read_bytes()).hexdigest(),
        }

    def bundle_members(self, name: str) -> list[str]:
        """Return requested membership without dependency expansion."""
        return self.catalog.query("bundle_members", name).splitlines()

    def catalog_snapshot(self) -> dict[str, object]:
        """Record source identities and required helpers for offline validation."""
        registry = (self.catalog.root / ".agents/dependencies.txt").read_text()
        dependencies = {
            fields[0]: fields[1:]
            for line in registry.splitlines()
            if (fields := line.split()) and not line.startswith("#")
        }
        explicit = set(
            (self.catalog.root / ".agents/providers/claude/explicit-only.txt")
            .read_text()
            .splitlines()
        )
        return {
            name: {
                "key": str(path.relative_to(self.catalog.root / ".agents/skills")),
                "requires": dependencies.get(name, []),
                "claude_explicit": str(
                    path.relative_to(self.catalog.root / ".agents/skills")
                )
                in explicit,
                "codex_explicit": "allow_implicit_invocation: false"
                in self.adapter_text(path),
            }
            for name, path in self.catalog.sources.items()
        }

    def adapter_text(self, source: Path) -> str:
        """Read optional provider policy for the offline catalog snapshot."""
        key = source.relative_to(self.catalog.root / ".agents/skills")
        adapter = (
            self.catalog.root
            / ".agents/providers/codex/skills"
            / key
            / "agents/openai.yaml"
        )
        return adapter.read_text() if adapter.exists() else ""

    def write_marketplace(
        self, root: Path, provider: str, selections: list[str]
    ) -> None:
        """Generate a provider-native local marketplace with relative sources."""
        names = [
            package_name(selection, self.catalog.bundles) for selection in selections
        ]
        marketplace: dict[str, object] = {
            "name": "agent-toolkit",
            "plugins": marketplace_entries(names, provider),
        }
        if provider == "claude":
            marketplace["owner"] = {"name": PUBLISHER}
            path = root / ".claude-plugin/marketplace.json"
        else:
            marketplace["interface"] = {"displayName": "agent-toolkit"}
            path = root / ".agents/plugins/marketplace.json"
        write_json(path, marketplace)
