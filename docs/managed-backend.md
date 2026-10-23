# Managed installation protocol

`scripts/managed.sh` exposes protocol version 1 for the TUI and other local
clients. Bash owns selection, dependency closure, ownership checks, revisions,
and filesystem changes. It requires the existing shell utilities, with no runtime
JSON parser. Clients pass argument arrays and parse one JSON document on stdout;
diagnostics go to stderr.

## Commands

| Command | Result |
| --- | --- |
| `catalog --protocol 1` | Skill names, categories, source keys, required helpers, invocation policies, and bundles |
| `status --protocol 1` | Shared desired settings, saved applied inventories, current observed skills, and state revision |
| `preview --protocol 1 [selection options]` | Resolved selection roles, operations, conflicts, removal confirmation, and state revision |
| `apply --protocol 1 [same selection options] --revision REVISION` | Actual applied inventory and the new revision after successful application |

The protocol defaults to 1 for CLI use. Clients should request it explicitly.
Catalog, status, and preview do not write managed state or provider files.
Temporary rendering files are removed after use. Catalog remains available
without reading saved settings.

## Selection and settings

| Option | Meaning |
| --- | --- |
| `--provider codex` or `--provider claude` | Replace the saved provider selection; repeat to select both |
| `--no-providers` | Select no providers and propose removing owned installations |
| `--skill NAME` | Replace requested skills; repeat for multiple skills |
| `--no-skills` | Clear requested skills; required and retained helpers still apply |
| `--remove-skill NAME` | Explicitly remove a skill or unused helper; reject removal while it is required |
| `--output-path PATH` | Save an absolute Markdown output root; an empty argument selects repo-local output |
| `--create-output-directory` | After explicit user confirmation, create a missing output root during apply; preview only checks its nearest existing parent |
| `--concise BOOL` | Set `true` or `false` to control concise-output loading independently of the full template |
| `--full-template BOOL` | Set `true` or `false` to control the full global guidance template |
| `--confirm-removals` | Authorize the removals shown in the preview |
| `--native-inventory FILE` | Read a supported provider-exported inventory for overlap notices |

Unspecified settings retain saved values. Fresh managed setup has no provider
selected. Legacy `.env` choices and differing owned installations are imported
without removal. Removing a parent retains installed helpers; disabling concise
loading retains `concise-output`. Use explicit removal to discard an unused skill.

`AGENTS_HOME`, `CODEX_HOME`, and `CLAUDE_HOME` retain their destination overrides.
`AGENT_TOOLKIT_ENV_FILE` selects the legacy import file. `MARKDOWN_OUTPUT_PATH`
overrides rendering for this invocation without changing saved preferences;
an explicit `--output-path` takes precedence. Keep the same arguments and
environment between preview and apply.

## Responses and revisions

Successful responses contain `protocol_version`, `ok: true`, and `kind`.
Status distinguishes `desired`, `applied`, and `observed`: saved preferences,
the last successful inventory, and currently observed toolkit-owned skill names.
Observed names alone do not prove that file contents are unchanged; preview
reports ownership and content conflicts.

Preview adds `selection`, `operations`, `conflicts`, and
`confirmation_required`. Selection roles are `requested`, `required`,
`retained`, or `automatic`. Operations also use `guidance` and
`previously_installed` roles. Actions are `install`, `update`, `keep`, `remove`,
`write-guidance`, `strip-guidance`, `remove-guidance`, or `conflict`.

The revision fingerprints saved state, legacy import settings, source resources,
provider destinations, and any supplied inventory export. Apply checks it under
`config/apply.lock` before mutation. A revision identifies the observed state;
clients must also resend the reviewed options unchanged. Conflicts stop the whole
apply. Failed or interrupted applies restore provider files and saved settings
together; incomplete restoration reports retained recovery backups on stderr.
Success is emitted only after state persistence and lock release.

The TUI requests output-directory creation only after a default-No confirmation.
Creation runs inside the same Bash transaction. Failed applies remove empty
directories created by that transaction; directories that acquired artifacts are
preserved and reported with incomplete rollback diagnostics. Changing output
mode never moves or deletes existing artifacts. The creation flag requires a
nonblank output path and a writable, searchable existing directory or ancestor.

Portable response examples are maintained in
[`tests/fixtures/managed/`](../tests/fixtures/managed/). Their `<root>` and
`<revision>` placeholders are normalized by the integration tests.

## Native coexistence

The backend never inspects private plugin caches or invokes a provider client
during read-only commands. Without a supported export, `native_overlap.status`
is `unknown` and the response includes a general coexistence notice.

An optional adapter recognizes the pretty-printed installed `pluginId` rows
exported by Codex CLI 0.163.0:

```bash
codex plugin list --marketplace agent-toolkit --json > native-inventory.json
./scripts/managed.sh preview --protocol 1 --provider codex --skill ticket-to-plan --native-inventory native-inventory.json
```

Generate that export separately using the provider's supported read-only command.
Recognized agent-toolkit packages and bundles expand to their catalog skills and
helpers. `status: reported` describes the supplied snapshot, including installed
packages that may be disabled; it does not verify current activation or freshness.
Unknown export layouts fall back to the general notice. No Claude export adapter
is enabled. Native plugins remain outside managed mutation.

## Errors

Nonzero exits return `ok: false` and an `error` object containing `code`,
`exit_code`, and `message`. Detailed diagnostics and recovery locations remain
on stderr. Clients must check both exit status and `ok`.

| Exit | Code | Action |
| --- | --- | --- |
| 1 or other operational failure | `operation_failed` | Display stderr; inspect lock or recovery information |
| 2 | `invalid_arguments` | Correct arguments or supply required removal confirmation |
| 3 | `stale_revision` | Request and review a new preview |
| 4 | `ownership_conflict` | Resolve the conflicts without overwriting user changes |
| 5 | `unsupported_protocol` | Use a supported protocol version |
| 130 or 143 | `interrupted` | Inspect rollback diagnostics before retrying |

The existing `scripts/install.sh` remains an additive CLI with human-readable
output. It never implicitly prunes unselected skills.
