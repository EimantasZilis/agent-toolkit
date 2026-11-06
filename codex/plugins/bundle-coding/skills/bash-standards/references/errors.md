# Bash errors, status, and cleanup

## Apply this first

- Define the script's success and failure contract before implementing it.
- Use `set -o errexit -o nounset -o pipefail` for executable scripts when the
  caller contract permits it, but understand the documented exceptions to
  `errexit` in conditionals, lists, pipelines, and `!` commands.
- Check expected failures explicitly with `if`, `case`, or `||` and keep the
  recovery path adjacent to the command that can fail.
- Preserve the status you intend: capture it immediately, use `if ! command;` 
  for diagnostics, and do not run an unrelated command before returning it.
- Install cleanup traps once, use a dedicated cleanup function, and make
  cleanup idempotent. Remove only paths created by the current invocation.
- Send actionable diagnostics to stderr and include the operation and safe
  context, never credentials or full secret-bearing commands.
- Add retries only for transient, bounded failures. Use backoff and a final
  failure status; do not retry validation or permission errors blindly.

## Quick checklist

- [ ] Expected non-zero statuses are handled intentionally.
- [ ] Pipeline failures cannot be masked unintentionally.
- [ ] Traps preserve or deliberately replace the original exit status.
- [ ] Cleanup is safe when setup failed halfway and when run twice.
- [ ] Retry count, delay, and retryable conditions are bounded.
- [ ] Error text goes to stderr and excludes sensitive values.

## DO

```bash
cleanup() {
    local status=$?
    rm -f -- "$temporary_file"
    trap - EXIT
    return "$status"
}

temporary_file="$(mktemp)"
trap cleanup EXIT

if ! result="$(fetch_result)"; then
    printf 'fetch_result failed\n' >&2
    exit 1
fi
```

## DO NOT

```bash
# This hides which command failed, retries permanent failures, and can delete
# an unintended path if the variable is empty or changed.
set -e
while ! deploy "$target"; do
    sleep 1
done
rm -rf "$WORKDIR"/*
```

## Concrete exceptions

- A library may avoid changing shell options because options are process-global;
  its public functions must still check statuses and document caller needs.
- A command in an `if` condition, `while` condition, `&&`/`||` list, or
  intentionally inverted `!` expression may return non-zero as control flow;
  make that intent obvious and test it.
- Cleanup may intentionally ignore a missing path (`rm -f`, `|| true`) only when
  the ignored failure cannot hide a real failure and the reason is documented.

## Source traceability

- `errexit`, `nounset`, `pipefail`, pipelines, and traps:
  [GNU Bash Reference Manual](https://www.gnu.org/software/bash/manual/bash.html) (accessed 2026-09-30).
- Return-value checks and diagnostic output:
  [Google Shell Style Guide](https://google.github.io/styleguide/shellguide.html) (accessed 2026-09-30).
- Failure-prone constructs and warnings:
  [ShellCheck](https://github.com/koalaman/shellcheck) (accessed 2026-09-30).

The Bash manual documents important `errexit` exceptions; therefore this pack
does not present strict mode as a substitute for explicit error handling.
