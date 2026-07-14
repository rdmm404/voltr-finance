# Finance dashboard

The read-only monthly dashboard is served at `https://finance.homelab.voltr.org/`. The canonical URL includes `month=YYYY-MM`; optional `userId` and `householdId` query parameters select bookmarkable report owners. Missing monthly budgets are shown as empty states and are never created by the UI.

## Access and trust boundary

Traefik BasicAuth protects the complete human-facing hostname. Set `VOLTR_UI_BASIC_AUTH_USERS` through deployment secret configuration to a bcrypt/MD5/SHA1-formatted Traefik users value; never commit credentials. BasicAuth grants installation-wide read access. Its username is not an application identity and does not select or authorize a finance owner.

The JSON API has a separate boundary at `https://finance-api.homelab.voltr.org/v1`: application bearer authentication remains authoritative there. The API hostname does not expose dashboard paths, and the UI does not call the API or require its bearer key. `/live` is reserved for the container-local health check.

Browser writes are intentionally excluded. Adding them requires a separate design for application authentication, durable audit identity, per-owner authorization, and CSRF protection; Traefik BasicAuth alone is insufficient.

## Configuration

The server requires positive `VOLTR_UI_DEFAULT_USER_ID` and `VOLTR_UI_DEFAULT_HOUSEHOLD_ID` values. Explicit query overrides that identify missing owners return a safe `404` rather than silently reverting to these defaults.

Set `TZ=America/Toronto` (or another IANA timezone) to define the current calendar month and rendered dates. Production includes IANA timezone data. Monetary values are presented consistently as `en-CA` CAD; no conversion or mixed-currency behavior is implied.

## Local development

Install no Node runtime. The project pins templ through Go's tool directive and downloads a checksum-verified standalone Tailwind executable.

```bash
make templ             # regenerate committed *_templ.go files
make css               # build production CSS
make generate          # run both generators
make dev               # templ watch + Tailwind watch + Air
make verify-generated  # verify committed templ output
```

Local Compose defaults the owner IDs to `1` and timezone to `America/Toronto`; override them through environment variables when your seed data differs. Compiled CSS and `.tools/tailwindcss` are generated artifacts and are not committed.
