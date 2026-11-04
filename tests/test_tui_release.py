"""Check complete release integrity using inert synthetic archives."""

from __future__ import annotations

import copy
import json
import stat
import struct
import subprocess
import sys
import tempfile
import unittest
import warnings
import zipfile
from pathlib import Path
from unittest.mock import patch

from scripts import build_tui
from scripts import validate_tui_release as validator

VERSION = "v0.0.0-validation"
ROOT = Path(__file__).resolve().parent.parent


class ReleaseTests(unittest.TestCase):
    """Create independent four-target releases without building runnable code."""

    def setUp(self) -> None:
        """Create a fixture using the production archive writer."""
        temporary = tempfile.TemporaryDirectory(prefix="release validation ")
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.directory = self.root / "release"
        self.directory.mkdir()
        self.binary = self.root / "binary"
        self.binary.write_bytes(b"inert executable bytes; never run")
        self.records = {}
        for target in validator.TARGETS:
            archive = self.directory / f"agent-toolkit-{VERSION}-{target}.zip"
            build_tui.archive_binary(self.binary, archive, self.metadata(target))
            self.records[target] = {
                "archive": archive.name,
                "sha256": build_tui.digest(archive),
                "binary_sha256": build_tui.digest(self.binary),
            }
        self.manifest = {
            "version": VERSION,
            "protocol_version": 1,
            "targets": self.records,
        }
        self.write_records()
        self.target = validator.TARGETS[0]
        self.archive = self.directory / self.records[self.target]["archive"]

    def metadata(self, target: str) -> dict[str, object]:
        """Return the builder's exact metadata contract."""
        return {"version": VERSION, "protocol_version": 1, "target": target}

    def write_records(self) -> None:
        """Refresh outer manifests while retaining independent executable hashes."""
        (self.directory / "release.json").write_text(json.dumps(self.manifest))
        (self.directory / "SHA256SUMS").write_text(
            "".join(
                f"{item['sha256']}  {item['archive']}\n"
                for item in self.records.values()
            )
        )

    def refresh_archive_hash(self) -> None:
        """Recompute outer hashes so validation must inspect archive contents."""
        self.records[self.target]["sha256"] = build_tui.digest(self.archive)
        self.write_records()

    def snapshot(self) -> dict[str, bytes | None]:
        """Capture membership and all file bytes beneath the fixture root."""
        return {
            str(path.relative_to(self.root)): path.read_bytes()
            if path.is_file()
            else None
            for path in self.root.rglob("*")
        }

    def assert_rejected(self, message: str) -> None:
        """Require a contextual failure with no input mutation or execution."""
        before = self.snapshot()

        with patch("subprocess.run", side_effect=AssertionError("must not execute")):
            with self.assertRaisesRegex(ValueError, message):
                validator.validate_release(VERSION, self.directory)

        self.assertEqual(self.snapshot(), before)

    def test_valid_release_and_retained_manifest_are_read_only(self) -> None:
        """GIVEN a complete release WHEN checked THEN inputs and unrelated files survive."""
        expected = self.root / "retained.json"
        expected.write_bytes((self.directory / "release.json").read_bytes())
        (self.directory / "notes.txt").write_text("unrelated")
        before = self.snapshot()

        with patch("subprocess.run", side_effect=AssertionError("must not execute")):
            validator.validate_release(VERSION, self.directory)
            validator.validate_release(VERSION, self.directory, expected)

        self.assertEqual(self.snapshot(), before)

    def test_invalid_version_precedes_file_access(self) -> None:
        """GIVEN an unsafe version WHEN checked THEN no input is opened."""
        with patch("pathlib.Path.open", side_effect=AssertionError("must not read")):
            with self.assertRaisesRegex(ValueError, "version"):
                validator.validate_release("../v1", self.directory)

    def test_manifest_contract_failures(self) -> None:
        """GIVEN invalid identity or targets WHEN checked THEN the field is reported."""
        original = copy.deepcopy(self.manifest)
        cases = [
            ("version", "v9.0.0", "version"),
            ("protocol_version", True, "protocol"),
            ("protocol_version", 2, "protocol"),
            ("targets", {}, "targets"),
            ("targets", dict(list(self.records.items())[1:]), "targets"),
            ("targets", dict(self.records, windows={}), "targets"),
            ("targets", [], "targets"),
        ]
        for field, value, message in cases:
            with self.subTest(field=field, value=value):
                self.manifest = dict(original, **{field: value})
                self.write_records()

                self.assert_rejected(message)

    def test_target_fields_are_checked(self) -> None:
        """GIVEN malformed target fields WHEN checked THEN names and digests fail."""
        for field, value in (
            ("archive", "../other.zip"),
            ("archive", "unexpected.zip"),
            ("sha256", "G" * 64),
            ("binary_sha256", "123"),
            ("sha256", None),
        ):
            with self.subTest(field=field, value=value):
                original = self.records[self.target][field]
                self.records[self.target][field] = value
                self.write_records()

                self.assert_rejected(field)
                self.records[self.target][field] = original

    def test_missing_release_files_fail(self) -> None:
        """GIVEN any missing release file WHEN checked THEN validation fails read-only."""
        for path in self.directory.iterdir():
            with self.subTest(file=path.name):
                data = path.read_bytes()
                path.unlink()

                self.assert_rejected(path.name)
                path.write_bytes(data)

    def test_checksum_list_must_match_exactly(self) -> None:
        """GIVEN malformed, duplicate or disagreeing sums WHEN checked THEN they fail."""
        path = self.directory / "SHA256SUMS"
        original = path.read_text()
        lines = original.splitlines(keepends=True)
        for data in (
            "bad\n",
            original + lines[0],
            "".join(lines[1:]),
            original.replace(self.archive.name, "extra.zip"),
            original.replace(self.records[self.target]["sha256"], "0" * 64),
        ):
            with self.subTest(data=data):
                path.write_text(data)

                self.assert_rejected("SHA256SUMS")

    def test_archive_and_binary_corruption_are_distinguished(self) -> None:
        """GIVEN changed bytes WHEN outer hashes are refreshed THEN inner hashes fail."""
        self.archive.write_bytes(b"truncated")

        self.assert_rejected("archive sha256")
        self.refresh_archive_hash()
        self.assert_rejected(self.target)

        self.binary.write_bytes(b"changed executable")
        build_tui.archive_binary(self.binary, self.archive, self.metadata(self.target))
        self.refresh_archive_hash()

        self.assert_rejected("binary_sha256")

    def test_metadata_corruption_with_matching_outer_hashes(self) -> None:
        """GIVEN pinned archives with wrong metadata WHEN inspected THEN they fail."""
        for field, value in (
            ("version", "v9.0.0"),
            ("protocol_version", True),
            ("protocol_version", 2),
            ("target", "linux-unknown"),
            ("extra", 1),
        ):
            with self.subTest(field=field, value=value):
                metadata = dict(self.metadata(self.target), **{field: value})
                build_tui.archive_binary(self.binary, self.archive, metadata)
                self.refresh_archive_hash()

                self.assert_rejected(self.target)

    def test_added_duplicate_and_traversal_members_fail(self) -> None:
        """GIVEN extra or unsafe ZIP names WHEN inspected THEN no files are extracted."""
        for name in ("extra", "agent-toolkit", "../escaped", "/absolute"):
            with self.subTest(name=name):
                build_tui.archive_binary(
                    self.binary, self.archive, self.metadata(self.target)
                )
                with (
                    warnings.catch_warnings(),
                    zipfile.ZipFile(self.archive, "a") as bundle,
                ):
                    warnings.simplefilter("ignore", UserWarning)
                    bundle.writestr(name, b"unsafe")
                self.refresh_archive_hash()

                self.assert_rejected("unsafe archive members")

    def replace_member(self, name: str, data: bytes, mode: int) -> None:
        """Rewrite one member while preserving the other fixture member."""
        with zipfile.ZipFile(self.archive) as bundle:
            entries = [(item, bundle.read(item)) for item in bundle.infolist()]
        with zipfile.ZipFile(self.archive, "w") as bundle:
            for item, original in entries:
                if item.filename == name:
                    item.external_attr = mode << 16
                    original = data
                bundle.writestr(item, original)
        self.refresh_archive_hash()

    def test_member_types_and_permissions_fail(self) -> None:
        """GIVEN unsafe executable types or modes WHEN inspected THEN validation fails."""
        for mode in (stat.S_IFLNK | 0o755, stat.S_IFDIR | 0o755, 0o100644, 0o104755):
            with self.subTest(mode=mode):
                self.replace_member("agent-toolkit", b"bad", mode)

                self.assert_rejected("permissions")

    def set_compression_header(self, name: str, compression: int) -> None:
        """Advertise a codec in both ZIP headers without invoking that codec."""
        with zipfile.ZipFile(self.archive) as bundle:
            members = bundle.infolist()

        data = bytearray(self.archive.read_bytes())
        central_offset = data.index(b"PK\x01\x02")
        for member in members:
            if member.filename == name:
                struct.pack_into("<H", data, member.header_offset + 8, compression)
                struct.pack_into("<H", data, central_offset + 10, compression)
            central_offset += (
                46
                + len(member.filename.encode())
                + len(member.extra)
                + len(member.comment)
            )

        self.archive.write_bytes(data)
        self.refresh_archive_hash()

    def test_unsupported_compression_fails_before_any_member_opens(self) -> None:
        """GIVEN unsupported ZIP codecs WHEN checked THEN no decompression starts."""
        for name in ("agent-toolkit", "metadata.json"):
            for compression in (
                zipfile.ZIP_STORED,
                zipfile.ZIP_BZIP2,
                zipfile.ZIP_LZMA,
                99,
            ):
                with self.subTest(name=name, compression=compression):
                    build_tui.archive_binary(
                        self.binary, self.archive, self.metadata(self.target)
                    )
                    self.set_compression_header(name, compression)

                    with patch.object(
                        zipfile.ZipFile,
                        "open",
                        side_effect=AssertionError("must not decompress"),
                    ):
                        self.assert_rejected(f"{name}: unsupported compression")

    def test_malformed_and_duplicate_json_fail(self) -> None:
        """GIVEN ambiguous or malformed JSON WHEN read THEN validation rejects it."""
        for data in (b"{", b"[]", b'{"version":1,"version":2}', b"\xff"):
            with self.subTest(data=data):
                self.replace_member("metadata.json", data, 0o100644)

                self.assert_rejected(self.target)
                (self.directory / "release.json").write_bytes(data)
                self.assert_rejected("release.json")
                self.write_records()

    def test_size_limits_apply_to_files_and_members(self) -> None:
        """GIVEN oversized files or members WHEN checked THEN reads stay bounded."""
        with patch.object(validator, "MAX_SIZE", 10):
            self.assert_rejected("limit")

        self.replace_member("agent-toolkit", b"x" * 10000, 0o100755)
        with patch.object(validator, "MAX_SIZE", 5000):
            self.assert_rejected("agent-toolkit.*limit")

    def test_retained_manifest_requires_exact_bytes(self) -> None:
        """GIVEN equivalent JSON with different bytes WHEN compared THEN it fails."""
        expected = self.root / "retained.json"
        expected.write_text(json.dumps(self.manifest, indent=2))
        before = self.snapshot()

        with self.assertRaisesRegex(ValueError, "retained manifest mismatch"):
            validator.validate_release(VERSION, self.directory, expected)

        self.assertEqual(self.snapshot(), before)

    def test_cli_reports_success_and_expected_errors_without_tracebacks(self) -> None:
        """GIVEN valid and invalid releases WHEN the CLI runs THEN exit status explains it."""
        command = [
            sys.executable,
            str(ROOT / "scripts/validate_tui_release.py"),
            "--version",
            VERSION,
            "--directory",
            str(self.directory),
        ]
        before = self.snapshot()

        success = subprocess.run(command, capture_output=True, text=True, check=False)
        self.archive.write_bytes(b"corrupt")
        failed_before = self.snapshot()
        failure = subprocess.run(command, capture_output=True, text=True, check=False)

        self.assertEqual(success.returncode, 0, success.stderr)
        self.assertIn("four archives", success.stdout)
        self.assertEqual(failure.returncode, 1)
        self.assertIn(self.target, failure.stderr)
        self.assertNotIn("Traceback", failure.stderr)
        self.assertEqual(self.snapshot(), failed_before)
        self.assertEqual(set(before), set(failed_before))


if __name__ == "__main__":
    unittest.main()
