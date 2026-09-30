"""Reject Python functions that exceed the local readability limit."""

from __future__ import annotations

import ast
import sys
from pathlib import Path

MAX_FUNCTION_LINES = 40


def check_file(path: Path) -> list[str]:
    """Return function-size violations found in one Python file."""
    tree = ast.parse(path.read_text(), filename=str(path))
    violations: list[str] = []

    for node in ast.walk(tree):
        if not isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
            continue

        function_lines = node.end_lineno - node.lineno + 1
        if function_lines > MAX_FUNCTION_LINES:
            violations.append(
                f"{path}:{node.lineno}: {node.name} is {function_lines} lines "
                f"(maximum {MAX_FUNCTION_LINES})"
            )

    return violations


def main() -> int:
    """Check the Python files passed on the command line."""
    violations = [
        violation
        for filename in sys.argv[1:]
        for violation in check_file(Path(filename))
    ]
    for violation in violations:
        print(violation)
    return 1 if violations else 0


if __name__ == "__main__":
    raise SystemExit(main())
