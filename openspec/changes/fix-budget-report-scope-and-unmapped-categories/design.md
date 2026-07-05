## Context

Budget report aggregation currently joins transactions to budget lines by category and limits dates correctly, but its personal-owner predicate accepts household transactions authored by the user. The separate uncategorized query repeats the same predicate and only detects null categories, so categorized transactions without a line mapping disappear from the report.

The report is assembled in the application service from sqlc query results and emitted as JSON by the CLI. The change crosses the database query, generated repository types, application DTOs, aggregation logic, and tests, but requires no schema migration.

## Goals / Non-Goals

**Goals:**

- Apply one exact owner-scope definition consistently to line totals, uncategorized totals, and unmapped transaction discovery.
- Return transaction-level details for both uncategorized and categorized-but-unmapped spending.
- Add an aggregate unmapped amount while preserving the existing uncategorized amount contract.
- Keep report ordering deterministic and behavior testable at query and service boundaries.

**Non-Goals:**

- Automatically assign categories to budget lines.
- Change transaction categorization or budget-line editing workflows.
- Change budget ownership rules or support budgets with both/neither owner populated.
- Redefine existing line `actualAmount`, total `actualAmount`, or `remainingAmount` semantics.

## Decisions

### Centralize the owner-scope SQL predicate

Every report query will use the same predicate: household budgets require `t.household_id = b.household_id`; personal budgets require both `t.author_id = b.user_id` and `t.household_id IS NULL`. This directly models the persisted ownership scope. Passing scope parameters from application code was considered, but deriving them from the selected budget avoids divergence and additional repository arguments.

### Discover unmapped transactions with an anti-join

A new sqlc query will select in-scope transactions and exclude any transaction for which a `budget_line_category` row exists for the same budget and category. A `NOT EXISTS` predicate expresses the rule without duplicate rows and naturally includes null categories. The query will join `category` for the optional category code and name and order results by transaction date and ID for stable output.

Extending the line aggregation query was considered, but combining line aggregates and transaction details would mix result granularities and complicate correctness.

### Add explicit unmapped DTOs without removing the uncategorized total

`BudgetReportDTO` will gain `unmappedTransactions`, containing transaction identity, date, description, amount, and an optional category reference. `BudgetReportTotalsDTO` will gain `unmappedActualAmount`. `uncategorizedActualAmount` remains a null-category-only subtotal for response compatibility.

The application service will derive `unmappedActualAmount` from the returned unmapped rows, keeping the list and total consistent. The existing uncategorized query remains, but receives the corrected scope predicate.

### Preserve existing aggregate semantics

Line actuals and total actual remain the sum of transactions mapped to budget lines; remaining remains allocation minus mapped actual. Unmapped spending is reported separately. Folding unmapped amounts into total actual was considered, but would silently alter established report semantics and make remaining allocation ambiguous.

## Risks / Trade-offs

- [The scope predicate could drift between queries later] → Cover household and personal cases for line totals, uncategorized totals, and unmapped rows with query-level integration tests where available, and keep predicates structurally identical.
- [The unmapped query adds work proportional to period transactions] → Use `NOT EXISTS` against the budget/category mapping keys and rely on existing transaction and mapping indexes; inspect the query plan if representative data shows a regression.
- [Keeping `uncategorizedActualAmount` alongside `unmappedActualAmount` can be confusing] → Document the former as a subset and the latter as the complete fall-through amount.

## Migration Plan

Regenerate sqlc code after changing queries, update the application response types and service, then deploy normally. No data backfill is required. Rollback consists of reverting the query and response additions; existing response fields remain compatible throughout.

## Open Questions

None.
