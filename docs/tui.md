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
