## Context

The completed hexagonal API rewrite made `cmd/api` the sole production composition root and preserved `net/http`, `embed`, and server-rendered HTML as a future inbound adapter. The current server exposes an authenticated JSON API under `/v1`; the CLI is a REST client. There is no browser UI or frontend toolchain.

Monthly review currently uses an external Python script that invokes the CLI, fetches aggregate budget reports and transaction lists separately, reconstructs category-to-line mappings, and writes a static HTML file. It demonstrates useful product behavior—combined personal and household summaries, line-level transaction drill-down, and unmapped spending—but duplicates reporting rules, hard-codes owner IDs, can observe inconsistent snapshots, and must be regenerated manually.

The deployment is private behind Traefik. The initial browser interface is intentionally a full-access administrative UI protected by Traefik BasicAuth, not an application-authenticated multi-user product. The JSON API retains its independent bearer-key boundary.

## Goals / Non-Goals

**Goals:**

- Serve a useful, responsive monthly finance dashboard from the existing Go process and binary.
- Preserve inward dependency direction by making HTML another inbound adapter over application services.
- Provide authoritative line-level transaction detail without UI-side mapping or multiple inconsistent report reads.
- Establish reusable page, component, styling, asset, and testing conventions for later UI sections.
- Keep the browser experience functional without JavaScript and keep the production runtime free of Node.
- Make the UI/API Traefik routing and temporary full-admin trust model explicit.

**Non-Goals:**

- Application-managed login, sessions, users-to-views authorization, roles, or CSRF-protected writes.
- Any create, update, delete, restore, ensure-budget, or other browser mutation.
- Charts, HTMX, a JavaScript framework, a JavaScript bundler, or browser calls to `/v1`.
- Multi-currency storage or conversion; this UI presents values as CAD.
- Replacing the existing REST/CLI aggregate budget report contract.
- Building a general-purpose design system or adopting a third-party UI component framework.

## Decisions

### 1. HTML is a same-process inbound adapter

The UI lives in a new `internal/webui` adapter family and calls narrow application-service interfaces directly. It does not call `/v1`, import PostgreSQL/sqlc packages, or reconstruct reporting rules. `internal/server` composes the web handler and existing API handler into one top-level `http.ServeMux`:

```text
Browser -> webui handlers -> application services -> ports -> Postgres/sqlc
CLI     -> REST client    -> /v1 handlers        -> application services
```

Human routes are root-level (`/` initially; `/transactions`, `/budgets`, and similar paths are reserved for later pages). `/assets/`, `/v1/`, and `/live` remain reserved infrastructure namespaces. The initial `GET /` is the monthly dashboard.

A separate sidecar or SPA would add an internal HTTP hop, browser credential problems, and a second runtime without providing value for conventional read pages and forms.

### 2. Detailed reporting belongs to the budget application feature

The budget application service gains a distinct detailed monthly query rather than making the web adapter combine `Report` and `transactions.List`. The repository loads the selected monthly budget, aggregate report lines, line/category mappings, mapped transaction detail, and unmapped transaction detail inside one repeatable-read, read-only transaction.

```text
DetailedMonthlyReport(owner, year, month)
  -> DetailedReport
       -> lines[]
            -> aggregate amounts
            -> categories[]
            -> transactions[]
       -> unmappedTransactions[]
       -> totals
```

Every in-scope transaction appears exactly once: under its mapped budget line or in the unmapped collection. Detailed transaction values include ID, date, decimal amount, description, notes, category, and author identity useful for display. Amounts remain decimal strings in the budget reporting model so the UI does not inherit the transaction API's `float32` representation.

The existing aggregate `Report(budgetID)` and its REST/CLI wire contract remain unchanged. The detailed query is initially consumed only by the web adapter. This avoids bloating existing machine responses while locating owner-scope and category mapping semantics in the authoritative reporting feature.

An `internal/app/dashboard` package is not introduced. Combining one personal and one household report is presently page composition, not an independent domain capability.

### 3. The web adapter owns page assembly and view models

The dashboard handler depends on narrow budget, user, and household reader interfaces. It resolves request state, invokes the two detailed monthly reports, and maps application models into UI-owned view models before rendering.

```text
application models -> dashboard assembler -> view models -> templ components
```

View models carry display-ready CAD values, canonical navigation URLs, percentages, and semantic states such as normal, warning, and danger. They do not carry arbitrary Tailwind class strings. Components map semantic state to appearance. Arithmetic and money parsing do not occur in templates.

Expected budget-not-found results become scope-specific empty states. If both budgets are absent, the page is still a successful empty dashboard. Invalid request values produce a safe `400`, absent selected owners produce `404`, and unexpected query failures fail the complete page with a safe `500`; partial totals are not rendered after an unexpected failure.

### 4. Month and owner state are canonical URL state

The dashboard accepts `month=YYYY-MM`, `userId`, and `householdId` query parameters. Missing owner parameters use positive server-configured default IDs. Select controls list available users and households and submit a GET form, making selections bookmarkable and preserving browser navigation. Explicit invalid or absent owners never silently fall back.

A request without `month` redirects to the canonical current month URL. “Current month” uses `time.Local`, configured through the standard `TZ` process environment (initially `America/Toronto`); the runtime must include or embed IANA timezone data. Previous and next links use calendar arithmetic and do not require period discovery. Navigating to a month without a budget is read-only and never invokes ensure/create behavior.

### 5. Headline totals include unmapped spending

For each scope and for the combined summary:

```text
total spent = mapped actual + unmapped actual
effective remaining = allocation - total spent
```

`UncategorizedActualAmount` is a subset of unmapped spending and is not added a second time. Budget-line rows retain mapped-only actuals. Scope summaries display unmapped amounts explicitly so the difference between headline totals and line sums is explainable.

The page places one combined full-width summary first, personal and household summary cards side-by-side on wide screens, and full-width detailed reports below. Narrow layouts stack all sections. Line transactions are rendered eagerly in native `<details>/<summary>` disclosure controls; no HTMX or custom JavaScript is required.

### 6. templ components and Tailwind establish the UI foundation

Human-authored `*.templ` files are authoritative, and generated `*_templ.go` files are committed so ordinary Go builds and tests do not require the templ generator. CI runs `templ generate` and fails on a generated diff. Generated files are excluded from development watch triggers to avoid loops.

Components are layered without a third-party framework:

```text
shell -> primitives -> reusable patterns -> dashboard feature -> page
```

Initial primitives include cards, buttons/links, badges, selects, progress indicators, alerts, and empty/error states. Patterns include metric cards, month navigation, owner selection, budget lines, and transaction lists. Components use semantic HTML, keyboard-visible focus, sufficient contrast, touch-sized controls, and text/icon cues in addition to color.

Tailwind is a build-time dependency pinned through `package.json` and `package-lock.json` using the Tailwind CLI. It scans templ and relevant Go/JS sources and emits production CSS. The Dracula-inspired dark theme uses semantic CSS tokens for background, surface, foreground, muted, accent, positive, warning, and danger roles; vivid purple, pink, cyan, green, orange, and red are restrained to meaningful emphasis. No compiled CSS, `node_modules`, or minified JavaScript is committed.

No JavaScript bundler is introduced. If later behavior needs authored JavaScript, same-origin native ES modules are the default. A future npm browser dependency such as Chart.js can justify adding esbuild then.

### 7. Static assets are embedded and same-origin

Compiled CSS and any source-controlled icons are embedded into the Go binary and served under `/assets/` with correct content types and conservative cache headers. The UI does not depend on a CDN. The production Docker build uses Node only in a build stage to produce CSS and copies only the final Go binary into the runtime image.

### 8. UI and API use distinct Traefik routers

The same container and port back two production hostnames:

```text
finance.homelab.voltr.org      -> all human routes, Traefik BasicAuth
finance-api.homelab.voltr.org  -> /v1 only, application bearer auth
```

The API router is path-restricted so root UI routes cannot bypass BasicAuth through the API hostname. `/live` remains available to the container-local health check and need not be routed publicly. BasicAuth users are supplied as deployment secret configuration and are never committed. Direct access to the application port remains prohibited by production network/port configuration.

This is explicitly installation-level full-admin protection. The application does not infer finance identity from BasicAuth; configured/selected user IDs remain report selectors. Deployment documentation records that future browser writes require application identity/audit decisions and CSRF defenses.

### 9. CAD is the initial presentation currency

The UI formats monetary values with `en-CA`/CAD semantics and consistently identifies the dashboard currency. This is presentation only: no currency column, conversion, or mixed-currency claim is introduced. Centralized money formatting in the view-model mapper leaves a future currency model possible without changing every component.

## Risks / Trade-offs

- [Traefik is the only UI access control] -> Restrict production routing to the authenticated UI hostname, do not publish the container port, keep API routing path-limited, and document the full-admin trust model.
- [BasicAuth is inadequate for future per-user authorization and audit attribution] -> Keep all UI operations read-only and require a separate auth/write design before adding mutations.
- [Eager transaction detail increases HTML size] -> Accept the bounded monthly payload for a private dashboard; preserve a separate aggregate report and revisit lazy fragments only with measured need.
- [A detailed report query adds SQL and model complexity] -> Reuse existing owner-scope predicates and test exact-once mapped/unmapped classification under one repeatable-read snapshot.
- [Committed templ output can become stale] -> Verify generation in CI and include generation in the documented development command.
- [Adding Node increases build tooling] -> Keep Node build-only, pin dependencies with a lockfile, and copy no Node runtime artifacts into the final image.
- [A dark colorful theme can reduce readability] -> Use semantic tokens, restrained accents, contrast tests, visible text states, and responsive component tests.
- [A fixed CAD assumption can misrepresent future mixed-currency data] -> Label the assumption, centralize formatting, and treat multi-currency as a future domain change rather than silently guessing.

## Migration Plan

1. Add detailed budget-report models, queries, repository behavior, application service behavior, and characterization tests without changing existing aggregate contracts.
2. Add templ and Tailwind build metadata, generation verification, embedded assets, and reusable web components.
3. Add dashboard configuration, assembler, handler, error pages, routing, and handler/render tests.
4. Extend server composition while verifying `/v1` bearer authentication and `/live` behavior remain unchanged.
5. Update development and production Docker builds, Compose environment, timezone data, and deployment documentation.
6. Configure DNS and Traefik routers for `finance.homelab.voltr.org` and `finance-api.homelab.voltr.org`; provide BasicAuth credentials out of band.
7. Update first-party CLI configuration if the API hostname changes, deploy, and verify container-local health, authenticated UI access, unauthenticated UI rejection, and bearer-authenticated API access.

Rollback deploys the preceding image and restores the prior Traefik API router/hostname. There is no schema or persisted-data migration to reverse.

## Open Questions

None. Exact spacing, typography, and decorative treatment can be refined during implementation within the specified semantic theme and responsive behavior.
