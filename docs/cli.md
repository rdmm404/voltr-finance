# Voltr Finance CLI

`voltr-finance` is a standalone client for the authenticated Voltr Finance REST API. It never connects to PostgreSQL. Commands emit JSON unless a command explicitly supports compact or CSV output.

## Install

Build from the repository root:

```bash
go build -o /tmp/voltr-finance ./cmd/cli
```

The Dockerfile also provides a `cli` target containing `/usr/local/bin/voltr-finance`.

## Configuration

The CLI reads a strict JSON config. Config-path precedence is:

1. `--config /path/to/config.json`
2. `VOLTR_CONFIG`
3. `$HOME/.config/voltr-finance/config.json`

```json
{
  "api": {
    "baseUrl": "https://finance.example.com",
    "apiKey": "replace-with-a-secret"
  }
}
```

Non-empty `VOLTR_API_URL` and `VOLTR_API_KEY` values override the corresponding file settings. Use HTTPS outside trusted local development; bearer credentials are sent on every finance request. Help commands do not require configuration.

Examples below use:

```bash
VOLTR="/tmp/voltr-finance --config /tmp/voltr-finance.json"
```

## Exit status and bulk results

- `0`: complete success
- `2`: usage/validation error or one or more failed bulk items
- `1`: configuration, authentication, transport, server, or unexpected failure

Bulk commands always print the complete `succeeded` and `failed` arrays before returning exit status `2` for item failures.

## Transactions

Transaction author selectors are `--author-id`, `--author-discord-id`, `--author-telegram-id`, `--author-phone-number`, and `--author-whatsapp-id`. Creates require exactly one author selector. Updates require exactly one only when changing the author.

Create a transaction:

```bash
$VOLTR transactions create \
  --amount 42.50 \
  --transaction-date 2026-05-05T14:30:00-04:00 \
  --description "Groceries" \
  --notes "Costco" \
  --category groceries \
  --author-telegram-id "123456789" \
  --household-id 1
```

For negative amounts, use `--amount=-12.34` so the value is not parsed as a flag.

Create transactions in bulk from a file, or omit `--input` to read from stdin:

```bash
$VOLTR transactions create-bulk --input /tmp/transactions-create.json
```

Expected JSON shape:

```json
{
  "transactions": [
    {
      "amount": 42.5,
      "transactionDate": "2026-05-05T14:30:00-04:00",
      "description": "Groceries",
      "notes": "Costco",
      "categoryCode": "groceries",
      "householdId": 1,
      "author": { "telegramId": "123456789" }
    }
  ]
}
```

Get transactions by ID:

```bash
$VOLTR transactions get --ids 101,102,103
$VOLTR transactions get --ids 101 --format compact
$VOLTR transactions get --ids 101 --include-deleted
```

List matching transactions as JSON:

```bash
$VOLTR transactions list \
  --from-date 2026-05-01T00:00:00-04:00 \
  --to-date 2026-05-31T23:59:59-04:00 \
  --search "Groceries" \
  --sort transaction_date \
  --order desc \
  --limit 100 \
  --offset 0
```

List matching transactions as CSV:

```bash
$VOLTR transactions list \
  --format csv \
  --household-id 1
```

Useful list filters:

- `--author-id INT-64`
- `--household-id INT-64`
- `--from-date RFC3339`
- `--to-date RFC3339`
- `--search STRING`
- `--include-deleted`
- `--only-deleted`

Sort fields are `transaction_date`, `created_at`, `amount`, and `id`. Sort order is `asc` or `desc`.

Update one transaction:

```bash
$VOLTR transactions update \
  --id 101 \
  --amount 45.00 \
  --description "Updated groceries" \
  --category groceries
```

Clear nullable fields:

```bash
$VOLTR transactions update \
  --id 101 \
  --clear-description \
  --clear-notes \
  --clear-category \
  --clear-household-id
```

Update transactions in bulk from a file, or omit `--input` to read from stdin:

```bash
$VOLTR transactions update-bulk --input /tmp/transactions-update.json
```

Expected JSON shape:

```json
{
  "transactions": [
    {
      "id": 101,
      "amount": 45.0,
      "categoryCode": "groceries"
    }
  ]
}
```

Soft-delete transactions:

```bash
$VOLTR transactions delete \
  --ids 123,124 \
  --reason "duplicate import" \
  --deleted-by-user-id 1
```

Restore soft-deleted transactions:

```bash
$VOLTR transactions restore \
  --ids 123,124 \
  --restored-by-user-id 1
```

## Users

Create a user:

```bash
$VOLTR users create \
  --name "Rafael" \
  --telegram-id "123456789"
```

Supported external identity flags are `--discord-id`, `--telegram-id`, `--phone-number`, and `--whatsapp-id`.

Update a user:

```bash
$VOLTR users update \
  --id 4 \
  --name "Rafael M" \
  --discord-id "987654321"
```

Clear external identities:

```bash
$VOLTR users update \
  --id 4 \
  --clear-discord-id \
  --clear-telegram-id \
  --clear-phone-number \
  --clear-whatsapp-id
```

Get or list users:

```bash
$VOLTR users get --id 4
$VOLTR users list
```

Resolve a user by exactly one identity selector:

```bash
$VOLTR users resolve --telegram-id "123456789"
$VOLTR users resolve --author-id 4
```

Other supported resolve selectors are `--discord-id`, `--phone-number`, and `--whatsapp-id`.

## Households

Look up a household by exactly one selector:

```bash
$VOLTR households get --name "Home"
$VOLTR households get --id 1
$VOLTR households get --guild-id "1234567890"
```

List households and household users:

```bash
$VOLTR households list
$VOLTR households users --household-id 1
```

A household `id` can be passed to transaction commands as `--household-id` and to household budget commands as `--household-id`.

## Categories

Create a category. If `--code` is omitted, the app generates a lowercase slug from the name.

```bash
$VOLTR categories create "Groceries" \
  --code groceries \
  --description "Food and household supplies"
```

List active categories, or include inactive ones:

```bash
$VOLTR categories list
$VOLTR categories list --include-inactive
```

Rename a category by code:

```bash
$VOLTR categories rename groceries "Groceries and Supplies"
```

Deactivate a category by code:

```bash
$VOLTR categories deactivate groceries
```

Category codes can be passed to transaction commands as `--category` and to budget line commands as comma-separated `--categories` values.

## Budgets

Budgets are monthly and owned by exactly one household or user. `--month` uses `YYYY-MM`.

Get an existing household monthly budget:

```bash
$VOLTR budgets get \
  --household-id 1 \
  --month 2026-05
```

Ensure a household monthly budget exists. `--create` calls the idempotent ensure endpoint directly; without it, the CLI performs a read-only GET and a missing budget remains missing. A newly ensured month copies the latest prior budget structure for the same owner, or starts empty when no prior budget exists.

```bash
$VOLTR budgets get \
  --household-id 1 \
  --month 2026-05 \
  --create
```

Get or create a personal monthly budget:

```bash
$VOLTR budgets get \
  --user-id 4 \
  --month 2026-05 \
  --create
```

Add a budget line. Amounts are decimal strings with at most two decimal places. Category inputs use comma-separated category codes. A category can appear on only one line within the same budget.

```bash
$VOLTR budgets lines add \
  --budget-id 12 \
  --name "Groceries" \
  --amount 800.00 \
  --categories groceries,costco \
  --sort-order 10
```

Update a budget line by line ID:

```bash
$VOLTR budgets lines update 44 \
  --name "Food" \
  --amount 900.00 \
  --categories groceries,restaurants \
  --sort-order 20
```

Passing `--categories` on update replaces the line's category mappings. Passing `--categories ""` clears them.

Delete a budget line by line ID:

```bash
$VOLTR budgets lines delete 44
```

Show budget actuals:

```bash
$VOLTR budgets report 12
```

The report returns budget metadata, report lines, and totals. Line actuals are derived from categorized transactions in the budget period. Transactions without categories are reported separately in `totals.uncategorizedActualAmount`.

## Nanobot Mapping

Map Nanobot sender metadata to exactly one CLI identity flag.

Discord:

```bash
$VOLTR transactions create \
  --amount 12.99 \
  --transaction-date 2026-05-05T14:30:00-04:00 \
  --description "Discord purchase" \
  --author-discord-id "$sender_id" \
  --household-id 1
```

Telegram:

```bash
$VOLTR users resolve \
  --telegram-id "${metadata_user_id:-$sender_id}"
```

When Nanobot falls back to sender IDs like `123456789|rafael`, the application normalizes Telegram identity matching to the stable numeric ID.

WhatsApp phone number:

```bash
$VOLTR transactions create \
  --amount 18.00 \
  --transaction-date 2026-05-05T14:30:00-04:00 \
  --description "WhatsApp phone sender" \
  --author-phone-number "$sender_id" \
  --household-id 1
```

WhatsApp LID/JID:

```bash
$VOLTR transactions create \
  --amount 18.00 \
  --transaction-date 2026-05-05T14:30:00-04:00 \
  --description "WhatsApp JID sender" \
  --author-whatsapp-id "$sender_id" \
  --household-id 1
```
