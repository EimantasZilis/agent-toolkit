# Native package compatibility

Native packages have structural checks for both providers and recorded Codex
model-session acceptance. Claude runtime acceptance remains pending. This is
the latest recorded evidence, dated 22 October 2026; it is not a claim about
untested client versions or public marketplace approval.

| Area | Recorded result |
| --- | --- |
| Package structure, resources, helper closure, and installer coexistence | Repository tests and `make verify` passed for both providers. |
| Codex CLI | Version `0.163.0` accepted the generated marketplace and renamed packages in an isolated home. |
| Codex model sessions | Standalone commit-message, planning with plan-review, Rails with Ruby guidance, duplicate-helper resources, and explicit-only behavior passed the recorded requests. |
| Source independence | Relocation fixtures passed with source fixtures unavailable. Model sessions selected package paths while the source clone remained on disk. |
| Claude | Manifests and resources passed structural checks. The client was unavailable; provider validation, installation, and model sessions remain unverified. |
| Public distribution | No public-directory submission or approval was attempted in these checks. |

The Codex sessions selected installed package paths explicitly. They do not
establish every picker interaction or all overlapping-helper behavior.

See the [archived test record](archive/native-packages-validation-2026-10-22.md)
for commands, fixtures, limitations, and earlier results. For the checks to run
before a release, see [release checks](native-publishing.md#release-checks).
