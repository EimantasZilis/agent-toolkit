#!/usr/bin/env python3
"""Check published launcher pins in disposable source and proposal-base checkouts."""

from __future__ import annotations

import argparse
import fcntl
import hashlib
import json
import os
import pty
import re
import shlex
import struct
import subprocess
import sys
import tempfile
import termios
from pathlib import Path

import publish_tui_release as publication
import smoke_tui_release as runtime
import tui_candidate as candidate
import validate_tui_release as validator

PIN = "scripts/tui-release.json"
PRODUCTION_REPOSITORY = "EimantasZilis/agent-toolkit"
CHECKS = {
    "fresh",
    "cached",
    "nonterminal",
    "arguments",
    "install_cli",
    "dry_run",
    "writes",
}


def check_publication(
    record: dict[str, object], report: dict[str, object], directory: Path
) -> bytes:
    """Bind successful publication evidence to independently retained candidate bytes."""
    publication.validate_candidate(record, directory, record["repository"])
    fields = (
        *candidate.IDENTITY,
        "repository",
        "hashes",
        "candidate_record_id",
        "verification_attempt",
        "report_artifact_ids",
        "jobs",
        "reports",
    )
    if any(report.get(field) != record[field] for field in fields):
        raise ValueError("publication candidate identity mismatch")
    if report.get("status") != "verified" or report.get("public") != "passed":
        raise ValueError("publication is not publicly verified")
    if report.get("draft") != "passed" or report.get("tag") != record["version"]:
        raise ValueError("publication draft or tag mismatch")
    expected_url = (
        f"https://github.com/{record['repository']}/releases/tag/{record['version']}"
    )
    if (
        report.get("release_url") != expected_url
        or type(report.get("release_id")) is not int
        or report["release_id"] <= 0
    ):
        raise ValueError("publication release identity mismatch")
    for stage in ("draft", "public"):
        expected = {
            name: {"expected": digest, "actual": digest}
            for name, digest in record["hashes"].items()
        }
        if report.get(stage + "_hashes") != expected:
            raise ValueError("publication download hashes mismatch")
    return validator.read_file(directory / "release.json")


def require_checkout(root: Path, sha: str) -> None:
    """Require a clean isolated checkout at the exact checked source or base commit."""
    if not re.fullmatch(r"[0-9a-f]{40}", sha):
        raise ValueError("invalid checkout SHA")
    if candidate.git(root, "rev-parse", "HEAD") != sha:
        raise ValueError("checkout SHA changed")
    if candidate.git(root, "status", "--porcelain", "--untracked-files=all"):
        raise ValueError("pin checks require a clean disposable checkout")


def terminal_command(
    command: list[str], root: Path, environment: dict[str, str], timeout: float
) -> None:
    """Reuse the bounded screen/quit driver through real Make and setup commands."""
    master, slave = pty.openpty()
    process = None
    try:
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 40, 120, 0, 0))
        process = subprocess.Popen(
            command,
            cwd=root,
            env=environment,
            stdin=slave,
            stdout=slave,
            stderr=slave,
            start_new_session=True,
        )
        os.close(slave)
        slave = -1
        runtime.drive_terminal(process, master, timeout)
    finally:
        if process is not None:
            runtime.stop_process(process)
        os.close(master)
        if slave != -1:
            os.close(slave)


def run_command(command: list[str], root: Path, environment: dict[str, str]) -> bytes:
    """Bound command output, lifetime, and descendant cleanup at the process boundary."""
    with subprocess.Popen(
        command,
        cwd=root,
        env=environment,
        stdin=subprocess.DEVNULL,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        start_new_session=True,
    ) as process:
        try:
            stdout, _ = runtime.version_output(process, 60)
        finally:
            runtime.stop_process(process)
    return stdout


def create_environment(home: Path, checkout: Path) -> dict[str, str]:
    """Isolate all writable state and put narrow instrumentation ahead of Bash/Python."""
    environment = runtime.isolated_environment(home)
    environment.update(
        PIN_CHECK_ROOT=str(checkout),
        PIN_CHECK_LOG=str(home / "events.jsonl"),
        PYTHONDONTWRITEBYTECODE="1",
        MARKDOWN_OUTPUT_PATH="",
        AGENT_TOOLKIT_ENV_FILE=str(home / "settings.env"),
        AGENT_TOOLKIT_HOME=str(checkout),
    )
    commands = home / "bin"
    commands.mkdir()
    harness = Path(__file__).with_name("tui_pin_harness.py")
    for name, kind in (("python3", "python"), ("bash", "bash")):
        path = commands / name
        command = shlex.join([sys.executable, str(harness), kind])
        path.write_text(f'#!/bin/sh\nexec {command} "$@"\n')
        path.chmod(0o700)
    environment["PATH"] = str(commands) + os.pathsep + environment["PATH"]
    return environment


def events(environment: dict[str, str]) -> list[dict[str, str]]:
    """Read observed route/download events without manufacturing passing outcomes."""
    path = Path(environment["PIN_CHECK_LOG"])
    return (
        [json.loads(line) for line in path.read_text().splitlines()]
        if path.exists()
        else []
    )


def launcher_cycle(
    checkout: Path, environment: dict[str, str], pin: bytes, target: str, offline: bool
) -> dict[str, object]:
    """Verify the native version and terminal setup with a fresh or populated cache."""
    manifest = validator.parse_json(pin)
    before = len(events(environment))
    environment["PIN_CHECK_OFFLINE"] = "1" if offline else "0"
    output = run_command(
        ["python3", "scripts/launch_tui.py", "--version"], checkout, environment
    )
    expected = f"agent-toolkit {manifest['version']}; managed protocol 1\n".encode()
    if output != expected:
        raise ValueError("launcher native version mismatch")
    terminal_command(["make", "install"], checkout, environment, 60)
    binary = (
        Path(environment["XDG_CACHE_HOME"])
        / "agent-toolkit-tui"
        / f"{manifest['version']}-{target}"
        / "agent-toolkit"
    )
    if (
        hashlib.sha256(binary.read_bytes()).hexdigest()
        != manifest["targets"][target]["binary_sha256"]
    ):
        raise ValueError("downloaded executable hash mismatch")
    observed = events(environment)[before:]
    archive = manifest["targets"][target]["archive"]
    repository = environment.get("PIN_CHECK_REPOSITORY", PRODUCTION_REPOSITORY)
    url = f"https://github.com/{repository}/releases/download/{manifest['version']}/{archive}"
    expected_events = [] if offline else [{"kind": "download", "value": url}]
    if observed != expected_events:
        raise ValueError("unexpected launcher download or Bash fallback")
    return {"status": "passed", "downloads": 0 if offline else 1, "url": url}


def bash_routes(checkout: Path, environment: dict[str, str]) -> dict[str, object]:
    """Observe no-argument installer routes, then exercise the real installer dry run."""
    before = len(events(environment))
    instrumented = dict(environment, PIN_CHECK_ROUTE="1", PIN_CHECK_OFFLINE="1")
    run_command(["make", "install"], checkout, instrumented)
    run_command(["bash", "scripts/setup.sh", "--dry-run"], checkout, instrumented)
    run_command(["make", "install-cli"], checkout, instrumented)
    expected = [
        {"kind": "installer", "value": "[]"},
        {"kind": "installer", "value": '["--dry-run"]'},
        {"kind": "installer", "value": "[]"},
    ]
    if events(environment)[before:] != expected:
        raise ValueError("Bash route mismatch or unexpected TUI download")
    dry_run = [
        "bash",
        "scripts/setup.sh",
        "--provider",
        "codex",
        "--skill",
        "concise-output",
        "--no-global-guidance",
        "--dry-run",
    ]
    output = run_command(dry_run, checkout, environment)
    if b"Dry run: no files changed." not in output:
        raise ValueError("real installer dry run did not complete")
    return dict(
        {
            name: {"status": "passed", "instrumented": True}
            for name in ("nonterminal", "arguments", "install_cli")
        },
        dry_run={"status": "passed", "instrumented": False},
    )


def provider_state(home: Path, environment: dict[str, str]) -> list[str]:
    """Detect provider/configuration writes while allowing the expected launcher cache."""
    roots = [
        Path(value)
        for key, value in environment.items()
        if key
        in (
            "HOME",
            "AGENTS_HOME",
            "CODEX_HOME",
            "CLAUDE_HOME",
            "XDG_CONFIG_HOME",
            "XDG_DATA_HOME",
            "XDG_STATE_HOME",
        )
    ]
    return [str(path) for root in roots for path in root.rglob("*")] + [
        str(path) for path in (home / "settings.env",) if path.exists()
    ]


def check_one(
    checkout: Path,
    sha: str,
    pin: bytes,
    target: str,
    test_repository: str | None = None,
    fixture: Path | None = None,
) -> dict[str, object]:
    """Temporarily pin one clean checkout and restore its original manifest on exit."""
    require_checkout(checkout, sha)
    path = checkout / PIN
    original = path.read_bytes() if path.exists() else None
    before = runtime.snapshot(checkout)
    try:
        path.write_bytes(pin)
        with tempfile.TemporaryDirectory(prefix="tui-pin-") as temporary:
            return isolated_checks(
                checkout, Path(temporary), pin, target, test_repository, fixture
            )
    finally:
        if original is None:
            path.unlink(missing_ok=True)
        else:
            path.write_bytes(original)
        if runtime.snapshot(checkout) != before:
            raise ValueError("terminal/routing checks changed managed checkout files")
        require_checkout(checkout, sha)


def isolated_checks(
    checkout: Path,
    home: Path,
    pin: bytes,
    target: str,
    test_repository: str | None,
    fixture: Path | None,
) -> dict[str, object]:
    """Run both launcher cycles and routes without allowing provider/settings changes."""
    environment = create_environment(home, checkout)
    if test_repository:
        environment["PIN_CHECK_REPOSITORY"] = test_repository
    if fixture:
        environment["PIN_CHECK_FIXTURE"] = str(fixture)
    checks = {}
    try:
        checks["fresh"] = launcher_cycle(checkout, environment, pin, target, False)
        checks["cached"] = launcher_cycle(checkout, environment, pin, target, True)
        checks.update(bash_routes(checkout, environment))
    finally:
        if provider_state(home, environment):
            raise ValueError("unexpected provider or settings writes")
    checks["writes"] = {"status": "passed"}
    return checks


def evidence_identity(
    record: dict[str, object],
    publication_report: dict[str, object],
    pin: bytes,
    base_sha: str,
    target: str,
) -> dict[str, object]:
    """Bind a launcher report to the exact manifest, publication, and checked base."""
    return dict(
        {field: record[field] for field in candidate.IDENTITY},
        repository=record["repository"],
        target=target,
        base_sha=base_sha,
        release_id=publication_report["release_id"],
        publication_sha256=hashlib.sha256(
            json.dumps(publication_report, sort_keys=True).encode()
        ).hexdigest(),
        pin_sha256=hashlib.sha256(pin).hexdigest(),
    )


def check_pin(
    record: dict[str, object],
    publication_report: dict[str, object],
    directory: Path,
    source: Path,
    base: Path,
    base_sha: str,
    target: str,
    test_repository: str | None = None,
    fixture: Path | None = None,
) -> dict[str, object]:
    """Return evidence bound to publication, both checkouts, target, and transport."""
    pin = check_publication(record, publication_report, directory)
    if test_repository and not re.fullmatch(
        r"[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*", test_repository
    ):
        raise ValueError("invalid acceptance test repository")
    if target != runtime.native_target():
        raise ValueError("pin check requires the native target")
    source_checks = check_one(
        source, record["source_sha"], pin, target, test_repository, fixture
    )
    require_checkout(base, base_sha)
    base_checks = (
        source_checks
        if base_sha == record["source_sha"]
        else check_one(base, base_sha, pin, target, test_repository, fixture)
    )
    return dict(
        evidence_identity(record, publication_report, pin, base_sha, target),
        transport="fixture" if fixture else "published-https",
        download_repository=test_repository or PRODUCTION_REPOSITORY,
        status="passed",
        checkouts={"source": source_checks, "base": base_checks},
    )


def main() -> int:
    """Persist failed diagnostics as well as successful native published-asset evidence."""
    parser = argparse.ArgumentParser(description=__doc__)
    for name in (
        "candidate-directory",
        "candidate-record",
        "publication-report",
        "checkout",
        "base-checkout",
        "report",
    ):
        parser.add_argument("--" + name, type=Path, required=True)
    parser.add_argument("--base-sha", required=True)
    parser.add_argument("--target", choices=validator.TARGETS, required=True)
    parser.add_argument(
        "--test-repository", help="override repository only in the acceptance harness"
    )
    args = parser.parse_args()
    report = {"status": "failed"}
    try:
        report = check_pin(
            candidate.load_json(args.candidate_record),
            candidate.load_json(args.publication_report),
            args.candidate_directory,
            args.checkout.resolve(),
            args.base_checkout.resolve(),
            args.base_sha,
            args.target,
            args.test_repository,
        )
    except (
        OSError,
        ValueError,
        KeyError,
        TypeError,
        subprocess.SubprocessError,
    ) as error:
        report["error"] = str(error)
    candidate.write_json(args.report, report)
    return int(report["status"] != "passed")


if __name__ == "__main__":
    sys.exit(main())
