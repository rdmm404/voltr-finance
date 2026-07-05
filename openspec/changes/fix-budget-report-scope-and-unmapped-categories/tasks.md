## 1. Report Queries

- [x] 1.1 Correct the owner-scope predicate in budget line and uncategorized report queries so personal budgets require transactions with no household assignment.
- [x] 1.2 Add a deterministic query that returns in-scope uncategorized and categorized-but-unmapped transactions with optional category details.
- [x] 1.3 Regenerate sqlc code and verify the generated repository methods and row types compile.

## 2. Report Response

- [x] 2.1 Add the unmapped transaction DTO/list and `unmappedActualAmount` total to the budget report response while retaining `uncategorizedActualAmount`.
- [x] 2.2 Extend the repository interface and test fake to retrieve unmapped transactions.
- [x] 2.3 Update `GetBudgetReport` to map unmapped rows, optional category references, deterministic amounts, and the unmapped subtotal.

## 3. Verification

- [x] 3.1 Add service tests covering unmapped transaction details, category presence/absence, totals, empty results, and preservation of the uncategorized subtotal.
- [x] 3.2 Add database-level coverage for personal versus household scope and for null-category, mapped-category, and unmapped-category transactions if the repository test harness supports query integration tests. (No database integration test harness is present; service coverage and generated-query verification apply.)
- [x] 3.3 Run formatting, sqlc generation consistency checks, and the full Go test suite.
