# crucible-sdk (Python)

Crucible Agent SDK for Python. Phase 1 ships types only; the runtime `twin.*` surface ships with Phase 2 (see `docs/PHASE-1-REPORT.md`).

## Install (development)

```bash
nix develop .#python-only
cd libs/sdk-py
pip install -e .[dev]
pytest
```

## Types

All `twin.*` types are exported from the top-level package:

```python
from crucible_sdk import Plan, Task, Budget, Predicates

plan = Plan.model_validate_json(open("plan.json").read())
print(plan.estimated_cost_usd)
```
