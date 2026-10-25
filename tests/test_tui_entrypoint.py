"""Verify terminal dispatch and safe failure through the actual setup entrypoint."""

from __future__ import annotations

import hashlib
import json
import os
import shlex
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

from scripts.build_tui import archive_binary
from scripts.launch_tui import TARGETS, host_target

ROOT = Path(__file__).resolve().parents[1]


class EntrypointTests(unittest.TestCase):
    """Run Make and Bash with real terminal descriptors in an isolated checkout."""

    def setUp(self) -> None:
        """Create a pinned release, read-only backend, and observable Bash fallback."""
        temporary = tempfile.TemporaryDirectory(prefix="setup entrypoint ")
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name).resolve()
        scripts = self.root / "scripts"
        scripts.mkdir()
        for name in ("setup.sh", "launch_tui.py"):
            shutil.copyfile(ROOT / "scripts" / name, scripts / name)
        shutil.copyfile(ROOT / "Makefile", self.root / "Makefile")
        (scripts / "install.sh").write_text("printf '%s\\n' \"$@\" > bash-arguments\n")
        self.backend = scripts / "managed.sh"
        self.backend.write_text('printf \'{"ok":true,"protocol_version":1}\\n\'\n')

        binary = self.root / "fixture-binary"
        binary.write_text('#!/bin/sh\nprintf \'%s\\n\' "$@" > "$2/launched"\n')
        self.archive = self.root / "release.zip"
        self.target = host_target()
        metadata = {"version": "v0.1.0", "protocol_version": 1, "target": self.target}
        archive_binary(binary, self.archive, metadata)
        self.pin = scripts / "tui-release.json"
        self.write_pin(binary)

        self.environment = dict(
            os.environ,
            XDG_CACHE_HOME=str(self.root / "cache"),
            FIXTURE_SYSTEM=os.uname().sysname,
        )
        self.install_network_fixture(scripts)

    def write_pin(self, binary: Path) -> None:
        """Pin archive and executable bytes without claiming a public release."""
        targets = {
            target: {
                "archive": f"agent-toolkit-v0.1.0-{target}.zip",
                "sha256": hashlib.sha256(self.archive.read_bytes()).hexdigest(),
                "binary_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
            }
            for target in TARGETS
        }
        self.pin.write_text(
            json.dumps({"version": "v0.1.0", "protocol_version": 1, "targets": targets})
        )

    def install_network_fixture(self, scripts: Path) -> None:
        """Keep the production launcher real while replacing the HTTPS response."""
        runner = scripts / "tui_network_runner.py"
        shutil.copyfile(ROOT / "tests/helpers/tui_network_runner.py", runner)
        commands = self.root / "bin"
        commands.mkdir()
        python = commands / "python3"
        command = f"{shlex.quote(sys.executable)} {shlex.quote(str(runner))}"
        python.write_text(f"#!/bin/sh\nexec {command}\n")
        python.chmod(0o700)
        go = commands / "go"
        go.write_text('#!/bin/sh\nprintf "unexpected Go invocation\\n" >&2\nexit 99\n')
        go.chmod(0o700)
        self.environment["PATH"] = f"{commands}{os.pathsep}{os.environ['PATH']}"

    def run_setup(
        self, arguments: list[str], terminal: bool = True
    ) -> subprocess.CompletedProcess:
        """Run the entrypoint with bounded output and optional real terminal FDs."""
        if not terminal:
            return subprocess.run(
                arguments,
                cwd=self.root,
                env=self.environment,
                capture_output=True,
                text=True,
                timeout=15,
            )

        master, slave = os.openpty()
        try:
            return subprocess.run(
                arguments,
                cwd=self.root,
                env=self.environment,
                stdin=slave,
                stdout=slave,
                stderr=subprocess.PIPE,
                text=True,
                timeout=15,
            )
        finally:
            os.close(slave)
            os.close(master)

    def assert_no_installation(self) -> None:
        """Ensure failures neither launch nor fall through to Bash installation."""
        for name in ("launched", "bash-arguments", "config", ".env"):
            self.assertFalse((self.root / name).exists(), name)

    def test_terminal_make_downloads_then_launches_offline(self) -> None:
        """GIVEN a usable pin WHEN terminal setup repeats THEN cache works offline."""
        first = self.run_setup(["make", "install"])
        self.environment["FIXTURE_DOWNLOAD_FAILURE"] = "1"

        second = self.run_setup(["make", "install"])

        self.assertEqual(
            (first.returncode, second.returncode), (0, 0), first.stderr + second.stderr
        )
        self.assertEqual(
            (self.root / "downloads").read_text().splitlines(), ["request"]
        )
        self.assertEqual(
            (self.root / "launched").read_text().splitlines(),
            ["--root", str(self.root)],
        )
        self.assertFalse((self.root / "bash-arguments").exists())

    def test_nonterminal_make_never_downloads(self) -> None:
        """GIVEN automation with a pin WHEN make install runs THEN Bash is used."""
        result = self.run_setup(["make", "install"], terminal=False)

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue((self.root / "bash-arguments").exists())
        self.assertFalse((self.root / "downloads").exists())

    def test_explicit_flags_and_cli_target_bypass_launcher(self) -> None:
        """GIVEN explicit CLI intent WHEN terminal setup runs THEN flags survive."""
        arguments = ["--provider", "codex", "--skill", "concise-output", "--dry-run"]

        result = self.run_setup(["bash", "scripts/setup.sh", *arguments])
        self.assertEqual(
            (self.root / "bash-arguments").read_text().splitlines(), arguments
        )
        explicit = self.run_setup(["make", "install-cli"])

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(explicit.returncode, 0, explicit.stderr)
        self.assertFalse((self.root / "downloads").exists())

    def test_explicit_paths_are_passed_as_single_arguments(self) -> None:
        """GIVEN paths with spaces WHEN dispatched THEN Bash receives exact values."""
        arguments = [
            "--output-root",
            str(self.root / "output with spaces"),
            "--dry-run",
        ]

        result = self.run_setup(["bash", "scripts/setup.sh", *arguments])

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(
            (self.root / "bash-arguments").read_text().splitlines(), arguments
        )

    def test_missing_pin_keeps_terminal_bash_route(self) -> None:
        """GIVEN no pin WHEN terminal setup runs THEN no download is attempted."""
        self.pin.unlink()

        result = self.run_setup(["make", "install"])

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue((self.root / "bash-arguments").exists())
        self.assertFalse((self.root / "downloads").exists())

    def test_missing_release_fails_without_bash_fallthrough(self) -> None:
        """GIVEN a missing release WHEN setup runs THEN no installation starts."""
        self.environment["FIXTURE_DOWNLOAD_FAILURE"] = "1"

        result = self.run_setup(["make", "install"])

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("release unavailable", result.stderr)
        self.assertIn("bash scripts/install.sh", result.stderr)
        self.assert_no_installation()

    def test_incompatible_backend_fails_before_download(self) -> None:
        """GIVEN a protocol mismatch WHEN setup runs THEN no installation occurs."""
        self.backend.write_text('printf \'{"ok":true,"protocol_version":2}\\n\'\n')

        result = self.run_setup(["make", "install"])

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("backend protocol mismatch", result.stderr)
        self.assertFalse((self.root / "downloads").exists())
        self.assert_no_installation()

    def test_invalid_pin_and_unsupported_platform_fail_safely(self) -> None:
        """GIVEN invalid pin/platform WHEN setup runs THEN nothing installs."""
        self.environment["FIXTURE_SYSTEM"] = "Windows"
        unsupported = self.run_setup(["make", "install"])
        self.pin.write_text("{}")

        invalid = self.run_setup(["make", "install"])

        self.assertNotEqual(unsupported.returncode, 0)
        self.assertIn("unsupported TUI platform", unsupported.stderr)
        self.assertNotEqual(invalid.returncode, 0)
        self.assertFalse((self.root / "downloads").exists())
        self.assert_no_installation()

    def test_real_bash_dry_run_preserves_configuration(self) -> None:
        """GIVEN real Bash WHEN explicit dry-run runs THEN isolated state stays clean."""
        environment = dict(
            os.environ,
            AGENT_TOOLKIT_ENV_FILE=str(self.root / "settings.env"),
            CODEX_HOME=str(self.root / "codex"),
            MARKDOWN_OUTPUT_PATH="",
        )

        result = subprocess.run(
            [
                "bash",
                str(ROOT / "scripts/setup.sh"),
                "--provider",
                "codex",
                "--skill",
                "concise-output",
                "--no-global-guidance",
                "--dry-run",
            ],
            cwd=self.root,
            env=environment,
            capture_output=True,
            text=True,
            timeout=15,
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("Dry run: no files changed.", result.stdout)
        self.assertFalse((self.root / "settings.env").exists())
        self.assertFalse((self.root / "codex").exists())


if __name__ == "__main__":
    unittest.main()
