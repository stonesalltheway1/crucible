"""Stack detection — auto-classify a repo into one of the 12 stacks.

The detector reads package manifests + filesystem-shape signals; it
deliberately doesn't run any LLM-class inference (the result drives
global_defaults selection so it must be deterministic + fast).
"""

from __future__ import annotations

import json
import os
import re
from dataclasses import dataclass, field


@dataclass
class StackDetection:
    primary: str = ""
    secondary: list[str] = field(default_factory=list)
    versions: dict[str, str] = field(default_factory=dict)
    confidence: float = 0.0


_SIGNALS = {
    # filename → (stack id, base_confidence_bump)
    "next.config.js":            ("nextjs", 0.4),
    "next.config.ts":            ("nextjs", 0.4),
    "next.config.mjs":           ("nextjs", 0.4),
    "manage.py":                 ("django", 0.5),
    "pyproject.toml":            ("python_generic", 0.05),
    "fastapi.toml":              ("fastapi", 0.5),
    "Gemfile":                   ("rails", 0.3),
    "config.ru":                 ("rails", 0.2),
    "pom.xml":                   ("spring_boot", 0.25),
    "build.gradle":              ("spring_boot", 0.25),
    "go.mod":                    ("go_services", 0.4),
    "Cargo.toml":                ("rust_services", 0.4),
    "mix.exs":                   ("phoenix_elixir", 0.5),
    "vue.config.js":             ("vue", 0.4),
    "nuxt.config.ts":            ("vue", 0.4),
    "artisan":                   ("laravel", 0.5),
}


def detect(repo_root: str) -> StackDetection:
    """Walk the top-level repo + first-level subdirs and accumulate
    signal weights per stack id. Highest stack id with confidence >= 0.4
    becomes ``primary``; anything else above 0.2 becomes ``secondary``.
    """
    scores: dict[str, float] = {}
    versions: dict[str, str] = {}

    def _bump(stack: str, by: float) -> None:
        scores[stack] = scores.get(stack, 0.0) + by

    if not os.path.isdir(repo_root):
        return StackDetection()

    # Top-level files
    for name in os.listdir(repo_root):
        if name in _SIGNALS:
            stack, bump = _SIGNALS[name]
            _bump(stack, bump)

    # package.json is overloaded — peek to disambiguate
    pj_path = os.path.join(repo_root, "package.json")
    if os.path.isfile(pj_path):
        try:
            with open(pj_path, "r", encoding="utf-8") as f:
                pj = json.load(f)
            deps = {**pj.get("dependencies", {}), **pj.get("devDependencies", {})}
            if "next" in deps:
                _bump("nextjs", 0.45)
                versions["next"] = _parse_version(deps["next"])
            if "express" in deps:
                _bump("express", 0.45)
            if "vue" in deps:
                _bump("vue", 0.40)
        except (OSError, json.JSONDecodeError):
            pass

    # FastAPI signal lives in requirements / pyproject
    py_paths = [
        os.path.join(repo_root, "requirements.txt"),
        os.path.join(repo_root, "pyproject.toml"),
    ]
    for path in py_paths:
        if os.path.isfile(path):
            try:
                with open(path, "r", encoding="utf-8", errors="replace") as f:
                    body = f.read().lower()
            except OSError:
                continue
            if "fastapi" in body:
                _bump("fastapi", 0.45)
            if "django" in body:
                _bump("django", 0.45)
            if "flask" in body:
                _bump("flask", 0.45)

    # Rails signal — Gemfile content
    gemfile = os.path.join(repo_root, "Gemfile")
    if os.path.isfile(gemfile):
        try:
            with open(gemfile, "r", encoding="utf-8", errors="replace") as f:
                if "rails" in f.read().lower():
                    _bump("rails", 0.5)
        except OSError:
            pass

    if not scores:
        return StackDetection()

    # Resolve primary + secondary.
    sorted_stacks = sorted(scores.items(), key=lambda kv: kv[1], reverse=True)
    primary, primary_score = sorted_stacks[0]
    if primary == "python_generic":
        # python_generic by itself is too vague; demote.
        if len(sorted_stacks) > 1:
            primary, primary_score = sorted_stacks[1]
        else:
            return StackDetection()
    secondary = [s for s, sc in sorted_stacks[1:] if sc >= 0.2 and s != primary]
    return StackDetection(
        primary=primary,
        secondary=secondary,
        versions=versions,
        confidence=min(primary_score, 1.0),
    )


def _parse_version(spec: str) -> str:
    m = re.search(r"\d+(?:\.\d+){1,2}", spec)
    return m.group(0) if m else spec
