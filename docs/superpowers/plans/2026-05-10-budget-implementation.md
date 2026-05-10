# Budget Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build monthly household and personal budgets with normalized budget lines, category mappings, auto-copy from the latest prior budget, line-level CLI operations, and derived budget reports.

**Architecture:** Budgets are monthly owner-scoped snapshots stored in the existing `budget` table, with `budget_line` and `budget_line_category` tables for planned allocations and category mappings. Transactions remain linked only to categories; budget actuals are derived by joining transaction categories to budget line category mappings for the budget owner and period.

**Tech Stack:** Go, PostgreSQL migrations managed by dbmate, sqlc v1.29, pgx/v5, Kong CLI, standard Go tests.

---

## File Map

- Create `db/migrations/20260510000000_monthly_budgets.sql`: reshape `budget`, add `budget_line`, add `budget_line_category`, and add indexes/constraints.
- Modify `internal/database/query.sql`: add budget, budget line, budget category mapping, copy, and report queries.
- Regenerate `internal/database/sqlc/models.go` and `internal/database/sqlc/query.sql.go` with `sqlc generate`.
- Modify `internal/app/service.go`: add `BudgetRepository` methods to the service repository interface.
- Create `internal/app/budgets.go`: budget DTOs, request types, monthly period calculation, owner validation, service methods, line validation, copy behavior, and report assembly.
- Create `internal/app/budgets_test.go`: service-level tests for monthly creation/copying, line operations, validation, and report math.
- Modify `internal/app/test_fakes_test.go`: add fake budget repository state and methods.
- Modify `internal/cli/commands.go`: add `budgets` command group and budget methods to `AppService`.
- Modify `internal/cli/commands_test.go`: add CLI parser tests for `budgets get`, `budgets report`, `budgets lines add`, `budgets lines update`, and `budgets lines delete`.
- Modify `docs/cli.md`: document basic budget CLI usage.

---

### Task 1: Database Migration and sqlc Queries

**Files:**
- Create: `db/migrations/20260510000000_monthly_budgets.sql`
- Modify: `internal/database/query.sql`
- Generate: `internal/database/sqlc/models.go`
- Generate: `internal/database/sqlc/query.sql.go`

- [ ] **Step 1: Write the migration**

Create `db/migrations/20260510000000_monthly_budgets.sql`:

```sql
-- migrate:up
SET search_path TO transactions, public;

ALTER TABLE budget
    DROP COLUMN type,
    ADD COLUMN period_start DATE,
    ADD COLUMN period_end DATE,
    ADD COLUMN source_budget_id BIGINT REFERENCES budget(id),
    ADD CONSTRAINT budget_exactly_one_owner CHECK (
        (household_id IS NOT NULL AND user_id IS NULL)
        OR
        (household_id IS NULL AND user_id IS NOT NULL)
    ),
    ADD CONSTRAINT budget_valid_period CHECK (period_end >= period_start);

UPDATE budget
SET period_start = CURRENT_DATE,
    period_end = CURRENT_DATE
WHERE period_start IS NULL
   OR period_end IS NULL;

ALTER TABLE budget
    ALTER COLUMN period_start SET NOT NULL,
    ALTER COLUMN period_end SET NOT NULL;

CREATE UNIQUE INDEX idx_budget_household_period
ON budget(household_id, period_start, period_end)
WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX idx_budget_user_period
ON budget(user_id, period_start, period_end)
WHERE user_id IS NOT NULL;

CREATE INDEX idx_budget_household_period_start
ON budget(household_id, period_start)
WHERE household_id IS NOT NULL;

CREATE INDEX idx_budget_user_period_start
ON budget(user_id, period_start)
WHERE user_id IS NOT NULL;

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

CREATE INDEX idx_budget_line_budget_id
ON budget_line(budget_id);

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

CREATE INDEX idx_budget_line_category_budget_id
ON budget_line_category(budget_id);

CREATE INDEX idx_budget_line_category_category_id
ON budget_line_category(category_id);

-- migrate:down
SET search_path TO transactions, public;

DROP INDEX IF EXISTS idx_budget_line_category_category_id;
DROP INDEX IF EXISTS idx_budget_line_category_budget_id;
DROP TABLE IF EXISTS budget_line_category;

DROP INDEX IF EXISTS idx_budget_line_budget_id;
DROP TABLE IF EXISTS budget_line;

DROP INDEX IF EXISTS idx_budget_user_period_start;
DROP INDEX IF EXISTS idx_budget_household_period_start;
DROP INDEX IF EXISTS idx_budget_user_period;
DROP INDEX IF EXISTS idx_budget_household_period;

ALTER TABLE budget
    DROP CONSTRAINT IF EXISTS budget_valid_period,
    DROP CONSTRAINT IF EXISTS budget_exactly_one_owner,
    DROP COLUMN source_budget_id,
    DROP COLUMN period_end,
    DROP COLUMN period_start,
    ADD COLUMN type VARCHAR(50) NOT NULL DEFAULT 'monthly';
```

- [ ] **Step 2: Run migration syntax generation check**

Run:

```bash
sqlc generate
```

Expected: this may fail because `internal/database/query.sql` does not reference the new tables yet, but it must not fail with migration syntax errors.

- [ ] **Step 3: Add budget queries to `internal/database/query.sql`**

Add this section before the transaction section:

```sql
-- ******************* budget *******************
-- READS

-- name: GetHouseholdBudgetByPeriod :one
SELECT * FROM budget
WHERE household_id = $1
  AND user_id IS NULL
  AND period_start = $2
  AND period_end = $3;

-- name: GetUserBudgetByPeriod :one
SELECT * FROM budget
WHERE user_id = $1
  AND household_id IS NULL
  AND period_start = $2
  AND period_end = $3;

-- name: GetBudgetById :one
SELECT * FROM budget
WHERE id = $1;

-- name: GetLatestPriorHouseholdBudget :one
SELECT * FROM budget
WHERE household_id = $1
  AND user_id IS NULL
  AND period_start < $2
ORDER BY period_start DESC, id DESC
LIMIT 1;

-- name: GetLatestPriorUserBudget :one
SELECT * FROM budget
WHERE user_id = $1
  AND household_id IS NULL
  AND period_start < $2
ORDER BY period_start DESC, id DESC
LIMIT 1;

-- name: ListBudgetLines :many
SELECT * FROM budget_line
WHERE budget_id = $1
ORDER BY sort_order ASC, id ASC;

-- name: ListBudgetLineCategories :many
SELECT
    blc.budget_id,
    blc.budget_line_id,
    c.id AS category_id,
    c.code AS category_code,
    c.name AS category_name
FROM budget_line_category blc
JOIN category c ON c.id = blc.category_id
WHERE blc.budget_id = $1
ORDER BY blc.budget_line_id ASC, c.name ASC, c.id ASC;

-- name: GetBudgetLineById :one
SELECT * FROM budget_line
WHERE id = $1;

-- name: GetMaxBudgetLineSortOrder :one
SELECT COALESCE(MAX(sort_order), 0)::INT
FROM budget_line
WHERE budget_id = $1;

-- name: ListBudgetTransactions :many
SELECT
    t.category_id,
    SUM(t.amount)::REAL AS actual_amount
FROM transaction t
WHERE t.deleted_at IS NULL
  AND t.category_id IS NOT NULL
  AND t.transaction_date >= sqlc.arg(period_start)::DATE
  AND t.transaction_date < (sqlc.arg(period_end)::DATE + INTERVAL '1 day')
  AND (
    (sqlc.narg(household_id)::BIGINT IS NOT NULL AND t.household_id = sqlc.narg(household_id)::BIGINT)
    OR
    (sqlc.narg(user_id)::BIGINT IS NOT NULL AND t.author_id = sqlc.narg(user_id)::BIGINT)
  )
GROUP BY t.category_id;

-- name: SumUncategorizedBudgetTransactions :one
SELECT COALESCE(SUM(t.amount), 0)::REAL AS actual_amount
FROM transaction t
WHERE t.deleted_at IS NULL
  AND t.category_id IS NULL
  AND t.transaction_date >= sqlc.arg(period_start)::DATE
  AND t.transaction_date < (sqlc.arg(period_end)::DATE + INTERVAL '1 day')
  AND (
    (sqlc.narg(household_id)::BIGINT IS NOT NULL AND t.household_id = sqlc.narg(household_id)::BIGINT)
    OR
    (sqlc.narg(user_id)::BIGINT IS NOT NULL AND t.author_id = sqlc.narg(user_id)::BIGINT)
  );

-- WRITES

-- name: CreateHouseholdBudget :one
INSERT INTO budget (household_id, period_start, period_end, source_budget_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateUserBudget :one
INSERT INTO budget (user_id, period_start, period_end, source_budget_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateBudgetLine :one
INSERT INTO budget_line (budget_id, name, allocation_amount, sort_order)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateBudgetLine :one
UPDATE budget_line
SET
    name = CASE
        WHEN sqlc.arg(set_name)::bool THEN sqlc.arg(name)::VARCHAR
        ELSE name
    END,
    allocation_amount = CASE
        WHEN sqlc.arg(set_allocation_amount)::bool THEN sqlc.arg(allocation_amount)::NUMERIC(12, 2)
        ELSE allocation_amount
    END,
    sort_order = CASE
        WHEN sqlc.arg(set_sort_order)::bool THEN sqlc.arg(sort_order)::INTEGER
        ELSE sort_order
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)::BIGINT
RETURNING *;

-- name: DeleteBudgetLine :exec
DELETE FROM budget_line
WHERE id = $1;

-- name: DeleteBudgetLineCategories :exec
DELETE FROM budget_line_category
WHERE budget_line_id = $1;

-- name: CreateBudgetLineCategory :exec
INSERT INTO budget_line_category (budget_id, budget_line_id, category_id)
VALUES ($1, $2, $3);
```

- [ ] **Step 4: Generate sqlc output**

Run:

```bash
sqlc generate
```

Expected: exit 0. `internal/database/sqlc/models.go` contains `BudgetLine` and `BudgetLineCategory`, and `Budget` has `PeriodStart`, `PeriodEnd`, and `SourceBudgetID` fields.

- [ ] **Step 5: Commit database and query changes**

Run:

```bash
git add db/migrations/20260510000000_monthly_budgets.sql internal/database/query.sql internal/database/sqlc
git commit -m "Add budget schema and queries"
```

Expected: commit succeeds.

---

### Task 2: App Budget Types and Monthly Budget Retrieval

**Files:**
- Modify: `internal/app/service.go`
- Create: `internal/app/budgets.go`
- Create: `internal/app/budgets_test.go`
- Modify: `internal/app/test_fakes_test.go`

- [ ] **Step 1: Add failing tests for owner validation and monthly period calculation**

Create `internal/app/budgets_test.go`:

```go
package app

import (
	"context"
	"testing"
	"time"

	"rdmm404/voltr-finance/internal/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestMonthlyBudgetPeriod(t *testing.T) {
	start, end, err := monthlyBudgetPeriod(2026, 5)
	if err != nil {
		t.Fatalf("monthlyBudgetPeriod returned error: %v", err)
	}
	if start != time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("start = %s, want 2026-05-01", start)
	}
	if end != time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("end = %s, want 2026-05-31", end)
	}
}

func TestMonthlyBudgetPeriodRejectsInvalidMonth(t *testing.T) {
	_, _, err := monthlyBudgetPeriod(2026, 13)
	if appErr, ok := err.(*AppError); !ok || appErr.Code != CodeValidationError {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestGetMonthlyBudgetRejectsMissingOwner(t *testing.T) {
	svc := NewService(&fakeRepo{}, &fakeTransactionService{})

	_, err := svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		Year:  2026,
		Month: 5,
	})

	if appErr, ok := err.(*AppError); !ok || appErr.Code != CodeValidationError {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestGetMonthlyBudgetRejectsMultipleOwners(t *testing.T) {
	householdID := int64(1)
	userID := int64(2)
	svc := NewService(&fakeRepo{}, &fakeTransactionService{})

	_, err := svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		HouseholdID: &householdID,
		UserID:      &userID,
		Year:        2026,
		Month:       5,
	})

	if appErr, ok := err.(*AppError); !ok || appErr.Code != CodeValidationError {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestGetMonthlyBudgetReturnsExistingHouseholdBudget(t *testing.T) {
	householdID := int64(1)
	repo := &fakeRepo{
		householdBudgetByPeriod: sqlc.Budget{
			ID:          12,
			HouseholdID: &householdID,
			PeriodStart: pgtype.Date{Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			PeriodEnd:   pgtype.Date{Time: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC), Valid: true},
		},
	}
	svc := NewService(repo, &fakeTransactionService{})

	budget, err := svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		HouseholdID: &householdID,
		Year:        2026,
		Month:       5,
	})

	if err != nil {
		t.Fatalf("GetMonthlyBudget returned error: %v", err)
	}
	if budget.ID != 12 || budget.HouseholdID == nil || *budget.HouseholdID != 1 {
		t.Fatalf("budget = %+v, want household budget 12", budget)
	}
	if !repo.lastGetHouseholdBudgetStart.Equal(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("period start = %s, want 2026-05-01", repo.lastGetHouseholdBudgetStart)
	}
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test ./internal/app
```

Expected: FAIL with undefined names such as `monthlyBudgetPeriod`, `GetMonthlyBudgetRequest`, and `GetMonthlyBudget`.

- [ ] **Step 3: Add budget repository methods to `internal/app/service.go`**

Modify the repository composition:

```go
type Repository interface {
	UserRepository
	HouseholdRepository
	TransactionRepository
	CategoryRepository
	BudgetRepository
}
```

Add the interface:

```go
type BudgetRepository interface {
	GetHouseholdBudgetByPeriod(context.Context, sqlc.GetHouseholdBudgetByPeriodParams) (sqlc.Budget, error)
	GetUserBudgetByPeriod(context.Context, sqlc.GetUserBudgetByPeriodParams) (sqlc.Budget, error)
	GetBudgetById(context.Context, int64) (sqlc.Budget, error)
	GetLatestPriorHouseholdBudget(context.Context, sqlc.GetLatestPriorHouseholdBudgetParams) (sqlc.Budget, error)
	GetLatestPriorUserBudget(context.Context, sqlc.GetLatestPriorUserBudgetParams) (sqlc.Budget, error)
	ListBudgetLines(context.Context, int64) ([]sqlc.BudgetLine, error)
	ListBudgetLineCategories(context.Context, int64) ([]sqlc.ListBudgetLineCategoriesRow, error)
	GetBudgetLineById(context.Context, int64) (sqlc.BudgetLine, error)
	GetMaxBudgetLineSortOrder(context.Context, int64) (int32, error)
	CreateHouseholdBudget(context.Context, sqlc.CreateHouseholdBudgetParams) (sqlc.Budget, error)
	CreateUserBudget(context.Context, sqlc.CreateUserBudgetParams) (sqlc.Budget, error)
	CreateBudgetLine(context.Context, sqlc.CreateBudgetLineParams) (sqlc.BudgetLine, error)
	UpdateBudgetLine(context.Context, sqlc.UpdateBudgetLineParams) (sqlc.BudgetLine, error)
	DeleteBudgetLine(context.Context, int64) error
	DeleteBudgetLineCategories(context.Context, int64) error
	CreateBudgetLineCategory(context.Context, sqlc.CreateBudgetLineCategoryParams) error
	ListBudgetTransactions(context.Context, sqlc.ListBudgetTransactionsParams) ([]sqlc.ListBudgetTransactionsRow, error)
	SumUncategorizedBudgetTransactions(context.Context, sqlc.SumUncategorizedBudgetTransactionsParams) (float32, error)
}
```

- [ ] **Step 4: Add budget DTOs and monthly retrieval in `internal/app/budgets.go`**

Create `internal/app/budgets.go`:

```go
package app

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"rdmm404/voltr-finance/internal/database/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

type GetMonthlyBudgetRequest struct {
	HouseholdID     *int64 `json:"householdId,omitempty"`
	UserID          *int64 `json:"userId,omitempty"`
	Year            int    `json:"year"`
	Month           int    `json:"month"`
	CreateIfMissing bool   `json:"createIfMissing,omitempty"`
}

type BudgetDTO struct {
	ID             int64           `json:"id"`
	HouseholdID    *int64          `json:"householdId,omitempty"`
	UserID         *int64          `json:"userId,omitempty"`
	PeriodStart    time.Time       `json:"periodStart"`
	PeriodEnd      time.Time       `json:"periodEnd"`
	SourceBudgetID *int64          `json:"sourceBudgetId,omitempty"`
	Lines          []BudgetLineDTO `json:"lines"`
}

type BudgetLineDTO struct {
	ID               int64            `json:"id"`
	BudgetID         int64            `json:"budgetId"`
	Name             string           `json:"name"`
	AllocationAmount string           `json:"allocationAmount"`
	SortOrder        int32            `json:"sortOrder"`
	Categories       []CategoryRefDTO `json:"categories"`
}

func (s *Service) GetMonthlyBudget(ctx context.Context, req GetMonthlyBudgetRequest) (BudgetDTO, error) {
	if err := validateBudgetOwner(req.HouseholdID, req.UserID); err != nil {
		return BudgetDTO{}, err
	}
	start, end, err := monthlyBudgetPeriod(req.Year, req.Month)
	if err != nil {
		return BudgetDTO{}, err
	}

	budget, err := s.findBudgetByPeriod(ctx, req.HouseholdID, req.UserID, start, end)
	if err == nil {
		return s.budgetDTO(ctx, budget)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return BudgetDTO{}, mapBudgetError(err)
	}
	if !req.CreateIfMissing {
		return BudgetDTO{}, NewError(CodeValidationError, "budget not found", err)
	}

	created, err := s.createMonthlyBudget(ctx, req.HouseholdID, req.UserID, start, end)
	if err != nil {
		return BudgetDTO{}, err
	}
	return s.budgetDTO(ctx, created)
}

func validateBudgetOwner(householdID, userID *int64) error {
	if householdID == nil && userID == nil {
		return NewError(CodeValidationError, "exactly one budget owner is required", nil)
	}
	if householdID != nil && userID != nil {
		return NewError(CodeValidationError, "only one budget owner is allowed", nil)
	}
	return nil
}

func monthlyBudgetPeriod(year, month int) (time.Time, time.Time, error) {
	if year < 1 || month < 1 || month > 12 {
		return time.Time{}, time.Time{}, NewError(CodeValidationError, "invalid budget month", nil)
	}
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, -1)
	return start, end, nil
}

func (s *Service) findBudgetByPeriod(ctx context.Context, householdID, userID *int64, start, end time.Time) (sqlc.Budget, error) {
	if householdID != nil {
		return s.repo.GetHouseholdBudgetByPeriod(ctx, sqlc.GetHouseholdBudgetByPeriodParams{
			HouseholdID:  householdID,
			PeriodStart: pgtype.Date{Time: start, Valid: true},
			PeriodEnd:   pgtype.Date{Time: end, Valid: true},
		})
	}
	return s.repo.GetUserBudgetByPeriod(ctx, sqlc.GetUserBudgetByPeriodParams{
		UserID:      userID,
		PeriodStart: pgtype.Date{Time: start, Valid: true},
		PeriodEnd:   pgtype.Date{Time: end, Valid: true},
	})
}

func (s *Service) createMonthlyBudget(ctx context.Context, householdID, userID *int64, start, end time.Time) (sqlc.Budget, error) {
	if householdID != nil {
		budget, err := s.repo.CreateHouseholdBudget(ctx, sqlc.CreateHouseholdBudgetParams{
			HouseholdID:  householdID,
			PeriodStart: pgtype.Date{Time: start, Valid: true},
			PeriodEnd:   pgtype.Date{Time: end, Valid: true},
		})
		if err != nil {
			return sqlc.Budget{}, mapBudgetError(err)
		}
		return budget, nil
	}
	budget, err := s.repo.CreateUserBudget(ctx, sqlc.CreateUserBudgetParams{
		UserID:      userID,
		PeriodStart: pgtype.Date{Time: start, Valid: true},
		PeriodEnd:   pgtype.Date{Time: end, Valid: true},
	})
	if err != nil {
		return sqlc.Budget{}, mapBudgetError(err)
	}
	return budget, nil
}

func (s *Service) budgetDTO(ctx context.Context, budget sqlc.Budget) (BudgetDTO, error) {
	lines, err := s.repo.ListBudgetLines(ctx, budget.ID)
	if err != nil {
		return BudgetDTO{}, mapBudgetError(err)
	}
	categories, err := s.repo.ListBudgetLineCategories(ctx, budget.ID)
	if err != nil {
		return BudgetDTO{}, mapBudgetError(err)
	}
	categoriesByLine := make(map[int64][]CategoryRefDTO)
	for _, row := range categories {
		categoriesByLine[row.BudgetLineID] = append(categoriesByLine[row.BudgetLineID], CategoryRefDTO{
			ID:   row.CategoryID,
			Code: row.CategoryCode,
			Name: row.CategoryName,
		})
	}
	dtoLines := make([]BudgetLineDTO, 0, len(lines))
	for _, line := range lines {
		dtoLines = append(dtoLines, budgetLineDTO(line, categoriesByLine[line.ID]))
	}
	return BudgetDTO{
		ID:             budget.ID,
		HouseholdID:    budget.HouseholdID,
		UserID:         budget.UserID,
		PeriodStart:    budget.PeriodStart.Time,
		PeriodEnd:      budget.PeriodEnd.Time,
		SourceBudgetID: budget.SourceBudgetID,
		Lines:          dtoLines,
	}, nil
}

func budgetLineDTO(line sqlc.BudgetLine, categories []CategoryRefDTO) BudgetLineDTO {
	return BudgetLineDTO{
		ID:               line.ID,
		BudgetID:         line.BudgetID,
		Name:             line.Name,
		AllocationAmount: line.AllocationAmount,
		SortOrder:        line.SortOrder,
		Categories:       categories,
	}
}

func mapBudgetError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return NewError(CodeValidationError, "budget not found", err)
	}
	return NewError(CodeDatabaseError, "database error", err)
}
```

Keep `AllocationAmount` as a JSON string in app DTOs and requests. `allocation_amount NUMERIC(12, 2)` should be converted at the app boundary with focused helpers in `internal/app/budgets.go`:

```go
func parseBudgetNumeric(value string) (pgtype.Numeric, error) {
	value = strings.TrimSpace(value)
	rat, ok := new(big.Rat).SetString(value)
	if !ok {
		return pgtype.Numeric{}, NewError(CodeValidationError, "allocation amount must be a decimal number", nil)
	}
	scaled := new(big.Rat).Mul(rat, big.NewRat(100, 1))
	if !scaled.IsInt() {
		return pgtype.Numeric{}, NewError(CodeValidationError, "allocation amount must have at most two decimal places", nil)
	}
	return pgtype.Numeric{Int: scaled.Num(), Exp: -2, Valid: true}, nil
}

func budgetNumericString(value pgtype.Numeric) string {
	if !value.Valid || value.Int == nil {
		return "0.00"
	}
	rat := new(big.Rat).SetInt(value.Int)
	if value.Exp < 0 {
		rat.Quo(rat, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-value.Exp)), nil)))
	}
	if value.Exp > 0 {
		rat.Mul(rat, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(value.Exp)), nil)))
	}
	out, _ := rat.Float64()
	return fmt.Sprintf("%.2f", out)
}
```

Add `math/big` to the import block when these helpers are needed. Use `parseBudgetNumeric` when setting sqlc params and `budgetNumericString` when building DTOs.

- [ ] **Step 5: Add fake repository budget fields and methods**

In `internal/app/test_fakes_test.go`, add fields to `fakeRepo`:

```go
householdBudgetByPeriod sqlc.Budget
userBudgetByPeriod      sqlc.Budget
budgetByID              sqlc.Budget
latestHouseholdBudget   sqlc.Budget
latestUserBudget        sqlc.Budget
createdHouseholdBudget  sqlc.Budget
createdUserBudget       sqlc.Budget
budgetLines             []sqlc.BudgetLine
budgetLineCategories    []sqlc.ListBudgetLineCategoriesRow

lastGetHouseholdBudgetStart time.Time
lastGetUserBudgetStart      time.Time
lastCreateHouseholdBudget   sqlc.CreateHouseholdBudgetParams
lastCreateUserBudget        sqlc.CreateUserBudgetParams
```

Add `time` to the test fake imports.

Add methods:

```go
func (f *fakeRepo) GetHouseholdBudgetByPeriod(_ context.Context, arg sqlc.GetHouseholdBudgetByPeriodParams) (sqlc.Budget, error) {
	f.lastGetHouseholdBudgetStart = arg.PeriodStart.Time
	if f.householdBudgetByPeriod.ID == 0 {
		return sqlc.Budget{}, sql.ErrNoRows
	}
	return f.householdBudgetByPeriod, nil
}

func (f *fakeRepo) GetUserBudgetByPeriod(_ context.Context, arg sqlc.GetUserBudgetByPeriodParams) (sqlc.Budget, error) {
	f.lastGetUserBudgetStart = arg.PeriodStart.Time
	if f.userBudgetByPeriod.ID == 0 {
		return sqlc.Budget{}, sql.ErrNoRows
	}
	return f.userBudgetByPeriod, nil
}

func (f *fakeRepo) GetBudgetById(context.Context, int64) (sqlc.Budget, error) {
	if f.budgetByID.ID == 0 {
		return sqlc.Budget{}, sql.ErrNoRows
	}
	return f.budgetByID, nil
}

func (f *fakeRepo) GetLatestPriorHouseholdBudget(context.Context, sqlc.GetLatestPriorHouseholdBudgetParams) (sqlc.Budget, error) {
	if f.latestHouseholdBudget.ID == 0 {
		return sqlc.Budget{}, sql.ErrNoRows
	}
	return f.latestHouseholdBudget, nil
}

func (f *fakeRepo) GetLatestPriorUserBudget(context.Context, sqlc.GetLatestPriorUserBudgetParams) (sqlc.Budget, error) {
	if f.latestUserBudget.ID == 0 {
		return sqlc.Budget{}, sql.ErrNoRows
	}
	return f.latestUserBudget, nil
}

func (f *fakeRepo) ListBudgetLines(context.Context, int64) ([]sqlc.BudgetLine, error) {
	return f.budgetLines, nil
}

func (f *fakeRepo) ListBudgetLineCategories(context.Context, int64) ([]sqlc.ListBudgetLineCategoriesRow, error) {
	return f.budgetLineCategories, nil
}

func (f *fakeRepo) CreateHouseholdBudget(_ context.Context, arg sqlc.CreateHouseholdBudgetParams) (sqlc.Budget, error) {
	f.lastCreateHouseholdBudget = arg
	if f.createdHouseholdBudget.ID != 0 {
		return f.createdHouseholdBudget, nil
	}
	return sqlc.Budget{ID: 1, HouseholdID: arg.HouseholdID, PeriodStart: arg.PeriodStart, PeriodEnd: arg.PeriodEnd, SourceBudgetID: arg.SourceBudgetID}, nil
}

func (f *fakeRepo) CreateUserBudget(_ context.Context, arg sqlc.CreateUserBudgetParams) (sqlc.Budget, error) {
	f.lastCreateUserBudget = arg
	if f.createdUserBudget.ID != 0 {
		return f.createdUserBudget, nil
	}
	return sqlc.Budget{ID: 1, UserID: arg.UserID, PeriodStart: arg.PeriodStart, PeriodEnd: arg.PeriodEnd, SourceBudgetID: arg.SourceBudgetID}, nil
}
```

Add stub methods for budget line/report repository calls that are used in later tasks:

```go
func (f *fakeRepo) GetBudgetLineById(context.Context, int64) (sqlc.BudgetLine, error) {
	return sqlc.BudgetLine{}, sql.ErrNoRows
}

func (f *fakeRepo) GetMaxBudgetLineSortOrder(context.Context, int64) (int32, error) {
	return 0, nil
}

func (f *fakeRepo) CreateBudgetLine(context.Context, sqlc.CreateBudgetLineParams) (sqlc.BudgetLine, error) {
	return sqlc.BudgetLine{}, nil
}

func (f *fakeRepo) UpdateBudgetLine(context.Context, sqlc.UpdateBudgetLineParams) (sqlc.BudgetLine, error) {
	return sqlc.BudgetLine{}, nil
}

func (f *fakeRepo) DeleteBudgetLine(context.Context, int64) error {
	return nil
}

func (f *fakeRepo) DeleteBudgetLineCategories(context.Context, int64) error {
	return nil
}

func (f *fakeRepo) CreateBudgetLineCategory(context.Context, sqlc.CreateBudgetLineCategoryParams) error {
	return nil
}

func (f *fakeRepo) ListBudgetTransactions(context.Context, sqlc.ListBudgetTransactionsParams) ([]sqlc.ListBudgetTransactionsRow, error) {
	return nil, nil
}

func (f *fakeRepo) SumUncategorizedBudgetTransactions(context.Context, sqlc.SumUncategorizedBudgetTransactionsParams) (float32, error) {
	return 0, nil
}
```

- [ ] **Step 6: Run tests**

Run:

```bash
go test ./internal/app
```

Expected: PASS for the new monthly period and existing budget tests.

- [ ] **Step 7: Commit app budget retrieval**

Run:

```bash
git add internal/app/service.go internal/app/budgets.go internal/app/budgets_test.go internal/app/test_fakes_test.go
git commit -m "Add monthly budget retrieval service"
```

Expected: commit succeeds.

---

### Task 3: Auto-Copy Latest Prior Budget

**Files:**
- Modify: `internal/app/budgets.go`
- Modify: `internal/app/budgets_test.go`
- Modify: `internal/app/test_fakes_test.go`

- [ ] **Step 1: Add failing test for empty budget creation**

Append to `internal/app/budgets_test.go`:

```go
func TestGetMonthlyBudgetCreatesEmptyBudgetWithoutPriorBudget(t *testing.T) {
	householdID := int64(1)
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeTransactionService{})

	budget, err := svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		HouseholdID:     &householdID,
		Year:            2026,
		Month:           5,
		CreateIfMissing: true,
	})

	if err != nil {
		t.Fatalf("GetMonthlyBudget returned error: %v", err)
	}
	if budget.ID != 1 || budget.HouseholdID == nil || *budget.HouseholdID != 1 {
		t.Fatalf("budget = %+v, want created household budget", budget)
	}
	if repo.lastCreateHouseholdBudget.SourceBudgetID != nil {
		t.Fatalf("SourceBudgetID = %v, want nil", repo.lastCreateHouseholdBudget.SourceBudgetID)
	}
}
```

- [ ] **Step 2: Add failing test for copying prior budget lines and category mappings**

Append:

```go
func TestGetMonthlyBudgetCopiesLatestPriorBudget(t *testing.T) {
	householdID := int64(1)
	sourceID := int64(7)
	repo := &fakeRepo{
		latestHouseholdBudget: sqlc.Budget{ID: sourceID, HouseholdID: &householdID},
		createdHouseholdBudget: sqlc.Budget{
			ID:             12,
			HouseholdID:    &householdID,
			SourceBudgetID: &sourceID,
			PeriodStart:    pgtype.Date{Time: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			PeriodEnd:      pgtype.Date{Time: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC), Valid: true},
		},
		budgetLines: []sqlc.BudgetLine{
			{ID: 101, BudgetID: sourceID, Name: "Groceries", AllocationAmount: "800.00", SortOrder: 1},
			{ID: 102, BudgetID: sourceID, Name: "Savings", AllocationAmount: "500.00", SortOrder: 2},
		},
		budgetLineCategories: []sqlc.ListBudgetLineCategoriesRow{
			{BudgetID: sourceID, BudgetLineID: 101, CategoryID: 3, CategoryCode: "groceries", CategoryName: "Groceries"},
		},
	}
	svc := NewService(repo, &fakeTransactionService{})

	budget, err := svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		HouseholdID:     &householdID,
		Year:            2026,
		Month:           7,
		CreateIfMissing: true,
	})

	if err != nil {
		t.Fatalf("GetMonthlyBudget returned error: %v", err)
	}
	if budget.SourceBudgetID == nil || *budget.SourceBudgetID != sourceID {
		t.Fatalf("SourceBudgetID = %v, want 7", budget.SourceBudgetID)
	}
	if len(repo.createdBudgetLines) != 2 {
		t.Fatalf("createdBudgetLines = %d, want 2", len(repo.createdBudgetLines))
	}
	if repo.createdBudgetLines[0].Name != "Groceries" || repo.createdBudgetLines[0].BudgetID != 12 {
		t.Fatalf("first copied line = %+v, want Groceries on budget 12", repo.createdBudgetLines[0])
	}
	if len(repo.createdBudgetLineCategories) != 1 || repo.createdBudgetLineCategories[0].CategoryID != 3 {
		t.Fatalf("created mappings = %+v, want category 3 mapping", repo.createdBudgetLineCategories)
	}
}
```

- [ ] **Step 3: Run tests and verify they fail**

Run:

```bash
go test ./internal/app
```

Expected: FAIL because the service does not copy prior budget lines yet and `fakeRepo` lacks tracking fields.

- [ ] **Step 4: Track created lines and mappings in the fake repo**

In `internal/app/test_fakes_test.go`, add fields:

```go
createdBudgetLines          []sqlc.CreateBudgetLineParams
createdBudgetLineRows      []sqlc.BudgetLine
createdBudgetLineCategories []sqlc.CreateBudgetLineCategoryParams
```

Replace the stub `CreateBudgetLine` and `CreateBudgetLineCategory` methods:

```go
func (f *fakeRepo) CreateBudgetLine(_ context.Context, arg sqlc.CreateBudgetLineParams) (sqlc.BudgetLine, error) {
	f.createdBudgetLines = append(f.createdBudgetLines, arg)
	id := int64(len(f.createdBudgetLines))
	if len(f.createdBudgetLineRows) >= len(f.createdBudgetLines) {
		return f.createdBudgetLineRows[len(f.createdBudgetLines)-1], nil
	}
	return sqlc.BudgetLine{
		ID:               id,
		BudgetID:         arg.BudgetID,
		Name:             arg.Name,
		AllocationAmount: arg.AllocationAmount,
		SortOrder:        arg.SortOrder,
	}, nil
}

func (f *fakeRepo) CreateBudgetLineCategory(_ context.Context, arg sqlc.CreateBudgetLineCategoryParams) error {
	f.createdBudgetLineCategories = append(f.createdBudgetLineCategories, arg)
	return nil
}
```

- [ ] **Step 5: Implement prior lookup and copy behavior**

In `internal/app/budgets.go`, update `createMonthlyBudget` so it finds a prior budget and passes `SourceBudgetID` into the create query:

```go
func (s *Service) createMonthlyBudget(ctx context.Context, householdID, userID *int64, start, end time.Time) (sqlc.Budget, error) {
	source, sourceID, err := s.latestPriorBudget(ctx, householdID, userID, start)
	if err != nil {
		return sqlc.Budget{}, err
	}

	var budget sqlc.Budget
	if householdID != nil {
		budget, err = s.repo.CreateHouseholdBudget(ctx, sqlc.CreateHouseholdBudgetParams{
			HouseholdID:     householdID,
			PeriodStart:    pgtype.Date{Time: start, Valid: true},
			PeriodEnd:      pgtype.Date{Time: end, Valid: true},
			SourceBudgetID: sourceID,
		})
	} else {
		budget, err = s.repo.CreateUserBudget(ctx, sqlc.CreateUserBudgetParams{
			UserID:         userID,
			PeriodStart:    pgtype.Date{Time: start, Valid: true},
			PeriodEnd:      pgtype.Date{Time: end, Valid: true},
			SourceBudgetID: sourceID,
		})
	}
	if err != nil {
		return sqlc.Budget{}, mapBudgetError(err)
	}
	if source.ID != 0 {
		if err := s.copyBudgetLines(ctx, source.ID, budget.ID); err != nil {
			return sqlc.Budget{}, err
		}
	}
	return budget, nil
}

func (s *Service) latestPriorBudget(ctx context.Context, householdID, userID *int64, start time.Time) (sqlc.Budget, *int64, error) {
	var budget sqlc.Budget
	var err error
	if householdID != nil {
		budget, err = s.repo.GetLatestPriorHouseholdBudget(ctx, sqlc.GetLatestPriorHouseholdBudgetParams{
			HouseholdID:  householdID,
			PeriodStart: pgtype.Date{Time: start, Valid: true},
		})
	} else {
		budget, err = s.repo.GetLatestPriorUserBudget(ctx, sqlc.GetLatestPriorUserBudgetParams{
			UserID:      userID,
			PeriodStart: pgtype.Date{Time: start, Valid: true},
		})
	}
	if errors.Is(err, sql.ErrNoRows) {
		return sqlc.Budget{}, nil, nil
	}
	if err != nil {
		return sqlc.Budget{}, nil, mapBudgetError(err)
	}
	return budget, &budget.ID, nil
}
```

Add copy helper:

```go
func (s *Service) copyBudgetLines(ctx context.Context, sourceBudgetID, targetBudgetID int64) error {
	lines, err := s.repo.ListBudgetLines(ctx, sourceBudgetID)
	if err != nil {
		return mapBudgetError(err)
	}
	mappings, err := s.repo.ListBudgetLineCategories(ctx, sourceBudgetID)
	if err != nil {
		return mapBudgetError(err)
	}
	mappingsByLine := make(map[int64][]sqlc.ListBudgetLineCategoriesRow)
	for _, mapping := range mappings {
		mappingsByLine[mapping.BudgetLineID] = append(mappingsByLine[mapping.BudgetLineID], mapping)
	}
	for _, sourceLine := range lines {
		targetLine, err := s.repo.CreateBudgetLine(ctx, sqlc.CreateBudgetLineParams{
			BudgetID:         targetBudgetID,
			Name:             sourceLine.Name,
			AllocationAmount: sourceLine.AllocationAmount,
			SortOrder:        sourceLine.SortOrder,
		})
		if err != nil {
			return mapBudgetError(err)
		}
		for _, mapping := range mappingsByLine[sourceLine.ID] {
			if err := s.repo.CreateBudgetLineCategory(ctx, sqlc.CreateBudgetLineCategoryParams{
				BudgetID:     targetBudgetID,
				BudgetLineID: targetLine.ID,
				CategoryID:   mapping.CategoryID,
			}); err != nil {
				return mapBudgetError(err)
			}
		}
	}
	return nil
}
```

- [ ] **Step 6: Run tests**

Run:

```bash
go test ./internal/app
```

Expected: PASS.

- [ ] **Step 7: Commit auto-copy behavior**

Run:

```bash
git add internal/app/budgets.go internal/app/budgets_test.go internal/app/test_fakes_test.go
git commit -m "Add monthly budget auto-copy"
```

Expected: commit succeeds.

---

### Task 4: Budget Line Create, Update, and Delete Services

**Files:**
- Modify: `internal/app/budgets.go`
- Modify: `internal/app/budgets_test.go`
- Modify: `internal/app/test_fakes_test.go`

- [ ] **Step 1: Add failing tests for line creation and category validation**

Append to `internal/app/budgets_test.go`:

```go
func TestCreateBudgetLineResolvesCategoryCodes(t *testing.T) {
	repo := &fakeRepo{
		budgetByID:      sqlc.Budget{ID: 12},
		categoryByCode:  sqlc.Category{ID: 3, Code: "groceries", Name: "Groceries", IsActive: true},
		maxSortOrder:    2,
		createdBudgetLineRows: []sqlc.BudgetLine{
			{ID: 44, BudgetID: 12, Name: "Groceries", AllocationAmount: "800.00", SortOrder: 3},
		},
	}
	svc := NewService(repo, &fakeTransactionService{})

	line, err := svc.CreateBudgetLine(context.Background(), CreateBudgetLineRequest{
		BudgetID:          12,
		Name:              "Groceries",
		AllocationAmount:  "800.00",
		CategoryCodes:     []string{"groceries"},
	})

	if err != nil {
		t.Fatalf("CreateBudgetLine returned error: %v", err)
	}
	if line.ID != 44 || line.SortOrder != 3 {
		t.Fatalf("line = %+v, want id 44 sort order 3", line)
	}
	if len(repo.createdBudgetLineCategories) != 1 || repo.createdBudgetLineCategories[0].CategoryID != 3 {
		t.Fatalf("created mappings = %+v, want category 3", repo.createdBudgetLineCategories)
	}
}

func TestCreateBudgetLineRejectsReusedCategory(t *testing.T) {
	repo := &fakeRepo{
		budgetByID:     sqlc.Budget{ID: 12},
		categoryByCode: sqlc.Category{ID: 3, Code: "groceries", Name: "Groceries", IsActive: true},
		budgetLineCategories: []sqlc.ListBudgetLineCategoriesRow{
			{BudgetID: 12, BudgetLineID: 40, CategoryID: 3, CategoryCode: "groceries", CategoryName: "Groceries"},
		},
	}
	svc := NewService(repo, &fakeTransactionService{})

	_, err := svc.CreateBudgetLine(context.Background(), CreateBudgetLineRequest{
		BudgetID:         12,
		Name:             "Food",
		AllocationAmount: "800.00",
		CategoryCodes:    []string{"groceries"},
	})

	if appErr, ok := err.(*AppError); !ok || appErr.Code != CodeValidationError {
		t.Fatalf("err = %v, want validation error", err)
	}
}
```

- [ ] **Step 2: Add failing tests for line update and delete**

Append:

```go
func TestUpdateBudgetLineReplacesOnlyThatLineCategories(t *testing.T) {
	repo := &fakeRepo{
		budgetLineByID: sqlc.BudgetLine{ID: 44, BudgetID: 12, Name: "Groceries", AllocationAmount: "800.00", SortOrder: 1},
		categoryByCode: sqlc.Category{ID: 3, Code: "groceries", Name: "Groceries", IsActive: true},
		updatedBudgetLine: sqlc.BudgetLine{ID: 44, BudgetID: 12, Name: "Groceries", AllocationAmount: "900.00", SortOrder: 1},
	}
	svc := NewService(repo, &fakeTransactionService{})

	line, err := svc.UpdateBudgetLine(context.Background(), UpdateBudgetLineRequest{
		LineID:           44,
		AllocationAmount: strPtr("900.00"),
		CategoryCodes:    &[]string{"groceries"},
	})

	if err != nil {
		t.Fatalf("UpdateBudgetLine returned error: %v", err)
	}
	if line.AllocationAmount != "900.00" {
		t.Fatalf("AllocationAmount = %q, want 900.00", line.AllocationAmount)
	}
	if repo.deletedBudgetLineCategoryID != 44 {
		t.Fatalf("deletedBudgetLineCategoryID = %d, want 44", repo.deletedBudgetLineCategoryID)
	}
	if len(repo.createdBudgetLineCategories) != 1 || repo.createdBudgetLineCategories[0].BudgetLineID != 44 {
		t.Fatalf("created mappings = %+v, want mapping for line 44", repo.createdBudgetLineCategories)
	}
}

func TestDeleteBudgetLineDeletesLine(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeTransactionService{})

	err := svc.DeleteBudgetLine(context.Background(), 44)

	if err != nil {
		t.Fatalf("DeleteBudgetLine returned error: %v", err)
	}
	if repo.deletedBudgetLineID != 44 {
		t.Fatalf("deletedBudgetLineID = %d, want 44", repo.deletedBudgetLineID)
	}
}
```

- [ ] **Step 3: Run tests and verify they fail**

Run:

```bash
go test ./internal/app
```

Expected: FAIL with undefined request types and service methods.

- [ ] **Step 4: Add request types and service methods**

In `internal/app/budgets.go`, add request types:

```go
type CreateBudgetLineRequest struct {
	BudgetID         int64    `json:"budgetId"`
	Name             string   `json:"name"`
	AllocationAmount string   `json:"allocationAmount"`
	CategoryIDs      []int64  `json:"categoryIds,omitempty"`
	CategoryCodes    []string `json:"categoryCodes,omitempty"`
	SortOrder        *int32   `json:"sortOrder,omitempty"`
}

type UpdateBudgetLineRequest struct {
	LineID           int64     `json:"lineId"`
	Name             *string   `json:"name,omitempty"`
	AllocationAmount *string   `json:"allocationAmount,omitempty"`
	CategoryIDs      *[]int64  `json:"categoryIds,omitempty"`
	CategoryCodes    *[]string `json:"categoryCodes,omitempty"`
	SortOrder        *int32    `json:"sortOrder,omitempty"`
}
```

Add methods:

```go
func (s *Service) CreateBudgetLine(ctx context.Context, req CreateBudgetLineRequest) (BudgetLineDTO, error) {
	if req.BudgetID == 0 {
		return BudgetLineDTO{}, NewError(CodeValidationError, "budget id is required", nil)
	}
	if _, err := s.repo.GetBudgetById(ctx, req.BudgetID); err != nil {
		return BudgetLineDTO{}, mapBudgetError(err)
	}
	name, err := validateBudgetLineName(req.Name)
	if err != nil {
		return BudgetLineDTO{}, err
	}
	if err := validateBudgetAmount(req.AllocationAmount); err != nil {
		return BudgetLineDTO{}, err
	}
	categoryIDs, err := s.resolveBudgetCategoryIDs(ctx, req.CategoryIDs, req.CategoryCodes)
	if err != nil {
		return BudgetLineDTO{}, err
	}
	if err := s.validateBudgetCategoryAvailability(ctx, req.BudgetID, 0, categoryIDs); err != nil {
		return BudgetLineDTO{}, err
	}
	sortOrder := req.SortOrder
	if sortOrder == nil {
		max, err := s.repo.GetMaxBudgetLineSortOrder(ctx, req.BudgetID)
		if err != nil {
			return BudgetLineDTO{}, mapBudgetError(err)
		}
		next := max + 1
		sortOrder = &next
	}
	line, err := s.repo.CreateBudgetLine(ctx, sqlc.CreateBudgetLineParams{
		BudgetID:          req.BudgetID,
		Name:              name,
		AllocationAmount:  req.AllocationAmount,
		SortOrder:         *sortOrder,
	})
	if err != nil {
		return BudgetLineDTO{}, mapBudgetError(err)
	}
	if err := s.replaceBudgetLineCategories(ctx, req.BudgetID, line.ID, categoryIDs); err != nil {
		return BudgetLineDTO{}, err
	}
	return s.budgetLineDTOWithCategories(ctx, line)
}

func (s *Service) UpdateBudgetLine(ctx context.Context, req UpdateBudgetLineRequest) (BudgetLineDTO, error) {
	if req.LineID == 0 {
		return BudgetLineDTO{}, NewError(CodeValidationError, "budget line id is required", nil)
	}
	existing, err := s.repo.GetBudgetLineById(ctx, req.LineID)
	if err != nil {
		return BudgetLineDTO{}, mapBudgetError(err)
	}
	params := sqlc.UpdateBudgetLineParams{ID: req.LineID}
	if req.Name != nil {
		name, err := validateBudgetLineName(*req.Name)
		if err != nil {
			return BudgetLineDTO{}, err
		}
		params.SetName = true
		params.Name = name
	}
	if req.AllocationAmount != nil {
		if err := validateBudgetAmount(*req.AllocationAmount); err != nil {
			return BudgetLineDTO{}, err
		}
		params.SetAllocationAmount = true
		params.AllocationAmount = *req.AllocationAmount
	}
	if req.SortOrder != nil {
		params.SetSortOrder = true
		params.SortOrder = *req.SortOrder
	}
	line, err := s.repo.UpdateBudgetLine(ctx, params)
	if err != nil {
		return BudgetLineDTO{}, mapBudgetError(err)
	}
	if req.CategoryIDs != nil || req.CategoryCodes != nil {
		ids := []int64(nil)
		codes := []string(nil)
		if req.CategoryIDs != nil {
			ids = *req.CategoryIDs
		}
		if req.CategoryCodes != nil {
			codes = *req.CategoryCodes
		}
		categoryIDs, err := s.resolveBudgetCategoryIDs(ctx, ids, codes)
		if err != nil {
			return BudgetLineDTO{}, err
		}
		if err := s.validateBudgetCategoryAvailability(ctx, existing.BudgetID, existing.ID, categoryIDs); err != nil {
			return BudgetLineDTO{}, err
		}
		if err := s.replaceBudgetLineCategories(ctx, existing.BudgetID, existing.ID, categoryIDs); err != nil {
			return BudgetLineDTO{}, err
		}
	}
	return s.budgetLineDTOWithCategories(ctx, line)
}

func (s *Service) DeleteBudgetLine(ctx context.Context, lineID int64) error {
	if lineID == 0 {
		return NewError(CodeValidationError, "budget line id is required", nil)
	}
	if err := s.repo.DeleteBudgetLine(ctx, lineID); err != nil {
		return mapBudgetError(err)
	}
	return nil
}
```

Add helpers:

```go
func validateBudgetLineName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", NewError(CodeValidationError, "budget line name is required", nil)
	}
	return name, nil
}

func validateBudgetAmount(amount string) error {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return NewError(CodeValidationError, "allocation amount is required", nil)
	}
	value, err := strconv.ParseFloat(amount, 64)
	if err != nil || value < 0 {
		return NewError(CodeValidationError, "allocation amount must be greater than or equal to zero", err)
	}
	return nil
}
```

Add imports:

```go
import (
	"strconv"
	"strings"
)
```

Merge these into the existing import block rather than creating a second one.

- [ ] **Step 5: Add category mapping helpers**

In `internal/app/budgets.go`, add:

```go
func (s *Service) resolveBudgetCategoryIDs(ctx context.Context, ids []int64, codes []string) ([]int64, error) {
	seen := make(map[int64]struct{})
	resolved := make([]int64, 0, len(ids)+len(codes))
	for _, id := range ids {
		category, err := s.repo.GetActiveCategoryById(ctx, id)
		if err != nil {
			return nil, mapCategoryError(err)
		}
		if _, ok := seen[category.ID]; !ok {
			seen[category.ID] = struct{}{}
			resolved = append(resolved, category.ID)
		}
	}
	for _, code := range codes {
		category, err := s.repo.GetActiveCategoryByCode(ctx, code)
		if err != nil {
			return nil, mapCategoryError(err)
		}
		if _, ok := seen[category.ID]; !ok {
			seen[category.ID] = struct{}{}
			resolved = append(resolved, category.ID)
		}
	}
	return resolved, nil
}

func (s *Service) validateBudgetCategoryAvailability(ctx context.Context, budgetID, currentLineID int64, categoryIDs []int64) error {
	existing, err := s.repo.ListBudgetLineCategories(ctx, budgetID)
	if err != nil {
		return mapBudgetError(err)
	}
	wanted := make(map[int64]struct{}, len(categoryIDs))
	for _, id := range categoryIDs {
		wanted[id] = struct{}{}
	}
	for _, mapping := range existing {
		if mapping.BudgetLineID == currentLineID {
			continue
		}
		if _, ok := wanted[mapping.CategoryID]; ok {
			return NewError(CodeValidationError, "category already mapped to another budget line", nil)
		}
	}
	return nil
}

func (s *Service) replaceBudgetLineCategories(ctx context.Context, budgetID, lineID int64, categoryIDs []int64) error {
	if err := s.repo.DeleteBudgetLineCategories(ctx, lineID); err != nil {
		return mapBudgetError(err)
	}
	for _, categoryID := range categoryIDs {
		if err := s.repo.CreateBudgetLineCategory(ctx, sqlc.CreateBudgetLineCategoryParams{
			BudgetID:     budgetID,
			BudgetLineID: lineID,
			CategoryID:   categoryID,
		}); err != nil {
			return mapBudgetError(err)
		}
	}
	return nil
}

func (s *Service) budgetLineDTOWithCategories(ctx context.Context, line sqlc.BudgetLine) (BudgetLineDTO, error) {
	categories, err := s.repo.ListBudgetLineCategories(ctx, line.BudgetID)
	if err != nil {
		return BudgetLineDTO{}, mapBudgetError(err)
	}
	lineCategories := []CategoryRefDTO{}
	for _, row := range categories {
		if row.BudgetLineID == line.ID {
			lineCategories = append(lineCategories, CategoryRefDTO{ID: row.CategoryID, Code: row.CategoryCode, Name: row.CategoryName})
		}
	}
	return budgetLineDTO(line, lineCategories), nil
}
```

- [ ] **Step 6: Update fake repo for line operations**

In `internal/app/test_fakes_test.go`, add fields:

```go
budgetLineByID              sqlc.BudgetLine
updatedBudgetLine           sqlc.BudgetLine
lastUpdateBudgetLine        sqlc.UpdateBudgetLineParams
maxSortOrder                int32
deletedBudgetLineID         int64
deletedBudgetLineCategoryID int64
```

Replace stubs:

```go
func (f *fakeRepo) GetBudgetLineById(context.Context, int64) (sqlc.BudgetLine, error) {
	if f.budgetLineByID.ID == 0 {
		return sqlc.BudgetLine{}, sql.ErrNoRows
	}
	return f.budgetLineByID, nil
}

func (f *fakeRepo) GetMaxBudgetLineSortOrder(context.Context, int64) (int32, error) {
	return f.maxSortOrder, nil
}

func (f *fakeRepo) UpdateBudgetLine(_ context.Context, arg sqlc.UpdateBudgetLineParams) (sqlc.BudgetLine, error) {
	f.lastUpdateBudgetLine = arg
	if f.updatedBudgetLine.ID != 0 {
		return f.updatedBudgetLine, nil
	}
	line := f.budgetLineByID
	if arg.SetName {
		line.Name = arg.Name
	}
	if arg.SetAllocationAmount {
		line.AllocationAmount = arg.AllocationAmount
	}
	if arg.SetSortOrder {
		line.SortOrder = arg.SortOrder
	}
	return line, nil
}

func (f *fakeRepo) DeleteBudgetLine(_ context.Context, id int64) error {
	f.deletedBudgetLineID = id
	return nil
}

func (f *fakeRepo) DeleteBudgetLineCategories(_ context.Context, id int64) error {
	f.deletedBudgetLineCategoryID = id
	return nil
}
```

- [ ] **Step 7: Run tests**

Run:

```bash
go test ./internal/app
```

Expected: PASS.

- [ ] **Step 8: Commit line service operations**

Run:

```bash
git add internal/app/budgets.go internal/app/budgets_test.go internal/app/test_fakes_test.go
git commit -m "Add budget line service operations"
```

Expected: commit succeeds.

---

### Task 5: Budget Report Service

**Files:**
- Modify: `internal/app/budgets.go`
- Modify: `internal/app/budgets_test.go`
- Modify: `internal/app/test_fakes_test.go`

- [ ] **Step 1: Add failing report test**

Append to `internal/app/budgets_test.go`:

```go
func TestGetBudgetReportDerivesActualsFromCategories(t *testing.T) {
	householdID := int64(1)
	repo := &fakeRepo{
		budgetByID: sqlc.Budget{
			ID:          12,
			HouseholdID: &householdID,
			PeriodStart: pgtype.Date{Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			PeriodEnd:   pgtype.Date{Time: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC), Valid: true},
		},
		budgetLines: []sqlc.BudgetLine{
			{ID: 44, BudgetID: 12, Name: "Groceries", AllocationAmount: "800.00", SortOrder: 1},
			{ID: 45, BudgetID: 12, Name: "Savings", AllocationAmount: "500.00", SortOrder: 2},
		},
		budgetLineCategories: []sqlc.ListBudgetLineCategoriesRow{
			{BudgetID: 12, BudgetLineID: 44, CategoryID: 3, CategoryCode: "groceries", CategoryName: "Groceries"},
		},
		budgetTransactions: []sqlc.ListBudgetTransactionsRow{
			{CategoryID: int64Ptr(3), ActualAmount: 570.25},
		},
		uncategorizedBudgetTransactions: 123.45,
	}
	svc := NewService(repo, &fakeTransactionService{})

	report, err := svc.GetBudgetReport(context.Background(), 12)

	if err != nil {
		t.Fatalf("GetBudgetReport returned error: %v", err)
	}
	if len(report.Lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(report.Lines))
	}
	if report.Lines[0].ActualAmount != "570.25" || report.Lines[0].RemainingAmount != "229.75" {
		t.Fatalf("groceries line = %+v, want actual 570.25 remaining 229.75", report.Lines[0])
	}
	if report.Lines[1].ActualAmount != "0.00" || report.Lines[1].RemainingAmount != "500.00" {
		t.Fatalf("savings line = %+v, want zero actual and full remaining", report.Lines[1])
	}
	if report.Totals.UncategorizedActualAmount != "123.45" {
		t.Fatalf("uncategorized = %q, want 123.45", report.Totals.UncategorizedActualAmount)
	}
}
```

Add helper if missing:

```go
func int64Ptr(value int64) *int64 {
	return &value
}
```

- [ ] **Step 2: Add failing refund test**

Append:

```go
func TestGetBudgetReportNegativeTransactionsReduceActuals(t *testing.T) {
	householdID := int64(1)
	repo := &fakeRepo{
		budgetByID: sqlc.Budget{
			ID:          12,
			HouseholdID: &householdID,
			PeriodStart: pgtype.Date{Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			PeriodEnd:   pgtype.Date{Time: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC), Valid: true},
		},
		budgetLines: []sqlc.BudgetLine{
			{ID: 44, BudgetID: 12, Name: "Groceries", AllocationAmount: "100.00", SortOrder: 1},
		},
		budgetLineCategories: []sqlc.ListBudgetLineCategoriesRow{
			{BudgetID: 12, BudgetLineID: 44, CategoryID: 3, CategoryCode: "groceries", CategoryName: "Groceries"},
		},
		budgetTransactions: []sqlc.ListBudgetTransactionsRow{
			{CategoryID: int64Ptr(3), ActualAmount: 60.00},
		},
	}
	svc := NewService(repo, &fakeTransactionService{})

	report, err := svc.GetBudgetReport(context.Background(), 12)

	if err != nil {
		t.Fatalf("GetBudgetReport returned error: %v", err)
	}
	if report.Lines[0].ActualAmount != "60.00" || report.Lines[0].RemainingAmount != "40.00" {
		t.Fatalf("line = %+v, want net actual 60.00 remaining 40.00", report.Lines[0])
	}
}
```

- [ ] **Step 3: Run tests and verify they fail**

Run:

```bash
go test ./internal/app
```

Expected: FAIL with undefined `GetBudgetReport`, `BudgetReportDTO`, and fake fields.

- [ ] **Step 4: Add report DTOs and formatting helpers**

In `internal/app/budgets.go`, add:

```go
type BudgetReportDTO struct {
	Budget BudgetSummaryDTO        `json:"budget"`
	Lines  []BudgetReportLineDTO   `json:"lines"`
	Totals BudgetReportTotalsDTO   `json:"totals"`
}

type BudgetSummaryDTO struct {
	ID             int64     `json:"id"`
	HouseholdID    *int64    `json:"householdId,omitempty"`
	UserID         *int64    `json:"userId,omitempty"`
	PeriodStart    time.Time `json:"periodStart"`
	PeriodEnd      time.Time `json:"periodEnd"`
	SourceBudgetID *int64    `json:"sourceBudgetId,omitempty"`
}

type BudgetReportLineDTO struct {
	ID               int64            `json:"id"`
	BudgetID         int64            `json:"budgetId"`
	Name             string           `json:"name"`
	AllocationAmount string           `json:"allocationAmount"`
	ActualAmount     string           `json:"actualAmount"`
	RemainingAmount  string           `json:"remainingAmount"`
	SortOrder        int32            `json:"sortOrder"`
	Categories       []CategoryRefDTO `json:"categories"`
}

type BudgetReportTotalsDTO struct {
	AllocationAmount           string `json:"allocationAmount"`
	ActualAmount               string `json:"actualAmount"`
	RemainingAmount            string `json:"remainingAmount"`
	UncategorizedActualAmount  string `json:"uncategorizedActualAmount"`
}

func moneyString(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func parseMoney(value string) (float64, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}
```

Add `fmt` to the import block.

- [ ] **Step 5: Implement `GetBudgetReport`**

In `internal/app/budgets.go`, add:

```go
func (s *Service) GetBudgetReport(ctx context.Context, budgetID int64) (BudgetReportDTO, error) {
	if budgetID == 0 {
		return BudgetReportDTO{}, NewError(CodeValidationError, "budget id is required", nil)
	}
	budget, err := s.repo.GetBudgetById(ctx, budgetID)
	if err != nil {
		return BudgetReportDTO{}, mapBudgetError(err)
	}
	lines, err := s.repo.ListBudgetLines(ctx, budgetID)
	if err != nil {
		return BudgetReportDTO{}, mapBudgetError(err)
	}
	mappings, err := s.repo.ListBudgetLineCategories(ctx, budgetID)
	if err != nil {
		return BudgetReportDTO{}, mapBudgetError(err)
	}
	actualRows, err := s.repo.ListBudgetTransactions(ctx, sqlc.ListBudgetTransactionsParams{
		PeriodStart: pgtype.Date{Time: budget.PeriodStart.Time, Valid: true},
		PeriodEnd:   pgtype.Date{Time: budget.PeriodEnd.Time, Valid: true},
		HouseholdID: budget.HouseholdID,
		UserID:      budget.UserID,
	})
	if err != nil {
		return BudgetReportDTO{}, mapBudgetError(err)
	}
	uncategorized, err := s.repo.SumUncategorizedBudgetTransactions(ctx, sqlc.SumUncategorizedBudgetTransactionsParams{
		PeriodStart: pgtype.Date{Time: budget.PeriodStart.Time, Valid: true},
		PeriodEnd:   pgtype.Date{Time: budget.PeriodEnd.Time, Valid: true},
		HouseholdID: budget.HouseholdID,
		UserID:      budget.UserID,
	})
	if err != nil {
		return BudgetReportDTO{}, mapBudgetError(err)
	}

	actualByCategory := make(map[int64]float64)
	for _, row := range actualRows {
		if row.CategoryID != nil {
			actualByCategory[*row.CategoryID] = float64(row.ActualAmount)
		}
	}
	categoriesByLine := make(map[int64][]CategoryRefDTO)
	categoryIDsByLine := make(map[int64][]int64)
	for _, row := range mappings {
		categoriesByLine[row.BudgetLineID] = append(categoriesByLine[row.BudgetLineID], CategoryRefDTO{
			ID:   row.CategoryID,
			Code: row.CategoryCode,
			Name: row.CategoryName,
		})
		categoryIDsByLine[row.BudgetLineID] = append(categoryIDsByLine[row.BudgetLineID], row.CategoryID)
	}

	reportLines := make([]BudgetReportLineDTO, 0, len(lines))
	var totalAllocation float64
	var totalActual float64
	for _, line := range lines {
		allocation, err := parseMoney(line.AllocationAmount)
		if err != nil {
			return BudgetReportDTO{}, NewError(CodeDatabaseError, "invalid budget allocation amount", err)
		}
		actual := 0.0
		for _, categoryID := range categoryIDsByLine[line.ID] {
			actual += actualByCategory[categoryID]
		}
		totalAllocation += allocation
		totalActual += actual
		reportLines = append(reportLines, BudgetReportLineDTO{
			ID:               line.ID,
			BudgetID:         line.BudgetID,
			Name:             line.Name,
			AllocationAmount: moneyString(allocation),
			ActualAmount:     moneyString(actual),
			RemainingAmount:  moneyString(allocation - actual),
			SortOrder:        line.SortOrder,
			Categories:       categoriesByLine[line.ID],
		})
	}

	return BudgetReportDTO{
		Budget: BudgetSummaryDTO{
			ID:             budget.ID,
			HouseholdID:    budget.HouseholdID,
			UserID:         budget.UserID,
			PeriodStart:    budget.PeriodStart.Time,
			PeriodEnd:      budget.PeriodEnd.Time,
			SourceBudgetID: budget.SourceBudgetID,
		},
		Lines: reportLines,
		Totals: BudgetReportTotalsDTO{
			AllocationAmount:          moneyString(totalAllocation),
			ActualAmount:              moneyString(totalActual),
			RemainingAmount:           moneyString(totalAllocation - totalActual),
			UncategorizedActualAmount: moneyString(float64(uncategorized)),
		},
	}, nil
}
```

- [ ] **Step 6: Update fake repo for reports**

In `internal/app/test_fakes_test.go`, add fields:

```go
budgetTransactions               []sqlc.ListBudgetTransactionsRow
uncategorizedBudgetTransactions  float32
```

Replace report stubs:

```go
func (f *fakeRepo) ListBudgetTransactions(context.Context, sqlc.ListBudgetTransactionsParams) ([]sqlc.ListBudgetTransactionsRow, error) {
	return f.budgetTransactions, nil
}

func (f *fakeRepo) SumUncategorizedBudgetTransactions(context.Context, sqlc.SumUncategorizedBudgetTransactionsParams) (float32, error) {
	return f.uncategorizedBudgetTransactions, nil
}
```

- [ ] **Step 7: Run tests**

Run:

```bash
go test ./internal/app
```

Expected: PASS.

- [ ] **Step 8: Commit report service**

Run:

```bash
git add internal/app/budgets.go internal/app/budgets_test.go internal/app/test_fakes_test.go
git commit -m "Add budget report service"
```

Expected: commit succeeds.

---

### Task 6: Budget CLI Commands

**Files:**
- Modify: `internal/cli/commands.go`
- Modify: `internal/cli/commands_test.go`

- [ ] **Step 1: Add failing CLI tests**

Append to `internal/cli/commands_test.go`:

```go
func TestKongBudgetsGetHouseholdCreate(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"budgets", "get",
		"--household-id", "1",
		"--month", "2026-05",
		"--create",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if svc.getMonthlyBudget.HouseholdID == nil || *svc.getMonthlyBudget.HouseholdID != 1 {
		t.Fatalf("HouseholdID = %v, want 1", svc.getMonthlyBudget.HouseholdID)
	}
	if svc.getMonthlyBudget.Year != 2026 || svc.getMonthlyBudget.Month != 5 || !svc.getMonthlyBudget.CreateIfMissing {
		t.Fatalf("getMonthlyBudget = %+v, want May 2026 create", svc.getMonthlyBudget)
	}
}

func TestKongBudgetsReport(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"budgets", "report", "12",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if svc.getBudgetReportID != 12 {
		t.Fatalf("report id = %d, want 12", svc.getBudgetReportID)
	}
}

func TestKongBudgetLinesAdd(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"budgets", "lines", "add",
		"--budget-id", "12",
		"--name", "Groceries",
		"--amount", "800.00",
		"--categories", "groceries,costco",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if svc.createBudgetLine.BudgetID != 12 || svc.createBudgetLine.Name != "Groceries" {
		t.Fatalf("createBudgetLine = %+v, want budget 12 groceries", svc.createBudgetLine)
	}
	if len(svc.createBudgetLine.CategoryCodes) != 2 || svc.createBudgetLine.CategoryCodes[1] != "costco" {
		t.Fatalf("CategoryCodes = %v, want groceries,costco", svc.createBudgetLine.CategoryCodes)
	}
}

func TestKongBudgetLinesUpdateUsesPositionalLineID(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"budgets", "lines", "update", "44",
		"--amount", "900.00",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if svc.updateBudgetLine.LineID != 44 || svc.updateBudgetLine.AllocationAmount == nil || *svc.updateBudgetLine.AllocationAmount != "900.00" {
		t.Fatalf("updateBudgetLine = %+v, want line 44 amount 900.00", svc.updateBudgetLine)
	}
}

func TestKongBudgetLinesDeleteUsesPositionalLineID(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"budgets", "lines", "delete", "44",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if svc.deleteBudgetLineID != 44 {
		t.Fatalf("deleteBudgetLineID = %d, want 44", svc.deleteBudgetLineID)
	}
}
```

- [ ] **Step 2: Run CLI tests and verify they fail**

Run:

```bash
go test ./internal/cli
```

Expected: FAIL because budget CLI commands and fake methods do not exist.

- [ ] **Step 3: Add budget methods to `AppService`**

In `internal/cli/commands.go`, extend `AppService`:

```go
	GetMonthlyBudget(context.Context, app.GetMonthlyBudgetRequest) (app.BudgetDTO, error)
	CreateBudgetLine(context.Context, app.CreateBudgetLineRequest) (app.BudgetLineDTO, error)
	UpdateBudgetLine(context.Context, app.UpdateBudgetLineRequest) (app.BudgetLineDTO, error)
	DeleteBudgetLine(context.Context, int64) error
	GetBudgetReport(context.Context, int64) (app.BudgetReportDTO, error)
```

- [ ] **Step 4: Add `budgets` command group**

In `internal/cli/commands.go`, add to `CLI`:

```go
	Budgets     BudgetsCmd     `cmd:"" help:"Manage budgets."`
```

Add command structs:

```go
type BudgetsCmd struct {
	Get    BudgetGetCmd    `cmd:"" help:"Get a monthly budget."`
	Report BudgetReportCmd `cmd:"" help:"Show a budget report."`
	Lines  BudgetLinesCmd  `cmd:"" help:"Manage budget lines."`
}

type BudgetGetCmd struct {
	HouseholdID *int64 `placeholder:"INT-64" help:"Household budget owner."`
	UserID      *int64 `placeholder:"INT-64" help:"Personal budget owner."`
	Month       string `required:"" help:"Budget month in YYYY-MM format."`
	Create      bool   `help:"Create the monthly budget if missing."`
}

func (c *BudgetGetCmd) Run(ctx *runContext) error {
	year, month, err := parseBudgetMonth(c.Month)
	if err != nil {
		return err
	}
	budget, err := ctx.svc.GetMonthlyBudget(ctx.Context, app.GetMonthlyBudgetRequest{
		HouseholdID:     c.HouseholdID,
		UserID:          c.UserID,
		Year:            year,
		Month:           month,
		CreateIfMissing: c.Create,
	})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, budget)
}

type BudgetReportCmd struct {
	ID int64 `arg:"" required:"" help:"Budget ID."`
}

func (c *BudgetReportCmd) Run(ctx *runContext) error {
	report, err := ctx.svc.GetBudgetReport(ctx.Context, c.ID)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, report)
}

type BudgetLinesCmd struct {
	Add    BudgetLineAddCmd    `cmd:"" help:"Add a budget line."`
	Update BudgetLineUpdateCmd `cmd:"" help:"Update a budget line."`
	Delete BudgetLineDeleteCmd `cmd:"" help:"Delete a budget line."`
}

type BudgetLineAddCmd struct {
	BudgetID   int64   `required:"" placeholder:"INT-64" help:"Budget ID."`
	Name       string  `required:"" help:"Budget line name."`
	Amount     string  `required:"" help:"Allocation amount."`
	Categories *string `help:"Comma-separated category codes."`
	SortOrder  *int32  `help:"Display sort order."`
}

func (c *BudgetLineAddCmd) Run(ctx *runContext) error {
	line, err := ctx.svc.CreateBudgetLine(ctx.Context, app.CreateBudgetLineRequest{
		BudgetID:         c.BudgetID,
		Name:             c.Name,
		AllocationAmount: c.Amount,
		CategoryCodes:    parseOptionalCSV(c.Categories),
		SortOrder:        c.SortOrder,
	})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, line)
}

type BudgetLineUpdateCmd struct {
	ID         int64   `arg:"" required:"" help:"Budget line ID."`
	Name       *string `help:"Replacement budget line name."`
	Amount     *string `help:"Replacement allocation amount."`
	Categories *string `help:"Replacement comma-separated category codes."`
	SortOrder  *int32  `help:"Replacement display sort order."`
}

func (c *BudgetLineUpdateCmd) Run(ctx *runContext) error {
	var categoryCodes *[]string
	if c.Categories != nil {
		parsed := parseOptionalCSV(c.Categories)
		categoryCodes = &parsed
	}
	line, err := ctx.svc.UpdateBudgetLine(ctx.Context, app.UpdateBudgetLineRequest{
		LineID:           c.ID,
		Name:             c.Name,
		AllocationAmount: c.Amount,
		CategoryCodes:    categoryCodes,
		SortOrder:        c.SortOrder,
	})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, line)
}

type BudgetLineDeleteCmd struct {
	ID int64 `arg:"" required:"" help:"Budget line ID."`
}

func (c *BudgetLineDeleteCmd) Run(ctx *runContext) error {
	return ctx.svc.DeleteBudgetLine(ctx.Context, c.ID)
}
```

- [ ] **Step 5: Add parsing helpers**

In `internal/cli/commands.go`, add:

```go
func parseBudgetMonth(value string) (int, int, error) {
	parsed, err := time.Parse("2006-01", value)
	if err != nil {
		return 0, 0, fmt.Errorf("month must be in YYYY-MM format")
	}
	return parsed.Year(), int(parsed.Month()), nil
}

func parseOptionalCSV(value *string) []string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	parts := strings.Split(*value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
```

`fmt`, `strings`, and `time` are already imported in `commands.go`; reuse the existing imports.

- [ ] **Step 6: Update fake CLI service**

In `internal/cli/commands_test.go`, add fields to `fakeAppService`:

```go
getMonthlyBudget  app.GetMonthlyBudgetRequest
createBudgetLine  app.CreateBudgetLineRequest
updateBudgetLine  app.UpdateBudgetLineRequest
deleteBudgetLineID int64
getBudgetReportID int64
```

Add methods:

```go
func (f *fakeAppService) GetMonthlyBudget(_ context.Context, req app.GetMonthlyBudgetRequest) (app.BudgetDTO, error) {
	f.getMonthlyBudget = req
	return app.BudgetDTO{ID: 12}, nil
}

func (f *fakeAppService) CreateBudgetLine(_ context.Context, req app.CreateBudgetLineRequest) (app.BudgetLineDTO, error) {
	f.createBudgetLine = req
	return app.BudgetLineDTO{ID: 44, BudgetID: req.BudgetID, Name: req.Name, AllocationAmount: req.AllocationAmount}, nil
}

func (f *fakeAppService) UpdateBudgetLine(_ context.Context, req app.UpdateBudgetLineRequest) (app.BudgetLineDTO, error) {
	f.updateBudgetLine = req
	amount := ""
	if req.AllocationAmount != nil {
		amount = *req.AllocationAmount
	}
	return app.BudgetLineDTO{ID: req.LineID, AllocationAmount: amount}, nil
}

func (f *fakeAppService) DeleteBudgetLine(_ context.Context, id int64) error {
	f.deleteBudgetLineID = id
	return nil
}

func (f *fakeAppService) GetBudgetReport(_ context.Context, id int64) (app.BudgetReportDTO, error) {
	f.getBudgetReportID = id
	return app.BudgetReportDTO{}, nil
}
```

- [ ] **Step 7: Run CLI tests**

Run:

```bash
go test ./internal/cli
```

Expected: PASS.

- [ ] **Step 8: Commit CLI commands**

Run:

```bash
git add internal/cli/commands.go internal/cli/commands_test.go
git commit -m "Add budget CLI commands"
```

Expected: commit succeeds.

---

### Task 7: CLI Documentation and Full Verification

**Files:**
- Modify: `docs/cli.md`

- [ ] **Step 1: Update CLI docs**

Add a `## Budgets` section to `docs/cli.md` after `## Households`:

```markdown
## Budgets

Get or create a household monthly budget. When `--create` is provided and the month does not exist, the app copies the latest prior budget for the same owner. If no prior budget exists, it creates an empty budget.

```bash
/tmp/voltr-finance --config /tmp/voltr-finance.json budgets get \
  --household-id 1 \
  --month 2026-05 \
  --create
```

Get or create a personal monthly budget:

```bash
/tmp/voltr-finance --config /tmp/voltr-finance.json budgets get \
  --user-id 4 \
  --month 2026-05 \
  --create
```

Add a budget line. Category inputs use category codes:

```bash
/tmp/voltr-finance --config /tmp/voltr-finance.json budgets lines add \
  --budget-id 12 \
  --name "Groceries" \
  --amount 800.00 \
  --categories groceries,costco
```

Update a budget line by line ID:

```bash
/tmp/voltr-finance --config /tmp/voltr-finance.json budgets lines update 44 \
  --amount 900.00
```

Delete a budget line by line ID:

```bash
/tmp/voltr-finance --config /tmp/voltr-finance.json budgets lines delete 44
```

Show budget actuals:

```bash
/tmp/voltr-finance --config /tmp/voltr-finance.json budgets report 12
```
```

- [ ] **Step 2: Run all tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 3: Build CLI**

Run:

```bash
go build ./cmd/cli
```

Expected: PASS with exit code 0.

- [ ] **Step 4: Search for stale budget category references**

Run:

```bash
grep -RIn "budget_category_id\|BudgetCategoryID\|budget_category" internal db/migrations docs/superpowers/specs/2026-05-10-budget-design.md
```

Expected: only historical migration down sections and existing category design/plan references may appear. No new budget implementation file should introduce `budget_category` or `BudgetCategoryID`.

- [ ] **Step 5: Commit docs and final verification state**

Run:

```bash
git add docs/cli.md
git commit -m "Document budget CLI commands"
```

Expected: commit succeeds.

Run:

```bash
git status --short
```

Expected: clean worktree.

---

## Self-Review Checklist

- Spec coverage: tasks cover normalized data model, monthly date ranges, household and personal owners, latest-prior auto-copy, line-level create/update/delete, category-code CLI inputs, derived reporting, refund netting, uncategorized visibility, and CLI docs.
- Placeholder scan: no implementation step depends on unspecified behavior; code snippets define the intended functions, methods, requests, and commands.
- Type consistency: all service and CLI names use `GetMonthlyBudget`, `CreateBudgetLine`, `UpdateBudgetLine`, `DeleteBudgetLine`, and `GetBudgetReport`; CLI update/delete use positional line IDs; CLI create uses `--budget-id`.
