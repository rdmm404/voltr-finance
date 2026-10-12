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

The dashboard requires positive `VOLTR_UI_DEFAULT_USER_ID` and `VOLTR_UI_DEFAULT_HOUSEHOLD_ID` settings. Set `TZ` to the IANA timezone used for month boundaries (production defaults to `America/Toronto`). See [Finance dashboard](dashboard.md) for CAD presentation. The current production Compose configuration does not attach a BasicAuth middleware to the read-only dashboard; authenticated mutation routes remain under `/v1`. Do not treat older BasicAuth documentation as evidence that the deployed dashboard is protected.

## Compose

Local development:

```bash
cp .env.example .env
# replace VOLTR_API_KEY before sharing the environment
docker compose up --build
```

Production uses `docker-compose.prd.yml`, an external `postgres-network`, and values supplied by `.env`. It exposes the read-only UI at `finance.homelab.voltr.org` and only `/v1` at `finance-api.homelab.voltr.org`. The current UI router has no BasicAuth middleware; `/v1` requires the API bearer key. Run migrations explicitly when required:

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

## Production Voltio CLI deployment

This workflow applies CLI changes from this repository to the production
Voltio instance in `/home/rdmm404/services/agents`. The live host binary is
`/home/rdmm404/.local/bin/voltr-finance`. The `voltio-gateway` service
bind-mounts that file read-only at `/usr/local/bin/voltr-finance`.

The current Compose deployment has only `voltio-gateway`; it does not define
a Docker `voltio-whatsapp-bridge` service. Recreate the gateway to refresh its
file-level binary mount. Separate systemd WhatsApp bridge templates are not
Docker services and are outside this workflow. If a future Compose deployment
adds a service sharing the gateway's network namespace, review that topology
and recreate its coupled services together.

### 1. Classify the change

- CLI-only changes require the test, dev-database verification, rebuild,
  recreation, and production verification steps below. An image build is not
  required for the bind-mounted binary.
- API behavior changes must be deployed and verified through
  `docker-compose.prd.yml` before deploying a CLI that depends on them.
- Database changes require a reviewed migration and an explicit migration run
  before dependent API or CLI behavior is enabled.
- Changes to the Voltio agent image, Compose file, profile, or configuration
  are a separate deployment. Do not use this CLI-only workflow as a substitute
  for rebuilding the affected image.

### 2. Test before touching production

From the Voltr Finance repository:

```bash
cd /home/rdmm404/apps/voltr-finance
git status --short --branch
git diff --check
go test ./...
go build -o /tmp/voltr-finance-candidate ./cmd/cli
/tmp/voltr-finance-candidate --help
```

Review the complete diff and ensure the tests cover the changed behavior. If
generated SQL changed, run `/home/rdmm404/go/bin/sqlc generate`, inspect the
generated diff, and rerun `go test ./...`. If templates or CSS changed, run
`make verify-generated` as well.

Do not continue when tests fail, generated files are stale, or the worktree
contains unexplained changes.

### 3. Verify against the development database

Start the local database, run migrations, and bring up the API from the current
worktree:

```bash
cd /home/rdmm404/apps/voltr-finance
VOLTR_API_KEY=development-only-key \
  docker compose up -d --build db migrate api
docker compose ps
curl --fail --silent --show-error http://127.0.0.1:8080/live
```

Create a temporary CLI config that cannot accidentally target production:

```bash
printf '%s\n' \
  '{"api":{"baseUrl":"http://127.0.0.1:8080","apiKey":"development-only-key"}}' \
  > /tmp/voltr-finance-dev.json
```

Use `/tmp/voltr-finance-candidate` to exercise the changed behavior through the
development API. Choose identifiers from `users list` and `households list`
rather than assuming seeded IDs:

```bash
/tmp/voltr-finance-candidate --config /tmp/voltr-finance-dev.json users list
/tmp/voltr-finance-candidate --config /tmp/voltr-finance-dev.json households list
```

For a transaction-creation change, the verification can look like this:

```bash
DEV_TIMESTAMP="$(date --iso-8601=seconds)"
DEV_MARKER="DEV deployment verification $(date +%s)"
/tmp/voltr-finance-candidate --config /tmp/voltr-finance-dev.json \
  transactions create \
  --amount 1.23 \
  --transaction-date "$DEV_TIMESTAMP" \
  --description "$DEV_MARKER" \
  --author-id <DEV_USER_ID>

/tmp/voltr-finance-candidate --config /tmp/voltr-finance-dev.json \
  transactions get --ids <CREATED_TRANSACTION_ID>
```

Verify the persisted state directly when the change concerns storage, owner
scope, migrations, or report aggregation:

```bash
docker compose exec -T db psql -U voltr -d voltr_finance -c \
  'SELECT id, author_id, household_id, amount
     FROM transactions.transaction
    WHERE id = <CREATED_TRANSACTION_ID>;'
```

The API response and database row must both match the expected behavior. Adapt
the CLI command and SQL query to the feature under test; do not treat a health
check alone as data-path verification.

Clean up every mutating verification before continuing. Transaction checks can
use the normal soft-delete path:

```bash
/tmp/voltr-finance-candidate --config /tmp/voltr-finance-dev.json \
  transactions delete \
  --ids <CREATED_TRANSACTION_ID> \
  --deleted-by-user-id <DEV_USER_ID> \
  --reason "completed DEV deployment verification"
```

Do not continue to production if the candidate fails against the development
database or if the direct SQL result differs from the intended model.

### 4. Rebuild the live CLI

Confirm the binary on `PATH` is the file mounted by Compose, save a temporary
rollback copy, and rebuild that exact path:

```bash
command -v voltr-finance
readlink -f "$(command -v voltr-finance)"
cp -p /home/rdmm404/.local/bin/voltr-finance /tmp/voltr-finance.previous
go build -o /home/rdmm404/.local/bin/voltr-finance ./cmd/cli
```

The resolved path must be `/home/rdmm404/.local/bin/voltr-finance`.

### 5. Recreate the production Voltio gateway

A running container can retain the old file-level bind mount after the host
binary is replaced. Restarting the process is insufficient; force-recreate the
currently deployed gateway:

```bash
cd /home/rdmm404/services/agents
docker compose config --quiet
docker compose up -d --no-build --force-recreate voltio-gateway
```

### 6. Verify without mutating finance data

```bash
docker compose ps
curl --fail --silent --show-error http://127.0.0.1:18791/health
sha256sum /home/rdmm404/.local/bin/voltr-finance
docker exec svc-voltio-gateway \
  sha256sum /usr/local/bin/voltr-finance
docker exec svc-voltio-gateway voltr-finance --help
docker compose logs --since=5m --tail=100 voltio-gateway
```

Deployment is complete only when the gateway container is running, the health
endpoint returns `{"status": "ok"}`, host and container checksums match, the
changed CLI surface works inside `svc-voltio-gateway`, and recent logs contain
no restart loop or connection failure.

Prefer help, list, or get commands for production smoke tests. Do not create,
update, or delete finance data merely to verify a deployment. When a data-path
check is genuinely required, use an explicitly identified test record and
clean it up deliberately.

### 7. Roll back a CLI-only deployment

If verification fails, restore the saved binary and recreate the gateway
again:

```bash
cp -p /tmp/voltr-finance.previous \
  /home/rdmm404/.local/bin/voltr-finance
cd /home/rdmm404/services/agents
docker compose up -d --no-build --force-recreate voltio-gateway
```

Repeat all verification commands after rollback. A CLI rollback does not roll
back API images or database migrations; those require their own compatible
rollback plan.
