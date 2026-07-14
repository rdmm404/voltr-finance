## 1. Detailed Budget Reporting Models and Service

- [x] 1.1 Add detailed budget-report application models for line-associated transactions, unmapped transactions, and exact decimal amounts
- [x] 1.2 Extend the budget repository port with one detailed monthly snapshot operation keyed by owner and month
- [x] 1.3 Implement detailed-report service mapping, normalization, validation, and not-found/error behavior without changing the existing aggregate report
- [x] 1.4 Add budget service tests for mapped, unmapped, uncategorized, out-of-scope, empty, and aggregate-equivalence behavior

## 2. Detailed PostgreSQL Reporting Adapter

- [x] 2.1 Add sqlc queries that load mapped transaction details under the existing exact personal and household scope rules
- [x] 2.2 Map detailed transaction rows to application-owned decimal, category, notes, and author fields
- [x] 2.3 Implement the detailed monthly repository operation in one repeatable-read, read-only transaction
- [x] 2.4 Ensure mapped and unmapped classification is exact-once and deterministic within each budget line
- [x] 2.5 Add PostgreSQL adapter/integration tests for scope boundaries, deleted transactions, category mappings, transaction detail, and snapshot consistency
- [x] 2.6 Regenerate sqlc output and verify existing aggregate budget-report tests remain unchanged

## 3. templ and Tailwind Build Foundation

- [x] 3.1 Add a pinned templ generator tool and document the generation command
- [x] 3.2 Add reproducible tooling that downloads an exact standalone Tailwind CLI version and verifies platform-specific SHA-256 checksums
- [x] 3.3 Add the Tailwind input stylesheet, source scanning configuration, production CSS command, and Dracula-inspired semantic color tokens
- [x] 3.4 Add development commands that coordinate templ generation, Tailwind watching, and Go reload without watching generated templ files
- [x] 3.5 Add CI verification that templ generation is clean and frontend assets build reproducibly
- [x] 3.6 Configure generated CSS and downloaded Tailwind executables according to repository source-control and ignore conventions

## 4. Embedded Web UI Foundation

- [x] 4.1 Create the `internal/webui` package structure for routing, dashboard assembly, view models, components, pages, and assets
- [x] 4.2 Embed and serve same-origin assets under `/assets/` with correct content types and cache behavior
- [x] 4.3 Build the shared document shell, metadata, responsive container, and accessible navigation foundation
- [x] 4.4 Build reusable templ primitives for cards, controls, badges, progress, alerts, and empty/error states
- [x] 4.5 Build reusable patterns for metric cards, month navigation, owner selectors, budget-line disclosures, and transaction lists
- [x] 4.6 Generate and commit `*_templ.go` output for the shared components and add component render tests
- [x] 4.7 Verify focus states, semantic disclosure behavior, non-color status cues, contrast, and touch target sizing

## 5. Dashboard Request and View-Model Assembly

- [x] 5.1 Add UI configuration for positive default user and household IDs and validate it at startup
- [x] 5.2 Parse and validate canonical `month`, `userId`, and `householdId` query state with explicit-override semantics
- [x] 5.3 Use `time.Local`/`TZ` to redirect missing months and generate previous/next calendar URLs
- [x] 5.4 Add narrow web-facing interfaces for detailed budgets, users, and households without importing persistence or REST wire contracts
- [x] 5.5 Implement the dashboard assembler that loads owner selectors and both detailed monthly reports while treating budget-not-found as an empty scope
- [x] 5.6 Map application values to UI-owned CAD view models with centralized `en-CA` money and date formatting
- [x] 5.7 Calculate scope and combined spent as mapped plus unmapped actual and calculate effective remaining without double-counting uncategorized spending
- [x] 5.8 Map progress and variance values to semantic normal, warning, and danger states without embedding Tailwind classes in view models
- [x] 5.9 Fail the complete page on unexpected report failures and map invalid inputs, missing owners, and internal errors to safe status pages
- [x] 5.10 Add unit tests for canonical URLs, timezone month boundaries, overrides, missing budgets, CAD values, combined totals, and all error mappings

## 6. Responsive Monthly Dashboard Page

- [x] 6.1 Render the full-width combined monthly summary and explicit unmapped contribution
- [x] 6.2 Render personal and household summary cards side-by-side on wide screens and stacked on narrow screens
- [x] 6.3 Render separate full-width detailed reports with allocation, mapped actual, effective remaining, progress, and categories
- [x] 6.4 Render mapped transaction details eagerly inside native line disclosures and unmapped transactions in prominent separate disclosures
- [x] 6.5 Render scope-specific and full-dashboard empty states without any ensure/create action
- [x] 6.6 Add responsive render/browser-level tests covering desktop and phone layouts, keyboard disclosure, no-JavaScript use, and long transaction content
- [x] 6.7 Refine the Dracula-inspired visual treatment with restrained accents, tabular monetary figures, and clear over-budget and unmapped states

## 7. HTTP Composition and Regression Coverage

- [x] 7.1 Register the root dashboard and `/assets/` handlers alongside the existing `/v1` and `/live` handlers in the top-level server mux
- [x] 7.2 Wire budget, user, and household services plus UI configuration into the web adapter at `cmd/api`
- [x] 7.3 Add handler tests for root rendering, canonical redirects, query validation, owner not-found, missing budgets, and safe internal errors
- [x] 7.4 Verify UI requests never call the REST client or require the bearer API key
- [x] 7.5 Add regression tests proving `/v1` still requires bearer authentication and `/live` retains its existing behavior
- [x] 7.6 Add routing tests proving reserved API, liveness, and asset paths cannot be shadowed by human page routes

## 8. Container and Traefik Deployment

- [x] 8.1 Add a frontend build stage to the Dockerfile that obtains the version/checksum-pinned standalone Tailwind CLI and compiles CSS before the Go binary build
- [x] 8.2 Keep the Tailwind executable, source templates, and build tools out of the final runtime image while including required IANA timezone data
- [x] 8.3 Update development Compose configuration with UI defaults and `TZ=America/Toronto`
- [x] 8.4 Add the BasicAuth-protected `finance.homelab.voltr.org` Traefik router with credentials supplied through deployment secret configuration
- [x] 8.5 Change the API router to `finance-api.homelab.voltr.org` and restrict it to `/v1` while retaining bearer authentication
- [x] 8.6 Keep `/live` available to the container-local health check without intentionally exposing it through either public router
- [x] 8.7 Validate rendered Compose configuration without exposing or committing BasicAuth users, API keys, or database credentials

## 9. Documentation and Final Verification

- [x] 9.1 Document dashboard URLs, BasicAuth trust boundaries, default owner configuration, `TZ`, CAD assumptions, and local development commands
- [x] 9.2 Document that browser writes require a separate application-authentication, audit-identity, authorization, and CSRF design
- [x] 9.3 Document API hostname migration and update first-party CLI deployment configuration examples
- [x] 9.4 Run templ generation verification, Tailwind production build, sqlc generation checks, formatting, static analysis, and the complete Go test suite
- [x] 9.5 Build and smoke-test the production image for authenticated UI access, rejected unauthenticated UI access, responsive dashboard rendering, bearer API access, and container health
