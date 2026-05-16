# fixtures/

Placeholder. The vitest suite synthesises its own minimal fixtures inline (see
`test/tier0.test.ts` and `test/tier1.test.ts`) — pulling a full toy project on
disk would balloon the runner image with no leverage at this stage of Phase 4.

When end-to-end fixtures are added in Phase 5 (the dispatcher-integration
suite owns that), they will land here as `fixtures/strong-suite/` and
`fixtures/weak-suite/` mini-projects with their own `package.json` and a
single mutable function in `src/`.
