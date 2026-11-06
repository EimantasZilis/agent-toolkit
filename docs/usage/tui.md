# Terminal setup

The TUI lets you choose providers, skills, Markdown output locations, and
automatic concise-output loading. Run `make install` to open it. Later runs
show your saved settings.

It requires Go at the version declared in `go.mod` and the
[local installation prerequisites](../../INSTALL.md#quick-start). The first launch
may download modules and a compatible toolchain. Use `make install-cli` for
Bash setup without Go. Both interfaces share the same settings and installer.

## Choose and save settings

First setup takes you through four pages, then shows a final review before
applying changes. Select Codex, Claude Code, or both; no provider is selected
initially. The starter skills are preselected.

| Control | Action |
| --- | --- |
| Arrow keys | Move between rows; Left/Right expand skill categories |
| Space | Toggle a skill or category |
| `a` | Select or deselect all skills |
| `s` | Save the page |
| Enter on the Path row | Edit the output path; Enter accepts, Escape cancels |
| Page Up / Page Down | Scroll long pages and previews |
| `t` / `T` | Cycle color themes for this run |

Leaving a changed page offers Save or Discard. Errors keep your draft so you
can correct it and press `s` to retry. Wait for installation or rollback to
finish before quitting.

## What the settings affect

- **Skills:** required helpers are included automatically. Removing a parent
  skill keeps its installed helpers; remove unused helpers individually.
- **Markdown output:** save artifacts in each repository or under a shared
  absolute path, grouped by repository, branch, and skill. Changing the location
  leaves existing artifacts in place.
- **Global settings:** enable automatic concise-output loading. Disabling it
  keeps the skill installed. Use the [CLI](../../INSTALL.md#configuration) to change
  the full global guidance template.

Removing providers or skills and creating a missing output directory require
confirmation. The default is No. User-edited installations cause a conflict
and stop the operation so you can resolve them.

Keep this checkout: installed supporting files link to it.
[Native plugins](native-packages.md) are managed separately by each provider.
