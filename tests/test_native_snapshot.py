"""Validate combined Git marketplace snapshots and clean-source provenance."""

from __future__ import annotations

import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

from scripts.native.release import file_inventory
from scripts.native.release import write_json
from scripts.native.snapshot import build_snapshot
from scripts.native.snapshot import inventory
from scripts.native.snapshot import modes
from scripts.native.snapshot import validate_snapshot

ROOT = Path(__file__).resolve().parents[1]


class NativeSnapshotTests(unittest.TestCase):
    """Build once from committed fixture inputs and corrupt isolated copies."""

    @classmethod
    def setUpClass(cls) -> None:
        """Commit an isolated source tree and generate a complete combined snapshot."""
        cls.temporary = tempfile.TemporaryDirectory(prefix="native snapshot ")
        cls.addClassCleanup(cls.temporary.cleanup)
        cls.root = Path(cls.temporary.name)
        cls.source = cls.root / "source"
        for name in ("scripts", ".agents"):
            shutil.copytree(
                ROOT / name,
                cls.source / name,
                ignore=shutil.ignore_patterns("__pycache__"),
            )
        for name in ("LICENSE", ".gitignore"):
            shutil.copyfile(ROOT / name, cls.source / name)
        cls.git("init", "-q")
        cls.git("add", ".")
        cls.git(
            "-c",
            "user.name=Fixture",
            "-c",
            "user.email=fixture@example.test",
            "commit",
            "-qm",
            "Snapshot source fixture",
        )
        cls.commit = cls.git("rev-parse", "HEAD")
        cls.snapshot = cls.root / "snapshot"
        build_snapshot(cls.source, "0.1.0", cls.snapshot)

    @classmethod
    def git(cls, *arguments: str) -> str:
        """Operate only on the disposable fixture's Git repository."""
        return subprocess.check_output(
            ["git", "-C", str(cls.source), *arguments], text=True
        ).strip()

    def copy(self, name: str) -> Path:
        """Give each mutation a separate snapshot tree."""
        destination = self.root / name
        shutil.copytree(self.snapshot, destination)
        return destination

    def reseal(self, root: Path) -> None:
        """Update checksums so hostile contract tests exercise deeper validation."""
        metadata = json.loads((root / "snapshot.json").read_text())
        metadata.update(files=inventory(root), modes=modes(root))
        write_json(root / "snapshot.json", metadata)

    def test_root_catalogs_resolve_distinct_provider_trees(self) -> None:
        """GIVEN one snapshot WHEN catalogs resolve THEN both providers and helpers validate."""
        validate_snapshot(self.snapshot, (self.source,))
        metadata = json.loads((self.snapshot / "snapshot.json").read_text())

        self.assertEqual(metadata["source_commit"], self.commit)
        self.assertEqual(metadata["version"], "0.1.0")
        for provider, relative in (
            ("codex", ".agents/plugins/marketplace.json"),
            ("claude", ".claude-plugin/marketplace.json"),
        ):
            catalog = json.loads((self.snapshot / relative).read_text())
            for entry in catalog["plugins"]:
                path = (
                    entry["source"]["path"] if provider == "codex" else entry["source"]
                )
                self.assertEqual(path, f"./{provider}/plugins/{entry['name']}")
                self.assertTrue((self.snapshot / path / "LICENSE").is_file())
            parent = self.snapshot / provider / "plugins/ticket-to-plan/skills"
            self.assertTrue((parent / "plan-review/SKILL.md").is_file())
            self.assertFalse((self.snapshot / provider / "zips").exists())

    def test_deterministic_output_and_existing_destination_protection(self) -> None:
        """GIVEN the same clean source WHEN rebuilt THEN content and Git modes match."""
        second = self.root / "second"

        build_snapshot(self.source, "0.1.0", second)

        self.assertEqual(file_inventory(second), file_inventory(self.snapshot))
        self.assertEqual(modes(second), modes(self.snapshot))
        with self.assertRaisesRegex(ValueError, "already exists"):
            build_snapshot(self.source, "0.1.0", second)

    def test_relocated_snapshot_validates_without_source_inside_git(self) -> None:
        """GIVEN relocated output WHEN the source disappears THEN standalone validation works."""
        relocated = self.copy("relocated")
        hidden = self.root / "hidden-source"
        self.source.rename(hidden)
        self.addCleanup(hidden.rename, self.source)
        subprocess.run(["git", "-C", str(relocated), "init", "-q"], check=True)

        validate_snapshot(relocated, (self.source,))
        result = subprocess.run(
            [
                os.sys.executable,
                str(ROOT / "scripts/package_native.py"),
                "validate-snapshot",
                str(relocated),
            ],
            capture_output=True,
            text=True,
            timeout=90,
        )

        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_dirty_untracked_and_ignored_source_inputs_are_rejected(self) -> None:
        """GIVEN uncommitted inputs WHEN generating THEN clean provenance is required."""
        manifest = self.source / ".agents/skills/productivity/commit-message/SKILL.md"
        original = manifest.read_bytes()
        manifest.write_bytes(original + b"\nUncommitted input\n")
        self.addCleanup(manifest.write_bytes, original)

        with self.assertRaisesRegex(ValueError, "Commit source inputs"):
            build_snapshot(self.source, "0.1.1", self.root / "dirty")
        manifest.write_bytes(original)
        ignored = manifest.parent / "__pycache__/untracked.pyc"
        ignored.parent.mkdir()
        ignored.write_bytes(b"ignored resource")
        self.addCleanup(shutil.rmtree, ignored.parent)
        with self.assertRaisesRegex(ValueError, "Untracked or ignored"):
            build_snapshot(self.source, "0.1.1", self.root / "ignored")
        ignored.unlink()
        untracked = self.source / "scripts/untracked-helper.sh"
        untracked.write_text("untracked input\n")
        self.addCleanup(untracked.unlink)
        with self.assertRaisesRegex(ValueError, "Commit source inputs"):
            build_snapshot(self.source, "0.1.1", self.root / "untracked")

    def test_corruption_missing_files_and_changed_modes_are_rejected(self) -> None:
        """GIVEN damaged snapshot bytes or permissions WHEN validated THEN integrity fails."""
        for kind in ("resource", "missing", "mode", "manifest"):
            with self.subTest(kind=kind):
                root = self.copy("corrupt-" + kind)
                resource = (
                    root / "codex/plugins/commit-message/skills/commit-message/SKILL.md"
                )
                if kind == "missing":
                    resource.unlink()
                elif kind == "mode":
                    resource.chmod(0o755)
                elif kind == "manifest":
                    (
                        root / "codex/plugins/commit-message/.codex-plugin/plugin.json"
                    ).write_text("{}")
                else:
                    resource.write_text("corrupt")

                with self.assertRaises(ValueError):
                    validate_snapshot(root)

    def test_escaped_cross_provider_and_missing_catalogs_are_rejected(self) -> None:
        """GIVEN resealed hostile catalogs WHEN validated THEN paths remain provider-confined."""
        for source in ("../escape", "./claude/plugins/commit-message", "/tmp/plugin"):
            with self.subTest(source=source):
                root = self.copy("path-" + str(len(list(self.root.glob("path-*")))))
                path = root / ".agents/plugins/marketplace.json"
                catalog = json.loads(path.read_text())
                catalog["plugins"][0]["source"]["path"] = source
                write_json(path, catalog)
                self.reseal(root)

                with self.assertRaisesRegex(ValueError, "marketplace path"):
                    validate_snapshot(root)
        missing = self.copy("missing-catalog")
        (missing / ".claude-plugin/marketplace.json").unlink()
        self.reseal(missing)
        with self.assertRaises(OSError):
            validate_snapshot(missing)

    def test_symlinks_and_bad_provenance_are_rejected(self) -> None:
        """GIVEN symlink payloads or malformed provenance WHEN validated THEN none are accepted."""
        root = self.copy("symlink")
        (root / "outside").symlink_to(self.source / "LICENSE")

        with self.assertRaisesRegex(ValueError, "Unsupported package member"):
            validate_snapshot(root)
        bad = self.copy("bad-provenance")
        metadata = json.loads((bad / "snapshot.json").read_text())
        metadata["source_commit"] = "main"
        write_json(bad / "snapshot.json", metadata)
        with self.assertRaisesRegex(ValueError, "provenance"):
            validate_snapshot(bad)


if __name__ == "__main__":
    unittest.main()
