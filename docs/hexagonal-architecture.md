# Hexagonal architecture

Voltr Finance uses feature-oriented ports and adapters with inward dependency direction.

```text
CLI -> REST client -> HTTP handlers -> application services -> repository ports -> Postgres/sqlc
```

## Packages

```text
cmd/api                 API composition and lifecycle
cmd/cli                 standalone REST-backed CLI
internal/api            versioned JSON wire contracts
internal/restclient     standard-library REST client
internal/httpapi        shared HTTP/auth/error infrastructure
internal/httpapi/*      feature-owned handlers and route registration
internal/app/*          application models, use cases, and outbound ports
internal/postgres/*     feature-owned sqlc adapters
internal/database/sqlc  generated SQL execution layer
internal/server         HTTP feature composition
```

Application feature packages import neither HTTP nor persistence infrastructure. Each feature owns the smallest repository interfaces needed by its use cases. PostgreSQL adapters translate sqlc rows, parameters, and driver errors into application-owned models and typed errors.

The `internal/api` package is an infrastructure-independent wire contract shared by HTTP handlers and the REST client. Application models are mapped explicitly rather than serialized directly.

The CLI is intentionally thin: it parses flags, calls `internal/restclient`, and renders API responses locally as JSON, compact text, or CSV. It has no PostgreSQL fallback or server application wiring.

## Composition

`cmd/api` is the only production process that wires database pools, sqlc queries, PostgreSQL adapters, application services, feature HTTP handlers, and server lifecycle. `cmd/cli` wires configuration to the REST client only.

The removed Discord, Genkit, and agent runtime is not part of the production graph. A future integration should be introduced as another inbound adapter calling existing application use cases, with external providers represented by application-owned outbound ports.
