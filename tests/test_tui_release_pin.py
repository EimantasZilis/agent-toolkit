"""Check pin acceptance with real setup/launcher processes and fixture HTTPS bytes."""

from __future__ import annotations

import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
import build_tui
import check_tui_release_pin as pin
import test_tui_release_publication as publishing
import test_tui_release_smoke as smoking

ROOT = Path(__file__).resolve().parents[1]


class PinTests(unittest.TestCase):
    """Combine complete candidate evidence with disposable Git checkouts."""

    def setUp(self) -> None:
        """Use existing inert-publication fixtures and replace bytes with a PTY fixture."""
        self.fixture = publishing.PublicationTests()
        self.fixture.setUp()
        self.addCleanup(self.fixture.doCleanups)
        self.directory = self.fixture.directory
        self.record = self.fixture.record
        self.make_release()
        self.report = self.fixture.publish()
        self.checkout = self.fixture.fixture.root / "checkout"
        self.checkout.mkdir()
        shutil.copytree(
            ROOT / "scripts",
            self.checkout / "scripts",
            ignore=shutil.ignore_patterns("__pycache__"),
        )
        shutil.copytree(ROOT / ".agents", self.checkout / ".agents")
        shutil.copyfile(ROOT / "Makefile", self.checkout / "Makefile")
        self.git("init", "-q")
        self.git("add", ".")
        self.git(
            "-c",
            "user.name=Fixture",
            "-c",
            "user.email=fixture@example.test",
            "commit",
            "-qm",
            "fixture",
        )
        self.sha = self.git("rev-parse", "HEAD")

    def git(self, *arguments: str) -> str:
        """Operate only on the disposable test repository."""
        return pin.candidate.git(self.checkout, *arguments)

    def make_release(self) -> None:
        """Rebind the approved candidate to an executable that handles launcher arguments."""
        fixture = self.fixture.fixture
        fixture.binary.write_text(
            "#!/bin/bash\n"
            "if [[ ${!#} == --version ]]; then\n"
            f"printf 'agent-toolkit {self.record['version']}; managed protocol 1\\n'; exit 0\n"
            "fi\n" + smoking.SmokeTests().screen()
        )
        for target, item in fixture.records.items():
            archive = self.directory / item["archive"]
            build_tui.archive_binary(fixture.binary, archive, fixture.metadata(target))
            item.update(
                sha256=build_tui.digest(archive),
                binary_sha256=build_tui.digest(fixture.binary),
            )
        fixture.write_records()
        self.record["hashes"] = pin.candidate.payload_hashes(
            self.directory, self.record["version"]
        )

    def accept(self) -> dict[str, object]:
        """Exercise one native target, marking local transport explicitly as a fixture."""
        self.record["source_sha"] = self.sha
        for report in self.record["reports"]:
            report["source_sha"] = self.sha
        self.fixture.api = publishing.FakeGitHub()
        self.report = self.fixture.publish()
        return pin.check_pin(
            self.record,
            self.report,
            self.directory,
            self.checkout,
            self.checkout,
            self.sha,
            pin.runtime.native_target(),
            fixture=self.directory,
        )

    def test_fresh_and_offline_routes_restore_checkout(self) -> None:
        """GIVEN published fixture bytes WHEN setup repeats THEN cache and Bash routes pass."""
        result = self.accept()

        self.assertEqual(result["transport"], "fixture")
        self.assertEqual(result["checkouts"]["source"]["fresh"]["downloads"], 1)
        self.assertEqual(result["checkouts"]["base"]["cached"]["downloads"], 0)
        self.assertTrue(
            all(
                item["status"] == "passed"
                for item in result["checkouts"]["source"].values()
            )
        )
        self.assertEqual(self.git("status", "--porcelain"), "")
        self.assertFalse((self.checkout / pin.PIN).exists())

    def test_forged_publication_is_rejected_before_checkout_writes(self) -> None:
        """GIVEN altered publication fields WHEN validated THEN no pin is written."""
        cases = (
            ("status", "unverified"),
            ("public", "failed"),
            ("draft", "failed"),
            ("source_sha", "f" * 40),
            ("payload_id", "999"),
            ("release_id", 0),
            ("release_url", "https://example.test/release"),
            ("public_hashes", {}),
        )
        for field, value in cases:
            with self.subTest(field=field):
                report = dict(self.report, **{field: value})

                with self.assertRaises(ValueError):
                    pin.check_publication(self.record, report, self.directory)

                self.assertFalse((self.checkout / pin.PIN).exists())

    def test_retained_manifest_substitution_fails(self) -> None:
        """GIVEN a replaced retained manifest WHEN validated THEN publication cannot authorize it."""
        (self.directory / "release.json").write_text("{}")

        with self.assertRaises(ValueError):
            pin.check_publication(self.record, self.report, self.directory)

    def test_checkout_race_and_dirty_state_are_rejected(self) -> None:
        """GIVEN a changed SHA or worktree WHEN checked THEN no acceptance runs."""
        with self.assertRaisesRegex(ValueError, "SHA changed"):
            pin.require_checkout(self.checkout, "f" * 40)
        (self.checkout / "human-file").write_text("keep")

        with self.assertRaisesRegex(ValueError, "clean disposable"):
            pin.require_checkout(self.checkout, self.sha)

    def test_wrong_host_and_missing_publication_cannot_execute(self) -> None:
        """GIVEN missing public verification or a foreign target WHEN checked THEN fail closed."""
        target = next(
            item
            for item in pin.validator.TARGETS
            if item != pin.runtime.native_target()
        )

        with self.assertRaisesRegex(ValueError, "native target"):
            pin.check_pin(
                self.record,
                self.report,
                self.directory,
                self.checkout,
                self.checkout,
                self.sha,
                target,
            )

    def test_download_corruption_restores_existing_pin(self) -> None:
        """GIVEN corrupt served bytes WHEN the launcher checks hashes THEN restore the source pin."""
        existing = self.checkout / pin.PIN
        existing.write_bytes(b"previous manifest\n")
        self.git("add", pin.PIN)
        self.git(
            "-c",
            "user.name=Fixture",
            "-c",
            "user.email=fixture@example.test",
            "commit",
            "-qm",
            "previous pin",
        )
        self.sha = self.git("rev-parse", "HEAD")
        served = self.checkout.parent / "served"
        shutil.copytree(self.directory, served)
        for path in served.glob("*.zip"):
            path.write_bytes(b"corrupt")

        with self.assertRaises((ValueError, subprocess.SubprocessError)):
            pin.check_one(
                self.checkout,
                self.sha,
                (self.directory / "release.json").read_bytes(),
                pin.runtime.native_target(),
                fixture=served,
            )

        self.assertEqual(existing.read_bytes(), b"previous manifest\n")
        self.assertEqual(self.git("status", "--porcelain"), "")

    def test_missing_or_corrupt_cache_cannot_pass_offline(self) -> None:
        """GIVEN populated cache WHEN removed/corrupted THEN an offline launch cannot pass."""
        manifest = (self.directory / "release.json").read_bytes()
        (self.checkout / pin.PIN).write_bytes(manifest)
        target = pin.runtime.native_target()
        with tempfile.TemporaryDirectory() as temporary:
            environment = pin.create_environment(Path(temporary), self.checkout)
            environment["PIN_CHECK_FIXTURE"] = str(self.directory)
            pin.launcher_cycle(self.checkout, environment, manifest, target, False)
            binary = next(Path(environment["XDG_CACHE_HOME"]).rglob("agent-toolkit"))
            binary.write_bytes(b"corrupt")

            with self.assertRaises(ValueError):
                pin.launcher_cycle(self.checkout, environment, manifest, target, True)
            binary.unlink()

            with self.assertRaises(ValueError):
                pin.launcher_cycle(self.checkout, environment, manifest, target, True)

    def test_incompatible_backend_blocks_download_and_restores_pin(self) -> None:
        """GIVEN an incompatible backend WHEN checked THEN launcher acceptance fails safely."""
        (self.checkout / "scripts/managed.sh").write_text(
            'printf \'{"ok":true,"protocol_version":2}\\n\'\n'
        )
        self.git("add", "scripts/managed.sh")
        self.git(
            "-c",
            "user.name=Fixture",
            "-c",
            "user.email=fixture@example.test",
            "commit",
            "-qm",
            "incompatible",
        )
        sha = self.git("rev-parse", "HEAD")

        with self.assertRaises(ValueError):
            pin.check_one(
                self.checkout,
                sha,
                (self.directory / "release.json").read_bytes(),
                pin.runtime.native_target(),
                fixture=self.directory,
            )

        self.assertFalse((self.checkout / pin.PIN).exists())
        self.assertEqual(self.git("status", "--porcelain"), "")


if __name__ == "__main__":
    unittest.main()
