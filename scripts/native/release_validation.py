"""Validate release contracts and ZIP contents without access to a source clone."""

from __future__ import annotations

import hashlib
import json
import re
import zipfile
from pathlib import Path

from .release import PROVIDERS
from .release import file_inventory
from .release import package_name
from .validation import validate_tree


def read_json(path: Path) -> dict[str, object]:
    """Read a JSON object, rejecting arrays and primitive roots."""
    data = json.loads(path.read_text())
    if not isinstance(data, dict):
        raise ValueError(f"Expected a JSON object: {path}")
    return data


def require(condition: bool, message: str) -> None:
    """Reject a broken release contract with an actionable diagnostic."""
    if not condition:
        raise ValueError(message)


def helper_closure(requested: list[str], catalog: dict[str, dict]) -> list[str]:
    """Expand required helper edges in the release's offline catalog snapshot."""
    selected: set[str] = set()
    pending = list(requested)
    while pending:
        name = pending.pop()
        require(name in catalog, f"Unknown catalog skill: {name}")
        if name in selected:
            continue
        selected.add(name)
        pending.extend(catalog[name]["requires"])
    return sorted(selected)


def validate_archive(archive: Path, package: Path, record: dict) -> None:
    """Check archive checksum and exact members without extracting untrusted paths."""
    digest = hashlib.sha256(archive.read_bytes()).hexdigest()
    require(digest == record["archive_sha256"], f"Archive checksum mismatch: {archive}")
    with zipfile.ZipFile(archive) as bundle:
        require(
            len(bundle.namelist()) == len(record["files"]),
            f"Duplicate or missing ZIP member: {archive}",
        )
        require(
            set(bundle.namelist()) == set(record["files"]),
            f"Unexpected ZIP members: {archive}",
        )
        for name in bundle.namelist():
            require(
                bundle.read(name) == (package / name).read_bytes(),
                f"ZIP content mismatch: {name}",
            )
            mode = bundle.getinfo(name).external_attr >> 16
            require(
                mode in {0o100644, 0o100755}, f"Unsupported ZIP member mode: {name}"
            )


def validate_manifest(package: Path, record: dict, version: str) -> None:
    """Check required provider fields and directory declarations."""
    provider = record["provider"]
    manifest = read_json(package / f".{provider}-plugin/plugin.json")
    require(
        manifest.get("name") == record["name"], f"Manifest identity mismatch: {package}"
    )
    require(manifest.get("version") == version, f"Manifest version mismatch: {package}")
    require(bool(manifest.get("description")), f"Missing description: {package}")
    require(bool(manifest.get("author", {}).get("name")), f"Missing author: {package}")
    if provider == "codex":
        require(
            manifest.get("skills") == "./skills/",
            f"Invalid skills declaration: {package}",
        )
        interface = manifest.get("interface", {})
        for field in (
            "displayName",
            "shortDescription",
            "longDescription",
            "developerName",
            "category",
        ):
            require(bool(interface.get(field)), f"Missing interface.{field}: {package}")
        require(
            isinstance(interface.get("capabilities"), list),
            f"Missing capabilities: {package}",
        )


def validate_provider_policy(
    package: Path, provider: str, name: str, info: dict
) -> None:
    """Check explicit invocation restrictions in the copied provider adapter."""
    skill = package / "skills" / name
    manifest = (skill / "SKILL.md").read_text()
    require(manifest.startswith("---\n"), f"Missing skill front matter: {skill}")
    front = manifest.split("---", 2)[1]
    require(
        bool(re.search(r"^name: " + re.escape(name) + r"$", front, re.MULTILINE)),
        f"Skill identity mismatch: {skill}",
    )
    require(
        bool(re.search(r"^description: .+", front, re.MULTILINE)),
        f"Missing skill description: {skill}",
    )
    if provider == "claude" and info["claude_explicit"]:
        require(
            "disable-model-invocation: true\n" in (skill / "SKILL.md").read_text(),
            f"Missing explicit-only policy: {skill}",
        )
    if provider == "codex" and info["codex_explicit"]:
        adapter = skill / "agents/openai.yaml"
        require(
            adapter.is_file()
            and "allow_implicit_invocation: false" in adapter.read_text(),
            f"Missing explicit-only policy: {skill}",
        )


def validate_package(root: Path, record: dict, release: dict) -> None:
    """Validate resource containment, helper closure, manifests, and hashes."""
    provider, name = record["provider"], record["name"]
    require(provider in PROVIDERS, f"Unknown provider: {provider}")
    require(
        bool(re.fullmatch(r"agent-toolkit-[a-z0-9]+(-[a-z0-9]+)*", name)),
        f"Invalid package name: {name}",
    )
    package = root / provider / "plugins" / name
    validate_tree(package)
    validate_manifest(package, record, release["version"])
    require(
        file_inventory(package) == record["files"],
        f"Package inventory mismatch: {name}",
    )
    requested = release["bundles"].get(record["selection"], [record["selection"]])
    expected = helper_closure(requested, release["catalog"])
    require(record["skills"] == expected, f"Incomplete helper closure: {name}")
    actual = sorted(path.parent.name for path in package.glob("skills/*/SKILL.md"))
    require(actual == expected, f"Skill membership mismatch: {name}")
    for skill in expected:
        validate_provider_policy(package, provider, skill, release["catalog"][skill])
    validate_archive(root / provider / "zips" / f"{name}.zip", package, record)


def validate_marketplace(root: Path, provider: str, names: list[str]) -> None:
    """Require each marketplace entry to resolve to its generated package root."""
    relative = (
        ".claude-plugin/marketplace.json"
        if provider == "claude"
        else ".agents/plugins/marketplace.json"
    )
    marketplace = read_json(root / provider / relative)
    require(marketplace.get("name") == "agent-toolkit", "Invalid marketplace name")
    entries = marketplace["plugins"]
    require(
        sorted(entry["name"] for entry in entries) == sorted(names),
        f"Marketplace membership mismatch: {provider}",
    )
    if provider == "claude":
        require(
            bool(marketplace.get("owner", {}).get("name")), "Missing marketplace owner"
        )
    for entry in entries:
        source = entry["source"]
        if provider == "codex":
            require(source["source"] == "local", "Invalid marketplace source type")
            require(
                entry["policy"]
                == {"installation": "AVAILABLE", "authentication": "ON_INSTALL"},
                "Invalid marketplace policy",
            )
            source = source["path"]
        require(
            source == f"./plugins/{entry['name']}",
            f"Invalid marketplace source: {source}",
        )


def validate_release(root: Path, forbidden_roots: tuple[Path, ...] = ()) -> None:
    """Validate a complete release after relocation, with no provider mutation."""
    validate_tree(root, forbidden_roots)
    release = read_json(root / "release.json")
    expected = {
        (provider, package_name(selection, list(release["bundles"])))
        for provider in PROVIDERS
        for selection in [*release["catalog"], *release["bundles"]]
    }
    actual = [(record["provider"], record["name"]) for record in release["packages"]]
    require(
        len(actual) == len(expected) and set(actual) == expected,
        "Release package membership mismatch",
    )
    for record in release["packages"]:
        validate_package(root, record, release)
    for provider in PROVIDERS:
        names = [name for source, name in expected if source == provider]
        validate_marketplace(root, provider, names)
        directories = sorted(
            path.name for path in (root / provider / "plugins").iterdir()
        )
        require(
            directories == sorted(names), f"Unexpected package directories: {provider}"
        )
