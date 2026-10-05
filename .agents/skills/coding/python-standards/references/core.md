# Python core and hygiene guidance

## Apply this first

- Format and lint only changed Python paths by default.
- After Python edits, run the repository's formatter and linter on changed
  paths, then inspect the diff for vertical spacing and other readability
  details that automated formatters do not enforce.
- Treat this diff inspection as an LLM implementation pass: make the needed
  readability edits rather than only reporting them.
- Run the smallest relevant test or verification command after the edit; do
  not treat formatting and lint success as behavioral validation.
- Follow formatting/import/complexity constraints from `pyproject.toml`.
- Do not add new dependencies unless necessary.
- If a dependency is added, document its license in `pyproject.toml` comments.
- Add/update docstrings for new or changed behavior.
- Keep docstrings behavior-accurate; do not drift from code.
- Prefer concise comments that explain why, not what.
- Never log secrets, tokens, passwords, or credentials.
- Include stable context fields (`extra`) in logs for correlation when available.

## Quick checklist

- [ ] `pyproject.toml` remains the hygiene source of truth.
- [ ] Only changed paths are formatted/linted by default.
- [ ] Changed code has readable vertical spacing between logical sections.
- [ ] Validation scope is minimal but sufficient.
- [ ] Relevant tests or verification commands have been run.
- [ ] New dependency is truly needed and justified.
- [ ] Dependency changes follow project conventions.

---

## Code constraints

- Follow pyproject.toml:
  - line length (~88)
  - import ordering
  - McCabe complexity ≤ 8
  - branches per function ≤ 8
  - statements per function ≤ 25

## Complexity enforcement

For projects using Ruff, enable `C901`, `PLR0912`, and `PLR0915` rather than
relying on review judgment alone. Preserve the target repository's Ruff
configuration and add these checks through CLI overrides:

```sh
ruff check \
  --extend-select C901,PLR0912,PLR0915 \
  --config lint.mccabe.max-complexity=8 \
  --config lint.pylint.max-branches=8 \
  --config lint.pylint.max-statements=25 \
  path/to/changed_file.py
```

Do not pass a pack-owned file with `--config`; a configuration file supplied
that way replaces Ruff's discovered target configuration rather than merging
with it. If the target configuration explicitly ignores one of these rules,
report that conflict instead of modifying the target repository.

These checks do not measure every form of size or responsibility. Have the
agent review changed Python functions for physical size, cohesion, and single
responsibility, using 40 physical lines as the default threshold. Distinguish
pre-existing violations from regressions and report the reasoning when a
function exceeds the threshold. Do not add bespoke AST checks or workflow code
to the target project for this review. If deterministic enforcement is needed,
run the checker bundled with this pack against the target files:

```sh
python3 .agents/skills/coding/python-standards/scripts/check_function_size.py \
  path/to/changed_file.py
```

Maintain the checker here rather than copying it into each project.

---

## Practical hygiene defaults

- Treat `pyproject.toml` as the single source of truth for formatter/linter/test config.
- Run the smallest verification scope that still proves correctness (changed files/tests first, full suite before merge when feasible).
- Use project virtual environments for dependency isolation; do not mix global packages into project workflows.
- Prefer standard hygiene output over custom wrappers unless the repo already standardizes a wrapper command.

---

## Dependency hygiene

- Prefer standard-library modules before adding third-party dependencies.
- Add dependencies only when they provide clear value over built-ins or existing project packages.
- Pin and document new dependency rationale in the project's existing dependency management workflow.
