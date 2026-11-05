#!/usr/bin/env python3
"""Launch current source twice in isolation and quit without changing settings."""

from __future__ import annotations

import errno
import fcntl
import hashlib
import os
from pathlib import Path
import pty
import re
import select
import shutil
import signal
import struct
import subprocess
import tempfile
import termios
import time


def snapshot(paths: list[Path]) -> dict[str, str]:
    """Hash watched settings and provider files, including link destinations."""
    result = {}
    for path in paths:
        for item in [path, *path.rglob("*")] if path.is_dir() else [path]:
            if item.is_symlink():
                result[str(item)] = "link:" + os.readlink(item)
            elif item.is_file():
                result[str(item)] = hashlib.sha256(item.read_bytes()).hexdigest()
    return result


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


def drive_terminal(process: subprocess.Popen, master: int, theme: bytes) -> None:
    """Observe setup, quit, and discard the unsaved initial wizard if prompted."""
    deadline = time.monotonic() + 90
    output = b""
    quitting = discarded = False
    while time.monotonic() < deadline:
        output = (output + read_terminal(master))[-262144:]
        plain = re.sub(rb"\x1b\[[0-?]*[ -/]*[@-~]", b"", output)
        if process.poll() is not None:
            if process.returncode != 0 or not quitting:
                raise ValueError(f"source startup/quit failed: {output[-2000:]!r}")
            return
        if not quitting and all(
            marker in plain for marker in (b"LLM providers", b"q quit", theme)
        ):
            os.write(master, b"q")
            quitting = True
        if quitting and not discarded and b"Discard" in plain:
            os.write(master, b"d")
            discarded = True
    raise ValueError(f"source startup/quit timeout: {output[-2000:]!r}")


def terminal_check(root: Path, environment: dict[str, str], theme: bytes) -> None:
    """Run real Make with a sized PTY and reap the process group on failure."""
    master, slave = pty.openpty()
    process = None
    try:
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 40, 120, 0, 0))
        process = subprocess.Popen(
            ["make", "install"],
            cwd=root,
            env=environment,
            stdin=slave,
            stdout=slave,
            stderr=slave,
            start_new_session=True,
        )
        os.close(slave)
        slave = -1
        drive_terminal(process, master, theme)
    finally:
        if process is not None:
            if process.poll() is None:
                os.killpg(process.pid, signal.SIGKILL)
            process.wait()
        os.close(master)
        if slave != -1:
            os.close(slave)


def prepare_checkout(base: Path) -> tuple[Path, dict[str, str], list[Path]]:
    """Copy source to a spaced path and isolate every provider and settings root."""
    source = Path(__file__).resolve().parents[1]
    root = base / "checkout with spaces"
    root.mkdir()
    for name in (".agents", "scripts", "cmd", "internal"):
        shutil.copytree(
            source / name, root / name, ignore=shutil.ignore_patterns("__pycache__")
        )
    for name in ("Makefile", "go.mod", "go.sum", ".env.template"):
        shutil.copyfile(source / name, root / name)
    environment = dict(os.environ, TERM="xterm-256color")
    for name in ("HOME", "AGENTS_HOME", "CODEX_HOME", "CLAUDE_HOME"):
        environment[name] = str(base / name.lower())
        Path(environment[name]).mkdir()
    environment["AGENT_TOOLKIT_HOME"] = str(root)
    environment["AGENT_TOOLKIT_ENV_FILE"] = str(root / ".env")
    for name in ("GOPATH", "GOCACHE"):
        environment[name] = subprocess.check_output(
            ["go", "env", name], text=True
        ).strip()
    watched = [
        Path(environment[name]) for name in ("AGENTS_HOME", "CODEX_HOME", "CLAUDE_HOME")
    ]
    return root, environment, [*watched, root / ".env", root / "config"]


def main() -> None:
    """Require current-source startup and read-only quit on both launches."""
    with tempfile.TemporaryDirectory(prefix="toolkit source smoke ") as directory:
        root, environment, watched = prepare_checkout(Path(directory))
        before = snapshot(watched)
        for theme in (b"Midnight", b"SourceChanged"):
            if theme == b"SourceChanged":
                model = root / "internal/tui/model.go"
                model.write_text(model.read_text().replace("Midnight", "SourceChanged"))
            terminal_check(root, environment, theme)
            if snapshot(watched) != before:
                raise ValueError(
                    "source setup changed provider files or toolkit settings"
                )
        print("OK: current-source startup, source change, and read-only quit")


if __name__ == "__main__":
    main()
