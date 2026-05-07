# Voltr Finance CLI

`voltr-finance` is a host-installed CLI for direct Postgres-backed finance operations.

## Install

Build the CLI entrypoint from the repository root:

```bash
go build -o /tmp/voltr-finance ./cmd/cli
```

The Dockerfile also provides a `cli` target that exports `/usr/local/bin/voltr-finance`.

## Configuration

The CLI reads a strict JSON config file. Config path lookup is:

1. `--config /path/to/config.json`
2. `VOLTR_CONFIG`
3. `$HOME/.config/voltr-finance/config.json`

Sample `config.json`:

```json
{
  "database": {
    "host": "localhost",
    "port": "5432",
    "name": "voltr_finance",
    "user": "voltr_cli_rw",
    "password": "change-me",
    "poolSize": 5
  }
}
```

The connection uses the `transactions` search path. `poolSize` defaults to `5` when omitted or zero.

## Transactions

Create a transaction:

```bash
/tmp/voltr-finance --config /tmp/voltr-finance.json transactions create \
  --amount 42.50 \
  --transaction-date 2026-05-05T14:30:00-04:00 \
  --description "Groceries" \
  --notes "Costco" \
  --author-telegram-id "123456789" \
  --household-id 1
```

List matching transactions as JSON:

```bash
/tmp/voltr-finance --config /tmp/voltr-finance.json transactions list \
  --from-date 2026-05-01T00:00:00-04:00 \
  --to-date 2026-05-31T23:59:59-04:00 \
  --search "Groceries" \
  --sort transaction_date \
  --order desc
```

List matching transactions as CSV:

```bash
/tmp/voltr-finance --config /tmp/voltr-finance.json transactions list \
  --format csv \
  --household-id 1
```

Soft-delete transactions:

```bash
/tmp/voltr-finance --config /tmp/voltr-finance.json transactions delete \
  --ids 123,124 \
  --reason "duplicate import" \
  --deleted-by-user-id 1
```

## Users

Resolve a user by provider identity:

```bash
/tmp/voltr-finance --config /tmp/voltr-finance.json users resolve \
  --telegram-id "123456789"
```

Other supported identity selectors are `--author-id`, `--discord-id`, `--phone-number`, and `--whatsapp-id`.

## Households

Look up a household by name:

```bash
/tmp/voltr-finance --config /tmp/voltr-finance.json households get \
  --name "Home"
```

The returned `id` can be passed to transaction commands as `--household-id`.

## Nanobot Mapping

Map Nanobot sender metadata to exactly one CLI identity flag.

Discord:

```bash
/tmp/voltr-finance --config /tmp/voltr-finance.json transactions create \
  --amount 12.99 \
  --transaction-date 2026-05-05T14:30:00-04:00 \
  --description "Discord purchase" \
  --author-discord-id "$sender_id" \
  --household-id 1
```

Telegram:

```bash
/tmp/voltr-finance --config /tmp/voltr-finance.json users resolve \
  --telegram-id "${metadata_user_id:-$sender_id}"
```

When Nanobot falls back to sender IDs like `123456789|rafael`, the application normalizes Telegram identity matching to the stable numeric ID.

WhatsApp phone number:

```bash
/tmp/voltr-finance --config /tmp/voltr-finance.json transactions create \
  --amount 18.00 \
  --transaction-date 2026-05-05T14:30:00-04:00 \
  --description "WhatsApp phone sender" \
  --author-phone-number "$sender_id" \
  --household-id 1
```

WhatsApp LID/JID:

```bash
/tmp/voltr-finance --config /tmp/voltr-finance.json transactions create \
  --amount 18.00 \
  --transaction-date 2026-05-05T14:30:00-04:00 \
  --description "WhatsApp JID sender" \
  --author-whatsapp-id "$sender_id" \
  --household-id 1
```
