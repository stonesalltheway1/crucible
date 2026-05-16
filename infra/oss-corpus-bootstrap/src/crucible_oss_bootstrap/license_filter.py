"""License filter — drops GPL/AGPL/SSPL/BUSL inputs at ingestion.

Per the Phase-5 brief's guardrail: "Do NOT include GPL/AGPL/SSPL/BUSL
inputs in the OSS-corpus bootstrap. License-filter at ingestion."
"""

from __future__ import annotations


ALLOWED_LICENSES: tuple[str, ...] = (
    "MIT", "MIT-0",
    "Apache-2.0", "Apache 2.0",
    "BSD-2-Clause", "BSD-3-Clause", "BSD-4-Clause",
    "BSD", "BSD-Original",
    "MPL-2.0", "MPL 2.0",
    "ISC",
    "Unlicense",
    "CC0-1.0", "CC0",
)


BLOCKED_LICENSES: tuple[str, ...] = (
    "GPL-2.0", "GPL-3.0", "GPL",
    "GPL-2.0-or-later", "GPL-3.0-or-later",
    "GPL-2.0-only", "GPL-3.0-only",
    "LGPL-2.1", "LGPL-3.0", "LGPL",
    "AGPL-3.0", "AGPL",
    "SSPL-1.0", "SSPL",
    "BUSL-1.1", "BUSL",
    "Elastic-2.0", "Elastic",
    "Confluent Community License",
)


def license_safe(license_spdx: str | None) -> bool:
    """Return True iff the license is in the allowlist."""
    if not license_spdx:
        return False
    s = license_spdx.strip()
    for blocked in BLOCKED_LICENSES:
        if blocked.lower() in s.lower():
            return False
    for allowed in ALLOWED_LICENSES:
        if allowed.lower() == s.lower():
            return True
    # Conservative default: refuse what we can't classify.
    return False
