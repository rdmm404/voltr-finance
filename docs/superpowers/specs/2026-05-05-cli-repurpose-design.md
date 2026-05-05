# Voltr Finance CLI Repurpose Design

## Goal

Repurpose the existing personal finance agent into a host-installed CLI that exposes the current finance data operations to a general-purpose agent such as Nanobot.

The initial release must preserve the useful behavior of the current agent without carrying forward the Discord/Genkit runtime as the primary integration surface. The CLI will connect directly to Postgres using limited credentials and will be deployed alongside the existing Docker Compose setup on the Debian host.

## Non-Goals

- No HTTP API in v1.
- No MCP server in v1.
- No arbitrary SQL execution command in v1.
- No LLM-backed natural-language transaction query in v1.
- No household mutation commands in v1.
- No hard delete for transactions in v1.

## Architecture

The repo will be organized around four conceptual layers:

1. `sqlc/database`
   - Owns generated SQL queries, DB connection setup, and low-level persistence.
   - Existing sqlc usage remains valid.

2. `transaction service`
   - Owns transaction business rules that already exist today: validation, hashing, duplicate handling, create, update, and get by ID.
   - This layer may be adjusted or wrapped, but it should not become CLI-specific.

3. `application service`
   - New API-shaped use-case boundary.
   - CLI commands call this layer, not sqlc directly.
   - Future HTTP or MCP transports should be able to call the same layer.
   - Owns request/response DTOs, identity resolution, stable error codes, and mapping from sqlc models to caller-facing models.

4. `entrypoints`
   - Current bot entrypoint can remain for legacy use while v1 is built.
   - New CLI entrypoint builds the host-installed `voltr-finance` binary.

The CLI should be thin: parse flags or JSON input, build a typed application request, call the application service, render output, and return a meaningful exit code.

## Deployment

The v1 artifact is a host-installed binary:

```text
voltr-finance
```

The normal config path is:

```text
$HOME/.config/voltr-finance/config.toml
```

Config lookup order:

```text
1. --config /path/to/config.toml
2. VOLTR_CONFIG=/path/to/config.toml
3. $HOME/.config/voltr-finance/config.toml
```

The binary must not embed database credentials. Runtime config contains the DB host, port, database name, and limited DB credentials.

The Docker image can still be used to build/package the binary. Later, CI can publish Linux release binaries to GitHub releases.

## Database Security

The CLI must not use migration/admin credentials.

Recommended roles:

- `voltr_cli_rw`: read/write only for approved app operations in the `transactions` schema.
- `voltr_cli_ro`: read-only role for read/list/export operations if split credentials are worth the extra setup.
- `voltr_migrate`: migration-only role used by dbmate, not available to the CLI runtime.

The CLI must not expose arbitrary write SQL. All writes go through typed application service methods and sqlc queries.

## Schema Changes

Extend `users` so it can identify people from Discord, Telegram, and WhatsApp without introducing a generic identity table:

```text
users.discord_id nullable unique
users.telegram_id nullable unique
users.phone_number nullable unique
users.whatsapp_id nullable unique
```

Implementation details:

- `discord_id` becomes nullable so users do not need a Discord identity.
- `telegram_id`, `phone_number`, and `whatsapp_id` are text identifiers.
- Unique indexes should only apply when the value is not null.
- `phone_number` should use an E.164-style value when available, for example `+1234567890`.
- `whatsapp_id` stores WhatsApp LID/JID-style identifiers when Nanobot cannot provide a phone number.

Add soft-delete fields to `transaction`:

```text
transaction.deleted_at nullable
transaction.deleted_by_user_id nullable references users(id)
transaction.delete_reason nullable
```

Default transaction reads exclude deleted rows unless explicitly requested.

## Identity Resolution

Transaction author resolution is provider-specific and deterministic.

Transaction commands accept exactly one of:

```text
--author-id
--author-discord-id
--author-telegram-id
--author-phone-number
--author-whatsapp-id
```

No generic `--author` resolver in v1.

Nanobot mapping:

```text
discord  -> --author-discord-id "$sender_id"
telegram -> --author-telegram-id "$metadata.user_id" when available, otherwise normalized "$sender_id"
whatsapp -> --author-phone-number "$sender_id" when it is a phone number
whatsapp -> --author-whatsapp-id "$sender_id" when it is a WhatsApp LID/JID
```

Telegram normalization may strip `|username` from Nanobot sender IDs such as `123456789|rafael`.

If identity resolution finds no matching user, the operation fails with a structured user-not-found error. If multiple provider-specific fields are supplied, the operation fails validation.

## CLI Surface

### Transactions

Single create from flags:

```text
voltr-finance transactions create
  --amount 42.50
  --transaction-date 2026-05-05T14:30:00-04:00
  --description "Groceries"
  --notes "Costco"
  --author-id 1
  --household-id 1
```

Bulk create from `--input` or stdin:

```text
voltr-finance transactions create-bulk --input transactions.json
voltr-finance transactions create-bulk < transactions.json
```

Single update from flags:

```text
voltr-finance transactions update
  --id 123
  --amount 40.00
  --description "Corrected groceries"
  --notes "Costco corrected"
  --clear-notes
  --author-id 1
  --household-id 1
  --clear-household
```

Bulk update from `--input` or stdin:

```text
voltr-finance transactions update-bulk --input updates.json
voltr-finance transactions update-bulk < updates.json
```

Reads and filtering:

```text
voltr-finance transactions get --ids 123
voltr-finance transactions get --ids 123,124 --json
voltr-finance transactions list --from 2026-01-01 --to 2026-01-31 --search paypal --sort transaction_date --order desc
voltr-finance transactions list --format csv --from 2026-01-01 --to 2026-01-31
```

List filters:

```text
--from
--to
--search
--description
--notes
--author-id
--author-discord-id
--author-telegram-id
--author-phone-number
--author-whatsapp-id
--household-id
--min-amount
--max-amount
--include-deleted
--only-deleted
--limit
--offset
--sort id|transaction_date|amount|created_at
--order asc|desc
--format json|csv
```

Default list ordering:

```text
--sort transaction_date --order desc
```

Soft delete and restore:

```text
voltr-finance transactions delete --ids 123,124 --reason "duplicate" --deleted-by-user-id 1
voltr-finance transactions restore --ids 123,124 --restored-by-user-id 1
```

### Users

User mutation commands use flags:

```text
voltr-finance users create --name "Rafael" --discord-id ... --telegram-id ... --phone-number ... --whatsapp-id ...
voltr-finance users update --id 1 --name "Rafael" --telegram-id ... --clear-phone-number
```

Clearing identity fields is explicit:

```text
--clear-discord-id
--clear-telegram-id
--clear-phone-number
--clear-whatsapp-id
```

User reads:

```text
voltr-finance users get --id 1
voltr-finance users resolve --discord-id ...
voltr-finance users resolve --telegram-id ...
voltr-finance users resolve --phone-number ...
voltr-finance users resolve --whatsapp-id ...
voltr-finance users list
```

No user delete in v1 because users are referenced by transactions and household membership.

### Households

Household operations are read-only in v1:

```text
voltr-finance households get --id 1
voltr-finance households get --guild-id ...
voltr-finance households get --name "Home"
voltr-finance households list
voltr-finance households users --id 1
```

Transaction commands can attach household context with:

```text
--household-id
```

Transaction mutation commands require `--household-id` in v1. Agents can call `households get --guild-id ...` first when they need to resolve a Discord guild into a household ID.
Agents can call `households get --name ...` when the household name is available instead of an ID.

## Bulk JSON Inputs

Bulk create input:

```json
{
  "transactions": [
    {
      "amount": 42.5,
      "authorTelegramId": "123456789",
      "transactionDate": "2026-05-05T14:30:00-04:00",
      "description": "Groceries",
      "notes": "Costco",
      "householdId": 1
    }
  ]
}
```

Bulk update input:

```json
{
  "transactions": [
    {
      "id": 123,
      "updates": {
        "amount": 40,
        "description": "Corrected groceries",
        "notes": null
      }
    }
  ]
}
```

Update DTOs must distinguish absent fields from explicit null fields.

## Output Contract

Write operations always return minimal JSON.

Example:

```json
{
  "createdIds": [101, 102],
  "updatedIds": [],
  "deletedIds": [],
  "restoredIds": [],
  "errors": []
}
```

Bulk partial failure:

```json
{
  "createdIds": [101],
  "updatedIds": [],
  "deletedIds": [],
  "restoredIds": [],
  "errors": [
    {
      "index": 1,
      "code": "duplicate_transaction",
      "message": "Transaction already exists",
      "id": 55
    }
  ]
}
```

Individual reads default to compact human-readable output and support `--json`.

`transactions get` defaults to compact human-readable output when exactly one ID is requested. When multiple IDs are requested, it returns JSON by default.

Example transaction text:

```text
Transaction #101
Amount: 42.50
Date: 2026-05-05 14:30
Author: Rafael
Household: Home
Description: Groceries
Notes: Costco
```

`transactions list` defaults to JSON and supports CSV:

```text
--format json
--format csv
```

Stable CSV columns:

```text
id,amount,transaction_date,author_id,author_name,household_id,household_name,description,notes,created_at,deleted_at
```

Other list operations return JSON in v1.

Exit codes:

```text
0 full success
1 unexpected runtime error
2 validation, not found, duplicate, or partial bulk failure
```

## Testing

Add focused tests around:

- Config path resolution.
- Identity resolution for Discord, Telegram, phone number, and WhatsApp ID.
- User create/update/resolve validation and uniqueness behavior.
- Transaction create/update/get/list/delete/restore use cases.
- Bulk JSON parsing and partial success errors.
- Soft-delete filtering defaults.
- CSV rendering for `transactions list`.
- CLI command parsing for representative commands.

Database behavior should be covered with integration tests where practical because identity resolution, unique indexes, and soft-delete filtering depend on SQL behavior.

## Open Decisions Deferred

- HTTP API transport.
- MCP transport.
- Natural-language transaction query using an LLM provider.
- CI release pipeline for host binaries.
- User delete semantics.
- Household mutation commands.
