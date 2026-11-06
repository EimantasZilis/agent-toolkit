# Local management TUI

Run `make run` to configure, refresh, or uninstall local skills for Codex and
Claude Code. The TUI previews paths and changes before applying them. Native
plugins remain managed separately by each provider.

The TUI requires Git and the Go version declared in `go.mod`. Make is optional:
`go run ./cmd/agent-toolkit` starts the same interface from the checkout. The
first launch may download modules and a compatible toolchain. Startup requires
an interactive terminal and fails without changing installations when redirected.

## Configure an installation

Fresh setup starts with no provider selected and the starter skills preselected.
Choose Codex, Claude Code, or both. Complete the settings pages and final review.
Saved installations open a management menu; choose **Configure** to edit settings.

| Control | Action |
| --- | --- |
| Arrow keys | Move between rows; Left/Right expand skill categories |
| Space | Toggle a setting, skill, or category |
| `a` | Select or deselect all skills |
| `s` | Preview and save settings |
| Enter on Path | Edit the output path; Enter accepts, Escape cancels |
| Page Up / Page Down | Scroll long pages and previews |
| `t` / `T` | Cycle color themes for this run |
| Escape | Return to the previous page or menu |
| `q` | Quit; changed drafts offer Save or Discard |

Errors preserve your draft. Conflicts show the affected paths and stop apply.
Wait for installation or rollback to finish before quitting. Handled signals
also let an active transaction finish or roll back. Failed recovery reports the
location of retained backups.

## Settings

- **Skills:** required helpers are included automatically. Removing a parent
  keeps its installed helpers; remove unused helpers individually.
- **Markdown output:** use each repository or a shared absolute path, grouped
  by repository, branch, and skill. Existing artifacts stay in place.
- **Automatic concise-output loading:** install the skill and add its standing
  instruction. Disabling the instruction keeps the skill installed.
- **Full guidance template:** add shared coding and documentation instructions
  independently of concise-output loading.

Removing skills or providers and creating an output directory require explicit
confirmation. Enter, No, and Escape cancel; the default is No. Apply checks that
source files and installed content still match the preview.

## Refresh source changes

1. Run `git pull --ff-only` separately to update the checkout.
2. Run `make run` and choose **Refresh installation**.
3. Review the proposed paths and confirm.
4. Start a new agent session.

Refresh preserves saved preferences. If a saved skill is unavailable or refresh
would remove managed content, use Configure to review the selection. Supporting
files link to this checkout and may change as soon as you pull. Installation
rollback cannot undo checkout changes.

## Uninstall

Choose **Uninstall**, review the managed paths, and explicitly confirm the final
prompt. Cancellation changes nothing. Successful uninstall removes verified
managed content and saved settings, then returns to fresh setup. User guidance
outside managed blocks and existing Markdown artifacts remain in place.

Start a new agent session after configuring, refreshing, or uninstalling.
Keep the checkout while local installations use its linked supporting files.

## Transitional commands

Legacy Bash installation, update, and uninstall commands remain available during
the Go migration. They share the saved state format. The TUI is the supported
entry point for new local workflows; see [INSTALL.md](../../INSTALL.md) for the
transitional command reference.
