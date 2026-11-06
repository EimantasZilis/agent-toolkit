# Verify repository tooling

Verification checks the shared skill catalog and exercises local installations
in disposable provider homes. It preserves the installation and packaging
contracts while keeping real provider directories untouched.

```sh
make validate
make test
make verify
```

`make verify` runs Go tests, vet, source validation, and isolated selective and
full installations for Codex and Claude. Tests inspect the bundled Python
checker as a resource; they do not execute it.

## Behavioral coverage after the Go migration

This mapping records the replacements for the legacy test harnesses. JSON
transport shapes and scripted management prompts are retired interfaces;
their selection, preservation, confirmation, and failure behavior is tested
through the typed service and TUI instead.

| Legacy coverage | Go replacement |
| --- | --- |
| Catalog, bundles, dependency closure, cycles, registries | `internal/catalog/catalog_test.go` |
| Source layout, front matter, relative links, optional and stale adapters | `internal/validate/source_test.go` |
| Both providers, selective and full installation, linked resources, metadata, guidance, refresh, uninstall | `internal/validate/verify.go` and `TestIsolatedVerification` |
| Selection roles, saved choices, retained helpers, automatic concise guidance, removal confirmation | `internal/install/preview_test.go`, `internal/tui/selection_test.go` |
| State compatibility, literal paths, malformed state, inventory ownership limits | `internal/install/preview_test.go`, `internal/install/compatibility_test.go` |
| Edited manifests, missing and dangling resources, foreign paths, symlink boundaries, stale previews | `internal/install/preview_test.go`, `internal/install/apply_test.go` |
| Transactions, cancellation, locking, both providers, state rollback, retained recovery backups | `internal/install/apply_test.go` |
| Guidance preservation, duplicate blocks, edited blocks, independent preferences | `internal/install/compatibility_test.go`, `internal/tui/settings_test.go` |
| Refresh from changed source, missing state, native helper overlap | `internal/install/compatibility_test.go`, `internal/tui/management_test.go` |
| TUI navigation, conflicts, drafts, output creation, viewport sizing, shutdown | Tests beside `internal/tui` |
| Local/native rendering, invocations, optional links, output policy | `internal/render/render_test.go` |
| Archives, manifests, resource bytes and modes, corruption, traversal, historical releases | `internal/native/release_test.go`, `internal/native/resources_test.go` |
| Snapshot provenance, committed bytes/modes, relocation, hostile catalogs, historical snapshots | `internal/native/snapshot_test.go` |
| Non-terminal startup, argument failures, version reporting | `cmd/agent-toolkit/main_test.go`, `cmd/agent-toolkit/native_test.go` |

Real terminal startup and current-source checks are covered by `make smoke-source`.
