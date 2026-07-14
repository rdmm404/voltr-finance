# API and dashboard deployment

## Runtime

Build the production API and CLI images:

```bash
docker build --target final -t voltr-api .
docker build --target cli -t voltr-finance-cli .
```

The Go server listens on `VOLTR_API_ADDRESS` (`:8080` by default) and serves both the dashboard and API. `/live` is unauthenticated for container-local health checks. Every `/v1` route requires:

```http
Authorization: Bearer <VOLTR_API_KEY>
```

Set a long random `VOLTR_API_KEY`; startup fails when it is empty. Terminate TLS at the load balancer or reverse proxy and use HTTPS for all non-local traffic so bearer credentials and finance data are encrypted in transit.

Required database settings are `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_PORT`, and `DB_NAME`. Pool bounds use `DB_POOL_SIZE` (default `5`) and `DB_MIN_POOL_SIZE` (default `0`). Connections force the `transactions` search path.

The dashboard requires positive `VOLTR_UI_DEFAULT_USER_ID` and `VOLTR_UI_DEFAULT_HOUSEHOLD_ID` settings. Set `TZ` to the IANA timezone used for month boundaries (production defaults to `America/Toronto`). See [Finance dashboard](dashboard.md) for CAD presentation and the temporary full-admin BasicAuth trust model.

## Compose

Local development:

```bash
cp .env.example .env
# replace VOLTR_API_KEY before sharing the environment
docker compose up --build
```

Production uses `docker-compose.prd.yml`, an external `postgres-network`, and values supplied by `.env`. It exposes the BasicAuth-protected UI at `finance.homelab.voltr.org` and only `/v1` at `finance-api.homelab.voltr.org`. Supply `VOLTR_UI_BASIC_AUTH_USERS` out of band; do not commit it. Run migrations explicitly when required:

```bash
docker compose -f docker-compose.prd.yml --profile migrate run --rm migrate
```

The API handles `SIGINT` and `SIGTERM` by stopping HTTP acceptance, allowing in-flight requests up to the shutdown deadline, and then closing the PostgreSQL pool.

## API behavior

Finance routes are versioned under `/v1` and cover transactions, users, households, categories, monthly budgets, budget lines, and reports. JSON errors have a stable shape:

```json
{"error":{"code":"validation_error","message":"safe message"}}
```

Bulk transaction endpoints return HTTP 200 with indexed `succeeded` and `failed` arrays; callers must inspect both. `GET /v1/budgets/monthly` is read-only. `POST /v1/budgets/monthly` idempotently ensures the month exists and returns 201 only when it creates one.

This release adds no destructive database migration. Rollback consists of restoring the previous API/CLI images and their matching configuration; existing schema and data remain compatible.
