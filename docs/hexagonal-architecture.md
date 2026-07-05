# Potential Hexagonal Architecture Direction

This document captures a possible direction for restructuring Voltr Finance toward a more hexagonal architecture. It is not an immediate migration plan or a requirement to rewrite the codebase. It is a reference for future changes, especially as the CLI moves toward becoming a REST API wrapper instead of talking to Postgres directly.

## Core idea

Hexagonal architecture is about dependency direction:

- application/use-case code should sit at the center;
- inbound technology, such as CLI, HTTP, Discord, or agent flows, should call into the application;
- outbound technology, such as Postgres/sqlc, Redis, LLM providers, or external APIs, should be called through application-owned ports;
- infrastructure should adapt to the application, not the other way around.

In short:

```text
CLI / HTTP / Discord / Agent
              ↓
        application use cases
              ↓
    app-owned outbound interfaces
              ↓
       Postgres / Redis / LLM / APIs
```

The app should know about budgets, transactions, users, categories, and rules. It should not need to know that budgets are loaded through `sqlc.GetHouseholdBudgetByPeriodParams` or that cache invalidation is implemented with Redis keys.

## Ports and adapters

A **port** is an interface or API shape owned by the application.

Example outbound port:

```go
type BudgetRepository interface {
    GetBudgetByPeriod(ctx context.Context, owner BudgetOwner, period BudgetPeriod) (Budget, error)
    CreateBudget(ctx context.Context, owner BudgetOwner, period BudgetPeriod, sourceID *int64) (Budget, error)
}
```

An **adapter** is a concrete implementation or caller at the edge.

Examples:

- CLI command parsing flags and calling app use cases;
- HTTP handlers parsing JSON and calling app use cases;
- Postgres/sqlc repository implementing `BudgetRepository`;
- Redis-backed cache wrapper around a repository;
- Genkit/LLM client implementing an app-owned `LLMClient` port.

Inbound adapters usually call the app directly. They do not necessarily implement an app interface. Outbound adapters usually implement app-owned interfaces because the app depends on their capabilities.

## Domain/application models

Models like `BudgetOwner` and `BudgetPeriod` are application concepts, not database concepts.

For example, `BudgetOwner` encodes the invariant that a budget belongs to exactly one owner: either a household or a user. That remains true regardless of whether the data is stored in Postgres, memory, or another system.

In a hexagonal approach, repository ports should receive and return application models, not `sqlc` generated models. The Postgres adapter translates between app models and generated sqlc params/rows.

## Relationship with sqlc

This architecture does **not** replace sqlc.

sqlc should remain the canonical SQL execution layer for Postgres:

- handwritten SQL remains explicit and reviewable;
- query params and rows remain compile-checked;
- scanning and generated boilerplate stay automated;
- complex SQL can stay in SQL instead of being rebuilt through a query builder.

The repository layer should not become a one-for-one wrapper around every sqlc method. That would add ceremony and defeat the purpose.

Instead, introduce app-level repository methods only when generated query shape leaks too much persistence detail into use-case code.

Good candidates:

```go
GetBudgetByPeriod(ctx, owner, period)
CreateMonthlyBudgetFromSource(ctx, owner, period, sourceBudgetID)
ReplaceBudgetLineCategories(ctx, budgetID, lineID, categoryIDs)
GetBudgetReport(ctx, budgetID)
```

Poor candidates:

```go
GetHouseholdBudgetByPeriod(...)
GetUserBudgetByPeriod(...)
DeleteBudgetLineCategories(...)
CreateBudgetLineCategory(...)
```

The first group describes application operations. The second group mirrors generated SQL details.

## Preferred package shape

We do not need folder names like `adapters/inbound` or `adapters/outbound`. The structure can stay Go-simple while preserving dependency direction.

A possible long-term shape:

```text
cmd/
  api/
    main.go              # wires HTTP server + app + Postgres repositories
  cli/
    main.go              # wires CLI + REST API client

internal/
  app/
    errors/              # optional shared app errors, if not kept per feature
      errors.go

    budgets/
      service.go         # budget use cases
      models.go          # BudgetOwner, BudgetPeriod, Budget, BudgetLine
      ports.go           # repository/other outbound ports needed by budgets

    transactions/
      service.go
      models.go
      ports.go

    categories/
      service.go
      models.go
      ports.go

    users/
      service.go
      models.go
      ports.go

  http/
    server.go
    budgets.go
    transactions.go
    errors.go

  cli/
    commands.go
    render.go

  database/
    conn.go
    transactor.go
    budgets.go          # sqlc-backed app.BudgetRepository
    transactions.go
    categories.go
    sqlc/
      ...generated...

  api/
    client.go           # REST client used by CLI
    budgets.go
    transactions.go

  cache/
    redis.go            # optional Redis implementations/decorators

  agent/
    ...                 # optional inbound agent/Discord orchestration if revived

  llm/
    genkit.go           # optional outbound LLM adapter if app calls an LLM
```

The important rule is not the folder name. The important rule is dependency direction:

```text
internal/app/... imports no http, cli, database, api, cache, agent, or llm packages.

internal/http imports specific app feature packages, such as internal/app/budgets.
internal/database imports app feature packages and internal/database/sqlc.
internal/cli may import internal/api once the CLI becomes a REST wrapper.
```

### App feature packages

The app layer should be organized by feature area rather than as one large package:

```text
internal/app/budgets
internal/app/transactions
internal/app/categories
internal/app/users
```

Each feature package owns its own use cases, models, and outbound ports. For example, `internal/app/budgets` owns `BudgetOwner`, `BudgetPeriod`, budget request/response models, and the repository interfaces its use cases need.

Be careful with dependencies between sibling app packages. If budgets need category data, prefer defining a small consumer-owned port in `internal/app/budgets`:

```go
type CategoryResolver interface {
    ResolveActiveCategoryIDs(ctx context.Context, codes []string) ([]int64, error)
}
```

Then the database adapter can implement that port. This avoids making `budgets` import `categories` unless the dependency is truly part of the domain model.

Use shared app packages sparingly. A shared package can make sense for genuinely universal concepts, such as app errors or a future `Money` type, but it should not become a dumping ground for miscellaneous models.

## CLI as REST wrapper

If the CLI becomes a REST API wrapper, the architecture naturally becomes more hexagonal.

Target runtime shape:

```text
Human
  ↓
CLI
  ↓
REST API client
  ↓
HTTP server
  ↓
application use cases
  ↓
app-owned repository ports
  ↓
Postgres/sqlc repository adapter
```

The CLI should become thin:

- parse flags;
- call the REST API;
- render responses.

The API server owns use-case execution:

- parse HTTP requests;
- call `internal/app` services;
- map app errors to HTTP status codes;
- encode HTTP responses.

## Agent, Discord, and Genkit

If the agent flow is revived, classify it by role:

- Discord receiving messages is an inbound adapter.
- An agent or assistant use case that decides what product action to take belongs in `internal/app` if it contains business/product behavior.
- Genkit as an LLM provider is an outbound adapter behind an app-owned interface.
- Genkit as an externally invoked flow is an inbound adapter that should call app use cases.

Prefer this shape:

```text
Discord / Genkit flow
        ↓
 app Assistant use case
        ↓
 budget/transaction/category use cases
        ↓
 LLM port, repositories, etc.
        ↓
 Genkit adapter, Postgres adapter, etc.
```

Avoid making one adapter call another adapter for core orchestration. If adapter-to-adapter calls start containing product behavior, move that behavior into `internal/app`.

## Migration philosophy

Do not big-bang rewrite the codebase.

Migrate feature by feature when there is a reason:

- adding the REST API;
- changing CLI to call the API;
- fixing service code that is too tightly coupled to sqlc shapes;
- introducing cache, external services, or LLM integrations;
- making a use case easier to test.

Rules of thumb:

1. Keep app code centered on use cases and domain/application concepts.
2. Keep sqlc as the explicit Postgres SQL layer.
3. Do not wrap sqlc one-for-one.
4. Add repository methods only when they represent meaningful app operations or read models.
5. Let app-owned interfaces define what infrastructure must provide.
6. Let adapters translate between technology-specific shapes and app models.
7. Prefer boring Go package structure over architecture-diagram purity.
