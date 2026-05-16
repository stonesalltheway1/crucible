"""crucible-distiller CLI.

Subcommands:
  selfcheck     Run the catch-rate audit against the adversarial corpus.
  process       Process one offline-corpus directory (used by the OSS
                bootstrap pipeline).
  daemon        Run the queue-consumer daemon (NotImplemented in Phase 5
                offline mode; the queue consumer ships behind the
                `kafka`/`sqs` extras).
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from typing import Sequence

from .judge.adversarial_corpus import ADVERSARIAL_CORPUS, HONEST_CORPUS
from .judge.deterministic import deterministic_verdict
from .judge.llm_judge import FakeJudge, judge
from .types import ConventionCandidate, ConventionCategory, ScopeFilter


def _conv(rule: str) -> ConventionCandidate:
    return ConventionCandidate(
        id="cand_corpus",
        tenant_id="ten_selfcheck",
        scope=ScopeFilter(),
        rule_nl=rule,
        category=ConventionCategory.SECURITY_DEFAULTS,
        rationale="",
        evidence_quote="",
        extractor_model="selfcheck",
    )


def _cmd_selfcheck(args: argparse.Namespace) -> int:
    """Run the adversarial + honest corpus through both judges and emit
    a JSON summary. Exit-code 1 on catch-rate failure."""
    fj = FakeJudge()
    adv_total = len(ADVERSARIAL_CORPUS)
    adv_caught_det = 0
    adv_caught_llm = 0
    adv_caught_either = 0
    for rule in ADVERSARIAL_CORPUS:
        c = _conv(rule)
        det = deterministic_verdict(c)
        llm = judge(fj, c)
        if det.quarantine:
            adv_caught_det += 1
        if llm.quarantine:
            adv_caught_llm += 1
        if det.quarantine or llm.quarantine:
            adv_caught_either += 1
    honest_total = len(HONEST_CORPUS)
    honest_falsepos_det = 0
    honest_falsepos_llm = 0
    for rule in HONEST_CORPUS:
        c = _conv(rule)
        det = deterministic_verdict(c)
        llm = judge(fj, c)
        if det.quarantine:
            honest_falsepos_det += 1
        if llm.quarantine:
            honest_falsepos_llm += 1

    summary = {
        "adversarial_total":      adv_total,
        "adversarial_caught_det": adv_caught_det,
        "adversarial_caught_llm": adv_caught_llm,
        "adversarial_caught_combined": adv_caught_either,
        "adversarial_catch_rate_combined": adv_caught_either / adv_total,
        "honest_total":           honest_total,
        "honest_falsepos_det":    honest_falsepos_det,
        "honest_falsepos_llm":    honest_falsepos_llm,
        "min_catch_rate_target":  args.min_catch_rate,
    }
    print(json.dumps(summary, indent=2))
    if summary["adversarial_catch_rate_combined"] < args.min_catch_rate:
        print(f"selfcheck FAILED: catch rate {summary['adversarial_catch_rate_combined']:.3f} < target {args.min_catch_rate:.3f}", file=sys.stderr)
        return 1
    if honest_falsepos_det + honest_falsepos_llm > 0:
        print(f"selfcheck WARNING: {honest_falsepos_det + honest_falsepos_llm} honest rules rejected; review filter tuning", file=sys.stderr)
    return 0


def main(argv: Sequence[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="crucible-distiller")
    sub = parser.add_subparsers(dest="cmd", required=True)

    sc = sub.add_parser("selfcheck", help="adversarial-corpus catch-rate audit")
    sc.add_argument("--min-catch-rate", type=float, default=0.99,
                    help="minimum acceptable catch rate (default 0.99)")
    sc.set_defaults(fn=_cmd_selfcheck)

    daemon = sub.add_parser("daemon", help="run the queue consumer (production)")
    daemon.set_defaults(fn=lambda _: (print("daemon mode requires kafka or sqs extras", file=sys.stderr) or 64))

    process = sub.add_parser("process", help="process one offline corpus")
    process.add_argument("--corpus", required=True, help="path to a corpus JSON file")
    process.set_defaults(fn=lambda _: (print("offline-corpus mode wires into infra/oss-corpus-bootstrap; see that package", file=sys.stderr) or 64))

    args = parser.parse_args(argv)
    return int(args.fn(args) or 0)


if __name__ == "__main__":
    sys.exit(main())
