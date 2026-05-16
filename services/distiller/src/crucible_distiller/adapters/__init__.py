"""Source-channel adapters.

Each adapter maps an upstream system into a stream of ExtractionInput
items the distiller pool consumes. The Adapter protocol is intentionally
narrow so swapping vendors (Rootly vs FireHydrant for incidents,
Confluence vs Notion for runbooks) only adds an adapter, not a code
fork.
"""

from .base import Adapter, AdapterError, RateLimitedError
from .github_pr import GitHubPRAdapter
from .github_squash import GitHubSquashAdapter
from .incident import IncidentAdapter
from .slack_incidents import SlackIncidentsAdapter
from .confluence import ConfluenceAdapter
from .notion import NotionAdapter
from .adr_file import ADRFileAdapter
from .lint_config import LintConfigAdapter

__all__ = [
    "Adapter",
    "AdapterError",
    "RateLimitedError",
    "GitHubPRAdapter",
    "GitHubSquashAdapter",
    "IncidentAdapter",
    "SlackIncidentsAdapter",
    "ConfluenceAdapter",
    "NotionAdapter",
    "ADRFileAdapter",
    "LintConfigAdapter",
]
