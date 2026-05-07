---
name: voltr-finance
description: Use when a chat or automation request needs Voltr Finance transaction tracking, lookup, export, correction, soft deletion, restoration, user identity resolution, or household lookup through the voltr-finance CLI
requires:
  bins: ["voltr-finance"]
---

# Voltr Finance

## Overview

Use the `voltr-finance` CLI as the only write surface for finance data. Do not run SQL, invent flags, or use old bot tool names. Convert chat requests into explicit CLI operations, preserve useful partial transaction data, and ask only when required data cannot be resolved.

## Identity

Transaction writes need exactly one author identity.

| Nanobot source | CLI identity flag |
|---|---|
| Discord sender | `--author-discord-id "$sender_id"` |
| Telegram metadata user ID | `--author-telegram-id "$metadata_user_id"` |
| Telegram fallback sender like `123456789|name` | `--author-telegram-id "$sender_id"`; the app normalizes it |
| WhatsApp phone sender | `--author-phone-number "$sender_id"` |
| WhatsApp LID/JID sender | `--author-whatsapp-id "$sender_id"` |
| Known user row ID | `--author-id <id>` |

For `users resolve`, use selector names without `author-` except `--author-id`:

```bash
voltr-finance users resolve --telegram-id "123456789"
voltr-finance users resolve --discord-id "$sender_id"
voltr-finance users resolve --phone-number "$sender_id"
voltr-finance users resolve --whatsapp-id "$sender_id"
```

If a message says someone else paid, resolve that person before writing. Use `households users --household-id <id>` or `users list`, then pass `--author-id <id>`. Do not pass names with fake flags such as `--paid-by`.

## Transactions

Create requires `--amount`, `--transaction-date`, `--household-id`, and one author selector. Use RFC3339 timestamps with timezone.

```bash
voltr-finance transactions create \
  --amount 42.50 \
  --transaction-date 2026-05-05T14:30:00-04:00 \
  --description "Costco groceries" \
  --notes "receipt text or useful context" \
  --author-telegram-id "123456789" \
  --household-id 1
```

For receipts or messages containing several transactions, prefer one `create-bulk` call:

```json
{
  "transactions": [
    {
      "amount": 42.5,
      "transactionDate": "2026-05-05T14:30:00-04:00",
      "description": "Costco groceries",
      "notes": "receipt line context",
      "householdId": 1,
      "author": { "telegramId": "123456789" }
    }
  ]
}
```

Update existing transactions with `transactions update --id <id>`. To clear optional fields use `--clear-description`, `--clear-notes`, `--clear-budget-category-id`, or `--clear-household-id`; do not send empty strings to mean clear.

```bash
voltr-finance transactions update \
  --id 123 \
  --amount 40.00 \
  --description "Corrected groceries" \
  --author-id 7 \
  --household-id 1
```

## Lookup And Export

Use implemented list flags exactly: `--from-date`, `--to-date`, `--search`, `--author-id`, `--household-id`, `--sort`, `--order`, `--limit`, `--offset`, `--include-deleted`, `--only-deleted`, and `--format json|csv`.

```bash
voltr-finance transactions list \
  --format csv \
  --search paypal \
  --from-date 2026-04-01T00:00:00-04:00 \
  --to-date 2026-04-30T23:59:59-04:00 \
  --sort transaction_date \
  --order asc \
  --limit 10000
```

Use `transactions get --ids 123` or `transactions get --ids 123,124`. Add `--include-deleted` when checking deleted rows. Use `--format compact` only for one transaction.

## Households And Users

Resolve household context before writes. Create currently requires a household ID, so do not invent personal/no-household creates.

```bash
voltr-finance households get --name "Home"
voltr-finance households get --guild-id "$guild_id"
voltr-finance households list
voltr-finance households users --household-id 1
```

User commands:

```bash
voltr-finance users create --name "Val" --telegram-id "123456789"
voltr-finance users update --id 7 --phone-number "+14165550123"
voltr-finance users get --id 7
voltr-finance users list
```

## Deletion And Restore

Deletes are soft deletes. Always identify who requested the delete with `--deleted-by-user-id`; resolve the sender first if needed. There is no `show`, `find-duplicates`, `--dry-run`, `--yes`, or hard-delete command.

```bash
voltr-finance transactions get --ids 123
voltr-finance transactions delete \
  --ids 123 \
  --reason "duplicate reported by user" \
  --deleted-by-user-id 7
voltr-finance transactions restore --ids 123 --restored-by-user-id 7
```

## Behavior Rules

- Reply in the same language as the user.
- Before each CLI call, say briefly what you are doing.
- Prefer storing extracted transaction data over dropping it because some fields are missing.
- Ask a follow-up only when required fields cannot be resolved: amount, date, household ID, or author identity.
- Interpret relative dates using the current conversation date and timezone; include exact dates in commands.
- Report write results from JSON output, including any `errors`.
- If the CLI returns `validation_error`, `user_not_found`, or `transaction_not_found`, explain the concrete missing or invalid value and ask for the smallest next input.

## Common Mistakes

| Mistake | Correct action |
|---|---|
| `transactions add`, `show`, or `find-duplicates` | Use `create`, `get`, or `list` |
| `--date`, `--merchant`, `--category`, `--paid-by` | Use `--transaction-date`, `--description`, `--notes`, and author selectors |
| `--from` / `--to` | Use `--from-date` / `--to-date` |
| Passing payer names directly | Resolve to user ID, then use `--author-id` |
| Deleting without actor ID | Resolve requester, then pass `--deleted-by-user-id` |
| Assuming no household is allowed | Resolve or ask for `--household-id` |
