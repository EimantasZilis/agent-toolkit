# Bash security-sensitive scripts

## Apply this first

- Treat every argument, environment variable, filename, command output, and
  sourced file as untrusted until validated for its intended use.
- Quote untrusted data and pass it as an argument. Do not construct shell code
  from data; avoid `eval`, `bash -c`, dynamic `source`, and command strings.
- Use `mktemp` for temporary files and directories, retain restrictive
  permissions, and clean them up with a safe trap. Never use predictable names
  in shared directories.
- Use `--` before user-controlled paths for commands that support it. Validate
  allowed values and canonicalize paths before destructive operations.
- Resolve dependencies and sourced files from trusted, explicit locations. Do
  not prepend writable directories to `PATH` for privileged work.
- Keep secrets out of command-line arguments, logs, traces, test output, and
  temporary files. Disable `set -x` around secret handling.
- Do not use SUID/SGID shell scripts. Prefer a narrow privileged helper with a
  separately reviewed interface when elevation is unavoidable.

## Quick checklist

- [ ] No user data is evaluated as shell syntax.
- [ ] Temporary paths are unique, private, and cleaned safely.
- [ ] Destructive targets are validated and bounded.
- [ ] Secrets are absent from arguments, logs, traces, and fixtures.
- [ ] `PATH`, `IFS`, locale, and sourced code are controlled where trust matters.
- [ ] Privilege boundaries and required permissions are explicit and reviewed.

## DO

```bash
input_path="$1"
[[ -f "$input_path" ]] || {
    printf 'input is not a regular file\n' >&2
    exit 2
}

temporary_dir="$(mktemp -d)"
trap 'rm -rf -- "$temporary_dir"' EXIT

cp -- "$input_path" "$temporary_dir/input"
```

## DO NOT

```bash
# User input becomes code and the predictable path is unsafe in a shared dir.
eval "tool $user_options"
temporary_file="/tmp/report.$$"
```

## Concrete exceptions

- `eval` may be unavoidable for a narrowly controlled, internally generated
  Bash declaration or completion hook; isolate it, prove the input is not
  user-controlled, and explain the invariant in review.
- `mktemp` may be unavailable in a constrained target; use the platform's
  documented secure temporary-file API or a private directory created with
  exclusive permissions, never a predictable shared path.
- A diagnostic may identify a resource by a stable non-secret identifier; mask
  or omit values whenever their sensitivity is uncertain.

## Source traceability

- Expansion, quoting, command execution, and shell options:
  [GNU Bash Reference Manual](https://www.gnu.org/software/bash/manual/bash.html) (accessed 2026-09-30).
- Quoting, `eval`, arrays, and unsafe patterns:
  [ShellCheck](https://github.com/koalaman/shellcheck) (accessed 2026-09-30).
- SUID/SGID prohibition, shell scope, and streams:
  [Google Shell Style Guide](https://google.github.io/styleguide/shellguide.html) (accessed 2026-09-30).

The SUID/SGID rule is adopted directly from Google guidance; the broader input,
temporary-file, and secret-handling checklist applies that principle to the
repository's general Bash use rather than claiming Bash itself provides a
security boundary.
