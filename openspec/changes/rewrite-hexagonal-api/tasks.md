## 1. Baseline and Contract Preparation

- [x] 1.1 Run and record the current test baseline, and add characterization coverage for finance behaviors that are not already protected before moving code.
- [x] 1.2 Add characterization tests for transaction partial-success indexing, missing delete/restore IDs, and deterministic result ordering.
- [x] 1.3 Add characterization coverage confirming the existing budget owner scope, prior-budget copy, line-category atomicity, and budget-reporting requirements.
- [x] 1.4 Define the versioned API route, request, response, error, and bulk-result contracts in feature-scoped `internal/api` types without importing application or database packages.

## 2. Application Center

- [x] 2.1 Introduce shared infrastructure-independent application error codes and helpers for validation, not-found, conflict, and internal failures.
- [x] 2.2 Create the `internal/app/users` feature models, ports, service, and unit tests, preserving all supported external identity selectors.
- [x] 2.3 Create the `internal/app/households` feature models, ports, service, and unit tests for list, ID lookup, external resolution, and household users.
- [x] 2.4 Create the `internal/app/categories` feature models, ports, service, error mapping contract, and unit tests for create, list, lookup, update, and deactivate behavior.
- [x] 2.5 Create the `internal/app/transactions` models and application-owned ports, moving transaction validation and hash behavior out of sqlc-shaped types.
- [x] 2.6 Implement and unit-test single transaction create, read, list, update, soft-delete, and restore use cases in the transactions feature.
- [x] 2.7 Implement and unit-test deterministic partial-success transaction batches with one indexed result per input, including duplicate inputs, missing IDs, and unattributable infrastructure failures.
- [x] 2.8 Create the `internal/app/budgets` models, ports, and application-owned transaction boundary without pgx, pgtype, or sqlc imports.
- [x] 2.9 Implement and unit-test separate monthly-budget read and idempotent ensure use cases, including prior-budget structure copying and concurrent creation recovery.
- [x] 2.10 Implement and unit-test budget-line create, update, delete, category replacement, and invariant enforcement use cases.
- [x] 2.11 Implement and unit-test budget report assembly against the existing `budget-reporting` requirements.

## 3. Postgres and sqlc Adapters

- [ ] 3.1 Refactor database configuration and connection construction into validated, non-global server configuration while retaining bounded pool settings and the transactions search path.
- [ ] 3.2 Implement common Postgres error translation and application-owned transaction callback support backed by pgx and transaction-scoped sqlc queries.
- [ ] 3.3 Implement and test the sqlc-backed users repository adapter with translation between generated and application models.
- [ ] 3.4 Implement and test the sqlc-backed households repository adapter.
- [ ] 3.5 Implement and test the sqlc-backed categories repository adapter, including unique-conflict and not-found translation.
- [ ] 3.6 Implement and test the sqlc-backed transactions repository adapter, preserving hashes, filtering, soft deletion, restoration, and item attribution.
- [ ] 3.7 Implement and test the sqlc-backed budgets repository adapter, including transactional line/category changes and monthly-budget copying.
- [ ] 3.8 Verify that existing SQL and schema continue to satisfy budget-report scoping and unmapped-transaction behavior without modifying historical migrations.

## 4. HTTP Server Foundation

- [ ] 4.1 Implement strict standard-library JSON decoding and encoding helpers, path/query parsing, empty-array normalization, and the stable top-level error envelope.
- [ ] 4.2 Implement centralized application-error-to-HTTP mapping for validation, not-found, conflict, and internal failures without leaking infrastructure details.
- [ ] 4.3 Implement bearer API-key middleware using constant-time comparison, generic unauthorized responses, and tests proving keys are not returned or logged.
- [ ] 4.4 Implement the standard-library server router, unauthenticated liveness route, authenticated `/v1` boundary, server timeouts, and method/not-found behavior.

## 5. Finance REST Handlers

- [ ] 5.1 Implement and test transaction create, read, filtered list, and single-update handlers with explicit wire-to-application mapping.
- [ ] 5.2 Implement and test transaction bulk create/update, bulk soft-delete, and bulk restore handlers with HTTP 200 indexed partial-success envelopes.
- [ ] 5.3 Implement and test user create, list, get, update, and exactly-one-selector resolution handlers.
- [ ] 5.4 Implement and test household list, ID lookup, external resolution, and household-user handlers.
- [ ] 5.5 Implement and test category create, list, code lookup, update, and deactivation-through-DELETE handlers.
- [ ] 5.6 Implement and test read-only monthly-budget GET and idempotent monthly-budget ensure POST handlers, including 200-versus-201 responses.
- [ ] 5.7 Implement and test budget-line create/update/delete and budget-report handlers while retaining existing report requirements.
- [ ] 5.8 Add end-to-end `httptest` coverage for authenticated success, malformed and unknown-field requests, missing resources, conflicts, internal errors, and empty collections across all feature routes.

## 6. REST Client

- [ ] 6.1 Implement the base standard-library REST client with normalized base URLs, request timeouts, bearer credentials, strict response decoding, and typed transport/API errors.
- [ ] 6.2 Implement and test transaction client methods, including all bulk result envelopes and list/query encoding.
- [ ] 6.3 Implement and test user, household, and category client methods and selector/query encoding.
- [ ] 6.4 Implement and test budget monthly read/ensure, line mutation, and report client methods.

## 7. API-Backed CLI

- [ ] 7.1 Replace the CLI database config schema with strict API base URL/API key configuration and `VOLTR_API_URL`/`VOLTR_API_KEY` overrides while preserving config-path precedence.
- [ ] 7.2 Rewire `cmd/cli` and `internal/cli` to depend only on the REST client and API wire types, with help remaining configuration-free.
- [ ] 7.3 Migrate transaction commands to the REST client while preserving useful flags, stdin bulk input, and JSON/compact/CSV rendering.
- [ ] 7.4 Migrate user, household, and category commands to the REST client while retaining supported Discord identity data as ordinary selectors.
- [ ] 7.5 Migrate budget commands so `budgets get --create` directly calls monthly ensure and the command without the flag performs only a read.
- [ ] 7.6 Normalize CLI rendering and enforce exit code 0 for complete success, 2 for usage/validation or item failures, and 1 for configuration/authentication/transport/server failures.
- [ ] 7.7 Add CLI-to-`httptest` integration coverage proving commands authenticate, never initialize Postgres, render partial successes before a non-zero exit, and select the correct budget endpoint.

## 8. API Composition and Deployment

- [ ] 8.1 Create `cmd/api` as the composition root that validates API/database configuration, wires feature services to Postgres adapters and HTTP handlers, and fails startup on an empty API key.
- [ ] 8.2 Add signal-aware API server startup and graceful shutdown that stops HTTP acceptance and closes the database pool cleanly.
- [ ] 8.3 Update the Dockerfile build/final targets to produce the API server and continue producing the standalone API-backed CLI.
- [ ] 8.4 Update development and production Compose configuration, environment examples, health checks, and exposed ports for the API runtime.
- [ ] 8.5 Rewrite CLI and deployment documentation for the REST endpoints, bearer authentication, API configuration, HTTPS expectation, partial-success responses, and monthly-budget semantics.

## 9. Runtime Cleanup

- [ ] 9.1 Remove the old Discord/Genkit composition entrypoint, `internal/bot`, `internal/ai`, agent development wiring, and Discord/agent-only runtime configuration.
- [ ] 9.2 Remove obsolete direct-database CLI wiring and the superseded monolithic application/transaction packages after all callers and tests have migrated.
- [ ] 9.3 Run `go mod tidy` and verify DiscordGo, Genkit, cloud storage, JSON-schema tooling, and unused transitive dependencies are removed while finance hashing and CLI dependencies remain.
- [ ] 9.4 Verify no database migration, historical table, Discord identity column, guild identifier, or stored data was removed as part of code cleanup.

## 10. Verification and Release Readiness

- [ ] 10.1 Add an architecture test or static import check proving application feature packages do not import HTTP, API client, CLI, database, transaction, pgx, or sqlc packages.
- [ ] 10.2 Add a dependency check proving the CLI production import graph contains no Postgres, pgx, sqlc, or server application wiring.
- [ ] 10.3 Run formatting, unit tests, race-enabled tests, API/CLI integration tests, and production builds for both `cmd/api` and `cmd/cli`.
- [ ] 10.4 Perform a local Postgres smoke test covering authentication, every REST feature family, bulk partial success, monthly budget read/ensure, budget reporting, and CLI rendering.
- [ ] 10.5 Confirm deployment rollback requires only the previous binaries and configuration because this change introduces no destructive database migration.
