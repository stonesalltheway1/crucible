# regression / opus-46-loop-bug

Past incident: Opus 4.6 looped on a trivial prompt. Crucible's
Bounded Budget Enforcer + retry-cap state machine fired correctly,
but the loop chewed cache. The regression case asserts ≤ 5 model
calls.
