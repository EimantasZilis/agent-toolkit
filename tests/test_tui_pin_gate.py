"""Check same-run artifact gates with local Git ancestry and fake GitHub responses."""

from __future__ import annotations

import copy
import os
import sys
import unittest
import urllib.error
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
import test_tui_pin_proposal as proposals
import tui_pin_gate as gate


class FakeGitHub(proposals.FakeGitHub):
    """Extend the proposal service only with the read-only Actions artifact boundary."""

    def __init__(self, releases: proposals.publishing.FakeGitHub) -> None:
        """Store exact artifact identities and expiry state."""
        super().__init__(releases)
        self.artifacts: dict[str, dict[str, object]] = {}

    def read(self, path: str) -> object:
        """Return complete run inventories or exact artifact objects."""
        if path.startswith("/actions/artifacts/"):
            artifact_id = path.rsplit("/", 1)[-1]
            if artifact_id not in self.artifacts:
                raise urllib.error.HTTPError(path, 404, "missing", {}, None)
            return self.artifacts[artifact_id]
        if path.startswith("/actions/runs/"):
            return {"artifacts": list(self.artifacts.values())}
        return super().read(path)


class GateTests(unittest.TestCase):
    """Exercise prepare/aggregate/write without faking internal gate decisions."""

    def setUp(self) -> None:
        """Create one local trusted commit and bind all retained evidence to it."""
        self.fixture = proposals.ProposalTests()
        self.fixture.setUp()
        self.addCleanup(self.fixture.doCleanups)
        self.root = self.fixture.fixture.fixture.root
        previous = Path.cwd()
        os.chdir(self.root)
        self.addCleanup(os.chdir, previous)
        sha = self.make_tooling()
        self.bind_candidate(sha)
        self.record = self.fixture.record
        self.api = FakeGitHub(self.fixture.fixture.api)
        self.api.branches["main"] = sha
        self.api.commits[sha] = {"tree": {"sha": "e" * 40}}
        self.seed_evidence(sha)

    def make_tooling(self) -> str:
        """Create the real paths required by the ancestry and backend-contract preflight."""
        root = self.root / "tooling"
        root.mkdir()
        for name in (
            "go.mod",
            "scripts/build_tui.py",
            "scripts/managed.sh",
            "cmd/agent-toolkit/main.go",
            ".agents/skills/fixture",
        ):
            path = root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text("fixture\n")
        git = gate.candidate.git
        git(root, "init", "-q")
        git(root, "add", ".")
        git(
            root,
            "-c",
            "user.name=Fixture",
            "-c",
            "user.email=fixture@example.test",
            "commit",
            "-qm",
            "trusted",
        )
        sha = git(root, "rev-parse", "HEAD")
        git(root, "update-ref", "refs/remotes/origin/main", sha)
        return sha

    def bind_candidate(self, sha: str) -> None:
        """Recreate publication and native identity snapshots for the local trusted commit."""
        fixture = self.fixture
        fixture.record.update(source_sha=sha, workflow_sha=sha)
        for report in fixture.record["reports"]:
            report.update(source_sha=sha, workflow_sha=sha)
        fixture.fixture.api = proposals.publishing.FakeGitHub()
        fixture.publication = fixture.fixture.publish()
        fixture.base = sha
        fixture.reports = [
            fixture.native_report(target) for target in gate.pin.validator.TARGETS
        ]
        (self.root / "payload").symlink_to(
            fixture.fixture.directory, target_is_directory=True
        )
        gate.candidate.write_json(Path("candidate-result.json"), fixture.record)
        gate.candidate.write_json(Path("publication-report.json"), fixture.publication)
        Path("pin-reports").mkdir()
        for report in fixture.reports:
            gate.candidate.write_json(
                Path("pin-reports") / f"pin-{report['target']}.json", report
            )

    def seed_evidence(self, sha: str) -> None:
        """Bind artifact names/IDs and workflow outputs before the gate runs."""
        names = gate.publication_gate.artifact_names(self.record, "458")
        names["800"] = "tui-publication-2"
        self.ids = {
            target: str(index + 900)
            for index, target in enumerate(gate.pin.validator.TARGETS)
        }
        names.update(
            {identity: f"tui-pin-{target}-2" for target, identity in self.ids.items()}
        )
        names["950"] = "tui-pin-result-2"
        self.api.artifacts = {
            identity: {
                "id": int(identity),
                "name": name,
                "expired": False,
                "workflow_run": {"id": 123},
            }
            for identity, name in names.items()
        }
        self.set_environment(sha)
        gate.candidate.write_json(Path("pin-report-ids.json"), self.ids)

    def set_environment(self, sha: str) -> None:
        """Supply the same trusted outputs that the workflow passes to downstream jobs."""
        environment = {
            field.upper(): self.record[field] for field in gate.candidate.IDENTITY
        }
        environment.update(
            GITHUB_REPOSITORY=self.record["repository"],
            GITHUB_RUN_ID="123",
            GITHUB_REF="refs/heads/main",
            DEFAULT_BRANCH="main",
            RECORD_ID="457",
            RESULT_ID="458",
            VERIFICATION_ATTEMPT="1",
            PUBLICATION_ID="800",
            PUBLICATION_ATTEMPT="2",
            BASE_SHA=sha,
            PIN_RESULT_ID="950",
            PROPOSE_PIN="true",
            PUBLISH="true",
            GITHUB_OUTPUT=str(self.root / "outputs"),
            GITHUB_STEP_SUMMARY=str(self.root / "summary"),
            PIN_NEEDS=gate.json.dumps(
                {
                    name: {"result": "success"}
                    for name in ("pin-preflight", "launcher", "publication")
                }
            ),
        )
        patcher = patch.dict(os.environ, environment)
        patcher.start()
        self.addCleanup(patcher.stop)

    def test_complete_gates_create_one_proposal_from_exact_reports(self) -> None:
        """GIVEN retained same-run evidence WHEN gates pass THEN create one unapproved pin PR."""
        gate.prepare(self.api)
        gate.aggregate(self.api)
        gate.write(self.api)

        result = gate.candidate.load_json(Path("proposal-report.json"))
        self.assertEqual(result["status"], "created")
        self.assertIn("base_sha=", Path("outputs").read_text())
        self.assertIn("/artifacts/800", self.api.pulls_state[0]["body"])
        self.assertIn("/artifacts/950", self.api.pulls_state[0]["body"])
        self.assertEqual(gate.resolve_reports(self.api), self.ids)

    def test_artifact_expiry_foreign_run_and_renaming_block_writes(self) -> None:
        """GIVEN substituted or expired publication evidence WHEN prepared THEN no mutation."""
        original = copy.deepcopy(self.api.artifacts["800"])
        cases = (
            ("expired", True),
            ("workflow_run", {"id": 999}),
            ("name", "tui-publication-1"),
            ("id", 801),
        )
        for field, value in cases:
            with self.subTest(field=field):
                self.api.artifacts["800"] = dict(original, **{field: value})

                with self.assertRaises(ValueError):
                    gate.prepare(self.api)

                self.assertEqual(self.api.calls, [])
        del self.api.artifacts["800"]
        with self.assertRaises(ValueError):
            gate.prepare(self.api)

    def test_changed_base_and_raw_report_substitution_fail_closed(self) -> None:
        """GIVEN a moving base or tampered raw report WHEN aggregated THEN reject approval."""
        original = self.api.branches["main"]
        self.api.branches["main"] = "f" * 40

        with self.assertRaisesRegex(ValueError, "base moved"):
            gate.aggregate(self.api)
        self.api.branches["main"] = original
        gate.aggregate(self.api)
        path = next(Path("pin-reports").glob("*.json"))
        report = gate.candidate.load_json(path)
        report["pin_sha256"] = "f" * 64
        gate.candidate.write_json(path, report)

        with self.assertRaisesRegex(ValueError, "raw native"):
            gate.write(self.api)
        self.assertEqual(self.api.calls, [])

    def test_failed_jobs_and_wrong_attempt_cannot_approve(self) -> None:
        """GIVEN skipped launcher jobs or stale publication WHEN aggregated THEN no approved result."""
        jobs = {
            name: {"result": "success"}
            for name in ("pin-preflight", "launcher", "publication")
        }
        jobs["launcher"]["result"] = "skipped"
        with patch.dict(os.environ, PIN_NEEDS=gate.json.dumps(jobs)):
            with self.assertRaises(ValueError):
                gate.aggregate(self.api)
        with patch.dict(os.environ, PUBLICATION_ATTEMPT="1"):
            with self.assertRaisesRegex(ValueError, "another attempt"):
                gate.prepare(self.api)

        self.assertFalse(Path("pin-result.json").exists())
        self.assertEqual(self.api.calls, [])

    def test_proposal_requires_explicit_inputs_and_trusted_default_ref(self) -> None:
        """GIVEN missing dispatch intent or an untrusted ref WHEN prepared THEN reject before writes."""
        for changes in (
            {"PUBLISH": "false"},
            {"PROPOSE_PIN": "false"},
            {"GITHUB_REF": "refs/heads/feature"},
        ):
            with patch.dict(os.environ, changes):
                with self.assertRaises(ValueError):
                    gate.prepare(self.api)

        self.assertEqual(self.api.calls, [])


if __name__ == "__main__":
    unittest.main()
