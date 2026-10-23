#!/usr/bin/env python3
"""Build CGO-free TUI release archives and a reviewable launcher pin."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import subprocess
import tempfile
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
TARGETS = ("darwin-amd64", "darwin-arm64", "linux-amd64", "linux-arm64")
PROTOCOL = 1


def digest(path: Path) -> str:
    """Return the SHA-256 of a file."""
    return hashlib.sha256(path.read_bytes()).hexdigest()


def archive_binary(binary: Path, archive: Path, metadata: dict[str, object]) -> None:
    """Package only the executable and its version/protocol metadata."""
    with zipfile.ZipFile(archive, "w", compression=zipfile.ZIP_DEFLATED) as output:
        for name, data, mode in (
            ("agent-toolkit", binary.read_bytes(), 0o100755),
            (
                "metadata.json",
                (json.dumps(metadata, sort_keys=True) + "\n").encode(),
                0o100644,
            ),
        ):
            info = zipfile.ZipInfo(name)
            info.external_attr = mode << 16
            info.compress_type = zipfile.ZIP_DEFLATED
            output.writestr(info, data)


def build_target(version: str, target: str, output: Path) -> dict[str, str]:
    """Cross-build one pinned target without a C toolchain."""
    goos, goarch = target.split("-")
    environment = dict(os.environ, CGO_ENABLED="0", GOOS=goos, GOARCH=goarch)
    with tempfile.TemporaryDirectory(prefix="toolkit-build-") as temporary:
        binary = Path(temporary) / "agent-toolkit"
        subprocess.run(
            [
                "go",
                "build",
                "-mod=readonly",
                "-trimpath",
                "-buildvcs=false",
                "-ldflags",
                f"-s -w -X main.version={version}",
                "-o",
                str(binary),
                "./cmd/agent-toolkit",
            ],
            cwd=ROOT,
            env=environment,
            check=True,
        )
        archive = output / f"agent-toolkit-{version}-{target}.zip"
        metadata = {"version": version, "protocol_version": PROTOCOL, "target": target}
        archive_binary(binary, archive, metadata)
        return {
            "archive": archive.name,
            "sha256": digest(archive),
            "binary_sha256": digest(binary),
        }


def build_release(version: str, output: Path) -> None:
    """Produce four archives, checksums, and a pin candidate without publishing."""
    if not re.fullmatch(r"v[0-9]+\.[0-9]+\.[0-9]+(?:-[a-zA-Z0-9.-]+)?", version):
        raise ValueError("version must be a v-prefixed release version")
    output.mkdir(parents=True, exist_ok=False)
    targets = {target: build_target(version, target, output) for target in TARGETS}
    manifest = {"version": version, "protocol_version": PROTOCOL, "targets": targets}
    (output / "release.json").write_text(json.dumps(manifest, indent=2) + "\n")
    (output / "SHA256SUMS").write_text(
        "".join(f"{item['sha256']}  {item['archive']}\n" for item in targets.values())
    )


def main() -> None:
    """Parse maintainer build options."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    build_release(args.version, args.output)


if __name__ == "__main__":
    main()
