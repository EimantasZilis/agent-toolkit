# Local management TUI

Run `make run` to configure, refresh, migrate, or uninstall local skills for Codex and
Claude Code. The TUI previews paths and changes before applying them. Native
plugins remain managed separately by each provider.

The TUI requires Git and the Go version declared in `go.mod`. Make is optional:
`go run ./cmd/agent-toolkit` starts the same interface from the checkout. The
first launch may download modules and a compatible toolchain. Startup requires
an interactive terminal and fails without changing installations when redirected.

## Configure an installation

Fresh setup starts with no provider selected and the starter skills preselected.
Choose Codex, Claude Code, or both. Complete the settings pages and final review.
Saved installations open the Overview tab. Section headings and rules separate
installation settings, source information, and actions. Muted labels distinguish
settings from bold values; enabled guidance uses green text. The focused action
has a filled background and a `>` marker. The same styles apply to settings
pages, keyboard help, review summaries, and unsaved-change dialogs. Skill categories use accent headings; checked items use
green markers. Detailed reviews show additions in green and removals in red,
while retaining their `+` and `-` markers. Font and text size follow your terminal
settings.

Use the Providers, Skills, Storage, and Guidance tabs to edit settings. Tabs share one draft and remember the focused
row. Switching tabs does not apply changes or prompt to save.

| Control | Action |
| --- | --- |
| Arrow keys | Move between rows; Left/Right expand skill categories |
| ↵ | Open a menu item or toggle a setting, skill, or category |
| Tab / Shift+Tab | Move to the next / previous tab |
| `1`–`5` | Open Overview, Providers, Skills, Storage, or Guidance |
| Space | Toggle the focused selection |
| `?` | Open or close keyboard help |
| `a` | Select or deselect all skills |
| `s` | Preview and save settings |
| ↵ on Path | Edit the output path; ↵ accepts, Esc or Ctrl+C cancels |
| Page Up / Page Down | Scroll long pages and previews |
| `v` in a confirmation | Switch between the change summary and detailed paths/diffs |
| Esc / Backspace | Return to the previous page or menu |
| `q` | Quit; changed drafts offer Save or Discard |

When leaving with unsaved changes, a centered dialog overlays the current
screen. Press `s` to save, `d` to discard, or Esc/Backspace to keep editing.

Errors preserve your draft. Conflicts show the affected paths and stop apply.
The conflict screen starts with the reason and any migration guidance. Use
Up/Down or Page Up/Page Down to scroll paths, then ↵, Esc, or Backspace to return
without applying changes.
Wait for installation or rollback to finish before quitting. Handled signals
also let an active transaction finish or roll back. Failed recovery reports the
location of retained backups.

While editing a path, Backspace deletes the previous character. Esc or Ctrl+C restores
the original path and leaves editing.

The header and footer stay visible while content scrolls. At 150 columns or
wider, Skills shows three panes: the skill list, its description, and selection
details. Descriptions come from each skill’s `SKILL.md` metadata and change as
you move through the list. At 110–149 columns, the description and selection
details share a pane beside the list. Smaller terminals show them below it. Long
descriptions wrap to the pane width; an ellipsis marks text that does not fit.
Partially selected categories use `[-]` and show selection counts. Review keeps its confirmation visible while paths and diffs scroll.

## Settings

- **Skills:** required helpers are included automatically. Removing a parent
  keeps its installed helpers; remove unused helpers individually.
  Use the first row to select or deselect all skills with ↵. The `a` shortcut
  performs the same action. Automatic concise loading and retained helpers
  follow the same rules as category selection.
- **Markdown output:** use each repository or a shared absolute path, grouped
  by repository, branch, and skill. Existing artifacts stay in place.
- **Global guidance and concise-output:** add shared coding, documentation,
  and Markdown artifact instructions together with concise-output automatic
  loading. Disabling removes these instructions and keeps the skill installed.
  When saving Configure, either enabled legacy option enables both together.
  Refresh and migration preserve the existing saved preferences.

Removing skills or providers and creating an output directory require explicit
confirmation. ↵, No, Esc, and Backspace cancel; the default is No. Apply checks that
source files and installed content still match the preview.

## Repeat the Development loop

Development installs the current working tree without a commit or eligibility
change. Draft skills remain selectable. Installed copies stay unchanged while
you edit source.

1. Edit skill instructions and resources without committing.
2. Run `make run` or `go run ./cmd/agent-toolkit` from that checkout.
3. Select skills in initial setup, or choose **Refresh installation** to keep
   saved providers, skills, helper selections, and guidance/output preferences.
4. Inspect the content review and press `y` to apply. ↵, No, Esc, and Backspace cancel.
5. Start a new agent session for each selected provider and test the skill.
6. Repeat before committing. Pulling upstream changes is an optional, separate
   source-management step.

Confirmations open a compact summary with the selected-skill count, output
location, guidance settings, and change counts. Press `v` to inspect paths and
file diffs, then `v` again to return to the summary. Up/Down and Page Up/Page
Down scroll the details. Each confirmation still requires `y`.

The detailed review groups changes by provider and skill or guidance. Each
changed path shows its action, before/after type, and permission mode. Instructions,
references, scripts, and Codex adapters include text changes. Binary resources
show sizes and SHA-256 checksums. Excerpts are limited to 100 lines or 8192 bytes
per file and show a truncation notice; the path list remains complete. Guidance
shows verified managed blocks and excludes surrounding user text. Terminal
control characters appear as visible escapes.

Use Page Up and Page Down to inspect long reviews, including in narrow windows.
An unchanged review says **Installed content: unchanged.** Existing agent
sessions do not reload automatically. Copied skills work without the checkout;
the manager still needs the checkout and its saved `config/` state.

Refresh can remove resources within retained skills. Use Configure to remove
whole skills or providers, or resolve an unavailable saved selection. Edited
installed files, changed modes, missing files, and unexpected content cause
ownership conflicts. Installation rollback cannot undo checkout changes.

If source, destination, or saved state changes after review, apply refuses.
Choose **Refresh installation** again for a new refresh review. Migration retries
through **Migrate linked installation**. Setup and Configure preserve the draft;
press `s` to preview it again. Each new preview requires confirmation.

Committed, eligible versioned snapshots belong to future lifecycle work. They
are separate from the existing [native plugin tools](native-packages.md).

## Migrate a linked installation

Saved linked installations offer **Migrate linked installation** in the
management menu. Migration is a separate review; Configure, Refresh installation,
and Uninstall never convert links automatically.

1. Back up managed provider directories, guidance and its markers, and both
   saved state files together. Keep the original checkout available.
2. Choose **Migrate linked installation**. Review link-to-copy changes, resource
   content and modes, rendered instruction changes, managed guidance, and saved
   providers, requested skills, retained and required helpers, output preferences,
   and guidance settings. Use Page Up and Page Down to inspect long previews.
3. Press `y` to convert. ↵, No, Esc, and Backspace cancel without writes.
4. Start a new agent session after successful conversion.

The supported layout is the previous Go installer's real skill directories,
checksummed rendered manifests, and individual resource and Codex adapter links
into this checkout. Migration requires intact saved settings and inventory.
It preserves preferences and the dependency closure. It uses the current
Development checkout; source switching is outside this action. Missing output
directories are left alone. In mixed installations, verified copied skills stay
unchanged while linked skills become complete copies.

Altered manifests, missing or extra resources, foreign links, symlinked parents,
unavailable skills, and changed dependency selections block migration. Diagnostics
identify the affected path and preserve the linked installation. Restore matching
source/state or resolve the affected path, then choose migration again. A stale
preview also requires another review.

Failure or cancellation during apply restores old links, modes, guidance, and
both state files. If rollback fails, keep the reported recovery backups and
restore the complete pre-migration provider/state backup. Reverting manager code
does not undo conversion. Use the copied-format manager or restore the full
backup; do not downgrade ownership records in place. After conversion, checkout
resource edits take effect only through normal reviewed refresh.

## Uninstall

Choose **Uninstall**, review the managed paths, and explicitly confirm the final
prompt. Cancellation changes nothing. Successful uninstall removes verified
managed content and saved settings, then returns to fresh setup. User guidance
outside managed blocks and existing Markdown artifacts remain in place.

Start a new agent session after configuring, refreshing, or uninstalling.
Keep the checkout and saved state to manage copied installations.

## Saved state and supported entry point

Settings and provider-home overrides retain their existing formats. Copied
installations use version 2 ownership and inventory records, including the
canonical Development checkout and a digest of current `.agents/` inputs.
Legacy records remain readable. Conversion requires a separate migration review
and independent verification of every linked skill.
The TUI reads saved values as data without shell evaluation. Configure changes
preferences; refresh reapplies them.

Local management uses the TUI. The former Bash scripts and installation, update,
and uninstall Make targets have been removed. See [INSTALL.md](../../INSTALL.md)
for installation paths, output rules, and recovery.
