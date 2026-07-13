## Context

Voltr Finance has two composition paths. `cmd/main.go` starts a Discord/Genkit agent and directly wires generated sqlc queries, transaction logic, cloud storage, and the bot. `cmd/cli/main.go` separately wires the CLI directly to Postgres and calls a monolithic `internal/app.Service`. Although `internal/app` contains useful application behavior, its repository interfaces and models expose sqlc rows, sqlc parameter types, pgx types, and a pgx-backed transactor. The current center therefore does not satisfy the dependency direction in `docs/hexagonal-architecture.md`.

The change replaces both runtime paths with a standard-library HTTP API and an API-backed CLI. Existing finance behavior and the `budget-reporting` specification remain authoritative. Existing database migrations and historical tables remain untouched, including dormant LLM tables and Discord-related identity columns.

## Goals / Non-Goals

**Goals:**

- Make the API server the only process that executes finance use cases or accesses Postgres.
- Expose every supported transaction, user, household, category, and budget operation through a versioned JSON REST API.
- Protect finance endpoints with one bearer API key.
- Make bulk mutation outcomes deterministic and explicit at the input-item level while retaining partial success.
- Split application logic into feature packages whose models and ports do not depend on HTTP, CLI, Postgres, pgx, or sqlc.
- Keep sqlc as the canonical generated SQL execution layer behind Postgres adapters.
- Make the CLI a thin authenticated HTTP client while retaining useful command names and local JSON, compact, and CSV rendering.
- Use `net/http` so JSON handlers and future `html/template` handlers can coexist as independent inbound adapters.
- Remove unused Discord, Genkit, agent, bot, and cloud-storage runtime code and dependencies.

**Non-Goals:**

- Dropping or rewriting existing database tables, columns, data, or migrations.
- Removing Discord IDs or guild IDs from finance identity models and CLI selectors.
- Adding browser authentication, user sessions, authorization roles, API-key persistence, key management, or multiple API keys.
- Adding server-rendered HTML pages in this change.
- Introducing an HTTP framework, OpenAPI generator, ORM, or replacing sqlc.
- Guaranteeing byte-for-byte compatibility with current CLI JSON, errors, or exit behavior.

## Decisions

### 1. Hexagonal package and dependency shape

The target package shape is feature-oriented at the application center:

```text
cmd/
  api/                 API composition root
  cli/                 CLI composition root

internal/
  app/
    transactions/      use cases, models, errors, outbound ports
    users/
    households/
    categories/
    budgets/
  http/                 routing, middleware, handlers, app/error mapping
  api/                  wire contracts and REST client; imports no app packages
  cli/                  command parsing and rendering; depends on API client
  database/             Postgres/sqlc adapters and transaction support
    sqlc/               generated code
```

Application feature packages SHALL own their input/output models and the narrow outbound ports needed by their use cases. They SHALL NOT import `internal/http`, `internal/api`, `internal/cli`, `internal/database`, `internal/transaction`, pgx, or sqlc. Shared application packages are allowed only for genuinely universal concepts such as typed application errors.

The HTTP handlers and API client use explicit wire contracts from `internal/api`; handlers translate between wire and application models. This small amount of mapping prevents an external JSON contract from becoming the domain model and prevents the standalone client from depending on server application packages.

The existing `internal/transaction` behavior, including validation and transaction hash generation, moves into the transactions application feature or an application-owned domain helper. Persistence-specific parameter construction moves into the database adapter.

Budget atomic operations use an application-owned transaction port. The Postgres adapter implements it with pgx and supplies transaction-scoped repository implementations. Application callbacks see only application repository ports.

Alternatives considered:

- Keeping one `internal/app.Service` would reduce initial movement but preserve broad interfaces and feature coupling.
- Wrapping every sqlc method one-for-one would retain persistence-shaped use cases and add ceremony without establishing a real port.
- Sharing application DTOs directly with the API client would be less mapping but would couple the external contract to server internals.

### 2. Standard-library HTTP server

The server uses Go's standard library:

- `net/http` and method-aware `http.ServeMux` patterns for routing;
- `encoding/json` with unknown-field rejection for request bodies;
- `http.Server` with explicit read-header, read, write, and idle timeouts;
- signal-aware graceful shutdown;
- `net/http/httptest` for handler and middleware tests;
- `html/template` and `embed` remain available for future web adapters.

No router or web framework is introduced because the route set and middleware needs are small. JSON API routes live under `/v1`. A liveness endpoint may live outside `/v1` and does not execute finance operations.

### 3. REST resource surface

The initial route surface is:

```text
Transactions
  POST   /v1/transactions
  POST   /v1/transactions/bulk
  GET    /v1/transactions/{id}
  GET    /v1/transactions
  PATCH  /v1/transactions/{id}
  PATCH  /v1/transactions/bulk
  DELETE /v1/transactions
  POST   /v1/transactions/restore

Users
  POST   /v1/users
  GET    /v1/users
  GET    /v1/users/{id}
  PATCH  /v1/users/{id}
  POST   /v1/users/resolve

Households
  GET    /v1/households
  GET    /v1/households/{id}
  GET    /v1/households/{id}/users
  GET    /v1/households/resolve

Categories
  POST   /v1/categories
  GET    /v1/categories
  GET    /v1/categories/{code}
  PATCH  /v1/categories/{id}
  DELETE /v1/categories/{code}

Budgets
  GET    /v1/budgets/monthly
  POST   /v1/budgets/monthly
  GET    /v1/budgets/{id}/report
  POST   /v1/budgets/{id}/lines
  PATCH  /v1/budget-lines/{id}
  DELETE /v1/budget-lines/{id}
```

`GET /v1/transactions` supports normal filters and an `ids` query for deterministic multi-ID retrieval. `DELETE /v1/transactions` accepts the IDs and audit fields in a JSON body. Restore remains an explicit action because restoration is not representable as ordinary resource creation or update. Category deletion retains current deactivation semantics and does not physically delete the category.

Household resolution accepts exactly one of `name` or `guildId`; direct ID retrieval uses the resource route. User resolution accepts exactly one supported identity selector.

Request IDs belong in path parameters for single-resource mutations and are not duplicated in JSON bodies. List filters, sorting, pagination, and include-deleted flags use query parameters. JSON field names remain lower camel case.

### 4. Monthly-budget read and ensure are separate

`GET /v1/budgets/monthly` is read-only and returns not-found when no budget exists. `POST /v1/budgets/monthly` idempotently ensures the selected owner's monthly budget exists, creating and optionally copying the latest prior structure according to existing application behavior. It returns `201 Created` when it creates a budget and `200 OK` when a concurrent or earlier request already created it.

The CLI keeps `budgets get --create`, but chooses the POST endpoint directly when the flag is present. It does not perform a GET-then-POST sequence.

### 5. Authentication uses one bearer API key

The server requires a non-empty API key at startup. All `/v1` routes require:

```text
Authorization: Bearer <api-key>
```

Middleware compares the supplied key with the configured key using `crypto/subtle.ConstantTimeCompare`. Missing, malformed, or incorrect credentials return a generic `401` response and are never logged. The liveness endpoint is unauthenticated. TLS termination is a deployment responsibility; production documentation requires HTTPS because a bearer key is reusable.

The API key authenticates the client installation, not a finance user. Existing audit user IDs and identity selectors remain explicit operation inputs. This avoids adding key/user persistence or authorization semantics.

Alternatives considered:

- `X-API-Key` is simple but bearer authorization is conventional and supported directly by HTTP tooling.
- Multiple persisted keys would improve rotation and attribution but add administration and data-model scope not needed now.

### 6. Stable JSON and error envelopes

A request-level failure uses a single error envelope:

```json
{
  "error": {
    "code": "validation_error",
    "message": "year must be greater than 0"
  }
}
```

Handlers map application errors consistently:

```text
malformed JSON or validation error   400
missing or invalid API key           401
resource not found                   404
duplicate/invariant conflict         409
unexpected internal/database error   500
```

Internal causes, SQL details, stack data, and secrets are logged server-side where appropriate but never returned. The application error taxonomy is expanded so users, households, categories, transactions, budgets, and budget lines can distinguish not-found, validation, conflict, and internal failures.

Single creates return the created representation with `201`; reads and updates return the representation with `200`; successful deletes may return `204` when there is no useful body. Collection responses use explicit arrays and return `[]`, not `null`.

### 7. Bulk mutations preserve partial success

Bulk create, update, delete, and restore process a syntactically valid envelope with item-level partial success. Their response is deterministic and keyed to original input indices:

```json
{
  "succeeded": [
    {"index": 0, "id": 101},
    {"index": 2, "id": 103}
  ],
  "failed": [
    {
      "index": 1,
      "id": 102,
      "error": {
        "code": "duplicate_transaction",
        "message": "duplicate transaction"
      }
    }
  ]
}
```

Both arrays are ordered by ascending input index. The `id` is omitted when no ID is known. Once the request envelope is valid, mixed, all-success, and all-item-failure results return `200`; malformed envelopes return a request-level `400`. Missing delete/restore IDs and duplicate inputs must be represented explicitly rather than disappearing from the result. If an infrastructure failure prevents item attribution, each unprocessed input is marked failed with a safe internal-error code.

Operations remain non-atomic across items. Application use cases may use a transaction within one item where its invariants require one, such as budget-line category replacement.

### 8. CLI is an HTTP adapter and renderer

`cmd/cli` wires command parsing to a standard-library REST client only. Neither it nor `internal/cli` imports database, sqlc, pgx, or server application packages. The client sets the bearer header, applies request timeouts, encodes requests, decodes success and error envelopes, and distinguishes transport failures from API operation failures.

The strict CLI config changes to:

```json
{
  "api": {
    "baseUrl": "https://finance.example.com",
    "apiKey": "secret"
  }
}
```

`VOLTR_API_URL` and `VOLTR_API_KEY` override corresponding file values for automation. Existing `--config`, `VOLTR_CONFIG`, and default path resolution remain. Help remains available without loading configuration.

Recognizable command names and JSON/compact/CSV rendering are retained where useful, but output contracts may be normalized. Exit codes become consistent: `0` for complete operation success, `2` for usage/validation or any item-level API failure, and `1` for authentication, transport, server, configuration, or unexpected failures. Bulk responses are still rendered so successful work is visible before a non-zero exit.

### 9. Runtime cleanup is code-only

The API composition root replaces the bot composition root. `internal/ai`, `internal/bot`, agent/Discord runtime configuration, and related development wiring are removed. `go mod tidy` removes dependencies no longer referenced, including DiscordGo, Genkit, cloud storage, and JSON-schema tooling as applicable.

No migration is edited or added solely to remove historical integration data. Discord identity fields, guild IDs, LLM tables, generated models, and historical rows remain. Finance operations that already support Discord IDs continue to support those identifiers as data, without importing or running Discord SDK code.

## Risks / Trade-offs

- [Large cross-cutting rewrite can introduce behavior regressions] → Characterize current use cases first, migrate one feature at a time behind ports, and run old behavioral tests plus new adapter/API tests before deleting old paths.
- [Sharing a single API key gives every holder full API access] → Keep the key server-side or in protected CLI configuration, require HTTPS, avoid logging it, and document that scoped credentials are future work.
- [Partial success can surprise callers expecting transactions] → Use explicit per-index results, deterministic ordering, and unambiguous CLI exit behavior.
- [DELETE request bodies are not uniformly supported by every intermediary] → The first-party standard-library client supports them; if deployment infrastructure rejects them, move the same bulk operation to an explicit action route without changing application semantics.
- [Handwritten contract mappings add boilerplate] → Keep contracts feature-scoped and test round trips; accept the mapping cost to preserve dependency direction.
- [Dormant LLM and Discord schema remains visible] → Document that cleanup is intentionally code-only and defer destructive schema decisions to a separate change.
- [One process becomes the sole database gateway] → Configure HTTP and database timeouts, graceful shutdown, health checks, and bounded connection pools.

## Migration Plan

1. Add characterization tests for current finance use cases and record existing budget-reporting behavior.
2. Introduce shared application errors and feature application packages with infrastructure-free models and ports.
3. Implement Postgres/sqlc adapters feature by feature, preserving SQL and schema behavior.
4. Add API wire contracts, standard-library handlers, error mapping, authentication middleware, and handler tests.
5. Add the API composition root, configuration, graceful lifecycle, health endpoint, and deployment wiring.
6. Add the REST client and migrate CLI commands and rendering to it; run CLI-to-HTTP integration tests.
7. Switch production/container entrypoints to the API server and distribute the API-backed CLI configuration.
8. Remove the old bot/agent code and runtime configuration, then tidy Go modules.
9. Run architecture/import checks and the complete test suite before release.

Rollback consists of deploying the preceding application/CLI binaries. Because this change does not migrate or delete database state, rollback does not require data restoration. CLI and server versions should be deployed together because the transport contract is new.

## Open Questions

None currently. The route contract may receive minor naming adjustments during implementation if standard-library route conflicts are discovered, but the resource semantics, separate monthly-budget endpoints, authentication model, and bulk-result behavior are fixed by the specifications.
