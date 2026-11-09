# review-branch: find issues before merging

[All skill examples](../../EXAMPLES.md)

[review-branch](../../.agents/skills/review/review-branch/SKILL.md) traces committed
changes through the surrounding code to find defects and gaps in requirements
before merge. It produces a prioritized list of actionable findings, each backed
by a triggering scenario, code evidence, impact, and a recommended fix.

Run it at the end of implementation to check your own work. It is also useful
for reviewing any other branch, including a teammate's pull request.

## Why use it

Catch defects that depend on code outside the diff before merging. Findings
include the trigger, evidence, impact, and suggested fix, so you can assess and
act on them.

## How it works

The review traces changes through related code to find issues beyond the diff.
Providing a ticket helps it check the changes against their purpose and
acceptance criteria. It also uses available PR discussions and plans, rechecking
existing concerns against the current code.

It works through four passes:

1. **Scope and contracts.** Connect changed files to callers, stored data,
   configuration, deployment, and tests. For an invoice export, that means following
   the download route into the invoice lookup, export worker, and schema change,
   then comparing them with the ownership, retry, and compatibility requirements.
2. **Behavior and edge cases.** Build a scenario matrix for each changed behavior.
   What happens with another customer's invoice ID, a repeated job, a failed
   upload, or an older invoice? Mark each relevant scenario as covered,
   unverified, contradicted, or not applicable.
3. **Data flow and invariants.** Trace inputs through side effects to outputs.
   Check whether authorization happens before PDF generation, whether a retry
   repeats a database insert, and whether older records can reach the new path.
   Follow transaction boundaries, exception handling, and cleanup where failures
   could leave inconsistent state.
4. **Evidence.** Inspect tests, fixtures, CI configuration, and relevant plan exit
   checks. Match that evidence to the scenarios: a successful owner download
   does not establish that another customer's request is rejected.

The review goes deeper on high-risk paths such as authorization, migrations,
concurrency, billing, and deployment changes. It inspects code and test coverage
statically; test execution is a separate request.

## Example prompt

> Use review-branch to review my teammate's committed branch against main for
> their pull request. Use the linked ticket and PR discussion as requirements
> context. 

## Example result

The findings table distinguishes confirmed defects from coverage gaps or
unresolved requirements. Each row explains what triggers the issue and what to
change. For the invoice branch, it could contain:

| Severity | Finding | Location | Why it matters | Recommended action | Issue summary |
| --- | --- | --- | --- | --- | --- |
| Critical — fix before merge | Another customer can download an invoice | `app/invoices/download.py:24` | The changed lookup filters only by invoice ID. A signed-in customer supplying another customer's ID reaches PDF generation without an ownership check. | Use the existing owner-scoped lookup before generating the PDF. | Another customer's invoice ID reaches PDF generation because this lookup does not check ownership. Use the existing owner-scoped lookup so the request is rejected before the PDF is generated. |
| Medium — fix before merge | Retrying an export creates a duplicate | `app/invoices/export_worker.py:51` | The worker inserts an export before uploading it. If the upload fails, retrying the same job inserts a second export because the insert has no job-identity guard. | Reuse the export for the same job, or enforce a unique job key and handle retries. | An upload failure followed by a retry creates a second export for the same job. Reuse the existing export, or enforce a unique job key and handle the existing record on retry. |
| Medium — fix before merge | Legacy-invoice compatibility lacks coverage | `tests/invoices/test_download.py:18` | All download fixtures populate the new storage key. They leave the requirement to download invoices created before that key existed unverified. | Add a legacy-invoice fixture and verify the download path against the compatibility requirement. | The download tests only cover invoices with the new storage key. Add a fixture for an invoice created before this change and check that it remains downloadable. |

Findings are ordered by severity and deduplicated by root cause. The final column
provides a comment ready to paste into the PR. The review reports actionable
concerns only; when none are found, it returns `No issues found.`
