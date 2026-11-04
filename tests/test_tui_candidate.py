"""Check candidate identity and aggregate approval using inert local fixtures."""

from __future__ import annotations

import copy
import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
import tui_candidate as candidate
import test_tui_release as releases


class CandidateTests(unittest.TestCase):
    """Require complete, correctly bound native evidence before approval."""

    def setUp(self) -> None:
        """Create one candidate with four independent successful native reports."""
        self.record = {
            "version": "v0.1.0",
            "source_sha": "a" * 40,
            "workflow_sha": "b" * 40,
            "run_id": "123",
            "run_attempt": "1",
            "payload_id": "456",
        }
        self.jobs = {name: "success" for name in ("preflight", "build", "runtime")}
        self.reports = [
            dict(
                self.record,
                target=target,
                verification_attempt="2",
                checks={
                    name: {"status": "passed", "reason": "fixture native check"}
                    for name in candidate.CHECKS
                },
            )
            for target in candidate.validator.TARGETS
        ]

    def approve(self) -> dict[str, object]:
        """Aggregate a retry while preserving the original payload attempt."""
        return candidate.aggregate(self.record, self.reports, self.jobs, "2")

    def test_complete_retry_preserves_payload_identity(self) -> None:
        """GIVEN four bound reports WHEN retried THEN original identity is retained."""
        result = self.approve()

        self.assertEqual(result["status"], "approved")
        self.assertEqual(result["run_attempt"], "1")
        self.assertEqual(result["verification_attempt"], "2")
        self.assertEqual(result["manual_ui_review"], "pending")

    def test_required_jobs_cannot_be_missing_or_unsuccessful(self) -> None:
        """GIVEN failed/cancelled/skipped/missing jobs WHEN aggregated THEN reject."""
        for value in ("failure", "cancelled", "skipped", "unknown", None):
            with self.subTest(value=value):
                self.jobs = {
                    "preflight": "success",
                    "build": "success",
                    "runtime": value,
                }
                if value is None:
                    del self.jobs["runtime"]

                with self.assertRaisesRegex(ValueError, "required job"):
                    self.approve()

    def test_substituted_candidate_fields_are_rejected(self) -> None:
        """GIVEN any changed candidate/report identity WHEN aggregated THEN reject."""
        for field in (*candidate.IDENTITY, "verification_attempt"):
            with self.subTest(field=field):
                reports = copy.deepcopy(self.reports)
                reports[0][field] = "other"

                with self.assertRaises(ValueError):
                    candidate.aggregate(self.record, reports, self.jobs, "2")

    def test_target_reports_must_be_exact(self) -> None:
        """GIVEN missing, duplicate, or unknown targets WHEN aggregated THEN reject."""
        variants = (
            self.reports[:-1],
            self.reports + [self.reports[0]],
            self.reports[:-1] + [self.reports[0]],
        )
        for reports in variants:
            with self.subTest(count=len(reports)):
                with self.assertRaisesRegex(ValueError, "target reports"):
                    candidate.aggregate(self.record, reports, self.jobs, "2")

    def test_invalid_check_evidence_is_rejected(self) -> None:
        """GIVEN unknown, failed, skipped, or incomplete checks WHEN checked THEN reject."""
        for name in candidate.CHECKS:
            for status in ("skipped", "failed", "cancelled", "unknown", None):
                with self.subTest(name=name, status=status):
                    reports = copy.deepcopy(self.reports)
                    reports[0]["checks"][name] = {
                        "status": status,
                        "reason": "fixture failure",
                    }

                    with self.assertRaises(ValueError):
                        candidate.aggregate(self.record, reports, self.jobs, "2")

    def test_supported_skip_policy_is_explicit(self) -> None:
        """GIVEN proven unsupported PTY capability WHEN checked THEN retain the skip."""
        self.reports[0]["checks"]["pty"] = {
            "status": "skipped",
            "reason": "host lacks PTY capability",
            "unsupported_capability": True,
        }

        result = self.approve()

        self.assertEqual(result["reports"][0]["checks"]["pty"]["status"], "skipped")
        for reason in ("startup error", "timeout", "missing runner"):
            self.reports[0]["checks"]["pty"]["reason"] = reason

            with self.assertRaises(ValueError):
                self.approve()

    def test_artifact_selection_requires_exact_names_and_ids(self) -> None:
        """GIVEN attempt-specific artifacts WHEN selected THEN preserve exact IDs."""
        artifacts = [
            {"name": f"tui-report-{target}-2", "id": index + 1, "expired": False}
            for index, target in enumerate(candidate.validator.TARGETS)
        ]

        result = candidate.require_artifacts(artifacts, "2")

        self.assertEqual(list(result.values()), ["1", "2", "3", "4"])
        for items in (
            artifacts[:-1],
            artifacts + [artifacts[0]],
            [dict(item, expired=True) for item in artifacts],
        ):
            with self.assertRaises(ValueError):
                candidate.require_artifacts(items, "2")
        with self.assertRaises(ValueError):
            candidate.require_artifacts(artifacts, "3")

    def test_invalid_version_never_invokes_git_or_build(self) -> None:
        """GIVEN hostile or invalid versions WHEN preflighted THEN no tools run."""
        for version in (
            "bad",
            "v1.2",
            "v1.2.3\n",
            "$(touch injected)",
            "v1.2.3;echo hostile",
        ):
            with (
                self.subTest(version=version),
                patch.object(
                    subprocess,
                    "check_output",
                    side_effect=AssertionError("must not execute"),
                ),
            ):
                with self.assertRaisesRegex(ValueError, "invalid release version"):
                    candidate.preflight(
                        Path("missing"), version, "main", "a" * 40, "main"
                    )

    def test_reports_must_match_trusted_job_outputs(self) -> None:
        """GIVEN substituted record identity WHEN compared with job outputs THEN reject."""
        environment = {
            field.upper(): self.record[field] for field in candidate.IDENTITY
        }
        with patch.dict(os.environ, environment):
            candidate.require_context(self.record)
            for field in candidate.IDENTITY:
                record = dict(self.record, **{field: "substituted"})

                with self.assertRaisesRegex(ValueError, "trusted workflow outputs"):
                    candidate.require_context(record)

    def test_payload_hash_changes_prevent_execution(self) -> None:
        """GIVEN a changed digest WHEN checked THEN reject before native execution."""
        fixture = releases.ReleaseTests()
        fixture.setUp()
        self.addCleanup(fixture.doCleanups)
        record = dict(self.record, version="v0.0.0-validation", hashes={})

        with self.assertRaisesRegex(ValueError, "hashes differ"):
            candidate.check_payload(record, fixture.directory)

    def test_cli_retains_failed_status_when_artifact_ids_are_missing(self) -> None:
        """GIVEN otherwise passing evidence WHEN report IDs are missing THEN fail closed."""
        fixture = releases.ReleaseTests()
        fixture.setUp()
        self.addCleanup(fixture.doCleanups)
        record = dict(self.record, version=releases.VERSION)
        record["hashes"] = candidate.payload_hashes(fixture.directory, releases.VERSION)
        report_root = fixture.root / "reports"
        report_root.mkdir()
        for report in self.reports:
            candidate.write_json(
                report_root / f"report-{report['target']}.json",
                dict(report, version=releases.VERSION),
            )
        candidate.write_json(fixture.root / "candidate.json", record)
        fixture.directory.rename(fixture.root / "payload")
        environment = dict(
            os.environ,
            **{field.upper(): str(record[field]) for field in candidate.IDENTITY},
        )
        environment.update(
            GITHUB_RUN_ATTEMPT="2",
            GITHUB_STEP_SUMMARY=str(fixture.root / "summary"),
            NEEDS=json.dumps(
                {name: {"result": status} for name, status in self.jobs.items()}
            ),
        )

        result = subprocess.run(
            [sys.executable, str(Path(candidate.__file__).resolve()), "aggregate"],
            cwd=fixture.root,
            env=environment,
            capture_output=True,
            text=True,
        )

        self.assertEqual(result.returncode, 1, result.stderr)
        evidence = candidate.load_json(fixture.root / "candidate-result.json")
        self.assertEqual(evidence["status"], "failed")
        self.assertIn("report-ids.json", evidence["error"])


class PreflightTests(unittest.TestCase):
    """Use real Git history to check trusted ancestry and source contracts."""

    def setUp(self) -> None:
        """Create a minimal trusted source commit and default-branch tracking ref."""
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        candidate.git(self.root, "init", "--initial-branch=main")
        for name in (
            "go.mod",
            "scripts/build_tui.py",
            "scripts/managed.sh",
            "cmd/agent-toolkit/main.go",
            ".agents/skills/fixture",
        ):
            path = self.root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text("fixture\n")
        candidate.git(self.root, "add", ".")
        self.commit()
        self.sha = candidate.git(self.root, "rev-parse", "HEAD")
        candidate.git(self.root, "update-ref", "refs/remotes/origin/main", self.sha)

    def commit(self) -> None:
        """Commit local fixture files without relying on user identity settings."""
        candidate.git(
            self.root,
            "-c",
            "user.name=Fixture",
            "-c",
            "user.email=fixture@example.invalid",
            "commit",
            "--allow-empty",
            "-m",
            "fixture",
        )

    def test_trusted_source_resolves_to_full_sha(self) -> None:
        """GIVEN a source on default history WHEN resolved THEN return one full SHA."""
        result = candidate.preflight(self.root, "v1.2.3", "main", self.sha, "main")

        self.assertEqual(result, self.sha)

    def test_untrusted_source_and_workflow_fail(self) -> None:
        """GIVEN an unmerged commit WHEN used as source or workflow THEN reject."""
        self.commit()
        unmerged = candidate.git(self.root, "rev-parse", "HEAD")

        for source, workflow in ((unmerged, self.sha), (self.sha, unmerged)):
            with self.assertRaises(subprocess.CalledProcessError):
                candidate.preflight(self.root, "v1.2.3", source, workflow, "main")

    def test_missing_source_contract_fails(self) -> None:
        """GIVEN a trusted source missing the builder WHEN checked THEN reject."""
        candidate.git(self.root, "rm", "scripts/build_tui.py")
        self.commit()
        sha = candidate.git(self.root, "rev-parse", "HEAD")
        candidate.git(self.root, "update-ref", "refs/remotes/origin/main", sha)

        with self.assertRaises(subprocess.CalledProcessError):
            candidate.preflight(self.root, "v1.2.3", sha, self.sha, "main")


if __name__ == "__main__":
    unittest.main()
