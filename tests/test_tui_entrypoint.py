"""Verify terminal dispatch and safe failure through the actual setup entrypoint."""

from __future__ import annotations

import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


class EntrypointTests(unittest.TestCase):
    """Run Make and Bash with real terminal descriptors in an isolated checkout."""

    def setUp(self) -> None:
        """Create a spaced checkout and observable Go and Bash boundaries."""
        temporary = tempfile.TemporaryDirectory(prefix="setup entrypoint ")
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name).resolve()
        scripts = self.root / "scripts"
        scripts.mkdir()
        shutil.copyfile(ROOT / "scripts/setup.sh", scripts / "setup.sh")
        shutil.copyfile(ROOT / "Makefile", self.root / "Makefile")
        (scripts / "install.sh").write_text(
            'printf "%s\\n" "$@" > "$(dirname "$0")/../bash-arguments"\n'
        )
        self.commands = self.root / "bin"
        self.commands.mkdir()
        self.go = self.commands / "go"
        self.go.write_text(
            "#!/bin/bash\n"
            'printf "%s\\n" "$PWD" "$@" > launched\n'
            "cat source >> launched\n"
            "if [[ ${FIXTURE_BUILD_FAILURE:-0} == 1 ]]; then\n"
            '  printf "compiler diagnostic\\n" >&2; exit 42\nfi\n'
        )
        self.go.chmod(0o700)
        (self.root / "source").write_text("first source\n")
        self.environment = dict(os.environ, PATH=str(self.commands))
        for name in ("bash", "dirname", "cat", "make"):
            (self.commands / name).symlink_to(shutil.which(name))

    def run_setup(
        self, arguments: list[str], terminal: bool = True, cwd: Path | None = None
    ) -> subprocess.CompletedProcess:
        """Run the entrypoint with bounded output and optional real terminal FDs."""
        if not terminal:
            return subprocess.run(
                arguments,
                cwd=cwd or self.root,
                env=self.environment,
                capture_output=True,
                text=True,
                timeout=15,
            )

        master, slave = os.openpty()
        try:
            return subprocess.run(
                arguments,
                cwd=cwd or self.root,
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

    def test_terminal_make_uses_current_source(self) -> None:
        """GIVEN unpinned source WHEN setup repeats THEN source edits are used."""
        first = self.run_setup(["make", "install"])
        (self.root / "source").write_text("second source\n")

        second = self.run_setup(["make", "install"])

        self.assertEqual((first.returncode, second.returncode), (0, 0))
        self.assertEqual(
            (self.root / "launched").read_text().splitlines(),
            [
                str(self.root),
                "run",
                "./cmd/agent-toolkit",
                "--root",
                str(self.root),
                "second source",
            ],
        )
        self.assertFalse((self.root / "bash-arguments").exists())

    def test_other_working_directory_uses_checkout_root(self) -> None:
        """GIVEN another cwd WHEN setup opens THEN Go runs in the spaced checkout."""
        result = self.run_setup(
            ["bash", str(self.root / "scripts/setup.sh")], cwd=Path("/tmp")
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(
            (self.root / "launched").read_text().splitlines()[0], str(self.root)
        )

    def test_bash_routes_work_without_go(self) -> None:
        """GIVEN no Go WHEN CLI or automation runs THEN Bash receives exact args."""
        self.go.unlink()
        arguments = [
            "--output-root",
            str(self.root / "output with spaces"),
            "--skill",
            "",
            "--dry-run",
        ]

        explicit = self.run_setup(["bash", "scripts/setup.sh", *arguments])

        self.assertEqual(explicit.returncode, 0, explicit.stderr)
        self.assertEqual(
            (self.root / "bash-arguments").read_text().splitlines(), arguments
        )
        for command, terminal in (
            (["make", "install-cli"], True),
            (["make", "install"], False),
        ):
            with self.subTest(command=command):
                result = self.run_setup(command, terminal=terminal)
                self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse((self.root / "launched").exists())

    def test_missing_go_stops_before_installation(self) -> None:
        """GIVEN no Go WHEN terminal setup runs THEN the Bash alternative is shown."""
        self.go.unlink()

        result = self.run_setup(["make", "install"])

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("requires Go", result.stderr)
        self.assertIn("make install-cli", result.stderr)
        self.assert_no_installation()

    def test_build_failure_keeps_diagnostics_and_stops(self) -> None:
        """GIVEN a compiler failure WHEN setup runs THEN no installer is invoked."""
        self.environment["FIXTURE_BUILD_FAILURE"] = "1"

        result = self.run_setup(["bash", "scripts/setup.sh"])

        self.assertEqual(result.returncode, 42)
        self.assertIn("compiler diagnostic", result.stderr)
        self.assertIn("make install-cli", result.stderr)
        for name in ("bash-arguments", "config", ".env"):
            self.assertFalse((self.root / name).exists(), name)

    def test_real_bash_dry_run_preserves_configuration(self) -> None:
        """GIVEN real Bash WHEN explicit dry-run runs THEN isolated state stays clean."""
        environment = dict(
            os.environ,
            AGENT_TOOLKIT_HOME=str(ROOT),
            AGENTS_HOME=str(self.root / "agents"),
            CODEX_HOME=str(self.root / "codex"),
            CLAUDE_HOME=str(self.root / "claude"),
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
                "--full-template",
                "false",
                "--concise",
                "false",
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
