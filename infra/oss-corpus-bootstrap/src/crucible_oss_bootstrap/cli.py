"""crucible-oss-bootstrap CLI."""

from __future__ import annotations

import argparse
import json
import sys
from typing import Sequence

from .pipeline import build_all, build_bundle, write_all


def _cmd_run(args: argparse.Namespace) -> int:
    paths = write_all(args.output)
    summary = {"output_dir": args.output, "bundles": paths, "count": len(paths)}
    print(json.dumps(summary, indent=2))
    return 0


def _cmd_build_stack(args: argparse.Namespace) -> int:
    bundle = build_bundle(args.stack)
    bundle.validate()
    out = bundle.to_dict()
    if args.out:
        with open(args.out, "w", encoding="utf-8") as f:
            json.dump(out, f, indent=2)
        print(f"wrote {args.out}", file=sys.stderr)
    else:
        json.dump(out, sys.stdout, indent=2)
        print()
    return 0


def _cmd_stats(args: argparse.Namespace) -> int:
    bundles = build_all()
    stats = {b.stack.value: b.stats.active_rules for b in bundles}
    stats["__total__"] = sum(stats.values())
    print(json.dumps(stats, indent=2))
    return 0


def main(argv: Sequence[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="crucible-oss-bootstrap")
    sub = parser.add_subparsers(dest="cmd", required=True)

    run = sub.add_parser("run", help="build all 12 stack bundles and write to disk")
    run.add_argument("--output", required=True, help="output directory (services/memory-router/global_defaults)")
    run.set_defaults(fn=_cmd_run)

    bs = sub.add_parser("build-stack", help="build a single stack bundle and emit JSON")
    bs.add_argument("--stack", required=True)
    bs.add_argument("--out", default="")
    bs.set_defaults(fn=_cmd_build_stack)

    stats = sub.add_parser("stats", help="report per-stack rule counts")
    stats.set_defaults(fn=_cmd_stats)

    args = parser.parse_args(argv)
    return int(args.fn(args) or 0)


if __name__ == "__main__":
    sys.exit(main())
