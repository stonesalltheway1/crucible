# Crucible — JetBrains plugin

Evidence-driven AI engineering inside IntelliJ IDEA, GoLand, PyCharm,
WebStorm, RubyMine, Rider, etc. Targets IntelliJ Platform 2024.3+
(since-build 243).

Per **ADR-011**, this plugin is a thin wrapper around the Crucible
control plane API. We do not replicate the IDE's chat, completion, or
multi-file rewrite UI.

## Surfaces

- **Tool window** (right rail, Junie-style): tabs for Tasks, Attestations, Plan
- **Status bar widget**: "Crucible · $0.42 / $150 (0.3%)" daily spend vs cap
- **Tools menu**: New Task, Approve Plan, Halt, Open Web Console
- **Settings**: API endpoint, bearer token, tenant id

## Build

```bash
./gradlew buildPlugin
# → build/distributions/crucible-jetbrains-2026.06.0-phase7.zip
```

Press *Plugins → Install Plugin from Disk* in the IDE to sideload.

## Tests

```bash
./gradlew test
```

## Publish

CI publishes to JetBrains Marketplace on tagged releases via
`./gradlew publishPlugin`.
