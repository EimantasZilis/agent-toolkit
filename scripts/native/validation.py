"""Reject clone dependencies and escaping or broken package resource links."""

from __future__ import annotations

import re
from pathlib import Path
from urllib.parse import unquote

from .render import LINK


def local_target(reference: str) -> str:
    """Return a local link path, excluding remote URLs and fragment-only links."""
    reference = reference.strip("<>")
    if reference.startswith("#") or re.match(r"https?://|mailto:", reference):
        return ""
    return unquote(reference.split("#")[0])


def validate_links(file: Path, root: Path) -> None:
    """Require every local Markdown resource to exist inside the package."""
    for match in LINK.finditer(file.read_text()):
        reference = local_target(match[2])
        if not reference:
            continue
        target = (file.parent / reference).resolve()
        if Path(reference).is_absolute() or root not in target.parents:
            raise ValueError(f"Escaping resource link in {file}: {reference}")
        if not target.exists():
            raise ValueError(f"Broken resource link in {file}: {reference}")


def validate_tree(root: Path, forbidden_roots: tuple[Path, ...] = ()) -> None:
    """Validate a relocated skill tree independently of its source catalog."""
    root = root.resolve()
    for path in sorted(root.rglob("*")):
        if path.is_symlink() or not (path.is_dir() or path.is_file()):
            raise ValueError(f"Unsupported package member: {path}")
        if not path.is_file():
            continue
        content = path.read_bytes()
        for forbidden in forbidden_roots:
            if str(forbidden).encode() in content:
                raise ValueError(f"Source clone path in {path}")
        if b".agents/skills/coding/python-standards/scripts/" in content:
            raise ValueError(f"Clone-dependent checker path in {path}")
        if path.suffix == ".md":
            validate_links(path, root)
