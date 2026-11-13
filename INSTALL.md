# Install and manage skills

Choose native plugins for provider-managed skills without a source checkout.
Use the local TUI when you want to select skills from a checkout and configure
global guidance or Markdown output. Prefer one route per skill to avoid duplicates.

Native installation, updates, and removal are covered in the
[native plugin guide](docs/usage/native-packages.md). The rest of this guide
covers local management.

## Start local management

Use an interactive macOS or Linux terminal with Git and the Go version in
`go.mod`. Install and sign in to Codex or Claude Code first. Make is optional.

```sh
git clone https://github.com/EimantasZilis/agent-toolkit.git
cd agent-toolkit
make run
```

Without Make, run `go run ./cmd/agent-toolkit`. The first launch may download
modules and a compatible toolchain. The TUI always runs current source.
Redirected startup fails clearly before changing installations.

Fresh setup starts with no provider selected and the starter skills preselected.
Work through the settings pages and review the content changes before applying.
Later launches open a management
menu with Configure, Refresh installation, and Uninstall. Linked installations
also offer Migrate linked installation. See the
[TUI controls](docs/usage/tui.md).

Local management is interactive. The former Bash scripts and the `install`,
`install-cli`, `update`, and `uninstall` Make targets have been removed.
Maintainer commands validate source and build plugins; they do not provide
scripted local installation.

## Choose skills and guidance

| Selection | Includes |
| --- | --- |
| Starter | `concise-output` and `commit-message` |
| Category, such as coding | Skills in that category |
| Individual skills | Selected skills and their required helpers |
| All skills | The full catalog |

Required helpers are included automatically: `plan-review` for `ticket-to-plan`
and `ruby-standards` for `ruby-rails-standards`. Optional integrations are separate.
Removing a parent keeps installed helpers; remove unused helpers individually.

Automatic concise-output loading adds a standing instruction and installs the
skill. Disabling the instruction keeps the skill installed. Installing the skill
alone does not make it a standing instruction.

The full global guidance template is an independent choice. It adds shared
coding and documentation guidance from
[instructions.md](.agents/global/instructions.md). A shared output directory or
concise loading can require a smaller managed guidance block even when the
full template is disabled.

## Installed files and checkout lifetime

| Provider | Skills | Global guidance |
| --- | --- | --- |
| Codex | `${AGENTS_HOME:-~/.agents}/skills/agent-toolkit/` | `${CODEX_HOME:-~/.codex}/AGENTS.md` |
| Claude | `${CLAUDE_HOME:-~/.claude}/skills/<skill-name>/` | `${CLAUDE_HOME:-~/.claude}/CLAUDE.md` |

Fresh installations contain owned copies of rendered Markdown, resources, and
provider adapters. They remain usable without the checkout. Source edits take
effect through a reviewed refresh. Keep the checkout and its saved `config/`
state to run the manager. Provider-home overrides still select destinations.

Legacy linked installations require their original checkout until you review
and apply migration to owned copies. Configure, Refresh installation, and
Uninstall refuse linked skills; use the separate migration action first.

The TUI preserves:

- The `config/settings.env` preferences and legacy inventory readability.
- Provider paths and home overrides.
- Existing ownership markers.

Settings are read as data
without shell evaluation. Change preferences through Configure; avoid editing
state files directly. Inventory records observations and cannot grant ownership
of user files.

## Migrate linked installations

Migration converts the linked layout created by the previous Go installer into
owned Development copies. It preserves saved provider choices, requested skills,
retained and required helpers, guidance settings, and Markdown output preferences.
It uses the current checkout and can update rendered instructions during conversion.
Source switching is outside migration.

1. Back up the managed skill directories, provider guidance files and their
   ownership markers, and both files under `config/` together.
2. Keep the original checkout available and start the current TUI from it.
3. Choose **Migrate linked installation**. Review every conversion path and the
   saved preferences. Page Up and Page Down scroll long previews.
4. Press `y` to apply. Enter, No, and Escape cancel without writes.
5. Start a new agent session. Future source edits require a reviewed refresh.

Supported linked skills have real directories, a regular rendered `SKILL.md`,
an exact checkout marker with its manifest checksum, and individual resource
links to canonical paths in this checkout. Nested resources and Codex adapter
links are supported. Saved settings and inventory must match the installed
selection and current dependency closure. Mixed copied and linked installations
are supported; migration keeps verified copied skills unchanged.

Missing state, unavailable selections, altered manifests, missing or unexpected
resources, foreign targets, and symlinked destination parents block conversion.
The diagnostic identifies the affected path. Refusal preserves both providers
and their state. Keep the checkout available while resolving the diagnostic;
inventory alone cannot authorize adopting user content.

Apply rechecks the review under the installation lock. Changes to source,
installed links, guidance, or saved state require a new preview. Failed or
cancelled apply restores the old links, guidance, modes, and both state files.
If rollback fails, retain the reported recovery directory and restore the
complete pre-migration provider and state backup before retrying. Reverting the
manager does not reverse migration; use the matching copied-format manager or
restore the complete backup. Do not downgrade ownership records in place.

## Markdown output

| Mode | Artifact location |
| --- | --- |
| Repository | `<repo-root>/<branch-slug>/<skill-name>/...` |
| Shared directory | `<output-root>/<repo-slug>/<branch-slug>/<skill-name>/...` |

Fresh TUI setup recommends a shared directory. Choose repository output to
keep artifacts with each project. Explicit user paths take precedence.
Slugs use lowercase names with unsafe characters replaced by `-`; detached
HEADs use `detached-<short-sha>`.

Changing the output location does not move existing artifacts. Uninstall does
not delete them. When a skill consumes earlier workflow output, it checks:

1. An explicit path.
2. The current repository and branch's producer-skill directory.
3. Its configured shared-directory equivalent.
4. A documented legacy location.

## Develop and refresh without committing

The Development installation reads your current working tree. Instructions,
resources, and provider adapters become installed copies only when you accept a
review. Draft skills remain selectable; Development needs no eligibility change.

1. Edit skill instructions and resources in the checkout without committing.
2. Run `make run` or `go run ./cmd/agent-toolkit` from that checkout.
3. Select skills during initial setup, or choose **Refresh installation** to use
   your saved selection and preferences.
4. Review additions, edits, removals, and permission transitions. Use Page Up and
   Page Down to inspect every changed path. Press `y` to apply; Enter, No, and
   Escape cancel.
5. Test the skill in a new agent session for each selected provider.
6. Repeat the edit, review, apply, and new-session test loop before committing.

The review includes instructions, nested resources, scripts, Codex adapters, and
managed guidance blocks. It shows text changes and old/new file types and modes.
Binary changes show byte counts and SHA-256 checksums. Text excerpts are limited
to 100 lines or 8192 bytes per file, with a visible truncation notice. Every
changed path remains listed. Surrounding user guidance stays outside the review.

Refresh preserves saved providers, skills, retained helpers, and guidance/output
preferences. Source edits leave installed copies unchanged until apply. Removing
a resource from a retained skill is a reviewed update. Removing whole skills or
providers requires Configure. Missing state, unavailable saved skills, and
ownership conflicts also require attention before apply.

Pull upstream changes separately with `git pull --ff-only` when wanted. Refresh
itself reads current source and does not pull. A stale review requires another
preview through **Refresh installation**, **Migrate linked installation**, or the
preserved Configure/setup draft, as appropriate. A new preview never applies
automatically. Installation rollback cannot undo checkout changes.

Committed, eligible versioned snapshots for a later lifecycle workflow are
future work. Existing native plugin build and publication tools remain a
separate workflow; see the [native plugin guide](docs/usage/native-packages.md).

## Conflicts and recovery

The TUI shows affected paths and preserves your draft when a conflict prevents
apply. Edited manifests, resources, modes, managed guidance, missing files, and
foreign content require review. Back up user changes outside provider skill
directories before resolving the affected paths.

Removing skills or providers and creating output directories require explicit
confirmation. The default is No. A stale preview is rejected if source or
installed content changes before apply.

Failed operations restore provider files and saved state together. Wait for
installation or rollback to finish. If recovery fails, the error reports retained
backup paths for manual recovery. Filesystem rollback does not cover machine
loss or uncatchable termination.

Copied ownership and inventory use schema version 2. Before reverting to a
manager that only understands linked installations, uninstall copied content
with the matching manager or restore a complete provider and state backup from
before installation. Do not downgrade ownership records in place.

## Uninstall local skills

Run the TUI and choose Uninstall. Review the managed paths and explicitly confirm
the final prompt. Successful uninstall removes verified managed skills, guidance,
and saved settings in one transaction. Surrounding user guidance, unrelated files,
Markdown artifacts, and native plugins remain.

Ownership conflicts stop uninstall before writes. Resolve them and retry.
Start a new agent session after configuring, refreshing, or uninstalling.
