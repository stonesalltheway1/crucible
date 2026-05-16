"""crucible-cartographer CLI.

  crucible-cartographer scan --repo acme/payments --path ./repo \
      [--tenant-id ten_x] [--stack nextjs] [--out result.json]

Used by the onboarding flow (Stage 2: Cartography). Runs inside the
customer's twin so the wall-clock budget is enforced by the twin
itself; the CLI exits with status 0 on success.
"""

from __future__ import annotations

import argparse
import dataclasses
import json
import sys
from datetime import datetime
from enum import Enum
from typing import Any, Sequence

from .scanner import CartographerJob, scan


def _serialize(obj: Any) -> Any:
    if isinstance(obj, datetime):
        return obj.strftime("%Y-%m-%dT%H:%M:%S.%fZ")
    if isinstance(obj, Enum):
        return obj.value
    if dataclasses.is_dataclass(obj):
        return {k: _serialize(v) for k, v in dataclasses.asdict(obj).items()}
    if isinstance(obj, dict):
        return {k: _serialize(v) for k, v in obj.items()}
    if isinstance(obj, list):
        return [_serialize(v) for v in obj]
    return obj


def _cmd_scan(args: argparse.Namespace) -> int:
    job = CartographerJob(
        tenant_id=args.tenant_id,
        repo=args.repo,
        repo_local_path=args.path,
        stack_hint=args.stack,
        include_pr_history=not args.no_pr_history,
    )
    res = scan(job)
    payload = _serialize(res)
    if args.out:
        with open(args.out, "w", encoding="utf-8") as f:
            json.dump(payload, f, indent=2)
        print(f"wrote {args.out}", file=sys.stderr)
    else:
        json.dump(payload, sys.stdout, indent=2)
        print()
    return 0


def main(argv: Sequence[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="crucible-cartographer")
    sub = parser.add_subparsers(dest="cmd", required=True)

    sc = sub.add_parser("scan", help="walk a repo and emit a cartographer result")
    sc.add_argument("--repo", required=True, help="logical repo identifier (e.g. acme/payments)")
    sc.add_argument("--path", required=True, help="local filesystem path to the repo")
    sc.add_argument("--tenant-id", default="ten_local", help="tenant_id for the admission writes")
    sc.add_argument("--stack", default="", help="override the detected stack")
    sc.add_argument("--out", default="", help="write the result JSON to this file")
    sc.add_argument("--no-pr-history", action="store_true", help="skip the PR-comment pass")
    sc.set_defaults(fn=_cmd_scan)

    args = parser.parse_args(argv)
    return int(args.fn(args) or 0)


if __name__ == "__main__":
    sys.exit(main())
