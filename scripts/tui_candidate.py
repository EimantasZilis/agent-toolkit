#!/usr/bin/env python3
"""Bind CI release payloads and native reports to one immutable candidate."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
import urllib.request
from pathlib import Path

import smoke_tui_release as runtime
import validate_tui_release as validator

IDENTITY = (
    "version",
    "source_sha",
    "workflow_sha",
    "run_id",
    "run_attempt",
    "payload_id",
)
CHECKS = {"host", "integrity", "version", "pty", "writes"}


def git(root: Path, *arguments: str) -> str:
    """Run Git without treating dispatch inputs as shell expressions."""
    return subprocess.check_output(
        ["git", "-C", str(root), *arguments], text=True
    ).strip()


def preflight(root: Path, version: str, source: str, workflow: str, branch: str) -> str:
    """Validate inputs, trusted ancestry, and the existing source backend contract."""
    if not re.fullmatch(validator.VERSION, version):
        raise ValueError("invalid release version")
    if not re.fullmatch(r"[0-9a-f]{40}", workflow):
        raise ValueError("invalid workflow SHA")
    if not source or source.startswith("-") or any(char.isspace() for char in source):
        raise ValueError("invalid source ref")
    trusted = f"refs/remotes/origin/{branch}"
    sha = git(root, "rev-parse", "--verify", "--end-of-options", source + "^{commit}")
    for commit in (workflow, sha):
        git(root, "merge-base", "--is-ancestor", commit, trusted)
    for path in (
        "go.mod",
        "scripts/build_tui.py",
        "scripts/managed.sh",
        "cmd/agent-toolkit/main.go",
        ".agents/skills",
    ):
        git(root, "cat-file", "-e", f"{sha}:{path}")
    return sha


def payload_hashes(directory: Path, version: str) -> dict[str, str]:
    """Hash exactly the six validated release files, without adding evidence files."""
    validator.validate_release(version, directory)
    names = ["release.json", "SHA256SUMS"] + [
        f"agent-toolkit-{version}-{target}.zip" for target in validator.TARGETS
    ]
    if {path.name for path in directory.iterdir()} != set(names):
        raise ValueError("payload must contain exactly six release files")
    return {
        name: hashlib.sha256(validator.read_file(directory / name)).hexdigest()
        for name in names
    }


def candidate(directory: Path) -> dict[str, object]:
    """Create a record only after upload has returned the immutable payload ID."""
    record = {field: os.environ[field.upper()] for field in IDENTITY}
    validate_identity(record)
    record["hashes"] = payload_hashes(directory, record["version"])
    record["retention_days"] = 30
    return record


def validate_identity(record: dict[str, object]) -> None:
    """Require safe, complete source, workflow, run, and artifact identifiers."""
    if not re.fullmatch(validator.VERSION, str(record.get("version", ""))):
        raise ValueError("invalid version identity")
    for field in ("source_sha", "workflow_sha"):
        if not re.fullmatch(r"[0-9a-f]{40}", str(record.get(field, ""))):
            raise ValueError(f"invalid {field}")
    for field in ("run_id", "run_attempt", "payload_id"):
        if not re.fullmatch(r"[1-9][0-9]*", str(record.get(field, ""))):
            raise ValueError(f"invalid {field}")


def check_payload(record: dict[str, object], directory: Path) -> None:
    """Require downloaded bytes to match the original candidate's six digests."""
    validate_identity(record)
    if payload_hashes(directory, str(record["version"])) != record.get("hashes"):
        raise ValueError("candidate payload hashes differ")


def require_context(record: dict[str, object]) -> None:
    """Bind evidence to trusted job outputs, including the original build attempt."""
    if any(record.get(field) != os.environ[field.upper()] for field in IDENTITY):
        raise ValueError("candidate identity differs from trusted workflow outputs")


def run_native(
    record: dict[str, object], directory: Path, checkout: Path, target: str
) -> dict[str, object]:
    """Verify candidate identity before executing the retained native binary."""
    require_context(record)
    check_payload(record, directory)
    report = runtime.smoke(str(record["version"]), target, directory, checkout)
    report.update({field: record[field] for field in IDENTITY})
    report["verification_attempt"] = os.environ["GITHUB_RUN_ATTEMPT"]
    return report


def check_report(
    record: dict[str, object], report: dict[str, object], attempt: str
) -> None:
    """Reject substituted identities, unknown checks/statuses, and required skips."""
    if any(report.get(field) != record[field] for field in IDENTITY):
        raise ValueError("runtime report candidate identity mismatch")
    if report.get("verification_attempt") != attempt or "error" in report:
        raise ValueError(
            "runtime report failed or belongs to another verification attempt"
        )
    checks = report.get("checks")
    if not isinstance(checks, dict) or set(checks) != CHECKS:
        raise ValueError("runtime report check set mismatch")
    for name, check in checks.items():
        check_status(name, check)


def check_status(name: str, check: object) -> None:
    """Permit only a documented unsupported-capability skip for the PTY check."""
    if (
        not isinstance(check, dict)
        or not isinstance(check.get("reason"), str)
        or not check["reason"].strip()
    ):
        raise ValueError(f"{name}: missing check reason")
    status = check.get("status")
    if status == "passed":
        return
    if (
        name == "pty"
        and status == "skipped"
        and check.get("unsupported_capability") is True
    ):
        if re.search(r"timeout|startup|missing runner", check["reason"], re.I):
            raise ValueError("runtime failure cannot be an unsupported PTY capability")
        return
    raise ValueError(f"{name}: required check did not pass ({status})")


def aggregate(
    record: dict[str, object],
    reports: list[dict[str, object]],
    jobs: dict[str, str],
    attempt: str,
) -> dict[str, object]:
    """Approve only complete native evidence and successful required jobs."""
    validate_identity(record)
    if set(jobs) != {"preflight", "build", "runtime"} or any(
        value != "success" for value in jobs.values()
    ):
        raise ValueError(f"required job failed, cancelled, or missing: {jobs}")
    targets = [report.get("target") for report in reports]
    if len(targets) != 4 or set(targets) != set(validator.TARGETS):
        raise ValueError("missing or duplicate native target reports")
    for report in reports:
        check_report(record, report, attempt)
    return dict(
        record,
        status="approved",
        verification_attempt=attempt,
        jobs=jobs,
        reports=reports,
        manual_ui_review="pending",
        skipped_pty_targets=[
            report["target"]
            for report in reports
            if report["checks"]["pty"]["status"] == "skipped"
        ],
    )


def report_artifacts(repository: str, run_id: str, attempt: str) -> dict[str, str]:
    """Resolve exact attempt-specific report names to IDs within this workflow run."""
    url = f"https://api.github.com/repos/{repository}/actions/runs/{run_id}/artifacts?per_page=100"
    artifacts = []
    for page in range(1, 101):
        request = urllib.request.Request(
            url + f"&page={page}",
            headers={
                "Authorization": "Bearer " + os.environ["GH_TOKEN"],
                "Accept": "application/vnd.github+json",
            },
        )
        with urllib.request.urlopen(request, timeout=30) as response:
            batch = json.load(response)["artifacts"]
        artifacts.extend(batch)
        if len(batch) < 100:
            break
    return require_artifacts(artifacts, attempt)


def require_artifacts(
    artifacts: list[dict[str, object]], attempt: str
) -> dict[str, str]:
    """Reject missing, ambiguous, or expired reports instead of selecting latest."""
    result = {}
    for target in validator.TARGETS:
        name = f"tui-report-{target}-{attempt}"
        matches = [item for item in artifacts if item["name"] == name]
        if len(matches) != 1 or matches[0]["expired"]:
            raise ValueError(f"missing, duplicate, or expired report artifact: {name}")
        result[target] = str(matches[0]["id"])
    return result


def write_json(path: Path, document: object) -> None:
    """Write stable human-readable evidence to a separate artifact file."""
    path.write_text(json.dumps(document, indent=2, sort_keys=True) + "\n")


def load_json(path: Path) -> dict[str, object]:
    """Read bounded evidence with duplicate-field rejection."""
    return validator.parse_json(validator.read_file(path))


def execute(command: str) -> None:
    """Dispatch trusted workflow operations using environment-bound inputs."""
    if command == "preflight":
        sha = preflight(
            Path("tooling"),
            os.environ["VERSION"],
            os.environ["SOURCE_REF"],
            os.environ["WORKFLOW_SHA"],
            os.environ["DEFAULT_BRANCH"],
        )
        with Path(os.environ["GITHUB_OUTPUT"]).open("a") as output:
            output.write(f"source_sha={sha}\n")
    elif command == "record":
        write_json(Path("candidate.json"), candidate(Path("payload")))
    elif command == "runtime":
        report = run_native(
            load_json(Path("candidate.json")),
            Path("payload"),
            Path("source"),
            os.environ["TARGET"],
        )
        write_json(Path(f"report-{os.environ['TARGET']}.json"), report)
        if "error" in report:
            raise ValueError(str(report["error"]))
    elif command == "reports":
        ids = report_artifacts(
            os.environ["GITHUB_REPOSITORY"],
            os.environ["GITHUB_RUN_ID"],
            os.environ["GITHUB_RUN_ATTEMPT"],
        )
        write_json(Path("report-ids.json"), ids)
        with Path(os.environ["GITHUB_OUTPUT"]).open("a") as output:
            output.write("ids=" + ",".join(ids.values()) + "\n")
    else:
        finish()


def finish() -> None:
    """Persist failed diagnostics or an approved candidate and job summary."""
    needs = json.loads(os.environ["NEEDS"])
    jobs = {name: item["result"] for name, item in needs.items()}
    result: dict[str, object] = {"status": "failed", "jobs": jobs}
    try:
        record = load_json(Path("candidate.json"))
        require_context(record)
        check_payload(record, Path("payload"))
        reports = [load_json(path) for path in Path("reports").glob("report-*.json")]
        result = aggregate(record, reports, jobs, os.environ["GITHUB_RUN_ATTEMPT"])
        result["report_artifact_ids"] = load_json(Path("report-ids.json"))
        result["candidate_record_id"] = os.environ["RECORD_ID"]
    except (OSError, ValueError) as error:
        result["status"] = "failed"
        result["error"] = str(error)
    write_json(Path("candidate-result.json"), result)
    with Path(os.environ["GITHUB_STEP_SUMMARY"]).open("a") as summary:
        summary.write(
            "## TUI candidate\n\n```json\n"
            + json.dumps(result, indent=2)
            + "\n```\n\nRetention: 30 days. Manual UI review: pending.\n"
        )
    if result["status"] != "approved":
        raise ValueError(str(result.get("error", "candidate failed")))


def main() -> int:
    """Fail closed with useful diagnostics at the workflow process boundary."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "command", choices=("preflight", "record", "runtime", "reports", "aggregate")
    )
    args = parser.parse_args()
    try:
        execute(args.command)
    except (OSError, ValueError, KeyError, subprocess.SubprocessError) as error:
        print(f"TUI candidate failed: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
