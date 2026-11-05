#!/usr/bin/env python3
"""Propose a verified manifest-only activation commit without executing source code."""

from __future__ import annotations

import argparse
import base64
import hashlib
import json
import os
import re
import sys
import urllib.parse
from pathlib import Path
from typing import cast

import check_tui_release_pin as pin_checks
import publish_tui_release as publication
import tui_candidate as candidate

BOT = {
    "name": "agent-toolkit release automation",
    "email": "41898282+github-actions[bot]@users.noreply.github.com",
}
MARKER = "<!-- agent-toolkit-pin:"


def require_sha(value: str) -> str:
    """Reject unsafe or abbreviated Git object identifiers."""
    if not re.fullmatch(r"[0-9a-f]{40}", value):
        raise ValueError("invalid proposal SHA")
    return value


def ownership(record: dict[str, object], pin: bytes, base_sha: str) -> str:
    """Bind an automation branch to its original base, candidate, and exact pin bytes."""
    identity = {
        field: record[field] for field in (*candidate.IDENTITY, "repository", "hashes")
    }
    identity.update(
        base_sha=require_sha(base_sha), pin_sha256=hashlib.sha256(pin).hexdigest()
    )
    return MARKER + json.dumps(identity, sort_keys=True, separators=(",", ":")) + " -->"


def original_base(body: str, record: dict[str, object], pin: bytes) -> str:
    """Read and verify the complete ownership marker before considering reuse."""
    line = body.splitlines()[0] if body else ""
    if not line.startswith(MARKER) or not line.endswith(" -->"):
        raise ValueError("proposal lacks automation ownership")
    identity = publication.validator.parse_json(line[len(MARKER) : -4].encode())
    base_sha = require_sha(str(identity.get("base_sha", "")))
    if line != ownership(record, pin, base_sha):
        raise ValueError("proposal belongs to a different candidate or pin")
    return base_sha


def check_launcher_reports(
    record: dict[str, object],
    report: dict[str, object],
    pin: bytes,
    base_sha: str,
    reports: list[dict[str, object]],
    attempt: str,
) -> None:
    """Require four real HTTPS native reports bound to this publication and base."""
    targets = [item.get("target") for item in reports]
    if len(targets) != 4 or set(targets) != set(publication.validator.TARGETS):
        raise ValueError("missing or duplicate launcher targets")
    for item in reports:
        expected = pin_checks.evidence_identity(
            record, report, pin, base_sha, item["target"]
        )
        if any(item.get(key) != value for key, value in expected.items()):
            raise ValueError("launcher candidate, publication, pin, or base mismatch")
        if (
            item.get("status") != "passed"
            or "error" in item
            or item.get("launcher_attempt") != attempt
            or item.get("transport") != "published-https"
            or item.get("download_repository") != record["repository"]
        ):
            raise ValueError(
                "launcher report is failed, stale, or uses fixture transport"
            )
        check_routes(record, item)


def check_routes(record: dict[str, object], report: dict[str, object]) -> None:
    """Reject missing checks, concealed instrumentation, and incorrect download counts."""
    checkouts = report.get("checkouts")
    if not isinstance(checkouts, dict) or set(checkouts) != {"source", "base"}:
        raise ValueError("missing source or proposal-base launcher evidence")
    target = report["target"]
    archive = f"agent-toolkit-{record['version']}-{target}.zip"
    url = f"https://github.com/{record['repository']}/releases/download/{record['version']}/{archive}"
    for checks in checkouts.values():
        if (
            not isinstance(checks, dict)
            or set(checks) != pin_checks.CHECKS
            or any(
                not isinstance(item, dict) or item.get("status") != "passed"
                for item in checks.values()
            )
        ):
            raise ValueError("missing or failed required launcher/routing checks")
        for name, count in (("fresh", 1), ("cached", 0)):
            if (
                type(checks[name].get("downloads")) is not int
                or checks[name].get("downloads") != count
                or checks[name].get("url") != url
            ):
                raise ValueError("fresh/cache launcher download evidence mismatch")
        for name in ("nonterminal", "arguments", "install_cli", "dry_run"):
            if checks[name].get("instrumented") is not (name != "dry_run"):
                raise ValueError("routing instrumentation evidence mismatch")


class GitHub(publication.GitHub):
    """Expose only reference creation, immutable Git objects, and PR creation."""

    def branch(self, name: str) -> str | None:
        """Read a branch reference without creating or updating it."""
        item = self.optional("/git/ref/heads/" + urllib.parse.quote(name, safe=""))
        if item is None:
            return None
        if item["object"]["type"] != "commit":
            raise ValueError("branch does not point to a commit")
        return require_sha(item["object"]["sha"])

    def default_base(self, branch: str) -> str:
        """Require the named base to remain the repository's current default branch."""
        repository = self.api("GET", "")
        if repository["default_branch"] != branch:
            raise ValueError("proposal default branch changed")
        return require_sha(self.branch(branch) or "")

    def pulls(self, head: str, base: str) -> list[dict[str, object]]:
        """Find open and closed same-repository proposals with bounded pagination."""
        result = []
        for page in range(1, 101):
            query = urllib.parse.urlencode(
                {
                    "state": "all",
                    "head": self.repository.split("/")[0] + ":" + head,
                    "base": base,
                    "per_page": 100,
                    "page": page,
                }
            )
            batch = cast(list[dict[str, object]], self.api("GET", "/pulls?" + query))
            result.extend(batch)
            if len(batch) < 100:
                return result
        raise ValueError("proposal inventory exceeds pagination limit")

    def pin_bytes(self, sha: str) -> bytes | None:
        """Read exact pin bytes at an immutable commit, rejecting non-file content."""
        item = self.optional("/contents/" + pin_checks.PIN + "?ref=" + require_sha(sha))
        if item is None:
            return None
        if item.get("type") != "file" or item.get("encoding") != "base64":
            raise ValueError("proposal pin is not a regular base64 file")
        return base64.b64decode(item["content"].replace("\n", ""), validate=True)

    def create_commit(self, base: str, pin: bytes, message: str) -> str:
        """Construct a single-file tree from the checked base without checking it out."""
        parent = self.api("GET", "/git/commits/" + require_sha(base))
        blob = self.api(
            "POST",
            "/git/blobs",
            {"encoding": "base64", "content": base64.b64encode(pin).decode()},
        )
        tree = self.api(
            "POST",
            "/git/trees",
            {
                "base_tree": parent["tree"]["sha"],
                "tree": [
                    {
                        "path": pin_checks.PIN,
                        "mode": "100644",
                        "type": "blob",
                        "sha": blob["sha"],
                    }
                ],
            },
        )
        commit = self.api(
            "POST",
            "/git/commits",
            {
                "message": message,
                "tree": tree["sha"],
                "parents": [base],
                "author": BOT,
                "committer": BOT,
            },
        )
        return require_sha(commit["sha"])

    def create_branch(self, name: str, sha: str) -> None:
        """Create a missing ref atomically; never update or force an existing branch."""
        self.api(
            "POST", "/git/refs", {"ref": "refs/heads/" + name, "sha": require_sha(sha)}
        )

    def create_pull(
        self, head: str, base: str, title: str, body: str
    ) -> dict[str, object]:
        """Open a review request without approval, merge, or edits to existing PRs."""
        return cast(
            dict[str, object],
            self.api(
                "POST",
                "/pulls",
                {
                    "head": head,
                    "base": base,
                    "title": title,
                    "body": body,
                    "maintainer_can_modify": False,
                },
            ),
        )


def commit_message(record: dict[str, object], pin: bytes, base: str) -> str:
    """Use stable ownership evidence for recovery after a denied PR request."""
    return f"Propose TUI {record['version']}\n\n" + ownership(record, pin, base)


def require_commit(api: GitHub, sha: str, base: str, pin: bytes, message: str) -> None:
    """Refuse human edits, additional commits/files, or a different manifest."""
    commit = api.api("GET", "/git/commits/" + require_sha(sha))
    if [parent["sha"] for parent in commit["parents"]] != [base] or commit[
        "message"
    ] != message:
        raise ValueError("proposal branch history or ownership changed")
    for field in ("author", "committer"):
        if any(commit[field].get(key) != value for key, value in BOT.items()):
            raise ValueError("proposal commit is not automation-owned")
    require_pin_mode(api, commit["tree"]["sha"])
    comparison = api.api("GET", f"/compare/{require_sha(base)}...{sha}")
    files = comparison.get("files", [])
    if comparison.get("total_commits") != 1 or len(files) != 1:
        raise ValueError("proposal must contain exactly one commit and file")
    expected_blob = hashlib.sha1(
        b"blob " + str(len(pin)).encode() + b"\0" + pin
    ).hexdigest()
    if (
        files[0].get("filename") != pin_checks.PIN
        or files[0].get("status") not in ("added", "modified")
        or files[0].get("sha") != expected_blob
        or api.pin_bytes(sha) != pin
    ):
        raise ValueError("proposal manifest-only diff or exact pin bytes mismatch")


def require_pin_mode(api: GitHub, tree_sha: str) -> None:
    """Reject executable pins or symlinks when reusing an automation branch."""
    root = api.api("GET", "/git/trees/" + require_sha(tree_sha))
    scripts = [entry for entry in root["tree"] if entry["path"] == "scripts"]
    if root.get("truncated") or len(scripts) != 1 or scripts[0].get("type") != "tree":
        raise ValueError("proposal scripts tree is missing or truncated")
    tree = api.api("GET", "/git/trees/" + require_sha(scripts[0]["sha"]))
    pins = [entry for entry in tree["tree"] if entry["path"] == "tui-release.json"]
    if (
        tree.get("truncated")
        or len(pins) != 1
        or pins[0].get("type") != "blob"
        or pins[0].get("mode") != "100644"
    ):
        raise ValueError("proposal pin must be a regular non-executable file")


def proposal_context(
    record: dict[str, object],
    report: dict[str, object],
    pin: bytes,
    base: str,
) -> list[str]:
    """Introduce the release and immutable source before its evidence tables."""
    repository = record["repository"]
    run = f"https://github.com/{repository}/actions/runs/{record['run_id']}"
    return [
        ownership(record, pin, base),
        "",
        f"Propose TUI {record['version']} for manual review.",
        "",
        f"Release: {report['release_url']}",
        f"Source: `{record['source_sha']}`",
        f"Checked proposal base: `{base}`",
        f"Candidate run: {run}",
        f"Payload: {run}/artifacts/{record['payload_id']}",
        "",
        "Retained SHA-256 hashes:",
        "",
        "| File | SHA-256 |",
        "| --- | --- |",
    ]


def proposal_body(
    record: dict[str, object],
    report: dict[str, object],
    pin: bytes,
    base: str,
    reports: list[dict[str, object]],
    artifacts: dict[str, str],
) -> str:
    """Keep manual UI review unchecked and link retained build/publication evidence."""
    run = f"https://github.com/{record['repository']}/actions/runs/{record['run_id']}"
    lines = proposal_context(record, report, pin, base)
    lines.extend(
        f"| {name} | `{digest}` |" for name, digest in record["hashes"].items()
    )
    lines += [
        "",
        "| Native target | Fresh download | Offline cache | Bash routes |",
        "| --- | --- | --- | --- |",
    ]
    lines.extend(f"| {item['target']} | passed | passed | passed |" for item in reports)
    lines += ["", "Retained acceptance artifacts:", ""]
    lines.extend(
        f"- [{name}]({run}/artifacts/{identity})"
        for name, identity in artifacts.items()
    )
    lines += manual_review()
    return "\n".join(lines) + "\n"


def manual_review() -> list[str]:
    """Keep visual review and activation under maintainer control."""
    return [
        "",
        "Both release source and checked base passed. Non-terminal, argument, and install-cli",
        "routes use instrumentation; a separate real installer dry run passed. No installation was applied.",
        "",
        "Bot-created PRs may need a maintainer to approve or start required workflows.",
        "Creating this PR does not establish that its required checks have passed.",
        "",
        "Manual review (leave pending until a maintainer completes it):",
        "",
        "- [ ] Review navigation and selection in the UI.",
        "- [ ] Check resizing, layout, and readability.",
        "- [ ] Check quit and Discard behavior.",
        "- [ ] Confirm that no installation was applied.",
        "- [ ] Confirm the current base still has compatible launcher/backend checks.",
        "- [ ] Review required checks, approve, and merge manually.",
    ]


class Proposer:
    """Create or reuse one owned proposal while preserving all conflicting state."""

    def __init__(
        self,
        api: GitHub,
        record: dict[str, object],
        report: dict[str, object],
        pin: bytes,
        base: str,
        branch: str,
        reports: list[dict[str, object]],
        artifacts: dict[str, str] | None = None,
    ) -> None:
        """Keep publication and native acceptance evidence separate from API credentials."""
        self.api, self.record, self.report = api, record, report
        self.pin, self.base, self.branch, self.reports = (
            pin,
            require_sha(base),
            branch,
            reports,
        )
        self.head = "tui-release/" + record["version"]
        if ".." in self.head or self.head.endswith((".", ".lock")):
            raise ValueError("release version cannot form a safe proposal branch name")
        self.artifacts = artifacts or {}

    def require_base(self) -> None:
        """Stop if the default branch moved after its native compatibility checks."""
        if self.api.default_base(self.branch) != self.base:
            raise ValueError(
                "proposal base changed; refresh native acceptance evidence"
            )

    def require_publication(self) -> None:
        """Recheck tag, release identity, ownership, and public state before proposal writes."""
        publication.require_tag(self.api, self.record)
        release = self.api.release(self.record["version"])
        publication.require_owned(release, self.record)
        if (
            release["id"] != self.report["release_id"]
            or release.get("draft") is not False
            or release.get("body") != publication.release_body(self.record, True)
        ):
            raise ValueError("publication is no longer the verified public release")
        publication.inventory(self.api, release["id"], self.record)

    def require_pull(self, pull: dict[str, object]) -> str:
        """Verify ownership and same-repository head/base before inspecting branch content."""
        if pull.get("user", {}).get("login") != "github-actions[bot]":
            raise ValueError("existing proposal is not automation-owned")
        for key, ref in (("head", self.head), ("base", self.branch)):
            if (
                pull[key]["ref"] != ref
                or pull[key]["repo"]["full_name"] != self.api.repository
            ):
                raise ValueError("proposal head/base repository mismatch")
        return original_base(pull.get("body", ""), self.record, self.pin)

    def reuse(self, pull: dict[str, object]) -> dict[str, object]:
        """Return open/merged proposals only after verifying immutable history and bytes."""
        base = self.require_pull(pull)
        sha = require_sha(pull["head"]["sha"])
        current = self.api.branch(self.head)
        if current is not None and current != sha:
            raise ValueError("proposal branch has human edits")
        require_commit(
            self.api, sha, base, self.pin, commit_message(self.record, self.pin, base)
        )
        self.require_base()
        if pull.get("merged_at"):
            if self.api.pin_bytes(self.base) != self.pin:
                raise ValueError(
                    "merged proposal no longer matches the default-branch pin"
                )
            return self.result(pull, "already_merged")
        if pull.get("state") != "open":
            raise ValueError(
                "proposal is closed unmerged; maintainer resolution required"
            )
        if current != sha or base != self.base or pull["base"]["sha"] != self.base:
            raise ValueError("open proposal branch or base changed")
        return self.result(pull, "reused")

    def result(self, pull: dict[str, object], status: str) -> dict[str, object]:
        """Report proposal identity without marking manual checks complete."""
        return {
            "status": status,
            "number": pull["number"],
            "url": pull["html_url"],
            "base_sha": self.base,
            "head_sha": pull["head"]["sha"],
            "manual_ui_review": "pending",
        }

    def run(self) -> dict[str, object]:
        """Create only missing automation state; never update, approve, or merge it."""
        check_launcher_reports(
            self.record,
            self.report,
            self.pin,
            self.base,
            self.reports,
            os.environ["GITHUB_RUN_ATTEMPT"],
        )
        self.require_base()
        self.require_publication()
        pulls = self.api.pulls(self.head, self.branch)
        if len(pulls) > 1:
            raise ValueError("ambiguous existing proposals")
        if pulls:
            return self.reuse(pulls[0])
        sha = self.prepare_branch()
        self.require_base()
        if self.api.branch(self.head) != sha or self.api.pulls(self.head, self.branch):
            raise ValueError("proposal branch or PR changed before creation")
        pull = self.api.create_pull(
            self.head,
            self.branch,
            f"Propose TUI {self.record['version']}",
            proposal_body(
                self.record,
                self.report,
                self.pin,
                self.base,
                self.reports,
                self.artifacts,
            ),
        )
        self.require_base()
        self.reuse(pull)
        return self.result(pull, "created")

    def prepare_branch(self) -> str:
        """Reuse a verified orphan branch after API failure or atomically create a new ref."""
        message = commit_message(self.record, self.pin, self.base)
        sha = self.api.branch(self.head)
        if sha is None:
            if self.api.pin_bytes(self.base) == self.pin:
                raise ValueError(
                    "base already has pin without matching merged proposal"
                )
            self.require_base()
            sha = self.api.create_commit(self.base, self.pin, message)
            require_commit(self.api, sha, self.base, self.pin, message)
            self.require_base()
            if self.api.branch(self.head) is not None:
                raise ValueError("proposal branch appeared during preparation")
            self.api.create_branch(self.head, sha)
        require_commit(self.api, sha, self.base, self.pin, message)
        return sha


def main() -> int:
    """Validate retained evidence before allowing any proposal mutation."""
    parser = argparse.ArgumentParser(description=__doc__)
    for name in (
        "candidate-directory",
        "candidate-record",
        "publication-report",
        "launcher-result",
        "report",
    ):
        parser.add_argument("--" + name, type=Path, required=True)
    parser.add_argument("--default-branch", required=True)
    args = parser.parse_args()
    result = {"status": "failed"}
    try:
        result = propose(args)
    except (OSError, ValueError, KeyError, TypeError) as error:
        result["error"] = str(error)
    candidate.write_json(args.report, result)
    print(json.dumps(result))
    return int(result["status"] == "failed")


def propose(args: argparse.Namespace) -> dict[str, object]:
    """Require complete native evidence and create a credentials-isolated API boundary."""
    record = candidate.load_json(args.candidate_record)
    report = candidate.load_json(args.publication_report)
    pin = pin_checks.check_publication(record, report, args.candidate_directory)
    acceptance = candidate.load_json(args.launcher_result)
    if acceptance.get("status") != "approved":
        raise ValueError("launcher aggregate is not approved")
    check_launcher_reports(
        record,
        report,
        pin,
        acceptance["base_sha"],
        acceptance["reports"],
        os.environ["GITHUB_RUN_ATTEMPT"],
    )
    api = GitHub(record["repository"], os.environ["GH_TOKEN"])
    return Proposer(
        api,
        record,
        report,
        pin,
        acceptance["base_sha"],
        args.default_branch,
        acceptance["reports"],
    ).run()


if __name__ == "__main__":
    sys.exit(main())
