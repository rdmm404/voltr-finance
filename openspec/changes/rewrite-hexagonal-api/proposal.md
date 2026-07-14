## Why

Voltr Finance currently has a CLI that connects directly to Postgres while the production entrypoint is tied to unused Discord, Genkit, and agent infrastructure. Replacing those runtime paths with a small authenticated HTTP API creates one application boundary for every finance operation and allows the codebase to adopt the dependency direction described in `docs/hexagonal-architecture.md`.

## What Changes

- Add a versioned REST API exposing transaction, user, household, category, and budget application operations.
- Protect API routes with one server-configured bearer API key.
- Separate monthly-budget retrieval from idempotent get-or-create behavior; preserve the CLI's `budgets get --create` convenience by having it call the creation endpoint directly.
- Define deterministic bulk mutation responses that explicitly identify successful and failed input items while preserving partial-success behavior.
- Rewrite the CLI as a thin HTTP API client that parses input and renders JSON, compact, or CSV output without connecting to Postgres.
- Reorganize application code into feature-focused services with application-owned models and ports, backed by Postgres/sqlc adapters.
- Replace the Discord/Genkit production entrypoint with an API server entrypoint and remove unused bot, agent, Genkit, Discord, cloud-storage, and related runtime code and Go dependencies.
- Preserve existing database migrations, tables, Discord identity columns, guild identifiers, and historical data; database cleanup is outside this change.
- Keep the HTTP server based on Go's standard library so future server-rendered HTML adapters can use `html/template` alongside the JSON API.
- **BREAKING** Replace the CLI's database configuration with API URL and API-key configuration.
- **BREAKING** Permit cleanup of command responses, JSON envelopes, error codes, and exit-code behavior rather than guaranteeing byte-for-byte CLI compatibility.

## Capabilities

### New Capabilities
- `finance-rest-api`: Versioned JSON endpoints for all supported transaction, user, household, category, and budget operations, including separate monthly-budget read and ensure endpoints.
- `api-key-authentication`: Bearer API-key protection for finance API routes.
- `partial-success-batches`: Deterministic bulk-operation results that clearly identify each successful and failed input.
- `api-backed-cli`: CLI configuration and behavior as a thin authenticated REST API wrapper with local output rendering.

### Modified Capabilities

None. The existing `budget-reporting` requirements remain unchanged and must continue to hold through the new API and architecture.

## Impact

- Replaces `cmd/main.go` with API-server composition and rewires `cmd/cli` away from Postgres.
- Adds standard-library HTTP handlers, middleware, server lifecycle code, and an HTTP API client.
- Splits `internal/app` into feature packages and moves sqlc/pgx translation and transaction handling behind database adapters.
- Changes CLI configuration, API/CLI response contracts, error mapping, and deployment/Docker configuration.
- Removes `internal/ai`, `internal/bot`, Discord/agent configuration, and unused direct dependencies such as DiscordGo, Genkit, cloud storage, and JSON-schema tooling where no longer referenced.
- Retains Postgres, sqlc, existing finance behavior, immutable migrations, and dormant historical schema.
