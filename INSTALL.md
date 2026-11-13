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
Work through the settings pages and review the proposed paths before applying.
Later launches open a management
menu with Configure, Refresh installation, and Uninstall. See the
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

Legacy linked installations require their original checkout and matching
manager. This version recognizes them but refuses conversion or removal;
migration is deferred.

The TUI preserves:

- The `config/settings.env` preferences and legacy inventory readability.
- Provider paths and home overrides.
- Existing ownership markers.

Settings are read as data
without shell evaluation. Change preferences through Configure; avoid editing
state files directly. Inventory records observations and cannot grant ownership
of user files.

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

## Refresh source changes

1. Review source changes and run `git pull --ff-only` in the checkout.
2. Run `make run` or `go run ./cmd/agent-toolkit`.
3. Choose Refresh installation and review the proposed paths.
4. Start a new agent session after applying.

Refresh reapplies saved providers, skills, retained helpers, and guidance/output
preferences from current source. It does not pull Git changes. Missing state,
unavailable saved skills, proposed removals, or ownership conflicts require
Configure to review the installation.

Copied resources remain unchanged until refresh applies reviewed source bytes
and modes. Rollback of an installation cannot undo checkout changes.

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
