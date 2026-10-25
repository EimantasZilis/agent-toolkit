#!/usr/bin/env python3
"""Launch a pinned, verified TUI without a Go installation."""

from __future__ import annotations

import argparse
import fcntl
import hashlib
import json
import os
import platform
import re
import stat
import subprocess
import sys
import tempfile
import urllib.request
import zipfile
from pathlib import Path
from typing import TypedDict

ROOT = Path(__file__).resolve().parent.parent
PIN = ROOT / "scripts/tui-release.json"
RELEASE_URL = "https://github.com/EimantasZilis/agent-toolkit/releases/download"
TARGETS = ("darwin-amd64", "darwin-arm64", "linux-amd64", "linux-arm64")
MAX_DOWNLOAD = 64 * 1024 * 1024


class TargetPin(TypedDict):
    """Describe locally reviewed archive and executable hashes."""

    archive: str
    sha256: str
    binary_sha256: str


class ReleasePin(TypedDict):
    """Describe the release identity and its supported targets."""

    version: str
    protocol_version: int
    targets: dict[str, TargetPin]


def host_target() -> str:
    """Map supported operating systems and CPU names to release targets."""
    machine = platform.machine().lower()
    architecture = {"x86_64": "amd64", "aarch64": "arm64"}.get(machine, machine)
    target = f"{platform.system().lower()}-{architecture}"
    if target not in TARGETS:
        raise ValueError(f"unsupported TUI platform: {target}")
    return target


def load_pin(path: Path) -> ReleasePin:
    """Read the locally reviewed pin, rejecting unsafe paths and missing hashes."""
    if not path.is_file():
        raise ValueError("no published TUI release is pinned in this checkout")
    pin = json.loads(path.read_text())
    if not re.fullmatch(r"v[0-9]+\.[0-9]+\.[0-9]+(?:-[a-zA-Z0-9.-]+)?", pin["version"]):
        raise ValueError("invalid pinned release version")
    if type(pin["protocol_version"]) is not int or pin["protocol_version"] != 1:
        raise ValueError("unsupported pinned backend protocol")
    for target in TARGETS:
        record = pin["targets"][target]
        expected = f"agent-toolkit-{pin['version']}-{target}.zip"
        if record["archive"] != expected:
            raise ValueError("invalid pinned archive name")
        for key in ("sha256", "binary_sha256"):
            if not re.fullmatch(r"[0-9a-f]{64}", record[key]):
                raise ValueError(f"invalid pinned {key}")
    return pin


def verify_backend(root: Path, protocol: int) -> None:
    """Check the backend's read-only catalog before any installation changes."""
    result = subprocess.run(
        [
            "bash",
            str(root / "scripts/managed.sh"),
            "catalog",
            "--protocol",
            str(protocol),
        ],
        cwd=root,
        check=True,
        capture_output=True,
        text=True,
    )
    response = json.loads(result.stdout)
    if response.get("protocol_version") != protocol or response.get("ok") is not True:
        raise ValueError("backend protocol mismatch")


def private_directory(path: Path) -> None:
    """Require a private, owned cache directory, never following its symlink."""
    path.mkdir(mode=0o700, parents=True, exist_ok=True)
    info = path.lstat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid():
        raise ValueError(f"cache directory is not owned: {path}")
    if stat.S_IMODE(info.st_mode) & 0o077:
        raise ValueError(f"cache directory must have mode 700: {path}")


def verify_binary(path: Path, expected: str) -> None:
    """Validate an owned regular executable against the source-pinned hash."""
    info = path.lstat()
    if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid():
        raise ValueError("cached executable must be an owned regular file")
    if info.st_mode & 0o022 or not info.st_mode & stat.S_IXUSR:
        raise ValueError("cached executable has unsafe permissions")
    if hashlib.sha256(path.read_bytes()).hexdigest() != expected:
        raise ValueError(
            "cached executable checksum mismatch; remove its cache directory"
        )


class HTTPSRedirects(urllib.request.HTTPRedirectHandler):
    """Reject redirects that downgrade a release download from HTTPS."""

    def redirect_request(
        self,
        request: urllib.request.Request,
        fp: object,
        code: int,
        message: str,
        headers: object,
        new_url: str,
    ) -> urllib.request.Request | None:
        """Permit GitHub's asset redirects only over HTTPS."""
        if not new_url.startswith("https://"):
            raise ValueError("release redirect must use HTTPS")
        return super().redirect_request(request, fp, code, message, headers, new_url)


def download(url: str, destination: Path) -> None:
    """Download a bounded HTTPS archive to a private temporary path."""
    if not url.startswith("https://"):
        raise ValueError("release download must use HTTPS")
    opener = urllib.request.build_opener(HTTPSRedirects())
    with opener.open(url, timeout=30) as response, destination.open("xb") as output:
        size = 0
        while chunk := response.read(1024 * 1024):
            size += len(chunk)
            if size > MAX_DOWNLOAD:
                raise ValueError("release archive exceeds size limit")
            output.write(chunk)


def validate_members(bundle: zipfile.ZipFile) -> None:
    """Require exactly two bounded regular files without duplicate member names."""
    members = bundle.infolist()
    if sorted(item.filename for item in members) != ["agent-toolkit", "metadata.json"]:
        raise ValueError("unsafe archive members")
    for item in members:
        if not stat.S_ISREG(item.external_attr >> 16) or item.file_size > MAX_DOWNLOAD:
            raise ValueError("unsafe archive member type or size")


def extract_binary(
    archive: Path, destination: Path, pin: ReleasePin, target: str
) -> None:
    """Verify the ZIP and copy only the expected regular executable member."""
    record = pin["targets"][target]
    if hashlib.sha256(archive.read_bytes()).hexdigest() != record["sha256"]:
        raise ValueError("release archive checksum mismatch")
    with zipfile.ZipFile(archive) as bundle:
        validate_members(bundle)
        metadata = json.loads(bundle.read("metadata.json"))
        expected = {
            "version": pin["version"],
            "protocol_version": pin["protocol_version"],
            "target": target,
        }
        if metadata != expected:
            raise ValueError("release metadata mismatch")
        with destination.open("xb") as output:
            output.write(bundle.read("agent-toolkit"))
    destination.chmod(0o700)
    verify_binary(destination, record["binary_sha256"])


class Launcher:
    """Serialize cache publication and execution for a single reviewed release."""

    def __init__(self, root: Path, cache: Path, pin: ReleasePin, target: str) -> None:
        self.root = root
        self.cache = cache
        self.pin = pin
        self.target = target

    def prepare(self, directory: Path) -> Path:
        """Return a verified cache hit or atomically publish a verified download."""
        binary = directory / "agent-toolkit"
        record = self.pin["targets"][self.target]
        if os.path.lexists(binary):
            verify_binary(binary, record["binary_sha256"])
            return binary

        with tempfile.TemporaryDirectory(
            prefix="download-", dir=directory
        ) as temporary:
            staging = Path(temporary)
            archive = staging / "release.zip"
            url = f"{RELEASE_URL}/{self.pin['version']}/{record['archive']}"
            download(url, archive)
            candidate = staging / "agent-toolkit"
            extract_binary(archive, candidate, self.pin, self.target)
            os.replace(candidate, binary)
        return binary

    def run(self, arguments: list[str]) -> int:
        """Check compatibility, lock the private cache, and execute verified bytes."""
        verify_backend(self.root, self.pin["protocol_version"])
        private_directory(self.cache)
        directory = self.cache / f"{self.pin['version']}-{self.target}"
        private_directory(directory)
        descriptor = os.open(
            directory / "launch.lock", os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600
        )
        with os.fdopen(descriptor, "r+") as lock:
            info = os.fstat(lock.fileno())
            if (
                not stat.S_ISREG(info.st_mode)
                or info.st_uid != os.getuid()
                or info.st_mode & 0o077
            ):
                raise ValueError("unsafe cache lock")
            fcntl.flock(lock, fcntl.LOCK_EX)
            binary = self.prepare(directory)
            command = [str(binary), "--root", str(self.root), *arguments]
            return subprocess.run(command, check=False).returncode


def main() -> int:
    """Launch from this source checkout or explain the explicit Bash fallback."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--version", action="store_true", help="print verified TUI version"
    )
    arguments = parser.parse_args()
    base = Path(os.environ.get("XDG_CACHE_HOME", Path.home() / ".cache"))
    try:
        pin = load_pin(PIN)
        launcher = Launcher(ROOT, base / "agent-toolkit-tui", pin, host_target())
        return launcher.run(["--version"] if arguments.version else [])
    except (
        OSError,
        ValueError,
        KeyError,
        TypeError,
        subprocess.CalledProcessError,
        zipfile.BadZipFile,
    ) as error:
        print(
            f"TUI launch failed: {error}\nBash fallback: bash scripts/install.sh",
            file=sys.stderr,
        )
        return 1


if __name__ == "__main__":
    sys.exit(main())
