## Why

Voltr Finance has an authenticated API and CLI but no durable human interface, so monthly budget review currently depends on a generated HTML report that duplicates reporting logic and requires manual regeneration. A private, server-rendered dashboard will make the existing finance data immediately useful in a browser while preserving the new hexagonal application boundary and leaving room for later administrative writes.

## What Changes

- Add a responsive, read-only monthly finance dashboard rendered by the existing Go server at root-level human-facing routes.
- Show combined, personal, and household budget summaries; budget-line progress; eagerly rendered transaction drill-down; and prominent unmapped spending.
- Treat mapped and unmapped transactions as spending in headline totals while retaining explicit unmapped detail.
- Add an authoritative detailed monthly budget-report query that associates mapped transactions with their budget lines in a consistent read snapshot instead of reconstructing mappings in the UI.
- Add configurable default personal and household scopes with bookmarkable owner and month overrides.
- Add reusable templ components, UI-owned view models, a Dracula-inspired responsive Tailwind theme, native semantic HTML interactions, and embedded same-origin assets.
- Add a reproducible build pipeline using committed templ-generated Go files and a version/checksum-pinned standalone Tailwind CLI; Node, a JavaScript bundler, and chart dependencies are not introduced initially.
- Expose the UI and JSON API through separate Traefik hostnames backed by the same container, with Traefik BasicAuth protecting the full-access UI and the existing bearer key continuing to protect `/v1`.
- Default server-rendered dates to the process `TZ` setting and monetary presentation to CAD.
- Keep charts, HTMX, application-managed authentication/authorization, multi-currency support, and all UI writes outside this change.

## Capabilities

### New Capabilities
- `finance-dashboard`: Private server-rendered monthly dashboard behavior, scope and period selection, financial summaries, responsive detail presentation, asset delivery, and deployment access boundary.

### Modified Capabilities
- `budget-reporting`: Add a detailed reporting mode that returns the in-scope transactions mapped to each budget line while preserving the existing aggregate report behavior and owner-scope rules.

## Impact

- Adds a web inbound adapter, templ-generated components, page/view models, styling assets, and route composition alongside `internal/httpapi`.
- Extends the budget application and PostgreSQL adapter with a detailed monthly report read path and mapped-transaction models/queries.
- Adds templ and standalone Tailwind generation to development, CI, and Docker build workflows, plus pinned tool metadata and embedded static assets.
- Updates production Traefik labels, hostnames, BasicAuth middleware wiring, timezone configuration, and deployment documentation.
- Does not change database schema, existing `/v1` authentication, existing aggregate REST/CLI report contracts, or application write behavior.
