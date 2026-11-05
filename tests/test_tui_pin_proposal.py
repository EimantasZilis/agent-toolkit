"""Exercise manifest-only GitHub proposals using an observable, stateful API boundary."""

from __future__ import annotations

import base64
import copy
import hashlib
import sys
import unittest
import urllib.error
import urllib.parse
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
import propose_tui_release_pin as proposal
import test_tui_release_publication as publishing
import tui_pin_gate as gate


class FakeGitHub(proposal.GitHub):
    """Implement only the Git/PR API operations that the proposal writer may use."""

    def __init__(self, releases: publishing.FakeGitHub) -> None:
        """Start with a default branch and an already verified release."""
        self.repository = releases.repository
        self.releases = releases
        self.branches = {"main": "d" * 40}
        self.commits = {"d" * 40: {"tree": {"sha": "e" * 40}}}
        self.blobs: dict[str, bytes] = {}
        self.pins: dict[str, bytes] = {}
        self.pulls_state: list[dict[str, object]] = []
        self.calls: list[tuple] = []
        self.tree: dict[str, object] = {}
        self.deny_pull = False
        self.race_branch = False
        self.race_base = False
        self.pin_mode = "100644"

    def tag_commit(self, version: str) -> str | None:
        """Delegate release state without permitting release mutations."""
        return self.releases.tag_commit(version)

    def release(self, version: str) -> dict[str, object] | None:
        """Read the verified publication's ownership marker and identity."""
        return self.releases.release(version)

    def assets(self, release_id: int) -> list[dict[str, object]]:
        """Expose the already published six-file inventory."""
        return self.releases.assets(release_id)

    def api(
        self, method: str, path: str, document: dict[str, object] | None = None
    ) -> object:
        """Log mutations and reject any operation outside the narrow proposal contract."""
        if method == "GET":
            return copy.deepcopy(self.read(path))
        if method != "POST":
            raise AssertionError("unexpected mutation: " + method + " " + path)
        self.calls.append((method, path, copy.deepcopy(document)))
        return copy.deepcopy(self.create(path, document))

    def read(self, path: str) -> object:
        """Model immutable commit/content reads and mutable refs/PR snapshots."""
        if path == "":
            return {"default_branch": "main"}
        if path.startswith("/git/ref/heads/"):
            return self.reference(path)
        if path.startswith("/git/commits/"):
            return self.commits[path.rsplit("/", 1)[-1]]
        if path.startswith("/contents/"):
            return self.content(path)
        if path.startswith("/compare/"):
            return self.comparison(path)
        if path.startswith("/git/trees/"):
            return self.read_tree(path)
        if path.startswith("/pulls?"):
            return self.pulls_state
        raise AssertionError("unexpected read: " + path)

    def read_tree(self, path: str) -> dict[str, object]:
        """Model regular pin mode separately from its exact content bytes."""
        if path.endswith("f" * 40):
            return {"tree": [{"path": "scripts", "type": "tree", "sha": "e" * 40}]}
        return {
            "tree": [
                {"path": "tui-release.json", "type": "blob", "mode": self.pin_mode}
            ]
        }

    def reference(self, path: str) -> dict[str, object]:
        """Return a missing-ref HTTP error rather than silently adopting a branch."""
        name = urllib.parse.unquote(path.removeprefix("/git/ref/heads/"))
        if name not in self.branches:
            raise urllib.error.HTTPError(path, 404, "missing", {}, None)
        return {"object": {"type": "commit", "sha": self.branches[name]}}

    def content(self, path: str) -> dict[str, object]:
        """Return exact base64 pin bytes or a genuine missing-file result."""
        sha = path.split("?ref=", 1)[1]
        if sha not in self.pins:
            raise urllib.error.HTTPError(path, 404, "missing", {}, None)
        return {
            "type": "file",
            "encoding": "base64",
            "content": base64.b64encode(self.pins[sha]).decode(),
        }

    def comparison(self, path: str) -> dict[str, object]:
        """Return one pin diff while allowing tests to inject extra/human changes."""
        sha = path.split("...", 1)[1]
        data = self.pins[sha]
        digest = hashlib.sha1(
            b"blob " + str(len(data)).encode() + b"\0" + data
        ).hexdigest()
        return {
            "total_commits": 1,
            "files": [
                {"filename": proposal.pin_checks.PIN, "status": "added", "sha": digest}
            ],
        }

    def create(self, path: str, document: dict[str, object]) -> object:
        """Create immutable objects, one new ref, or one PR; never update existing state."""
        if path == "/git/blobs":
            data = base64.b64decode(document["content"])
            digest = hashlib.sha1(
                b"blob " + str(len(data)).encode() + b"\0" + data
            ).hexdigest()
            self.blobs[digest] = data
            return {"sha": digest}
        if path == "/git/trees":
            self.tree = document
            return {"sha": "f" * 40}
        if path == "/git/commits":
            return self.create_commit_object(document)
        if path == "/git/refs":
            name = document["ref"].removeprefix("refs/heads/")
            if name in self.branches:
                raise ValueError("branch race")
            self.branches[name] = document["sha"]
            return {}
        if path == "/pulls":
            return self.create_pull_object(document)
        raise AssertionError("unexpected mutation: " + path)

    def create_commit_object(self, document: dict[str, object]) -> dict[str, str]:
        """Retain exact parent, authorship, and manifest bytes for later reuse checks."""
        sha = "c" * 40
        self.commits[sha] = dict(
            document,
            parents=[{"sha": item} for item in document["parents"]],
            tree={"sha": document["tree"]},
        )
        self.pins[sha] = self.blobs[self.tree["tree"][0]["sha"]]
        if self.race_base:
            self.branches["main"] = "a" * 40
        if self.race_branch:
            self.branches["tui-release/" + publishing.releases.VERSION] = "a" * 40
        return {"sha": sha}

    def create_pull_object(self, document: dict[str, object]) -> dict[str, object]:
        """Simulate repository PR permission denial or a bot-created open proposal."""
        if self.deny_pull:
            raise PermissionError("repository denies PR creation")
        head, base = document["head"], document["base"]
        pull = dict(
            document,
            number=7,
            html_url="https://github.com/fixture/toolkit/pull/7",
            state="open",
            merged_at=None,
            user={"login": "github-actions[bot]"},
            head={
                "ref": head,
                "sha": self.branches[head],
                "repo": {"full_name": self.repository},
            },
            base={
                "ref": base,
                "sha": self.branches[base],
                "repo": {"full_name": self.repository},
            },
        )
        self.pulls_state.append(pull)
        return pull


class ProposalTests(unittest.TestCase):
    """Require real report contracts while replacing only the external GitHub boundary."""

    def setUp(self) -> None:
        """Create complete candidate/publication evidence and four synthetic native reports."""
        self.fixture = publishing.PublicationTests()
        self.fixture.setUp()
        self.addCleanup(self.fixture.doCleanups)
        self.record = self.fixture.record
        self.publication = self.fixture.publish()
        self.pin = (self.fixture.directory / "release.json").read_bytes()
        self.api = FakeGitHub(self.fixture.api)
        self.base = self.api.branches["main"]
        self.reports = [
            self.native_report(target)
            for target in proposal.publication.validator.TARGETS
        ]
        environment = patch.dict("os.environ", GITHUB_RUN_ATTEMPT="2")
        environment.start()
        self.addCleanup(environment.stop)

    def native_report(self, target: str) -> dict[str, object]:
        """Make explicit fixture evidence for testing the aggregate's validation logic."""
        url = f"https://github.com/{self.record['repository']}/releases/download/{self.record['version']}/agent-toolkit-{self.record['version']}-{target}.zip"
        checks = {name: {"status": "passed"} for name in proposal.pin_checks.CHECKS}
        for name, count in (("fresh", 1), ("cached", 0)):
            checks[name].update(downloads=count, url=url)
        for name in ("nonterminal", "arguments", "install_cli", "dry_run"):
            checks[name]["instrumented"] = name != "dry_run"
        return dict(
            proposal.pin_checks.evidence_identity(
                self.record, self.publication, self.pin, self.base, target
            ),
            status="passed",
            launcher_attempt="2",
            transport="published-https",
            download_repository=self.api.repository,
            checkouts={"source": copy.deepcopy(checks), "base": copy.deepcopy(checks)},
        )

    def writer(self) -> proposal.Proposer:
        """Use the actual writer with its fake GitHub collaborator."""
        return proposal.Proposer(
            self.api,
            self.record,
            self.publication,
            self.pin,
            self.base,
            "main",
            self.reports,
        )

    def test_new_proposal_changes_only_exact_manifest_and_reuses_open_pr(self) -> None:
        """GIVEN four matching reports WHEN proposed twice THEN reuse one manifest-only PR."""
        first = self.writer().run()
        writes = copy.deepcopy(self.api.calls)

        second = self.writer().run()

        self.assertEqual((first["status"], second["status"]), ("created", "reused"))
        self.assertEqual(first["number"], second["number"])
        self.assertEqual(self.api.calls, writes)
        self.assertEqual(self.api.pins[first["head_sha"]], self.pin)
        self.assertEqual(self.api.tree["base_tree"], "e" * 40)
        self.assertEqual(
            [item["path"] for item in self.api.tree["tree"]], [proposal.pin_checks.PIN]
        )
        body = self.api.pulls_state[0]["body"]
        self.assertIn("- [ ] Check resizing", body)
        self.assertNotIn("- [x]", body)
        self.assertTrue(all(call[0] == "POST" for call in writes))

    def test_merged_matching_pin_is_reported_without_mutation(self) -> None:
        """GIVEN a merged owned proposal WHEN pin still matches THEN report completion."""
        self.writer().run()
        self.api.pulls_state[0].update(state="closed", merged_at="2026-11-04T12:00:00Z")
        self.api.pins[self.base] = self.pin
        writes = copy.deepcopy(self.api.calls)

        result = self.writer().run()

        self.assertEqual(result["status"], "already_merged")
        self.assertEqual(self.api.calls, writes)
        self.api.pins[self.base] = b"changed"
        with self.assertRaisesRegex(ValueError, "no longer matches"):
            self.writer().run()

    def test_closed_human_edited_and_divergent_proposals_stop(self) -> None:
        """GIVEN a closed/conflicting PR WHEN retried THEN preserve every human change."""
        self.writer().run()
        original = copy.deepcopy(self.api.pulls_state[0])
        cases = (
            ("state", "closed"),
            ("user", {"login": "human"}),
            ("body", "human-edited ownership"),
        )
        writes = copy.deepcopy(self.api.calls)
        for field, value in cases:
            with self.subTest(field=field):
                self.api.pulls_state[0] = dict(original, **{field: value})

                with self.assertRaises(ValueError):
                    self.writer().run()

                self.assertEqual(self.api.calls, writes)
        self.api.pulls_state[0] = original
        self.api.branches[original["head"]["ref"]] = "a" * 40

        with self.assertRaisesRegex(ValueError, "human edits"):
            self.writer().run()

    def test_base_and_branch_races_never_overwrite(self) -> None:
        """GIVEN state changes during object creation WHEN rechecked THEN no branch is overwritten."""
        for race in ("race_base", "race_branch"):
            with self.subTest(race=race):
                self.api = FakeGitHub(self.fixture.api)
                setattr(self.api, race, True)

                with self.assertRaises(ValueError):
                    self.writer().run()

                self.assertFalse(
                    any(call[1] in ("/git/refs", "/pulls") for call in self.api.calls)
                )

    def test_denied_pr_permissions_leave_recoverable_owned_branch(self) -> None:
        """GIVEN denied PR permission WHEN restored THEN reuse the owned commit without rewriting it."""
        self.api.deny_pull = True

        with self.assertRaises(PermissionError):
            self.writer().run()
        original_commit_calls = [
            call for call in self.api.calls if call[1] == "/git/commits"
        ]
        self.api.deny_pull = False
        result = self.writer().run()

        self.assertEqual(result["status"], "created")
        self.assertEqual(
            [call for call in self.api.calls if call[1] == "/git/commits"],
            original_commit_calls,
        )
        self.assertEqual(self.fixture.api.state["draft"], False)

    def test_forged_missing_and_fixture_reports_prevent_all_writes(self) -> None:
        """GIVEN incomplete/substituted native evidence WHEN proposed THEN no GitHub writes occur."""
        valid = copy.deepcopy(self.reports)
        cases = (
            ("base_sha", "a" * 40),
            ("pin_sha256", "f" * 64),
            ("transport", "fixture"),
            ("launcher_attempt", "1"),
            ("publication_sha256", "f" * 64),
            ("status", "failed"),
        )
        for field, value in cases:
            with self.subTest(field=field):
                self.reports = copy.deepcopy(valid)
                self.reports[0][field] = value

                with self.assertRaises(ValueError):
                    self.writer().run()

                self.assertEqual(self.api.calls, [])
        self.reports = valid[:-1]
        with self.assertRaisesRegex(ValueError, "missing or duplicate"):
            self.writer().run()

    def test_failed_route_or_wrong_offline_count_prevents_proposal(self) -> None:
        """GIVEN failed/skipped route checks or an offline download WHEN aggregated THEN block writes."""
        original = copy.deepcopy(self.reports)
        for status in ("failed", "skipped", "unknown"):
            self.reports = copy.deepcopy(original)
            self.reports[0]["checkouts"]["base"]["writes"]["status"] = status

            with self.assertRaises(ValueError):
                self.writer().run()
        self.reports = original
        self.reports[0]["checkouts"]["source"]["cached"]["downloads"] = 1

        with self.assertRaises(ValueError):
            self.writer().run()
        self.assertEqual(self.api.calls, [])

    def test_unverified_release_and_tampered_branch_stop(self) -> None:
        """GIVEN a changed public release or pin commit WHEN retried THEN never repair it automatically."""
        self.fixture.api.state["draft"] = True

        with self.assertRaises(ValueError):
            self.writer().run()
        self.assertEqual(self.api.calls, [])
        self.fixture.api.state["draft"] = False
        self.writer().run()
        self.api.pins["c" * 40] = b"human pin"
        writes = copy.deepcopy(self.api.calls)

        with self.assertRaises(ValueError):
            self.writer().run()
        self.assertEqual(self.api.calls, writes)

    def test_failed_missing_or_skipped_workflow_jobs_block_aggregate(self) -> None:
        """GIVEN absent or unsuccessful prerequisites WHEN aggregated THEN no approval is possible."""
        for status in ("failure", "cancelled", "skipped"):
            jobs = {
                name: {"result": "success"}
                for name in ("pin-preflight", "launcher", "publication")
            }
            jobs["launcher"]["result"] = status

            with self.assertRaises(ValueError):
                gate.require_jobs(jobs)

        with self.assertRaises(ValueError):
            gate.require_jobs({})

    def test_changed_pin_file_mode_and_extra_diff_are_conflicts(self) -> None:
        """GIVEN a human mode change or added file WHEN reused THEN leave the branch intact."""
        self.writer().run()
        writes = copy.deepcopy(self.api.calls)
        self.api.pin_mode = "100755"

        with self.assertRaisesRegex(ValueError, "regular non-executable"):
            self.writer().run()
        self.api.pin_mode = "100644"
        difference = {
            "total_commits": 1,
            "files": [{"filename": "human.md"}, {"filename": proposal.pin_checks.PIN}],
        }
        with patch.object(self.api, "comparison", return_value=difference):
            with self.assertRaisesRegex(ValueError, "exactly one"):
                self.writer().run()

        self.assertEqual(self.api.calls, writes)


if __name__ == "__main__":
    unittest.main()
