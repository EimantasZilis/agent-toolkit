# Managed setup and settings TUI

The TUI and Bash setup share one managed installation engine and canonical
settings. Both reconcile the desired providers and skills from this clone.
Keep the clone for the installer and supporting files.
[Native plugins](native-packages.md) are managed separately by each provider.

Terminal `make install` runs the current Go source from this clone. It requires
Go at the version declared in `go.mod` (currently 1.27.3) and the Bash installer
prerequisites listed in the README. First launch may need network access to
download modules and a compatible toolchain. Later launches use Go’s build
cache and pick up source changes without a manual build.

Use `make install-cli` to run Bash directly without Go. Non-terminal setup and
`scripts/setup.sh` with arguments also use Bash. Missing Go or build errors stop
setup with diagnostics and the Bash alternative before installation starts.

Contributors can build and check the application with:

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
Select Codex, Claude Code, or both before continuing. Saved choices
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

## Source verification and retired binary releases

`make smoke-source` opens real setup twice in a disposable checkout with a space
in its path. It quits without applying, checks provider files and saved settings,
and verifies that a harmless source change appears on the second launch.
Source CI runs this check on macOS and Linux alongside Go tests, vet, and
`make verify`. Python 3.9 or later is a contributor prerequisite for these checks.

The former binary-release workflow, launcher, pins, and download service are
retired. Setup uses Go’s cache. Old toolkit executable caches are unused:
`${XDG_CACHE_HOME:-$HOME/.cache}/agent-toolkit-tui`. You may remove that directory
manually; setup does not delete it. Native plugin packaging remains supported.
Historical Git commits and external planning/evidence artifacts may still name
the retired release commands. They describe the old delivery process.
