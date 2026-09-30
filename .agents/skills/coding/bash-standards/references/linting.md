# Bash linting and verification

## Apply this first

- Parse every changed script with `bash -n path/to/script` before running it.
- Run ShellCheck with Bash mode, normally `shellcheck --shell=bash path...`,
  and fix warnings rather than suppressing them globally.
- Use shfmt for formatting when the repository adopts it; pin the version in
  CI or the tool manifest when reproducibility matters.
- Keep shellcheck directives narrow and explain each justified suppression next
  to the directive. Never suppress a warning merely to make CI green.
- Test the script from a clean environment with representative arguments,
  empty values, paths containing spaces, command failures, and interrupted or
  partially completed work where relevant.
- Inspect executable permissions, shebangs, sourced paths, external command
  dependencies, and changed-file scope in review.
- Run the repository's own verification command in addition to language tools.

## Quick checklist

- [ ] `bash -n` passes for every changed Bash script.
- [ ] ShellCheck passes at the repository's configured severity.
- [ ] shfmt output is clean when shfmt is configured.
- [ ] Tool versions and local suppressions are reproducible and justified.
- [ ] Relevant tests and `make verify` (or the local equivalent) pass.
- [ ] No secrets, generated files, or unrelated formatting changes entered the diff.

## DO

```bash
bash -n scripts/deploy.sh
shellcheck --shell=bash scripts/deploy.sh
shfmt -d scripts/deploy.sh
```

```bash
# shellcheck disable=SC2034  # Exported for the sourced deployment helper.
export DEPLOY_REGION
```

## DO NOT

```bash
# Disables useful analysis for every file and every future warning.
shellcheck --exclude='SC2086,SC2046,SC2155,all' scripts/*.sh
```

## Concrete exceptions

- Skip `shfmt` only when the repository has no formatter policy, or when a
  generated/vendor file cannot be formatted; document the path and reason.
- Use a narrow ShellCheck suppression for a documented false positive or an
  intentional shell construct; retain the warning code and a concrete reason.
- A script that intentionally targets another Bash version may use syntax not
  available locally, but validation must run in the declared target image.

## Source traceability

- Bash syntax and exit behavior: [GNU Bash Reference Manual](https://www.gnu.org/software/bash/manual/bash.html) (accessed 2026-09-30).
- Static analysis, exit codes, and build integration:
  [ShellCheck README](https://github.com/koalaman/shellcheck) (accessed 2026-09-30).
- Formatting: [shfmt project](https://github.com/mvdan/sh) (accessed 2026-09-30).

ShellCheck explicitly states that it does not enforce formatting, so this pack
keeps linting and formatting as separate checks rather than treating one as a
replacement for the other.
