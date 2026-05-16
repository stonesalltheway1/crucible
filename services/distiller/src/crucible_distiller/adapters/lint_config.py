"""Lint-config adapter — deterministic, zero-cost.

Per docs/06-research/memory-bootstrap.md §"Tier B", ~30% of conventions
fall out of lint configs alone (no LLM cost). The adapter parses the
common config files we recognise and emits one ExtractionInput per
recognised pattern. The distiller pool passes these through the
`deterministic_extract` extractor — no LLM call.

Recognised files (Phase 5 minimum set):
  .editorconfig
  .prettierrc                .prettierrc.json   .prettierrc.yaml
  .eslintrc.{js,json,yaml}
  tsconfig.json
  pyproject.toml             (tool.ruff, tool.black, tool.isort)
  .rubocop.yml
  rustfmt.toml               clippy.toml
  .golangci.yml
  commitlint.config.{js,cjs}
"""

from __future__ import annotations

import json
import os
import re
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Iterable

from ..types import ConventionCategory, ExtractionInput, ScopeFilter, SourceChannel, SourceRef


_RECOGNISED = {
    ".editorconfig", ".prettierrc", ".prettierrc.json", ".prettierrc.yaml",
    ".eslintrc.js", ".eslintrc.cjs", ".eslintrc.json", ".eslintrc.yaml",
    "tsconfig.json", "pyproject.toml",
    ".rubocop.yml", "rustfmt.toml", "clippy.toml",
    ".golangci.yml", "commitlint.config.js", "commitlint.config.cjs",
    ".markdownlint.json", "renovate.json", ".gitleaks.toml",
}


@dataclass
class LintConfigAdapter:
    """Adapter producing per-file ExtractionInput items.

    Each item's raw_text is the raw config file body; the
    deterministic_extract function inside the extractor handles the
    actual rule extraction.
    """

    repo_root: str
    repo: str = ""

    def name(self) -> str:
        return f"lint_config:{self.repo}"

    def iter_items(self, *, tenant_id: str, cursor: str = "") -> Iterable[ExtractionInput]:
        if not os.path.isdir(self.repo_root):
            return
        for entry in os.listdir(self.repo_root):
            if entry not in _RECOGNISED:
                continue
            full = os.path.join(self.repo_root, entry)
            try:
                with open(full, "r", encoding="utf-8", errors="replace") as f:
                    body = f.read()
            except OSError:
                continue
            yield ExtractionInput(
                tenant_id=tenant_id,
                repo=self.repo,
                source_channel=SourceChannel.LINT_CONFIG,
                source=SourceRef(kind="adr", path=entry, commit=""),
                raw_text=body,
                extracted_at=datetime.now(timezone.utc),
            )
