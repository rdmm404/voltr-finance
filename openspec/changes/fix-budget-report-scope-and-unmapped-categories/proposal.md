## Why

Budget reports can currently include personal transactions in a household budget, or household transactions in a personal budget, because ownership is inferred without requiring the transaction's scope to match the budget. Reports also hide categorized transactions when their categories are not assigned to a budget line, leaving unexplained gaps in reported spending.

## What Changes

- Require report transactions to match the budget owner scope: household budgets include only transactions assigned to that household, while personal budgets include only the owner's transactions that are not assigned to any household.
- Identify every in-scope, non-deleted transaction in the budget period whose category is either absent or not mapped to any line in that budget.
- Expose those unmapped transactions in the budget report, with enough transaction and category information to find and correct the missing mapping.
- Include all unmapped transaction amounts in report totals so line totals and unallocated spending can be reconciled against in-scope spending.

## Capabilities

### New Capabilities

- `budget-reporting`: Defines ownership-scope filtering, budget-line aggregation, and explicit reporting of transactions whose categories are not mapped to a budget line.

### Modified Capabilities

None.

## Impact

- Budget report SQL queries and generated sqlc code.
- Budget application DTOs, repository interface, aggregation logic, and tests.
- CLI JSON budget report output.
- No database schema migration or external dependency change is expected.
