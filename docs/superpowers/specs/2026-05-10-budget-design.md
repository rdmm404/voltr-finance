# Budget Design

## Purpose

Add simple monthly budgets for both households and individual users. A budget is a dated allocation plan made of budget lines such as rent, utilities, groceries, dates, dog food, and savings.

Budgets integrate with the existing transaction category feature. Transactions do not point directly to budget lines. Instead, budget actuals are derived from transaction categories mapped to each budget line.

## Goals

- Support household budgets and personal budgets.
- Store budgets as monthly snapshots with explicit date ranges.
- Keep the app surface monthly-only for now, even though the data model stores date ranges.
- Automatically create a missing monthly budget from the latest prior budget for the same owner when requested with create behavior.
- Let each budget line map to zero, one, or many existing categories.
- Prevent one category from counting against multiple lines in the same budget.
- Keep savings possible as an unmapped budget line.
- Keep salary out of the first budget model.
- Derive actual spending from categorized transactions.

## Non-Goals

- No income or salary tracking.
- No budget-line split allocation by user or percentage.
- No direct `transaction.budget_line_id` reference.
- No custom non-monthly periods in the app API.
- No scheduler that pre-creates future budgets.
- No hard requirement to support bulk replacement as the primary editing path.

## Data Model

Reuse the existing `budget` table concept, but make it the monthly budget snapshot record.

```sql
CREATE TABLE budget (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    household_id BIGINT REFERENCES household(id),
    user_id BIGINT REFERENCES users(id),
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    source_budget_id BIGINT REFERENCES budget(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CHECK (
        (household_id IS NOT NULL AND user_id IS NULL)
        OR
        (household_id IS NULL AND user_id IS NOT NULL)
    ),
    CHECK (period_end >= period_start)
);
```

Use partial unique indexes so each owner has at most one budget for a date range:

```sql
CREATE UNIQUE INDEX idx_budget_household_period
ON budget(household_id, period_start, period_end)
WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX idx_budget_user_period
ON budget(user_id, period_start, period_end)
WHERE user_id IS NOT NULL;
```

Budget lines are normalized records. Each line is a planned allocation in one monthly budget.

```sql
CREATE TABLE budget_line (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    budget_id BIGINT NOT NULL REFERENCES budget(id) ON DELETE CASCADE,
    name VARCHAR NOT NULL,
    allocation_amount NUMERIC(12, 2) NOT NULL,
    sort_order INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CHECK (allocation_amount >= 0),
    UNIQUE (budget_id, id),
    UNIQUE (budget_id, sort_order)
);
```

Budget line category mappings connect allocation lines to existing transaction categories.

```sql
CREATE TABLE budget_line_category (
    budget_id BIGINT NOT NULL REFERENCES budget(id) ON DELETE CASCADE,
    budget_line_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL REFERENCES category(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (budget_line_id, category_id),
    FOREIGN KEY (budget_id, budget_line_id)
        REFERENCES budget_line(budget_id, id)
        ON DELETE CASCADE,
    UNIQUE (budget_id, category_id)
);
```

The denormalized `budget_id` on `budget_line_category` is intentional. It lets the database enforce that a category can only be mapped once within a budget, which prevents double-counted actuals.

## Monthly Period Rules

The database stores `period_start` and `period_end`, but the app only exposes monthly budgets in the first version.

For a requested `year` and `month`, the service computes:

```text
period_start = first day of requested month
period_end = last day of requested month
```

Example:

```text
May 2026 -> period_start 2026-05-01, period_end 2026-05-31
```

The first implementation does not need database overlap constraints because app-created budgets are monthly and exact owner/range uniqueness is enough for the supported behavior.

## Service Behavior

### Get Monthly Budget

```go
GetMonthlyBudget(ctx, req GetMonthlyBudgetRequest) (BudgetDTO, error)
```

Request fields:

- `HouseholdID *int64`
- `UserID *int64`
- `Year int`
- `Month int`
- `CreateIfMissing bool`

Behavior:

- Validate exactly one owner selector is present.
- Convert `Year` and `Month` into the monthly period range.
- Return the existing budget when found.
- If no budget exists and `CreateIfMissing` is false, return a not-found error.
- If no budget exists and `CreateIfMissing` is true:
  - find the latest prior budget for the same owner by `period_start`;
  - create a new budget for the requested month;
  - set `source_budget_id` to the copied budget when one exists;
  - copy all budget lines as new rows;
  - copy all category mappings for those copied lines;
  - create an empty budget when no prior budget exists.

Copying from the latest prior budget should work even when there are month gaps. If May exists and June does not, requesting July with create behavior should copy May.

### Create Budget Line

```go
CreateBudgetLine(ctx, req CreateBudgetLineRequest) (BudgetLineDTO, error)
```

Request fields:

- `BudgetID int64`
- `Name string`
- `AllocationAmount decimal`
- `CategoryIDs []int64`
- `CategoryCodes []string`
- `SortOrder *int`

Behavior:

- Validate the budget exists.
- Validate name is present.
- Validate allocation amount is greater than or equal to zero.
- Resolve supplied category codes to active categories.
- Validate supplied category IDs reference active categories.
- Validate no category is already mapped to a different line in the same budget.
- If `SortOrder` is omitted, append the line after the current maximum sort order for the budget.
- Create the line and mappings in one transaction.

Category codes are expected for CLI ergonomics. Database rows store category IDs.

### Update Budget Line

```go
UpdateBudgetLine(ctx, req UpdateBudgetLineRequest) (BudgetLineDTO, error)
```

Request fields:

- `LineID int64`
- `Name *string`
- `AllocationAmount *decimal`
- `CategoryIDs *[]int64`
- `CategoryCodes *[]string`
- `SortOrder *int`

Behavior:

- Validate the line exists.
- Apply only provided fields.
- If categories are provided, replace mappings for that line only.
- Validate replacement categories are active and not already mapped to another line in the same budget.
- Do not affect other budget lines.

### Delete Budget Line

```go
DeleteBudgetLine(ctx, lineID int64) error
```

Behavior:

- Delete the line and its category mappings.
- Do not modify transactions.
- Hard delete is acceptable in the first version because budget lines are editable planning rows within a monthly snapshot.

### Get Budget Report

```go
GetBudgetReport(ctx, budgetID int64) (BudgetReportDTO, error)
```

Behavior:

- Load the budget, lines, and category mappings.
- Load transactions for the same owner and period.
- Ignore deleted transactions.
- Ignore transactions without categories for line actuals.
- Match transactions to budget lines through `transaction.category_id` and `budget_line_category.category_id`.
- Sum transaction amounts directly:
  - positive expense amounts increase actual spending;
  - negative refund amounts reduce actual spending.
- Return allocated amount, actual amount, and remaining amount for each line.
- Include report totals.
- Include uncategorized actual amount separately if useful for visibility, but do not count it against any line.

Remaining amount is:

```text
allocation_amount - actual_amount
```

## DTO Shape

Budget responses should include the owner, period, source budget, and lines.

```json
{
  "id": 12,
  "householdId": 1,
  "periodStart": "2026-05-01",
  "periodEnd": "2026-05-31",
  "sourceBudgetId": 11,
  "lines": [
    {
      "id": 44,
      "name": "Groceries",
      "allocationAmount": "800.00",
      "sortOrder": 1,
      "categories": [
        { "id": 1, "code": "groceries", "name": "Groceries" }
      ]
    }
  ]
}
```

Report responses should include the same line structure with actuals.

```json
{
  "budget": {
    "id": 12,
    "householdId": 1,
    "periodStart": "2026-05-01",
    "periodEnd": "2026-05-31",
    "sourceBudgetId": 11
  },
  "lines": [
    {
      "id": 44,
      "name": "Groceries",
      "allocationAmount": "800.00",
      "actualAmount": "570.25",
      "remainingAmount": "229.75",
      "sortOrder": 1,
      "categories": [
        { "id": 1, "code": "groceries", "name": "Groceries" }
      ]
    }
  ],
  "totals": {
    "allocationAmount": "4000.00",
    "actualAmount": "2850.00",
    "remainingAmount": "1150.00",
    "uncategorizedActualAmount": "123.45"
  }
}
```

## CLI Behavior

Use a `budgets` command group.

Get or create a household monthly budget:

```bash
voltr-finance budgets get --household-id 1 --month 2026-05 --create
```

Get or create a personal monthly budget:

```bash
voltr-finance budgets get --user-id 4 --month 2026-05 --create
```

Get a report:

```bash
voltr-finance budgets report 12
```

Add a line:

```bash
voltr-finance budgets lines add \
  --budget-id 12 \
  --name "Groceries" \
  --amount 800 \
  --categories groceries,costco
```

Update a line by positional line ID:

```bash
voltr-finance budgets lines update 44 --amount 900
voltr-finance budgets lines update 44 --categories groceries,costco,supermarket
```

Delete a line by positional line ID:

```bash
voltr-finance budgets lines delete 44
```

Category inputs in the CLI should use category codes. The service resolves them to category IDs before storing mappings.

## Validation and Errors

Budget validation:

- Exactly one owner selector is required.
- Year and month must identify a valid calendar month.
- The owner must exist.
- A duplicate owner/month budget should be treated as the existing budget during create-if-missing behavior.

Line validation:

- Name is required.
- Allocation amount must be greater than or equal to zero.
- Category IDs/codes must reference active categories.
- A category cannot appear in more than one line in the same budget.
- Sort order must be unique within a budget when explicitly provided.

Use existing app error patterns for validation, not-found, and database errors.

## Testing

Add focused tests for:

- Monthly period calculation.
- Budget owner validation for household and personal budgets.
- Creating an empty monthly budget when no prior budget exists.
- Auto-copying from the latest prior budget with copied lines and category mappings.
- Auto-copying from latest prior budget when month gaps exist.
- Creating a budget line with category codes.
- Rejecting inactive or unknown categories.
- Rejecting category reuse across lines in the same budget.
- Updating one line without affecting other lines.
- Replacing category mappings for one line.
- Deleting a line without touching transactions.
- Budget report actuals derived from transaction categories.
- Negative refund transactions reducing actual spending.
- Uncategorized transactions excluded from line actuals and surfaced separately.
- CLI parsing for `budgets get`, `budgets report`, `budgets lines add`, `budgets lines update`, and `budgets lines delete`.

## Future Extensions

- Bulk import or set-all-lines helper for initial setup.
- Split allocation per budget line by user, percentage, or fixed amount.
- Budget templates if monthly copy behavior stops being enough.
- Manual transaction-to-budget override for exceptional transactions.
- Income/cashflow tracking for salary and other income sources.
- Gross spend and refund breakdown in budget reports.
