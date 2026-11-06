# Bash core

## Apply this first

- Use Bash deliberately: start executable scripts with `#!/usr/bin/env bash`
  when the repository supports env-based discovery, or its locally required
  Bash shebang. Do not claim POSIX `sh` compatibility for a Bash script.
- Keep scripts small and orchestration-focused. Move substantial parsing,
  stateful logic, or performance-sensitive work to a language with stronger
  data types.
- Put a short file header above the implementation. Use `main` for executable
  flow and keep reusable functions focused with explicit inputs and outputs.
- Use `local` for function variables. Use lowercase names for locals and
  uppercase names only for exported configuration or constants.
- Quote parameter expansions, command arguments, redirections, and paths by
  default. Use arrays for argument lists and `"${array[@]}"` to preserve word
  boundaries.
- Prefer `[[ ... ]]` for Bash conditionals, `(( ... ))` for arithmetic, `$(...)`
  for command substitution, and `printf` for predictable output.
- Use `case` for multi-way dispatch. Prefer a `while IFS= read -r` loop for
  line-oriented input and do not parse `ls` output.
- Make stdout the machine-readable or successful result stream and stderr the
  diagnostic stream. Return meaningful non-zero statuses from functions.
- Keep command names, variable expansions, and control flow visually clear;
  use four-space indentation unless local style says otherwise.

## Quick checklist

- [ ] The shebang, Bash version assumptions, and executable bit match the repo.
- [ ] Expansions are quoted or the unquoted splitting/globbing is intentional.
- [ ] Lists of arguments use arrays, not string concatenation.
- [ ] Functions have focused responsibilities and do not leak accidental globals.
- [ ] Output streams and return statuses are part of the script's interface.
- [ ] The changed script remains readable without relying on aliases or caller state.

## DO

```bash
#!/usr/bin/env bash

set -o errexit -o nounset -o pipefail

main() {
    local -r output_dir="$1"
    local -a command=(tool --output "$output_dir")

    "${command[@]}"
}

main "$@"
```

## DO NOT

```bash
# Unquoted strings lose boundaries and globals leak out of the function.
run() {
  output_dir=$1
  tool --output $output_dir
}
```

## Concrete exceptions

- Use `#!/bin/bash` when the repository or deployment contract requires that
  exact path; portability of the interpreter is a local deployment decision.
- Leave an expansion unquoted only when it is deliberately used for globbing,
  word splitting, or an arithmetic/pattern context, and make that intent clear
  in nearby code or a comment.
- Do not force `set -o errexit -o nounset -o pipefail` into sourced libraries or
  snippets whose caller owns shell options; document the caller contract.
- A shell script may exceed the usual small-script boundary when it is an
  established, well-tested repository entry point and a rewrite would add
  operational risk; record that trade-off in review.

## Source traceability

- Bash behavior and expansion rules: [GNU Bash Reference Manual](https://www.gnu.org/software/bash/manual/bash.html) (accessed 2026-09-30).
- Shebangs, naming, functions, formatting, streams, and when to use shell:
  [Google Shell Style Guide](https://google.github.io/styleguide/shellguide.html) (accessed 2026-09-30).
- Quoting, arrays, command substitution, and common pitfalls:
  [ShellCheck](https://github.com/koalaman/shellcheck) (accessed 2026-09-30).

These sources agree on explicit interpreter choice, careful quoting, focused
functions, and predictable streams. This pack chooses Bash-native `[[` and
arrays over POSIX portability because its scope is explicitly Bash; a local
deployment contract may override that choice.
