"""Verify native skill trees with their source fixture removed from reach."""

from __future__ import annotations

import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

from scripts.native import Catalog
from scripts.native import Renderer
from scripts.native import validate_tree

REPO = Path(__file__).resolve().parents[1]


class NativeRenderTests(unittest.TestCase):
    """Exercise package relocation, provider adapters, and hostile resources."""

    def setUp(self) -> None:
        """Create an isolated source clone and output directory."""
        self.temporary = tempfile.TemporaryDirectory(prefix="native render ")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.source = self.root / "source"
        shutil.copytree(REPO / ".agents", self.source / ".agents")
        shutil.copytree(REPO / "scripts", self.source / "scripts")
        self.catalog = Catalog(self.source)

    def test_relocated_catalog_preserves_resources_and_provider_behavior(self) -> None:
        """GIVEN both providers WHEN the clone is hidden THEN resources stay usable."""
        selected = self.catalog.select("all")
        for provider in ("codex", "claude"):
            Renderer(self.catalog, provider, "test-bundle").render(
                self.root / provider, selected
            )
        self.source.rename(self.root / "unavailable")

        for provider in ("codex", "claude"):
            package = self.root / provider
            validate_tree(package, (self.source,))
            self.assertEqual(
                len(list(package.glob("skills/*/SKILL.md"))), len(selected)
            )
            self.assert_provider_metadata(package, provider)
            checker = package / "skills/python-standards/scripts/check_function_size.py"
            result = subprocess.run(
                ["python3", str(checker), str(checker)],
                cwd=self.root,
                capture_output=True,
                text=True,
                check=False,
            )
            self.assertEqual(result.returncode, 0, result.stderr)
            for name in selected:
                original = self.root / "unavailable/.agents/skills"
                directory = next(original.glob(f"*/{name}"))
                for resource in directory.rglob("*"):
                    if resource.is_file() and resource.suffix != ".md":
                        copied = (
                            package / "skills" / name / resource.relative_to(directory)
                        )
                        self.assertEqual(copied.read_bytes(), resource.read_bytes())

    def assert_provider_metadata(self, package: Path, provider: str) -> None:
        """Assert explicit-only behavior and the same-package helper link."""
        manifest = (package / "skills/ticket-to-plan/SKILL.md").read_text()
        self.assertIn("../plan-review/SKILL.md", manifest)
        if provider == "claude":
            self.assertIn("disable-model-invocation: true", manifest)
        else:
            metadata = package / "skills/plan-review/agents/openai.yaml"
            self.assertIn("allow_implicit_invocation: false", metadata.read_text())

    def test_standalone_keeps_optional_fallback_and_explicit_output_guidance(
        self,
    ) -> None:
        """GIVEN a standalone workflow WHEN rendered THEN optional fallbacks survive."""
        package = self.root / "standalone"
        selected = self.catalog.select("ticket-to-plan")

        Renderer(self.catalog, "claude", "agent-toolkit-ticket-to-plan").render(
            package, selected
        )

        validate_tree(package, (self.source,))
        self.assertEqual(selected, ["plan-review", "ticket-to-plan"])
        text = (package / "skills/ticket-to-plan/SKILL.md").read_text()
        self.assertIn("`quick-plan` (optional integration)", text)
        self.assertIn("otherwise silently follow", text)
        self.assertIn("Honor explicit user guidance", text)
        self.assertIn("<repo-root>/<branch-slug>/<skill-name>/...", text)
        self.assertFalse((self.root / "settings.env").exists())

    def test_rails_contains_ruby_once(self) -> None:
        """GIVEN Rails WHEN rendered THEN Ruby guidance is local and deduplicated."""
        selected = self.catalog.select("ruby-rails-standards ruby-standards")
        package = self.root / "rails"

        Renderer(self.catalog, "codex", "rails").render(package, selected)

        validate_tree(package)
        self.assertEqual(selected, ["ruby-rails-standards", "ruby-standards"])
        text = (package / "skills/ruby-rails-standards/SKILL.md").read_text()
        self.assertIn("../ruby-standards/SKILL.md", text)

    def test_rejects_symlink_resources(self) -> None:
        """GIVEN an external symlink WHEN copied THEN packaging fails closed."""
        source = self.catalog.sources["commit-message"]
        (source / "secret").symlink_to("/etc/passwd")

        with self.assertRaisesRegex(ValueError, "Unsupported source resource"):
            Renderer(self.catalog, "codex", "test").render(
                self.root / "unsafe", ["commit-message"]
            )

    def test_rejects_broken_escaping_and_clone_paths(self) -> None:
        """GIVEN hostile package content WHEN validated THEN it is rejected."""
        package = self.root / "hostile"
        package.mkdir()
        file = package / "SKILL.md"
        cases = ["[missing](absent.md)", "[escape](../secret)", str(self.source)]
        for content in cases:
            with self.subTest(content=content):
                file.write_text(content)

                with self.assertRaises(ValueError):
                    validate_tree(package, (self.source,))
