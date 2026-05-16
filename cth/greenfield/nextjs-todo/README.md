# greenfield / nextjs-todo

Brand-new Next.js 15 + App Router + Prisma SQLite + shadcn project the
agent builds end-to-end. Exercises the full path: scaffolding, schema,
routing, UI, tests.

**What this case probes:**
- The agent can build a real app from prompt without overrunning the
  budget.
- The verifier runs Vitest in the twin and confirms a clean test suite.
- The agent correctly picks Tailwind + 2px-corners brand voice if no
  customer override exists (Crucible's anti-vibe brand cue applies to
  generated UIs too).

**Why it's a good first case:**
Greenfield is the easiest end-to-end check; if this fails, nothing
else works.
