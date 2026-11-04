#!/usr/bin/env python3
"""Check native release startup and exit without saving or applying settings."""

from __future__ import annotations

import argparse
import errno
import fcntl
import hashlib
import io
import json
import os
import platform
import pty
import re
import select
import signal
import struct
import subprocess
import sys
import tempfile
import termios
import time
import zipfile
from pathlib import Path

import validate_tui_release as validator


def native_target() -> str:
    """Return the supported host mapping, rejecting unknown architectures."""
    architecture = {"x86_64": "amd64", "arm64": "arm64", "aarch64": "arm64"}
    return (
        platform.system().lower()
        + "-"
        + architecture.get(platform.machine(), "unknown")
    )


def snapshot(root: Path) -> dict[str, str]:
    """Record managed files and links to detect settings or installation writes."""
    result = {}
    for name in ("config", ".env", ".agents", ".codex", ".claude"):
        path = root / name
        paths = [path, *path.rglob("*")] if path.is_dir() else [path]
        for item in paths:
            if item.is_symlink():
                result[str(item)] = "link:" + os.readlink(item)
            elif item.is_file():
                result[str(item)] = hashlib.sha256(item.read_bytes()).hexdigest()
    return result


def isolated_environment(root: Path) -> dict[str, str]:
    """Keep provider homes, configuration, and cache away from user settings."""
    environment = {
        "PATH": os.environ.get("PATH", "/usr/bin:/bin"),
        "TERM": "xterm-256color",
    }
    for variable in (
        "HOME",
        "AGENTS_HOME",
        "CODEX_HOME",
        "CLAUDE_HOME",
        "AGENT_TOOLKIT_HOME",
        "XDG_CONFIG_HOME",
        "XDG_CACHE_HOME",
        "XDG_DATA_HOME",
        "XDG_STATE_HOME",
    ):
        destination = root / variable.lower()
        destination.mkdir()
        environment[variable] = str(destination)
    return environment


def read_terminal(master: int) -> bytes:
    """Read available terminal output, treating a closed slave as EOF."""
    if not select.select([master], [], [], 0.05)[0]:
        return b""
    try:
        return os.read(master, 65536)
    except OSError as error:
        if error.errno != errno.EIO:
            raise
        return b""


def drive_terminal(
    process: subprocess.Popen[bytes], master: int, timeout: float
) -> None:
    """Recognize a loaded screen, request quit, and discard only if prompted."""
    deadline = time.monotonic() + timeout
    output = b""
    quitting = False
    discarded = False
    while time.monotonic() < deadline:
        output = (output + read_terminal(master))[-262144:]
        plain = re.sub(rb"\x1b\[[0-?]*[ -/]*[@-~]", b"", output)
        if not quitting and all(
            marker in plain
            for marker in (
                b"setup and settings",
                b"LLM providers",
                b"q quit",
            )
        ):
            os.write(master, b"q")
            quitting = True
        if quitting and not discarded and b"Discard" in plain:
            os.write(master, b"d")
            discarded = True
        if process.poll() is not None:
            if process.returncode != 0 or not quitting:
                raise ValueError(
                    f"PTY startup/exit failed ({process.returncode}): {output[-2000:]!r}"
                )
            return
    raise ValueError(
        f"PTY {'exit' if quitting else 'startup'} timeout: {output[-2000:]!r}"
    )


def terminal_check(
    binary: Path, checkout: Path, environment: dict[str, str], timeout: float
) -> None:
    """Run a sized PTY and always kill and reap its process group on failure."""
    master, slave = pty.openpty()
    process = None
    try:
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 40, 120, 0, 0))
        process = subprocess.Popen(
            [str(binary), "--root", str(checkout)],
            cwd=checkout,
            env=environment,
            stdin=slave,
            stdout=slave,
            stderr=slave,
            start_new_session=True,
        )
        os.close(slave)
        slave = -1
        drive_terminal(process, master, timeout)
    finally:
        if process is not None:
            try:
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass  # The entire session already exited.
            process.wait()
        os.close(master)
        if slave != -1:
            os.close(slave)


def verified_binary(directory: Path, version: str, target: str) -> bytes:
    """Validate the complete release and bind extraction to verified bytes."""
    validator.validate_release(version, directory)
    records = validator.validate_manifest(
        validator.read_file(directory / "release.json"), version
    )
    record = records[target]
    archive = validator.read_file(directory / record["archive"])
    validator.validate_archive(archive, record, version, target)
    with zipfile.ZipFile(io.BytesIO(archive)) as bundle:
        return validator.read_member(bundle, "agent-toolkit")


def version_check(
    binary: Path, environment: dict[str, str], version: str, timeout: float
) -> None:
    """Require the exact protocol/version line from the native executable."""
    result = subprocess.run(
        [str(binary), "--version"],
        env=environment,
        cwd=environment["AGENT_TOOLKIT_HOME"],
        capture_output=True,
        timeout=timeout,
        check=True,
    )
    expected = f"agent-toolkit {version}; managed protocol 1\n".encode()
    if result.stdout != expected or result.stderr:
        raise ValueError(
            f"version output mismatch: {result.stdout!r} {result.stderr!r}"
        )


def runtime_checks(
    binary: Path,
    checkout: Path,
    home: Path,
    version: str,
    timeout: float,
    checks: dict[str, dict[str, str]],
) -> None:
    """Record independent version, interactive lifecycle, and write outcomes."""
    environment = isolated_environment(home)
    environment["AGENT_TOOLKIT_HOME"] = str(checkout)
    before = snapshot(checkout)
    try:
        version_check(binary, environment, version, timeout)
        checks["version"] = {
            "status": "passed",
            "reason": "exact native version output",
        }
        terminal_check(binary, checkout, environment, timeout)
        checks["pty"] = {"status": "passed", "reason": "loaded screen and clean quit"}
    finally:
        if snapshot(checkout) != before or any(home.rglob("*/*")):
            raise ValueError("unexpected managed/configuration/cache writes")
        checks["writes"] = {"status": "passed", "reason": "no managed writes"}


def smoke(
    version: str, target: str, directory: Path, checkout: Path, timeout: float = 20
) -> dict[str, object]:
    """Return explicit required-check outcomes, including diagnostics on failure."""
    report: dict[str, object] = {"version": version, "target": target, "checks": {}}
    checks = report["checks"] = {
        name: {"status": "failed", "reason": "not reached"}
        for name in ("host", "integrity", "version", "pty", "writes")
    }
    try:
        if target != native_target():
            raise ValueError(
                f"host mismatch: expected {target}, found {native_target()}"
            )
        checks["host"] = {"status": "passed", "reason": "native host matches"}
        data = verified_binary(directory, version, target)
        checks["integrity"] = {
            "status": "passed",
            "reason": "complete release verified",
        }
        with tempfile.TemporaryDirectory(prefix="tui-smoke-") as temporary:
            root = Path(temporary)
            binary = root / "agent-toolkit"
            binary.write_bytes(data)
            binary.chmod(0o755)
            home = root / "isolated"
            home.mkdir()
            runtime_checks(binary, checkout.resolve(), home, version, timeout, checks)
    except (OSError, ValueError, subprocess.SubprocessError) as error:
        report["error"] = str(error)
        for check in checks.values():
            if check["status"] == "failed":
                check["reason"] = str(error)
    return report


def main() -> int:
    """Write a machine-readable report even when required runtime checks fail."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", required=True)
    parser.add_argument("--target", choices=validator.TARGETS, required=True)
    parser.add_argument("--directory", type=Path, required=True)
    parser.add_argument("--checkout", type=Path, required=True)
    parser.add_argument("--report", type=Path, required=True)
    args = parser.parse_args()
    report = smoke(args.version, args.target, args.directory, args.checkout)
    args.report.write_text(json.dumps(report, indent=2) + "\n")
    return int("error" in report)


if __name__ == "__main__":
    sys.exit(main())
