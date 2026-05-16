"""Generate an inferred AGENTS.md proposal from cartographer output.

Customer reviews it in the web console; edits persist as repo_overrides.
The brief mandates: "Generate inferred AGENTS.md if one doesn't exist."
"""

from __future__ import annotations

from typing import Iterable

from .stack_detect import StackDetection


def infer_agents_md_markdown(stack: StackDetection, candidates: Iterable) -> str:
    """Compose a Markdown AGENTS.md draft from extracted candidates.

    Output structure follows the canonical AGENTS.md pattern (per
    docs/06-research/memory-bootstrap.md §"AGENTS.md ecosystem"):

      # AGENTS.md (inferred by Crucible)
      ## Stack
      ## Conventions
        ### {12 taxonomy sections}
    """
    out: list[str] = []
    out.append("# AGENTS.md")
    out.append("")
    out.append("> Inferred by Crucible's cartographer at onboarding. Review + edit; saved edits override defaults.")
    out.append("")
    if stack.primary:
        out.append(f"## Stack")
        sv = f"**Primary:** {stack.primary}"
        if stack.versions:
            v = ", ".join(f"{k}={v}" for k, v in stack.versions.items())
            sv += f" ({v})"
        out.append(sv)
        if stack.secondary:
            out.append(f"**Secondary:** {', '.join(stack.secondary)}")
        out.append("")

    # Group by category.
    by_cat: dict[str, list[str]] = {}
    for c in candidates:
        cat = getattr(c.category, "value", c.category)
        by_cat.setdefault(cat, []).append(c.rule_nl.strip())

    out.append("## Conventions")
    out.append("")
    for cat, rules in sorted(by_cat.items()):
        out.append(f"### {cat}")
        for r in rules[:25]:
            out.append(f"- {r}")
        out.append("")

    if not by_cat:
        out.append("_(No conventions extracted yet. Crucible will continue learning from your PR review activity.)_")
        out.append("")
    return "\n".join(out)
