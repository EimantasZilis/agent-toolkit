"""Verify unattended saved-state refresh, refusals and rollback."""

from __future__ import annotations

import hashlib
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]


class ManagedUpdateTests(unittest.TestCase):
    """Run the update entrypoint against isolated provider homes without Go."""

    def setUp(self) -> None:
        """Create disposable source, canonical settings, and provider roots."""
        temporary = tempfile.TemporaryDirectory(prefix="managed update ")
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.toolkit = self.root / "toolkit"
        for name in ("scripts", ".agents"):
            shutil.copytree(ROOT / name, self.toolkit / name)
        shutil.copyfile(ROOT / "Makefile", self.toolkit / "Makefile")
        self.environment = dict(os.environ)
        self.environment.pop("MARKDOWN_OUTPUT_PATH", None)
        self.environment.update(
            PATH="/usr/bin:/bin:/usr/sbin:/sbin",
            AGENT_TOOLKIT_HOME=str(self.toolkit),
            AGENTS_HOME=str(self.root / "agents"),
            CODEX_HOME=str(self.root / "codex"),
            CLAUDE_HOME=str(self.root / "claude"),
        )
        self.source = self.toolkit / ".agents/skills/scoping/ticket-to-plan"

    def command(self, name: str, *arguments: str, expected: int = 0) -> str:
        """Execute a real shell command and assert its status with diagnostics."""
        result = subprocess.run(
            [str(self.toolkit / "scripts" / name), *arguments],
            cwd=self.root,
            env=self.environment,
            capture_output=True,
            text=True,
            timeout=90,
        )
        self.assertEqual(result.returncode, expected, result.stdout + result.stderr)
        return result.stdout + result.stderr

    def install(self) -> None:
        """Save two providers and non-default output and guidance preferences."""
        self.command(
            "install.sh",
            "--provider",
            "codex",
            "--provider",
            "claude",
            "--skill",
            "ticket-to-plan",
            "--concise",
            "true",
            "--output-path",
            str(self.root / "saved output"),
        )

    def destinations(self) -> list[Path]:
        """Return the installed parent skill for each saved provider."""
        return [
            self.root / "agents/skills/agent-toolkit/ticket-to-plan",
            self.root / "claude/skills/ticket-to-plan",
        ]

    def snapshot(self) -> dict[str, str]:
        """Hash managed files and literal links without following source resources."""
        result = {}
        paths = [self.toolkit / "config"]
        paths.extend(self.root / name for name in ("agents", "codex", "claude"))
        for path in paths:
            for item in [path, *path.rglob("*")] if path.is_dir() else [path]:
                if item.is_symlink():
                    result[str(item)] = "link:" + os.readlink(item)
                elif item.is_file():
                    result[str(item)] = hashlib.sha256(item.read_bytes()).hexdigest()
        return result

    def test_refreshes_both_providers_and_preserves_preferences(self) -> None:
        """GIVEN changed source WHEN updated THEN instructions and new links refresh."""
        self.install()
        settings = self.toolkit / "config/settings.env"
        saved = settings.read_bytes()
        manifest = self.source / "SKILL.md"
        manifest.write_text(manifest.read_text() + "\nUpdated instruction.\n")
        (self.source / "new resource.md").write_text("New resource.\n")
        before = self.snapshot()
        self.environment["MARKDOWN_OUTPUT_PATH"] = "/temporary-override"

        self.command("update.sh", "--dry-run")
        self.assertEqual(self.snapshot(), before)
        output = self.command("update.sh")

        for destination in self.destinations():
            text = (destination / "SKILL.md").read_text()
            self.assertIn("Updated instruction.", text)
            self.assertIn(str(self.root / "saved output"), text)
            self.assertNotIn("/temporary-override", text)
            self.assertTrue((destination / "new resource.md").is_symlink())
        self.assertEqual(settings.read_bytes(), saved)
        self.assertIn("Start new agent sessions", output)
        refreshed = self.snapshot()
        self.command("update.sh")
        self.assertEqual(self.snapshot(), refreshed)

    def test_retained_helpers_stay_installed(self) -> None:
        """GIVEN a retained helper WHEN refreshed THEN its saved role and files remain."""
        self.install()
        self.command("install.sh", "--skill", "commit-message", "--confirm-removals")
        saved = (self.toolkit / "config/settings.env").read_bytes()

        self.command("update.sh")

        self.assertIn(b"RETAINED_HELPERS=concise-output plan-review", saved)
        self.assertEqual((self.toolkit / "config/settings.env").read_bytes(), saved)
        for destination in self.destinations():
            self.assertTrue((destination.parent / "plan-review/SKILL.md").is_file())
            self.assertFalse(destination.exists())

    def test_missing_state_does_not_install_defaults(self) -> None:
        """GIVEN no observed installation WHEN updating THEN setup is required."""
        before = self.snapshot()

        output = self.command("update.sh", expected=2)
        self.assertEqual(self.snapshot(), before)
        self.assertIn("make install-cli", output)
        self.assertFalse((self.toolkit / "config").exists())

    def test_observed_files_without_saved_state_require_setup(self) -> None:
        """GIVEN installed files without settings WHEN updated THEN no choices are inferred."""
        self.install()
        shutil.rmtree(self.toolkit / "config")
        before = self.snapshot()

        output = self.command("update.sh", expected=2)

        self.assertIn("make install-cli", output)
        self.assertEqual(self.snapshot(), before)
        self.assertFalse((self.toolkit / "config").exists())

    def test_unknown_saved_skill_and_removed_catalog_entry_stop_read_only(self) -> None:
        """GIVEN stale saved skills WHEN updating THEN no provider or state writes occur."""
        self.install()
        settings = self.toolkit / "config/settings.env"
        original = settings.read_text()
        settings.write_text(original.replace("ticket-to-plan", "unknown-skill"))
        before = self.snapshot()

        output = self.command("update.sh", expected=2)
        self.assertEqual(self.snapshot(), before)
        settings.write_text(original)
        other = self.toolkit / ".agents/skills/productivity/commit-message"
        self.command("install.sh", "--skill", "commit-message", "--confirm-removals")
        shutil.rmtree(other)
        before = self.snapshot()
        self.command("update.sh", expected=2)

        self.assertIn("review setup", output)
        self.assertEqual(self.snapshot(), before)

    def test_user_edits_and_proposed_removals_stop_read_only(self) -> None:
        """GIVEN conflicts or a narrower desired set WHEN updating THEN nothing applies."""
        self.install()
        destination = self.destinations()[0] / "SKILL.md"
        original = destination.read_bytes()
        destination.write_bytes(original + b"\nUser edits\n")
        before = self.snapshot()

        self.command("update.sh", expected=4)
        self.assertEqual(self.snapshot(), before)
        destination.write_bytes(original)
        settings = self.toolkit / "config/settings.env"
        settings.write_text(
            settings.read_text().replace("PROVIDERS=claude codex", "PROVIDERS=codex")
        )
        before = self.snapshot()
        self.command("update.sh", expected=4)

        self.assertEqual(self.snapshot(), before)

    def test_moved_resource_requires_review_instead_of_claiming_a_dangling_link(
        self,
    ) -> None:
        """GIVEN a moved owned resource WHEN updating THEN ambiguous ownership stops writes."""
        (self.source / "old resource.md").write_text("Resource.\n")
        self.install()
        (self.source / "old resource.md").rename(self.source / "moved resource.md")
        before = self.snapshot()

        output = self.command("update.sh", expected=4)

        self.assertIn("review make install-cli", output)
        self.assertEqual(self.snapshot(), before)
        for destination in self.destinations():
            self.assertTrue((destination / "old resource.md").is_symlink())
            self.assertFalse((destination / "moved resource.md").exists())

    def test_apply_failures_restore_both_providers_and_state(self) -> None:
        """GIVEN changed instructions WHEN provider or state persistence fails THEN rollback restores."""
        self.install()
        manifest = self.source / "SKILL.md"
        manifest.write_text(manifest.read_text() + "\nNew instructions.\n")
        before = self.snapshot()

        for variable, value, status in (
            ("AGENT_TOOLKIT_FAIL_PROVIDER", "claude", 97),
            ("AGENT_TOOLKIT_FAIL_STATE", "after-settings", 1),
        ):
            self.environment[variable] = value
            self.command("update.sh", expected=status)
            del self.environment[variable]
            self.assertEqual(self.snapshot(), before)

    def test_source_change_after_preview_rejects_stale_apply(self) -> None:
        """GIVEN a source race WHEN update reaches apply THEN its revision is rejected."""
        self.install()
        commands = self.root / "bin"
        commands.mkdir()
        shasum = commands / "shasum"
        shasum.write_text(
            "#!/bin/bash\n"
            "if [[ $# == 2 ]]; then\n"
            "  input=$(cat)\n"
            '  printf "%s\\n" "$input" | /usr/bin/shasum "$@"\n'
            '  if [[ "$input" == *"PATH $AGENT_TOOLKIT_HOME/config/settings.env"* && ! -e "$UPDATE_RACE" ]]; then\n'
            '    touch "$UPDATE_RACE"\n'
            '    printf "\\nSource changed during preview.\\n" >> "$UPDATE_SOURCE"\n'
            "  fi\n"
            'else exec /usr/bin/shasum "$@"; fi\n'
        )
        shasum.chmod(0o700)
        self.environment.update(
            PATH=str(commands) + ":" + self.environment["PATH"],
            UPDATE_RACE=str(self.root / "raced"),
            UPDATE_SOURCE=str(self.source / "SKILL.md"),
        )
        before = self.snapshot()

        output = self.command("update.sh", expected=3)

        self.assertIn("stale preview", output)
        self.assertEqual(self.snapshot(), before)

    def test_make_update_invokes_the_saved_refresh(self) -> None:
        """GIVEN saved providers WHEN make update runs THEN Bash refresh succeeds without Go."""
        self.install()

        result = subprocess.run(
            ["make", "update"],
            cwd=self.toolkit,
            env=self.environment,
            capture_output=True,
            text=True,
            timeout=90,
        )

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("Updated saved installation", result.stdout)


if __name__ == "__main__":
    unittest.main()
