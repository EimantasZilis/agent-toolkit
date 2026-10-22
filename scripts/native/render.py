"""Copy and render provider-native skills with package-relative resources."""

from __future__ import annotations

import os
import re
import shutil
from pathlib import Path
from typing import Match

from .catalog import Catalog

LINK = re.compile(r"\[([^\]]+)\]\(([^)]+)\)")
OUTPUT_SKILLS = {
    "developer-highlights",
    "next-commit",
    "next-phase",
    "plan-writing-conventions",
    "plan-execution-conventions",
    "plan-review",
    "requirements-to-tickets",
    "review-branch",
    "test-cases",
    "ticket-context",
    "ticket-to-plan",
    "to-markdown",
}


class Renderer:
    """Render one self-contained skill tree for a named provider package."""

    def __init__(self, catalog: Catalog, provider: str, package: str) -> None:
        if provider not in {"codex", "claude"}:
            raise ValueError(f"Unknown provider: {provider}")
        self.catalog = catalog
        self.provider = provider
        self.package = package
        explicit_file = catalog.root / ".agents/providers/claude/explicit-only.txt"
        self.explicit = set(explicit_file.read_text().splitlines())

    def render(self, destination: Path, selected: list[str]) -> None:
        """Copy selected skills and adapters into a new package directory."""
        destination.mkdir(parents=True, exist_ok=False)
        for name in selected:
            source = self.catalog.sources[name]
            target = destination / "skills" / name
            self.copy_resources(source, target)
            self.render_markdown(source, target, selected)
            self.copy_metadata(source, target)

    def copy_resources(self, source: Path, target: Path) -> None:
        """Reject symlinks and special files before copying source resources."""
        for path in [source, *source.rglob("*")]:
            if path.is_symlink() or not (path.is_file() or path.is_dir()):
                raise ValueError(f"Unsupported source resource: {path}")
        shutil.copytree(source, target)

    def render_markdown(self, source: Path, target: Path, selected: list[str]) -> None:
        """Rewrite all supporting Markdown, including executable examples."""
        for path in source.rglob("*.md"):
            output = target / path.relative_to(source)
            text = LINK.sub(
                lambda match: self.render_link(
                    match, path, output, target.parent, selected
                ),
                path.read_text(),
            )
            text = self.render_invocations(text, selected)
            text = self.render_checker(text, output, target)
            if path.name == "SKILL.md":
                text = self.render_manifest(text, source)
            output.write_text(text)

    def render_link(
        self,
        match: Match[str],
        source: Path,
        output: Path,
        skills_root: Path,
        selected: list[str],
    ) -> str:
        """Keep local resources relative and name absent optional integrations."""
        label, reference = match.groups()
        reference = reference.strip("<>")
        if re.match(r"[a-zA-Z][a-zA-Z0-9+.-]*:", reference) or reference.startswith(
            "#"
        ):
            return match.group()
        resolved = (source.parent / reference.split("#")[0]).resolve()
        for name, directory in self.catalog.sources.items():
            if resolved == directory / "SKILL.md":
                if name not in selected:
                    return f"`{name}` (optional integration)"
                target = skills_root / name / "SKILL.md"
                relative = os.path.relpath(target, output.parent)
                return f"[{label}]({relative})"
        return match.group()

    def render_invocations(self, text: str, selected: list[str]) -> str:
        """Render Claude invocations using this package's namespace."""
        if self.provider != "claude":
            return text
        for name in sorted(self.catalog.sources, key=len, reverse=True):
            invocation = f"/{self.package}:{name}" if name in selected else f"`{name}`"
            text = re.sub(r"\$" + re.escape(name) + r"(?![a-z0-9-])", invocation, text)
        return text

    def render_checker(self, text: str, output: Path, target: Path) -> str:
        """Resolve the bundled Python checker relative to its reference file."""
        original = (
            ".agents/skills/coding/python-standards/scripts/check_function_size.py"
        )
        if original in text:
            relative = os.path.relpath(
                target / "scripts/check_function_size.py", output.parent
            )
            text = text.replace(original, f'"<loaded-reference-directory>/{relative}"')
            text += (
                "\nResolve `<loaded-reference-directory>` to the directory containing "
                "this loaded reference before running the bundled checker.\n"
            )
        return text

    def render_manifest(self, text: str, source: Path) -> str:
        """Add invocation restrictions and task-local output guidance."""
        key = str(source.relative_to(self.catalog.root / ".agents/skills"))
        if self.provider == "claude" and key in self.explicit:
            text = text.replace("\n---", "\ndisable-model-invocation: true\n---", 1)
        if source.name in OUTPUT_SKILLS:
            policy = (self.catalog.root / ".agents/global/instructions.md").read_text()
            policy = (
                "## Markdown artifact outputs"
                + policy.split("## Markdown artifact outputs", 1)[1]
            )
            text += "\n" + policy
            text += (
                "\n### Native output policy\n\n"
                "Honor explicit user guidance configuring an output location, including "
                "a centralized output root. Otherwise use "
                "`<repo-root>/<branch-slug>/<skill-name>/...`. "
                "These task-local defaults do not modify global settings.\n"
            )
        return text

    def copy_metadata(self, source: Path, target: Path) -> None:
        """Copy the provider adapter rather than linking to the source clone."""
        key = source.relative_to(self.catalog.root / ".agents/skills")
        adapter = (
            self.catalog.root
            / ".agents/providers/codex/skills"
            / key
            / "agents/openai.yaml"
        )
        if self.provider == "codex" and adapter.exists():
            if any(path.is_symlink() for path in [adapter, *adapter.parents]):
                raise ValueError(f"Symlinked adapter: {adapter}")
            (target / "agents").mkdir(exist_ok=True)
            shutil.copyfile(adapter, target / "agents/openai.yaml")
