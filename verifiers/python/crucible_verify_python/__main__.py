"""``python -m crucible_verify_python`` / console-script entrypoint."""

from __future__ import annotations

import sys

from .cli import run


def main(argv: list[str] | None = None) -> int:
    """Console-script entrypoint.

    Returns the process exit code. The dispatched body is responsible for
    writing the TestReport to stdout; this wrapper only translates
    Python-level errors into stable exit codes.
    """
    try:
        return run(argv if argv is not None else sys.argv[1:])
    except KeyboardInterrupt:
        print("crucible-verify-python: interrupted", file=sys.stderr)
        return 130


if __name__ == "__main__":  # pragma: no cover — module entrypoint
    raise SystemExit(main())
