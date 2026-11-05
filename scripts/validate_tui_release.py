#!/usr/bin/env python3
"""Validate a complete TUI release without extracting or executing its contents."""

from __future__ import annotations

import argparse
import hashlib
import io
import json
import re
import stat
import sys
import zipfile
import zlib
from pathlib import Path
from typing import cast

TARGETS = ("darwin-amd64", "darwin-arm64", "linux-amd64", "linux-arm64")
MAX_SIZE = 64 * 1024 * 1024
VERSION = r"v[0-9]+\.[0-9]+\.[0-9]+(?:-[a-zA-Z0-9.-]+)?"
SHA256 = r"[0-9a-f]{64}"


def read_file(path: Path) -> bytes:
    """Read at most 64 MiB from a regular input file."""
    if not stat.S_ISREG(path.stat().st_mode):
        raise ValueError(f"{path}: expected a regular file")
    with path.open("rb") as source:
        data = source.read(MAX_SIZE + 1)
    if len(data) > MAX_SIZE:
        raise ValueError(f"{path}: exceeds 64 MiB limit")
    return data


def unique_object(pairs: list[tuple[str, object]]) -> dict[str, object]:
    """Reject ambiguous duplicate JSON fields at every nesting level."""
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError(f"duplicate JSON field: {key}")
        result[key] = value
    return result


def parse_json(data: bytes) -> dict[str, object]:
    """Decode an unambiguous JSON object with bounded nesting failures."""
    value = json.loads(data, object_pairs_hook=unique_object)
    if not isinstance(value, dict):
        raise ValueError("expected a JSON object")
    return value


def require_identity(document: dict[str, object], version: str) -> None:
    """Require the requested version and integer backend protocol 1."""
    if document.get("version") != version:
        raise ValueError("version mismatch")
    protocol = document.get("protocol_version")
    if type(protocol) is not int or protocol != 1:
        raise ValueError("unsupported protocol_version; expected integer 1")


def validate_record(record: object, version: str, target: str) -> dict[str, str]:
    """Check one target's safe archive name and both digest fields."""
    if not isinstance(record, dict):
        raise ValueError(f"{target}: expected a target object")
    expected = f"agent-toolkit-{version}-{target}.zip"
    if record.get("archive") != expected:
        raise ValueError(f"{target}: archive must be {expected}")
    for field in ("sha256", "binary_sha256"):
        value = record.get(field)
        if not isinstance(value, str) or not re.fullmatch(SHA256, value):
            raise ValueError(f"{target}: invalid {field}")
    return cast(dict[str, str], record)


def validate_manifest(data: bytes, version: str) -> dict[str, dict[str, str]]:
    """Validate release identity and the exact four-target contract."""
    document = parse_json(data)
    require_identity(document, version)
    targets = document.get("targets")
    if not isinstance(targets, dict) or set(targets) != set(TARGETS):
        raise ValueError("targets must contain exactly: " + ", ".join(TARGETS))
    return {
        target: validate_record(targets[target], version, target) for target in TARGETS
    }


def validate_checksums(data: bytes, records: dict[str, dict[str, str]]) -> None:
    """Require one matching checksum for every archive and no extra entries."""
    expected = {item["archive"]: item["sha256"] for item in records.values()}
    entries = {}
    for line in data.decode("utf-8").splitlines():
        match = re.fullmatch(rf"({SHA256})  (.+)", line)
        if not match:
            raise ValueError("invalid checksum entry")
        digest, name = match.groups()
        if name in entries:
            raise ValueError(f"duplicate checksum entry: {name}")
        entries[name] = digest
    if entries != expected:
        raise ValueError("checksum entries disagree with release.json archives/hashes")


def validate_members(bundle: zipfile.ZipFile) -> None:
    """Require two bounded DEFLATE members with builder names and permissions."""
    members = bundle.infolist()
    modes = {"agent-toolkit": 0o755, "metadata.json": 0o644}
    if sorted(item.filename for item in members) != sorted(modes):
        raise ValueError(
            "unsafe archive members; expected agent-toolkit, metadata.json"
        )
    for item in members:
        # Other codecs may allocate unbounded output before read() applies its limit.
        if item.compress_type != zipfile.ZIP_DEFLATED:
            raise ValueError(
                f"{item.filename}: unsupported compression; expected DEFLATE"
            )
        mode = item.external_attr >> 16
        if not stat.S_ISREG(mode) or stat.S_IMODE(mode) != modes[item.filename]:
            raise ValueError(f"{item.filename}: unsafe member type or permissions")
        if item.file_size > MAX_SIZE:
            raise ValueError(f"{item.filename}: exceeds 64 MiB limit")


def read_member(bundle: zipfile.ZipFile, name: str) -> bytes:
    """Read a bounded member and let ZIP CRC checks reject corrupted data."""
    with bundle.open(name) as member:
        data = member.read(MAX_SIZE + 1)
    if len(data) > MAX_SIZE:
        raise ValueError(f"{name}: exceeds 64 MiB limit")
    return data


def validate_archive(
    data: bytes, record: dict[str, str], version: str, target: str
) -> None:
    """Check archive and executable hashes plus exact embedded metadata."""
    if hashlib.sha256(data).hexdigest() != record["sha256"]:
        raise ValueError("archive sha256 mismatch")
    with zipfile.ZipFile(io.BytesIO(data)) as bundle:
        validate_members(bundle)
        metadata = parse_json(read_member(bundle, "metadata.json"))
        require_identity(metadata, version)
        expected = {"version": version, "protocol_version": 1, "target": target}
        if metadata != expected:
            raise ValueError("metadata.json: target or fields mismatch")
        binary = read_member(bundle, "agent-toolkit")
    if hashlib.sha256(binary).hexdigest() != record["binary_sha256"]:
        raise ValueError("agent-toolkit: binary_sha256 mismatch")


INPUT_ERRORS = (
    OSError,
    ValueError,
    UnicodeError,
    RecursionError,
    zipfile.BadZipFile,
    NotImplementedError,
    RuntimeError,
    EOFError,
    zlib.error,
)


def validate_release(
    version: str, directory: Path, expected_manifest: Path | None = None
) -> None:
    """Validate all six inputs read-only, optionally binding retained manifest bytes."""
    if not re.fullmatch(VERSION, version):
        raise ValueError("version must be a v-prefixed release version")
    manifest_path = directory / "release.json"
    try:
        manifest = read_file(manifest_path)
        records = validate_manifest(manifest, version)
        if expected_manifest is not None and manifest != read_file(expected_manifest):
            raise ValueError(f"retained manifest mismatch: {expected_manifest}")
    except INPUT_ERRORS as error:
        raise ValueError(f"{manifest_path}: {error}") from error

    checksums = directory / "SHA256SUMS"
    try:
        validate_checksums(read_file(checksums), records)
    except INPUT_ERRORS as error:
        raise ValueError(f"{checksums}: {error}") from error
    for target, record in records.items():
        archive = directory / record["archive"]
        try:
            validate_archive(read_file(archive), record, version, target)
        except INPUT_ERRORS as error:
            raise ValueError(f"{target} ({archive}): {error}") from error


def main() -> int:
    """Report expected input failures without a traceback."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", required=True)
    parser.add_argument("--directory", type=Path, required=True)
    parser.add_argument("--expected-manifest", type=Path)
    args = parser.parse_args()
    try:
        validate_release(args.version, args.directory, args.expected_manifest)
    except ValueError as error:
        print(f"TUI release validation failed: {error}", file=sys.stderr)
        return 1
    print(f"Validated {args.version}: four archives, SHA256SUMS, release.json")
    return 0


if __name__ == "__main__":
    sys.exit(main())
