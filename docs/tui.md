# Managed setup and settings TUI

Development requires Go 1.27.3 or later and the existing Bash installer
prerequisites. From this source clone:

```sh
go run ./cmd/agent-toolkit --root .
go build -o /tmp/agent-toolkit ./cmd/agent-toolkit
go test ./...
go test -race ./...
go vet ./...
make verify
```

`--version` prints the application and managed protocol versions. `--root`
selects the source clone; retain it for the Bash backend and installed resources.
Bubble Tea v2 and its transitive modules are pinned in `go.mod` and `go.sum`.

First setup stages four pages and applies only after final review. Later runs
show a settings menu. Press `s` to save a page; leaving a dirty page offers
Save/Discard, and Escape cancels the dialog. Backend operations run asynchronously
and prevent duplicate saves. During apply, wait for completion or rollback
before quitting. Errors preserve the draft; `s` requests a new preview for retry.

Fresh setup starts with no provider selected and the starter skills selected.
Select Codex, Claude Code, or both before continuing. Saved and imported choices
are preserved on later runs. Expand skill categories with Left/Right, toggle a
category or skill with Space, and use `a` to select/deselect all. Required helpers
remain selected; removing a parent keeps its installed helpers. Remove unused
helpers individually when desired.

Markdown output offers repo-local mode or a recommended global path, initially
`~/agent-toolkit-output` for fresh setup. Global output is separated by repository,
branch, and skill. Select the Path row and press Enter to edit it; Enter accepts,
Escape restores the previous text, and Ctrl+u clears the field. Arrow keys,
Home/End, Backspace/Delete, and single-line paste work within the field. While
editing, letters such as `s` and `q` are ordinary path characters. The path must
be absolute and writable. Editing and preview do not create directories. A
missing directory requires a default-No confirmation before Bash creates it
during apply. Switching to repo-local leaves existing global artifacts in place.

Global settings controls only automatic `concise-output` loading. Enabling it
installs the skill and its standing instruction for every selected provider;
disabling it removes only the instruction. Existing full-template preferences
are preserved; use the CLI to change them. Provider and skill removals also
require default-No confirmation after Save. Enter, `n`, and Escape decline these
confirmations and keep the draft.

The view is framed with a title bar and a separated command footer. The view
follows the selected row and wraps within the current terminal size.
Use Page Up/Page Down to inspect long pages, previews, errors, and confirmation
lists. Very small terminals can require scrolling to read the full prompt;
resize at any point. Backend ownership conflicts and rollback diagnostics remain
visible for review and retry.

Press `t` to cycle the Midnight, Ocean, and Daylight palettes; `T` cycles
backwards. The selection is for the current run and does not change saved
installation settings. [Lip Gloss](https://github.com/charmbracelet/lipgloss)
renders the palettes and adapts colors to the terminal's color profile.
