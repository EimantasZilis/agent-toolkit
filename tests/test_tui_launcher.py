"""Exercise verified launch and failure boundaries without network or Go."""

from __future__ import annotations

import hashlib
import json
import shutil
import stat
import tempfile
import unittest
import urllib.error
import warnings
import zipfile
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
from unittest.mock import patch

from scripts import launch_tui
from scripts.build_tui import archive_binary


class LauncherTests(unittest.TestCase):
    """Use local ZIP downloads and a read-only backend in paths with spaces."""

    def setUp(self) -> None:
        """Create a source-pinned executable fixture and isolated repository."""
        temporary = tempfile.TemporaryDirectory(prefix="tui launcher ")
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        (self.root / "scripts").mkdir()
        backend = self.root / "scripts/managed.sh"
        backend.write_text('printf \'{"protocol_version":1,"ok":true}\\n\'\n')
        self.binary = self.root / "fixture"
        self.binary.write_text('#!/bin/sh\nprintf \'%s\\n\' "$@" >> "$2/launched"\n')
        self.archive = self.root / "fixture.zip"
        self.metadata = {
            "version": "v0.1.0",
            "protocol_version": 1,
            "target": "darwin-arm64",
        }
        archive_binary(self.binary, self.archive, self.metadata)
        self.pin = self.pin_for_archive()
        self.cache = self.root / "cache"
        self.launcher = launch_tui.Launcher(
            self.root, self.cache, self.pin, "darwin-arm64"
        )

    def pin_for_archive(self) -> launch_tui.ReleasePin:
        """Pin the fixture bytes for all supported platform records."""
        targets = {}
        for target in launch_tui.TARGETS:
            targets[target] = {
                "archive": f"agent-toolkit-v0.1.0-{target}.zip",
                "sha256": hashlib.sha256(self.archive.read_bytes()).hexdigest(),
                "binary_sha256": hashlib.sha256(self.binary.read_bytes()).hexdigest(),
            }
        return {"version": "v0.1.0", "protocol_version": 1, "targets": targets}

    def download(self, url: str, destination: Path) -> None:
        """Replace only the network boundary with a local archive copy."""
        self.assertEqual(
            url,
            f"{launch_tui.RELEASE_URL}/v0.1.0/agent-toolkit-v0.1.0-darwin-arm64.zip",
        )
        shutil.copyfile(self.archive, destination)

    def cached_binary(self) -> Path:
        """Return the expected published executable path."""
        return self.cache / "v0.1.0-darwin-arm64/agent-toolkit"

    def assert_no_launch(self) -> None:
        """Ensure failures never ran the executable or changed installation state."""
        self.assertFalse((self.root / "launched").exists())
        self.assertFalse((self.root / "config").exists())
        self.assertFalse(self.cached_binary().exists())

    def test_platform_mappings_and_unsupported_hosts(self) -> None:
        """GIVEN OS/CPU identities WHEN selected THEN all four targets map safely."""
        for system, machine, target in (
            ("Darwin", "x86_64", "darwin-amd64"),
            ("Darwin", "arm64", "darwin-arm64"),
            ("Linux", "x86_64", "linux-amd64"),
            ("Linux", "aarch64", "linux-arm64"),
        ):
            with (
                self.subTest(target=target),
                patch("platform.system", return_value=system),
                patch("platform.machine", return_value=machine),
            ):
                self.assertEqual(launch_tui.host_target(), target)

        with patch("platform.system", return_value="Windows"):
            with self.assertRaisesRegex(ValueError, "unsupported TUI platform"):
                launch_tui.host_target()

    def test_download_then_offline_cached_launch(self) -> None:
        """GIVEN a pinned ZIP WHEN launched twice THEN the second launch is offline."""
        with patch("scripts.launch_tui.download", side_effect=self.download) as network:
            first = self.launcher.run(["--version"])
        with patch(
            "scripts.launch_tui.download", side_effect=AssertionError("offline")
        ):
            second = self.launcher.run(["--version"])

        self.assertEqual((first, second), (0, 0))
        network.assert_called_once()
        self.assertEqual(
            (self.root / "launched").read_text().splitlines(),
            ["--root", str(self.root), "--version"] * 2,
        )

    def test_bad_archive_checksum_never_executes(self) -> None:
        """GIVEN changed download bytes WHEN verified THEN no executable is published."""
        self.archive.write_bytes(b"untrusted")

        with patch("scripts.launch_tui.download", side_effect=self.download):
            with self.assertRaisesRegex(ValueError, "archive checksum mismatch"):
                self.launcher.run([])

        self.assert_no_launch()

    def test_bad_binary_checksum_never_executes(self) -> None:
        """GIVEN a pinned archive with wrong binary hash WHEN extracted THEN it fails."""
        self.pin["targets"]["darwin-arm64"]["binary_sha256"] = "0" * 64

        with patch("scripts.launch_tui.download", side_effect=self.download):
            with self.assertRaisesRegex(ValueError, "executable checksum mismatch"):
                self.launcher.run([])

        self.assert_no_launch()

    def test_corrupt_cache_is_rejected_without_download(self) -> None:
        """GIVEN changed cached bytes WHEN launched offline THEN they never execute."""
        with patch("scripts.launch_tui.download", side_effect=self.download):
            self.launcher.run([])
        (self.root / "launched").unlink()
        self.cached_binary().write_bytes(b"corrupt")

        with patch(
            "scripts.launch_tui.download", side_effect=AssertionError("offline")
        ):
            with self.assertRaisesRegex(ValueError, "executable checksum mismatch"):
                self.launcher.run([])

        self.assertFalse((self.root / "launched").exists())

    def test_interrupted_and_missing_downloads_leave_no_binary(self) -> None:
        """GIVEN a partial download or HTTP 404 WHEN fetched THEN staging is removed."""

        def interrupted(url: str, destination: Path) -> None:
            destination.write_bytes(b"partial")
            raise OSError("interrupted")

        missing = urllib.error.HTTPError(
            "https://example.com", 404, "missing", {}, None
        )
        for failure in (interrupted, missing):
            with (
                self.subTest(failure=failure),
                patch("scripts.launch_tui.download", side_effect=failure),
            ):
                with self.assertRaises(OSError):
                    self.launcher.run([])

                self.assert_no_launch()
                self.assertFalse(list(self.cache.glob("*/download-*")))

    def test_unsafe_members_never_extract(self) -> None:
        """GIVEN pinned ZIPs with traversal, duplicates or symlinks THEN none extract."""
        for name in ("../escaped", "/absolute", "agent-toolkit", "link"):
            with self.subTest(name=name):
                archive_binary(self.binary, self.archive, self.metadata)
                with (
                    warnings.catch_warnings(),
                    zipfile.ZipFile(self.archive, "a") as bundle,
                ):
                    warnings.filterwarnings(
                        "ignore", message="Duplicate name:", category=UserWarning
                    )
                    entry = zipfile.ZipInfo(name)
                    entry.external_attr = (stat.S_IFLNK | 0o777) << 16
                    bundle.writestr(entry, "../escaped")
                self.pin["targets"]["darwin-arm64"]["sha256"] = hashlib.sha256(
                    self.archive.read_bytes()
                ).hexdigest()

                with patch("scripts.launch_tui.download", side_effect=self.download):
                    with self.assertRaisesRegex(ValueError, "unsafe archive"):
                        self.launcher.run([])

                self.assert_no_launch()
                self.assertFalse((self.root / "escaped").exists())

    def test_metadata_and_truncated_zip_fail_before_publication(self) -> None:
        """GIVEN corrupt but pinned archives WHEN inspected THEN no bytes execute."""
        self.metadata["protocol_version"] = 2
        archive_binary(self.binary, self.archive, self.metadata)
        for truncated in (False, True):
            if truncated:
                self.archive.write_bytes(self.archive.read_bytes()[:100])
            self.pin["targets"]["darwin-arm64"]["sha256"] = hashlib.sha256(
                self.archive.read_bytes()
            ).hexdigest()

            with patch("scripts.launch_tui.download", side_effect=self.download):
                with self.assertRaises((ValueError, zipfile.BadZipFile)):
                    self.launcher.run([])

            self.assert_no_launch()

    def test_concurrent_launches_publish_once(self) -> None:
        """GIVEN concurrent cache misses WHEN launched THEN only one download occurs."""
        with patch("scripts.launch_tui.download", side_effect=self.download) as network:
            with ThreadPoolExecutor(max_workers=4) as workers:
                results = list(workers.map(self.launcher.run, [[]] * 4))

        self.assertEqual(results, [0] * 4)
        network.assert_called_once()
        launch_tui.verify_binary(
            self.cached_binary(), self.pin["targets"]["darwin-arm64"]["binary_sha256"]
        )

    def test_backend_mismatch_fails_before_download_or_cache_creation(self) -> None:
        """GIVEN incompatible backend metadata WHEN launched THEN nothing mutates."""
        (self.root / "scripts/managed.sh").write_text(
            'printf \'{"protocol_version":2,"ok":true}\\n\'\n'
        )

        with patch(
            "scripts.launch_tui.download", side_effect=AssertionError("no download")
        ):
            with self.assertRaisesRegex(ValueError, "backend protocol mismatch"):
                self.launcher.run([])

        self.assert_no_launch()
        self.assertFalse(self.cache.exists())

    def test_private_cache_and_symlink_boundaries(self) -> None:
        """GIVEN an unsafe cache root WHEN prepared THEN external paths stay intact."""
        external = self.root / "external"
        external.mkdir(mode=0o700)
        self.cache.symlink_to(external, target_is_directory=True)

        with self.assertRaisesRegex(ValueError, "cache directory is not owned"):
            self.launcher.run([])

        self.assertEqual(list(external.iterdir()), [])
        self.cache.unlink()
        self.cache.mkdir(mode=0o755)
        with self.assertRaisesRegex(ValueError, "mode 700"):
            self.launcher.run([])

    def test_missing_and_unsafe_pins_are_rejected(self) -> None:
        """GIVEN absent or malformed pins WHEN loaded THEN no download is attempted."""
        path = self.root / "pin.json"
        with self.assertRaisesRegex(ValueError, "no published TUI release"):
            launch_tui.load_pin(path)
        path.write_text(json.dumps(self.pin))
        self.assertEqual(launch_tui.load_pin(path), self.pin)

        for key, value in (
            ("version", "../v1"),
            ("protocol_version", 2),
            ("targets", {}),
        ):
            invalid = dict(self.pin, **{key: value})
            path.write_text(json.dumps(invalid))

            with self.assertRaises((ValueError, KeyError)):
                launch_tui.load_pin(path)

    def test_cache_leaf_symlinks_are_rejected(self) -> None:
        """GIVEN symlinked lock or binary WHEN launched THEN its target stays intact."""
        launch_tui.private_directory(self.cache)
        directory = self.cached_binary().parent
        launch_tui.private_directory(directory)
        external = self.root / "external"
        external.write_text("user owned")
        external.chmod(0o700)

        for name in ("launch.lock", "agent-toolkit"):
            entry = directory / name
            if entry.exists():
                entry.unlink()
            entry.symlink_to(external)

            with self.assertRaises((ValueError, OSError)):
                self.launcher.run([])

            self.assertEqual(external.read_text(), "user owned")
            self.assertFalse((self.root / "launched").exists())
            entry.unlink()

    def test_https_downgrades_are_rejected(self) -> None:
        """GIVEN insecure URLs WHEN downloaded or redirected THEN HTTP is rejected."""
        with self.assertRaisesRegex(ValueError, "must use HTTPS"):
            launch_tui.download("http://example.com/archive", self.root / "download")
        handler = launch_tui.HTTPSRedirects()

        with self.assertRaisesRegex(ValueError, "must use HTTPS"):
            handler.redirect_request(
                None, None, 302, "redirect", {}, "http://example.com"
            )


if __name__ == "__main__":
    unittest.main()
