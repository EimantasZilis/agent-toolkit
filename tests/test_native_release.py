"""Verify complete native releases, reproducibility, and offline validation."""

from __future__ import annotations

import hashlib
import json
import shutil
import tempfile
import unittest
import zipfile
from pathlib import Path

from scripts.native import Catalog
from scripts.native import validate_tree
from scripts.native.release import ReleaseBuilder
from scripts.native.release import file_inventory
from scripts.native.release_validation import validate_release

REPO = Path(__file__).resolve().parents[1]


class NativeReleaseTests(unittest.TestCase):
    """Share a real generated release across isolated corruption checks."""

    @classmethod
    def setUpClass(cls) -> None:
        """Build once from an isolated source tree, then make it unavailable."""
        cls.temporary = tempfile.TemporaryDirectory(prefix="native release ")
        cls.addClassCleanup(cls.temporary.cleanup)
        cls.root = Path(cls.temporary.name)
        cls.source = cls.root / "source"
        for directory in (".agents", "scripts"):
            shutil.copytree(REPO / directory, cls.source / directory)
        shutil.copyfile(REPO / "LICENSE", cls.source / "LICENSE")
        cls.catalog = Catalog(cls.source)
        cls.release = cls.root / "release"
        cls.builder = ReleaseBuilder(cls.catalog, "0.1.0")
        cls.builder.build(cls.release)
        cls.source.rename(cls.root / "hidden-source")

    def test_all_packages_extract_without_source_and_preserve_resources(self) -> None:
        """GIVEN generated ZIPs WHEN extracted without a clone THEN all validate."""
        metadata = json.loads((self.release / "release.json").read_text())
        self.assertFalse(self.source.exists())

        validate_release(self.release, (self.source,))

        self.assertEqual(
            len(metadata["packages"]),
            2 * (len(metadata["catalog"]) + len(metadata["bundles"])),
        )
        for archive in self.release.glob("*/zips/*.zip"):
            extracted = (
                self.root / "extracted" / archive.parent.parent.name / archive.stem
            )
            with zipfile.ZipFile(archive) as bundle:
                bundle.extractall(extracted)
            validate_tree(extracted, (self.source,))
            original = archive.parent.parent / "plugins" / archive.stem
            self.assertEqual(file_inventory(extracted), file_inventory(original))

    def test_release_bytes_are_reproducible(self) -> None:
        """GIVEN identical inputs WHEN rebuilt THEN all release bytes match."""
        hidden = self.root / "hidden-source"
        hidden.rename(self.source)
        self.addCleanup(self.source.rename, hidden)
        second = self.root / "second"

        self.builder.build(second)

        self.assertEqual(file_inventory(second), file_inventory(self.release))

    def test_plugins_use_skill_and_bundle_names_in_both_providers(self) -> None:
        """GIVEN a release WHEN inspected THEN plugins use unprefixed identifiers."""
        names = (
            "ticket-to-plan",
            "commit-message",
            "bundle-starter",
            "bundle-all",
            "bundle-coding",
        )

        for provider in ("codex", "claude"):
            for name in names:
                with self.subTest(provider=provider, name=name):
                    package = self.release / provider / "plugins" / name
                    manifest = json.loads(
                        (package / f".{provider}-plugin/plugin.json").read_text()
                    )
                    archive = self.release / provider / "zips" / f"{name}.zip"

                    self.assertEqual(manifest["name"], name)
                    self.assertTrue(archive.is_file())
            self.assertFalse(
                list((self.release / provider / "plugins").glob("agent-toolkit-*"))
            )

    def test_existing_output_is_preserved(self) -> None:
        """GIVEN existing output WHEN rebuilding THEN no files are overwritten."""
        before = (self.release / "release.json").read_bytes()

        with self.assertRaisesRegex(ValueError, "already exists"):
            self.builder.build(self.release)

        self.assertEqual((self.release / "release.json").read_bytes(), before)

    def test_invalid_versions_are_rejected(self) -> None:
        """GIVEN unsafe versions WHEN configured THEN no build is attempted."""
        for version in ("../release", "01.2.3", "1.0", "1.0.0\n"):
            with self.subTest(version=version):
                with self.assertRaisesRegex(ValueError, "Release version"):
                    ReleaseBuilder(self.catalog, version)

    def test_corruption_is_rejected(self) -> None:
        """GIVEN damaged release boundaries WHEN validated THEN each fails closed."""
        for boundary in ("resource", "archive", "marketplace", "closure", "manifest"):
            with self.subTest(boundary=boundary):
                destination = self.root / f"damaged-{boundary}"
                shutil.copytree(self.release, destination)
                self.corrupt(destination, boundary)

                with self.assertRaises(ValueError):
                    validate_release(destination)

    def corrupt(self, root: Path, boundary: str) -> None:
        """Change one independently validated release boundary."""
        plugin = root / "codex/plugins/ticket-to-plan"
        if boundary == "resource":
            (plugin / "skills/plan-review/SKILL.md").write_text("corrupt\n")
        elif boundary == "archive":
            archive = root / "codex/zips/ticket-to-plan.zip"
            archive.write_bytes(b"corrupt")
        elif boundary == "marketplace":
            path = root / "claude/.claude-plugin/marketplace.json"
            metadata = json.loads(path.read_text())
            metadata["plugins"][0]["source"] = "../escape"
            path.write_text(json.dumps(metadata))
        elif boundary == "closure":
            path = root / "release.json"
            metadata = json.loads(path.read_text())
            metadata["packages"][0]["skills"] = []
            path.write_text(json.dumps(metadata))
        else:
            (plugin / ".codex-plugin/plugin.json").write_text("{}\n")

    def test_archive_traversal_is_rejected_before_extraction(self) -> None:
        """GIVEN a ZIP traversal member WHEN validated THEN it is never extracted."""
        destination = self.root / "traversal"
        shutil.copytree(self.release, destination)
        archive = destination / "codex/zips/commit-message.zip"
        with zipfile.ZipFile(archive, "a") as bundle:
            bundle.writestr("../../escaped", "untrusted")
        metadata = json.loads((destination / "release.json").read_text())
        for record in metadata["packages"]:
            if record["provider"] == "codex" and record["name"] == archive.stem:
                record["archive_sha256"] = hashlib.sha256(
                    archive.read_bytes()
                ).hexdigest()
        (destination / "release.json").write_text(json.dumps(metadata))

        with self.assertRaisesRegex(ValueError, "ZIP member"):
            validate_release(destination)

        self.assertFalse((self.root / "escaped").exists())
