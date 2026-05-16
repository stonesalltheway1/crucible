"""Tier-A curated style-guide list + per-stack seed scaffolding.

The full Tier-A corpus (~40 style guides) is mined offline; this file
records the canonical pointers + per-stack seed *rules* the bootstrap
pipeline uses to assemble each bundle. The pipeline's deterministic
extractor walks the seeds verbatim into conventions; the LLM
extractor adds breadth from full document scans.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Mapping


@dataclass(frozen=True)
class StyleGuide:
    name: str
    stack_tags: tuple[str, ...]
    url: str
    license_spdx: str
    confidence_multiplier: float = 1.5  # Tier-A ×1.5 per the brief


TIER_A_STYLE_GUIDES: tuple[StyleGuide, ...] = (
    StyleGuide("Google TypeScript Style Guide", ("nextjs", "vue", "express"),
               "https://google.github.io/styleguide/tsguide.html", "Apache-2.0"),
    StyleGuide("Google JavaScript Style Guide", ("nextjs", "vue", "express"),
               "https://google.github.io/styleguide/jsguide.html", "Apache-2.0"),
    StyleGuide("Airbnb JavaScript", ("nextjs", "vue", "express"),
               "https://github.com/airbnb/javascript", "MIT"),
    StyleGuide("Airbnb React", ("nextjs",),
               "https://github.com/airbnb/javascript/tree/master/react", "MIT"),
    StyleGuide("Microsoft TypeScript Coding Guidelines", ("nextjs", "express"),
               "https://github.com/microsoft/TypeScript/wiki/Coding-guidelines", "Apache-2.0"),
    StyleGuide("PEP 8", ("django", "fastapi", "flask"),
               "https://peps.python.org/pep-0008/", "Public Domain"),
    StyleGuide("PEP 257", ("django", "fastapi", "flask"),
               "https://peps.python.org/pep-0257/", "Public Domain"),
    StyleGuide("PEP 484", ("django", "fastapi", "flask"),
               "https://peps.python.org/pep-0484/", "Public Domain"),
    StyleGuide("Effective Go", ("go_services",),
               "https://go.dev/doc/effective_go", "BSD-3-Clause"),
    StyleGuide("Uber Go Style Guide", ("go_services",),
               "https://github.com/uber-go/guide", "MIT"),
    StyleGuide("google/styleguide go", ("go_services",),
               "https://google.github.io/styleguide/go", "Apache-2.0"),
    StyleGuide("Rust API Guidelines", ("rust_services",),
               "https://rust-lang.github.io/api-guidelines/", "Apache-2.0"),
    StyleGuide("tokio style notes", ("rust_services",),
               "https://github.com/tokio-rs/tokio", "MIT"),
    StyleGuide("rubocop rails style", ("rails",),
               "https://github.com/rubocop/rails-style-guide", "MIT"),
    StyleGuide("rubocop ruby style", ("rails",),
               "https://github.com/rubocop/ruby-style-guide", "MIT"),
    StyleGuide("HackSoft Django Styleguide", ("django",),
               "https://github.com/HackSoftware/Django-Styleguide", "MIT"),
    StyleGuide("Django coding-style docs", ("django",),
               "https://docs.djangoproject.com/en/5.0/internals/contributing/writing-code/coding-style/", "BSD-3-Clause"),
    StyleGuide("Spring framework code quality", ("spring_boot",),
               "https://github.com/spring-projects/spring-framework/wiki/Code-Style", "Apache-2.0"),
    StyleGuide("spring-petclinic reference", ("spring_boot",),
               "https://github.com/spring-projects/spring-petclinic", "Apache-2.0"),
    StyleGuide("Elixir style guide", ("phoenix_elixir",),
               "https://github.com/christopheradams/elixir_style_guide", "MIT"),
    StyleGuide("Credo defaults", ("phoenix_elixir",),
               "https://github.com/rrrene/credo", "MIT"),
    StyleGuide("Swift API design guidelines", (),
               "https://www.swift.org/documentation/api-design-guidelines/", "Apache-2.0"),
    StyleGuide("FastAPI best practices", ("fastapi",),
               "https://github.com/zhanymkanov/fastapi-best-practices", "MIT"),
    StyleGuide("Pallets cookiecutter-flask", ("flask",),
               "https://github.com/cookiecutter-flask/cookiecutter-flask", "MIT"),
    StyleGuide("Vercel commerce reference", ("nextjs",),
               "https://github.com/vercel/commerce", "MIT"),
    StyleGuide("shadcn/ui", ("nextjs",),
               "https://github.com/shadcn-ui/ui", "MIT"),
    StyleGuide("react.dev hooks rules", ("nextjs",),
               "https://react.dev/reference/rules", "CC-BY-4.0"),
)


# Per-stack seed rules (raw text the extractor walks deterministically).
# These were derived offline from the Tier-A guides; deriving them at
# bootstrap time would require running the LLM extractor which is
# offline-only. Phase 5 ships the pre-derived set so the bundle build
# is reproducible. Each rule is paraphrased to neutral / categorical
# form so no source's verbatim expression is shipped.
STACK_SEEDS: Mapping[str, tuple[tuple[str, str, str], ...]] = {
    "nextjs": (
        ("Naming", "Component files use PascalCase; hooks use camelCase prefixed with `use`.", "**/*.tsx"),
        ("Layering", "Server-only modules import only from `lib/server/**`.", "app/**/*.{ts,tsx}"),
        ("LibraryPreferences", "Use `date-fns` for date handling; don't introduce `moment`.", "**/*.{ts,tsx,js}"),
        ("TestPatterns", "Tests colocate with source in `__tests__/`.", "**/*.{ts,tsx}"),
        ("ErrorHandling", "API routes return `Response.json({ error: { code, message } })` envelopes.", "app/api/**/*.ts"),
        ("Logging", "Use a structured logger (pino / winston); no console.log in production code.", "app/**/*.ts"),
        ("MigrationPatterns", "Database migrations are additive-only.", "db/migrations/**/*.{sql,ts}"),
        ("PrCommitHygiene", "Conventional Commits required.", "*"),
        ("SecurityDefaults", "Auth middleware precedes any route handler.", "app/**/*.ts"),
        ("PerformanceDefaults", "Use cursor pagination; avoid offset.", "app/api/**/*.ts"),
        ("Concurrency", "Pass an AbortSignal through every async chain.", "app/**/*.ts"),
        ("ApiShape", "Idempotency keys required on mutation endpoints.", "app/api/**/route.ts"),
    ),
    "fastapi": (
        ("Naming", "Pydantic models use PascalCase; instances use snake_case.", "**/*.py"),
        ("Layering", "Domain layer cannot import from `api/`.", "app/**/*.py"),
        ("LibraryPreferences", "Use `pydantic` v2 idioms; avoid raw dict response models.", "app/api/**/*.py"),
        ("TestPatterns", "Tests live in `tests/` mirroring the package layout.", "tests/**/*.py"),
        ("ErrorHandling", "Use `HTTPException` for 4xx; structured Result types internally.", "app/**/*.py"),
        ("Logging", "Structured logs via `structlog`; no print() in service code.", "app/**/*.py"),
        ("MigrationPatterns", "Alembic migrations are additive-only with `op.execute` guard rails.", "alembic/versions/**/*.py"),
        ("PrCommitHygiene", "Black-formatted; ruff-clean; Conventional Commits.", "*"),
        ("SecurityDefaults", "Authentication dependency precedes route handlers.", "app/api/**/*.py"),
        ("PerformanceDefaults", "Use `select_related`/`asyncio.gather` to avoid N+1.", "app/**/*.py"),
        ("Concurrency", "Pass `asyncio` contextvars through async chains.", "app/**/*.py"),
        ("ApiShape", "OpenAPI tags + response_model are mandatory on each route.", "app/api/**/*.py"),
    ),
    "django": (
        ("Naming", "Models use PascalCase; managers end in `Manager`.", "**/models.py"),
        ("Layering", "Models, services, selectors, apis layering (HackSoft).", "**/*.py"),
        ("LibraryPreferences", "Use `django-extensions` for ULID PKs.", "**/*.py"),
        ("TestPatterns", "Tests use `pytest-django` factories; no Django TestCase by default.", "tests/**/*.py"),
        ("ErrorHandling", "Service-layer raises domain exceptions; views translate to HTTP.", "**/services.py"),
        ("Logging", "Structured logs via `structlog`.", "**/*.py"),
        ("MigrationPatterns", "Migrations are additive-only and have a `RunPython.noop` reverse.", "**/migrations/**/*.py"),
        ("PrCommitHygiene", "Conventional Commits + isort + black.", "*"),
        ("SecurityDefaults", "Use Django's CSRF middleware on session views.", "**/*.py"),
        ("PerformanceDefaults", "Use `select_related`/`prefetch_related` on QuerySets.", "**/*.py"),
        ("Concurrency", "ASGI views must remain `async def` end-to-end.", "**/views.py"),
        ("ApiShape", "DRF errors use `{ detail, code }` envelope.", "**/serializers.py"),
    ),
    "flask": (
        ("Naming", "Blueprint names match the URL prefix.", "**/*.py"),
        ("Layering", "Blueprints don't import from each other.", "app/blueprints/**/*.py"),
        ("LibraryPreferences", "Use `Flask-SQLAlchemy`; avoid raw SQL outside repos.", "**/*.py"),
        ("TestPatterns", "Use `pytest` fixtures for app factory + db.", "tests/**/*.py"),
        ("ErrorHandling", "Use `errorhandler` per blueprint with structured envelopes.", "**/*.py"),
        ("Logging", "Configure `dictConfig` once at startup; no print().", "app/__init__.py"),
        ("MigrationPatterns", "Alembic migrations are additive-only.", "migrations/**/*.py"),
        ("PrCommitHygiene", "Conventional Commits.", "*"),
        ("SecurityDefaults", "Auth via `before_request` blueprint hook.", "app/blueprints/**/*.py"),
        ("PerformanceDefaults", "Bulk queries via `db.session.scalars().all()`.", "**/*.py"),
        ("Concurrency", "Don't mix sync and async views without `asgiref`.", "**/*.py"),
        ("ApiShape", "JSON envelope: `{ data | error }`.", "**/*.py"),
    ),
    "rails": (
        ("Naming", "Models PascalCase; tables plural snake_case.", "app/models/**/*.rb"),
        ("Layering", "Service objects under `app/services/`; controllers stay thin.", "app/**/*.rb"),
        ("LibraryPreferences", "Use `dry-monads` for Result; no `rescue StandardError` for control flow.", "app/**/*.rb"),
        ("TestPatterns", "Use rspec; avoid Minitest unless legacy.", "spec/**/*.rb"),
        ("ErrorHandling", "Errors derive from `ApplicationError`; controllers `rescue_from`.", "app/controllers/**/*.rb"),
        ("Logging", "Tagged logging with `Rails.logger.tagged`.", "app/**/*.rb"),
        ("MigrationPatterns", "Migrations are reversible.", "db/migrate/**/*.rb"),
        ("PrCommitHygiene", "Conventional Commits.", "*"),
        ("SecurityDefaults", "Use Strong Params on controllers.", "app/controllers/**/*.rb"),
        ("PerformanceDefaults", "Cursor-paginated lists via `pagy`.", "app/controllers/**/*.rb"),
        ("Concurrency", "Use Sidekiq for background jobs; no in-process threads.", "app/jobs/**/*.rb"),
        ("ApiShape", "JSON:API or `{ data, errors }` envelope.", "app/controllers/api/**/*.rb"),
    ),
    "spring_boot": (
        ("Naming", "Service classes end in `Service`; configs in `Config`.", "**/*.java"),
        ("Layering", "Domain classes don't import from `web/` or `api/`.", "src/main/java/**/*.java"),
        ("LibraryPreferences", "Use `slf4j-api`; never `System.out.println` in production code.", "**/*.java"),
        ("TestPatterns", "JUnit 5 + `@SpringBootTest` for integration; unit tests free of Spring.", "src/test/java/**/*.java"),
        ("ErrorHandling", "Use typed exceptions + `@ControllerAdvice` envelopes.", "**/*.java"),
        ("Logging", "Use `Logger logger = LoggerFactory.getLogger(...)`; structured fields.", "**/*.java"),
        ("MigrationPatterns", "Flyway migrations are additive-only.", "src/main/resources/db/migration/**/*.sql"),
        ("PrCommitHygiene", "Conventional Commits + Google Java Format.", "*"),
        ("SecurityDefaults", "Use Spring Security; never custom auth filters.", "**/*.java"),
        ("PerformanceDefaults", "Use JPA fetch-joins to avoid N+1.", "**/*.java"),
        ("Concurrency", "Use `@Async` only with explicit `Executor`; no orphan thread starts.", "**/*.java"),
        ("ApiShape", "OpenAPI annotations on every controller method.", "**/*.java"),
    ),
    "go_services": (
        ("Naming", "Test files end in `_test.go`; benchmarks in `_bench_test.go`.", "**/*.go"),
        ("Layering", "Internal packages cannot import from `cmd/`.", "**/*.go"),
        ("LibraryPreferences", "Use `slog` for logging; never `log.Println` in service code.", "**/*.go"),
        ("TestPatterns", "Table tests with `t.Run`; subtests for fixtures.", "**/*_test.go"),
        ("ErrorHandling", "Wrap errors with `%w`; never panic in library code.", "**/*.go"),
        ("Logging", "Structured `slog` calls with key-value pairs.", "**/*.go"),
        ("MigrationPatterns", "Use Goose / sql-migrate; additive-only migrations.", "db/migrations/**/*.sql"),
        ("PrCommitHygiene", "Conventional Commits + gofmt.", "*"),
        ("SecurityDefaults", "Auth middleware wraps every handler.", "internal/api/**/*.go"),
        ("PerformanceDefaults", "Use cursor pagination via opaque tokens.", "internal/api/**/*.go"),
        ("Concurrency", "Pass `context.Context` as the first argument through every async chain.", "**/*.go"),
        ("ApiShape", "Error envelope: `{ error: { code, message } }`.", "internal/api/**/*.go"),
    ),
    "rust_services": (
        ("Naming", "Module names snake_case; types CamelCase.", "**/*.rs"),
        ("Layering", "Application crate cannot import from binary crates.", "**/*.rs"),
        ("LibraryPreferences", "Use `tracing` for logs; never `println!` in service code.", "**/*.rs"),
        ("TestPatterns", "Unit tests in `#[cfg(test)] mod tests`; integration tests in `tests/`.", "**/*.rs"),
        ("ErrorHandling", "Use `thiserror` for libraries; `anyhow` for binaries.", "**/*.rs"),
        ("Logging", "`tracing::info!`/`warn!`/`error!` with structured fields.", "**/*.rs"),
        ("MigrationPatterns", "Use `sqlx::migrate!`; never edit applied migrations.", "migrations/**/*.sql"),
        ("PrCommitHygiene", "Conventional Commits + rustfmt + clippy.", "*"),
        ("SecurityDefaults", "Don't `unsafe` outside FFI boundary modules.", "**/*.rs"),
        ("PerformanceDefaults", "Avoid `clone()` in hot loops; prefer borrowed forms.", "**/*.rs"),
        ("Concurrency", "Pass `tokio::CancellationToken` through async chains.", "**/*.rs"),
        ("ApiShape", "axum handlers return `Result<Json<T>, ApiError>` envelopes.", "src/api/**/*.rs"),
    ),
    "phoenix_elixir": (
        ("Naming", "Modules use `MyApp.SubModule`; functions snake_case.", "**/*.ex"),
        ("Layering", "Contexts hide schemas from controllers.", "lib/**/*.ex"),
        ("LibraryPreferences", "Use `Ecto.Multi` for transactional writes.", "lib/**/*.ex"),
        ("TestPatterns", "Use ExUnit with `async: true` where state allows.", "test/**/*.exs"),
        ("ErrorHandling", "Return `{:ok, _} | {:error, _}` tuples; raise only in tests.", "lib/**/*.ex"),
        ("Logging", "Structured logs via `Logger.metadata`.", "lib/**/*.ex"),
        ("MigrationPatterns", "Ecto migrations are additive-only.", "priv/repo/migrations/**/*.exs"),
        ("PrCommitHygiene", "Conventional Commits + mix format.", "*"),
        ("SecurityDefaults", "Pipelines apply `:fetch_session` + auth plug.", "lib/**/*.ex"),
        ("PerformanceDefaults", "Preload associations via `Repo.preload`.", "lib/**/*.ex"),
        ("Concurrency", "Use `Task.Supervisor` for fan-out; never raw `spawn`.", "lib/**/*.ex"),
        ("ApiShape", "JSON envelope `{data, errors}`.", "lib/**_web/controllers/**/*.ex"),
    ),
    "vue": (
        ("Naming", "Components in PascalCase; composables prefixed `use`.", "src/**/*.{vue,ts}"),
        ("Layering", "Stores under `src/stores/`; pages do not import store internals.", "src/**/*.ts"),
        ("LibraryPreferences", "Use `pinia` for state; avoid Vuex in new code.", "src/**/*.ts"),
        ("TestPatterns", "Vitest + `@testing-library/vue` for component tests.", "src/**/__tests__/**/*"),
        ("ErrorHandling", "Reject promises with typed errors; surfaces via `useError()` composable.", "src/**/*.ts"),
        ("Logging", "Use `consola` for client logs; structured.", "src/**/*.ts"),
        ("MigrationPatterns", "Schema changes additive-only.", "db/migrations/**/*"),
        ("PrCommitHygiene", "Conventional Commits.", "*"),
        ("SecurityDefaults", "Sanitize HTML via `DOMPurify` before v-html.", "src/**/*.vue"),
        ("PerformanceDefaults", "Use `defineAsyncComponent` for route-level code-splitting.", "src/**/*.ts"),
        ("Concurrency", "Cancel inflight requests via AbortController.", "src/**/*.ts"),
        ("ApiShape", "API errors envelope `{ error: { code, message } }`.", "src/**/*.ts"),
    ),
    "express": (
        ("Naming", "Routers grouped per resource under `src/routes/`.", "src/**/*.{ts,js}"),
        ("Layering", "Controllers don't access DB; service layer does.", "src/**/*.{ts,js}"),
        ("LibraryPreferences", "Use `zod` for input validation.", "src/**/*.{ts,js}"),
        ("TestPatterns", "Vitest or jest; isolated supertest for HTTP.", "src/**/__tests__/**/*"),
        ("ErrorHandling", "Centralized error middleware emits `{ error: { code, message } }`.", "src/**/*.{ts,js}"),
        ("Logging", "Use `pino` with `pino-http`.", "src/**/*.{ts,js}"),
        ("MigrationPatterns", "Migrations are additive-only.", "db/migrations/**/*"),
        ("PrCommitHygiene", "Conventional Commits.", "*"),
        ("SecurityDefaults", "Use `helmet` + `express-rate-limit` on every router.", "src/**/*.{ts,js}"),
        ("PerformanceDefaults", "Cursor pagination on list endpoints.", "src/**/*.{ts,js}"),
        ("Concurrency", "Use `Promise.allSettled` for fan-out.", "src/**/*.{ts,js}"),
        ("ApiShape", "Standard error envelope.", "src/**/*.{ts,js}"),
    ),
    "laravel": (
        ("Naming", "Controllers single-action with `__invoke`.", "app/Http/Controllers/**/*.php"),
        ("Layering", "Services under `app/Services/`; controllers stay thin.", "app/**/*.php"),
        ("LibraryPreferences", "Use Eloquent over query builder for typed scopes.", "app/**/*.php"),
        ("TestPatterns", "Use Pest; feature tests under `tests/Feature/`.", "tests/**/*.php"),
        ("ErrorHandling", "Throw domain exceptions; `Handler::render` envelopes.", "app/**/*.php"),
        ("Logging", "Use `Log::info` with context arrays.", "app/**/*.php"),
        ("MigrationPatterns", "Migrations additive-only; no down-only ops.", "database/migrations/**/*.php"),
        ("PrCommitHygiene", "Conventional Commits.", "*"),
        ("SecurityDefaults", "Auth middleware on every route group.", "routes/**/*.php"),
        ("PerformanceDefaults", "Eager load with `with()`.", "app/**/*.php"),
        ("Concurrency", "Queue jobs via Horizon; no inline `dispatch_sync` in hot paths.", "app/Jobs/**/*.php"),
        ("ApiShape", "JSON:API or `{ data, errors }` envelope.", "app/Http/Controllers/Api/**/*.php"),
    ),
}


def stacks_with_seeds() -> tuple[str, ...]:
    return tuple(STACK_SEEDS.keys())
