# Transaction Categories Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add optional global transaction categories backed by the database and exposed through the app service and CLI.

**Architecture:** Categories are global reference records in the `transactions` schema. Transactions point to categories through nullable `transaction.category_id`; category assignment can be supplied by id or stable code. The old `budget_category` table and `transaction.budget_category_id` column are removed from the active schema and app surface.

**Tech Stack:** Go, PostgreSQL migrations managed by dbmate, sqlc v1.29, pgx/v5, Kong CLI, standard Go tests.

---

## File Map

- Create `db/migrations/20260508000000_transaction_categories.sql`: add `category`, add `transaction.category_id`, drop `transaction.budget_category_id`, drop `budget_category`, and reverse those changes in down migration.
- Modify `internal/database/query.sql`: add category queries, remove `budget_category_id` from transaction writes, include category details in transaction read/list queries, and update transaction assignment fields.
- Regenerate `internal/database/sqlc/models.go` and `internal/database/sqlc/query.sql.go` with `sqlc generate`.
- Create `internal/app/categories.go`: category DTOs, requests, slug generation, validation helpers, service methods.
- Modify `internal/app/service.go`: add category repository methods to `Repository`.
- Modify `internal/app/transactions.go`: replace budget category request fields with category fields and render category details.
- Modify `internal/app/transactions_test.go` and `internal/app/test_fakes_test.go`: cover transaction category assignment and response rendering.
- Modify `internal/transaction/types.go`, `internal/transaction/transaction_service.go`, `internal/transaction/hashing.go`, and existing tests in `internal/transaction`: replace budget category update flow with category update flow.
- Modify `internal/cli/commands.go`: add `categories` command group, add `--category` and `--clear-category` transaction flags.
- Modify `internal/cli/render.go`: include category columns/details in compact and CSV renderers.
- Modify `internal/cli/commands_test.go`: cover category commands and transaction category flags.

---

### Task 1: Database Migration and SQL Queries

**Files:**
- Create: `db/migrations/20260508000000_transaction_categories.sql`
- Modify: `internal/database/query.sql`
- Generate: `internal/database/sqlc/models.go`
- Generate: `internal/database/sqlc/query.sql.go`

- [ ] **Step 1: Add migration**

Create `db/migrations/20260508000000_transaction_categories.sql`:

```sql
-- migrate:up
SET search_path TO transactions, public;

CREATE TABLE category (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR UNIQUE NOT NULL,
    name VARCHAR NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_category_code ON category(code);
CREATE INDEX idx_category_is_active ON category(is_active);

ALTER TABLE transaction
    ADD COLUMN category_id BIGINT REFERENCES category(id);

CREATE INDEX idx_transaction_category_id ON transaction(category_id);

DROP INDEX IF EXISTS idx_transaction_budget_category_id;

ALTER TABLE transaction
    DROP COLUMN budget_category_id;

DROP TABLE budget_category;

-- migrate:down
SET search_path TO transactions, public;

CREATE TABLE budget_category (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    budget_id BIGINT,
    category_name VARCHAR NOT NULL,
    allocation REAL NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (budget_id) REFERENCES budget(id)
);

ALTER TABLE transaction
    ADD COLUMN budget_category_id BIGINT REFERENCES budget_category(id);

CREATE INDEX idx_transaction_budget_category_id ON transaction(budget_category_id);

DROP INDEX IF EXISTS idx_transaction_category_id;

ALTER TABLE transaction
    DROP COLUMN category_id;

DROP INDEX IF EXISTS idx_category_is_active;
DROP INDEX IF EXISTS idx_category_code;
DROP TABLE category;
```

- [ ] **Step 2: Update category queries**

Add this section to `internal/database/query.sql` before the transaction section:

```sql
-- ******************* category *******************
-- READS

-- name: CreateCategory :one
INSERT INTO category (code, name, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListCategories :many
SELECT * FROM category
WHERE (sqlc.arg(include_inactive)::bool OR is_active)
ORDER BY name ASC, id ASC;

-- name: GetCategoryById :one
SELECT * FROM category
WHERE id = $1;

-- name: GetActiveCategoryById :one
SELECT * FROM category
WHERE id = $1 AND is_active;

-- name: GetCategoryByCode :one
SELECT * FROM category
WHERE code = $1;

-- name: GetActiveCategoryByCode :one
SELECT * FROM category
WHERE code = $1 AND is_active;

-- name: UpdateCategory :one
UPDATE category
SET
    name = CASE
        WHEN sqlc.arg(set_name)::bool THEN sqlc.arg(name)::VARCHAR
        ELSE name
    END,
    description = CASE
        WHEN sqlc.arg(set_description)::bool THEN sqlc.narg(description)::TEXT
        ELSE description
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)::BIGINT
RETURNING *;

-- name: DeactivateCategory :one
UPDATE category
SET is_active = false,
    updated_at = CURRENT_TIMESTAMP
WHERE code = $1
RETURNING *;
```

- [ ] **Step 3: Update transaction queries**

In `internal/database/query.sql`, remove `budget_category_id` from `CreateTransaction` and `UpdateTransactionById`, and add `category_id`.

Use this `CreateTransaction` body:

```sql
-- name: CreateTransaction :one
INSERT INTO transaction
(
    amount,
    category_id,
    description,
    transaction_date,
    transaction_id,
    author_id,
    household_id,
    notes
)
VALUES
($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;
```

In `UpdateTransactionById`, replace the budget category assignment block with:

```sql
    category_id = CASE
        WHEN sqlc.arg(set_category_id)::bool THEN sqlc.narg(category_id)::BIGINT
        ELSE category_id
    END,
```

Update `GetTransactionsByIdWithDetails` to left join category and return category columns:

```sql
-- name: GetTransactionsByIdWithDetails :many
SELECT
    sqlc.embed(t),
    u.id AS author_id,
    u.name AS author_name,
    h.id AS household_id,
    h.name AS household_name,
    c.id AS category_id,
    c.code AS category_code,
    c.name AS category_name
FROM transaction t
JOIN users u ON u.id = t.author_id
LEFT JOIN household h ON h.id = t.household_id
LEFT JOIN category c ON c.id = t.category_id
WHERE t.id = ANY(sqlc.arg(ids)::BIGINT[])
  AND (sqlc.arg(include_deleted)::bool OR t.deleted_at IS NULL)
ORDER BY array_position(sqlc.arg(ids)::BIGINT[], t.id);
```

Update `ListTransactions` similarly by adding the same selected category columns and `LEFT JOIN category c ON c.id = t.category_id`.

- [ ] **Step 4: Generate sqlc output**

Run:

```bash
sqlc generate
```

Expected: command exits 0 and updates `internal/database/sqlc/models.go` plus `internal/database/sqlc/query.sql.go`. `sqlc.Transaction` has `CategoryID *int64` and no `BudgetCategoryID`.

- [ ] **Step 5: Run package tests to expose compile failures**

Run:

```bash
go test ./...
```

Expected: tests fail to compile where Go code still references `BudgetCategoryID` or lacks category repository methods. Keep the failures for Task 2.

- [ ] **Step 6: Commit database and generated query changes**

```bash
git add db/migrations/20260508000000_transaction_categories.sql internal/database/query.sql internal/database/sqlc
git commit -m "feat: add category schema and queries"
```

---

### Task 2: App Category Service

**Files:**
- Create: `internal/app/categories.go`
- Modify: `internal/app/service.go`
- Modify: `internal/app/test_fakes_test.go`
- Test: `internal/app/categories_test.go`

- [ ] **Step 1: Write category service tests**

Create `internal/app/categories_test.go`:

```go
package app

import (
	"context"
	"testing"

	"rdmm404/voltr-finance/internal/database/sqlc"
)

func TestCreateCategoryGeneratesCode(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeTransactionService{})

	category, err := svc.CreateCategory(context.Background(), CreateCategoryRequest{
		Name: "Restaurants & Takeout",
	})

	if err != nil {
		t.Fatalf("CreateCategory returned error: %v", err)
	}
	if category.Code != "restaurants-takeout" {
		t.Fatalf("Code = %q, want restaurants-takeout", category.Code)
	}
	if repo.lastCreateCategory.Code != "restaurants-takeout" || repo.lastCreateCategory.Name != "Restaurants & Takeout" {
		t.Fatalf("CreateCategoryParams = %+v, want generated code and name", repo.lastCreateCategory)
	}
}

func TestCreateCategoryAcceptsExplicitCode(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeTransactionService{})

	category, err := svc.CreateCategory(context.Background(), CreateCategoryRequest{
		Name: "Restaurants & Takeout",
		Code: strPtr("restaurants"),
	})

	if err != nil {
		t.Fatalf("CreateCategory returned error: %v", err)
	}
	if category.Code != "restaurants" {
		t.Fatalf("Code = %q, want restaurants", category.Code)
	}
}

func TestCreateCategoryRejectsInvalidCode(t *testing.T) {
	svc := NewService(&fakeRepo{}, &fakeTransactionService{})

	_, err := svc.CreateCategory(context.Background(), CreateCategoryRequest{
		Name: "Groceries",
		Code: strPtr("Groceries!"),
	})

	if appErr, ok := err.(*AppError); !ok || appErr.Code != CodeValidationError {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestListCategoriesMapsRows(t *testing.T) {
	repo := &fakeRepo{listCategoriesResult: []sqlc.Category{
		{ID: 1, Code: "groceries", Name: "Groceries", IsActive: true},
		{ID: 2, Code: "utilities", Name: "Utilities", IsActive: true},
	}}
	svc := NewService(repo, &fakeTransactionService{})

	categories, err := svc.ListCategories(context.Background(), ListCategoriesRequest{})

	if err != nil {
		t.Fatalf("ListCategories returned error: %v", err)
	}
	if len(categories) != 2 || categories[0].Code != "groceries" || categories[1].Code != "utilities" {
		t.Fatalf("categories = %+v, want mapped category DTOs", categories)
	}
}
```

- [ ] **Step 2: Run category tests and confirm missing implementation**

Run:

```bash
go test ./internal/app -run 'TestCreateCategory|TestListCategories' -count=1
```

Expected: FAIL because `CreateCategoryRequest`, `CategoryDTO`, and service methods are undefined.

- [ ] **Step 3: Add category service implementation**

Create `internal/app/categories.go`:

```go
package app

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"unicode"

	"rdmm404/voltr-finance/internal/database/sqlc"
)

var categoryCodePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type CategoryDTO struct {
	ID          int64   `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	IsActive    bool    `json:"isActive"`
}

type CategoryRefDTO struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type CreateCategoryRequest struct {
	Name        string  `json:"name"`
	Code        *string `json:"code,omitempty"`
	Description *string `json:"description,omitempty"`
}

type ListCategoriesRequest struct {
	IncludeInactive bool `json:"includeInactive,omitempty"`
}

type UpdateCategoryRequest struct {
	ID               int64   `json:"id"`
	Name             *string `json:"name,omitempty"`
	Description      *string `json:"description,omitempty"`
	ClearDescription bool    `json:"clearDescription,omitempty"`
}

func (s *Service) CreateCategory(ctx context.Context, req CreateCategoryRequest) (CategoryDTO, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return CategoryDTO{}, NewError(CodeValidationError, "category name is required", nil)
	}
	code := ""
	if req.Code != nil {
		code = strings.TrimSpace(*req.Code)
	} else {
		code = categoryCodeFromName(name)
	}
	if !categoryCodePattern.MatchString(code) {
		return CategoryDTO{}, NewError(CodeValidationError, "category code must be a lowercase slug", nil)
	}
	category, err := s.repo.CreateCategory(ctx, sqlc.CreateCategoryParams{
		Code:        code,
		Name:        name,
		Description: req.Description,
	})
	if err != nil {
		return CategoryDTO{}, mapCategoryError(err)
	}
	return categoryDTO(category), nil
}

func (s *Service) ListCategories(ctx context.Context, req ListCategoriesRequest) ([]CategoryDTO, error) {
	rows, err := s.repo.ListCategories(ctx, req.IncludeInactive)
	if err != nil {
		return nil, mapCategoryError(err)
	}
	categories := make([]CategoryDTO, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, categoryDTO(row))
	}
	return categories, nil
}

func (s *Service) GetCategoryByCode(ctx context.Context, code string) (CategoryDTO, error) {
	category, err := s.repo.GetCategoryByCode(ctx, code)
	if err != nil {
		return CategoryDTO{}, mapCategoryError(err)
	}
	return categoryDTO(category), nil
}

func (s *Service) UpdateCategory(ctx context.Context, req UpdateCategoryRequest) (CategoryDTO, error) {
	if req.ID == 0 {
		return CategoryDTO{}, NewError(CodeValidationError, "category id is required", nil)
	}
	if req.Name == nil && req.Description == nil && !req.ClearDescription {
		return CategoryDTO{}, NewError(CodeValidationError, "at least one category field is required", nil)
	}
	name := ""
	setName := false
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
		if name == "" {
			return CategoryDTO{}, NewError(CodeValidationError, "category name is required", nil)
		}
		setName = true
	}
	category, err := s.repo.UpdateCategory(ctx, sqlc.UpdateCategoryParams{
		ID:             req.ID,
		SetName:        setName,
		Name:           name,
		SetDescription: req.Description != nil || req.ClearDescription,
		Description:    req.Description,
	})
	if err != nil {
		return CategoryDTO{}, mapCategoryError(err)
	}
	return categoryDTO(category), nil
}

func (s *Service) DeactivateCategory(ctx context.Context, code string) (CategoryDTO, error) {
	if !categoryCodePattern.MatchString(code) {
		return CategoryDTO{}, NewError(CodeValidationError, "category code must be a lowercase slug", nil)
	}
	category, err := s.repo.DeactivateCategory(ctx, code)
	if err != nil {
		return CategoryDTO{}, mapCategoryError(err)
	}
	return categoryDTO(category), nil
}

func categoryCodeFromName(name string) string {
	var b strings.Builder
	lastHyphen := true
	for _, r := range strings.ToLower(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastHyphen = false
		default:
			if !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func categoryDTO(category sqlc.Category) CategoryDTO {
	return CategoryDTO{
		ID:          category.ID,
		Code:        category.Code,
		Name:        category.Name,
		Description: category.Description,
		IsActive:    category.IsActive,
	}
}

func mapCategoryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return NewError(CodeValidationError, "category not found", err)
	}
	return NewError(CodeDatabaseError, "database error", err)
}
```

- [ ] **Step 4: Extend repository interface**

In `internal/app/service.go`, update `Repository`:

```go
type Repository interface {
	UserRepository
	HouseholdRepository
	TransactionRepository
	CategoryRepository
}
```

Add:

```go
type CategoryRepository interface {
	CreateCategory(context.Context, sqlc.CreateCategoryParams) (sqlc.Category, error)
	ListCategories(context.Context, bool) ([]sqlc.Category, error)
	GetCategoryById(context.Context, int64) (sqlc.Category, error)
	GetActiveCategoryById(context.Context, int64) (sqlc.Category, error)
	GetCategoryByCode(context.Context, string) (sqlc.Category, error)
	GetActiveCategoryByCode(context.Context, string) (sqlc.Category, error)
	UpdateCategory(context.Context, sqlc.UpdateCategoryParams) (sqlc.Category, error)
	DeactivateCategory(context.Context, string) (sqlc.Category, error)
}
```

- [ ] **Step 5: Extend app fakes**

In `internal/app/test_fakes_test.go`, add fields to `fakeRepo`:

```go
lastCreateCategory  sqlc.CreateCategoryParams
lastUpdateCategory  sqlc.UpdateCategoryParams
categoryByID        sqlc.Category
categoryByCode      sqlc.Category
listCategoriesResult []sqlc.Category
```

Add fake methods:

```go
func (f *fakeRepo) CreateCategory(_ context.Context, arg sqlc.CreateCategoryParams) (sqlc.Category, error) {
	f.lastCreateCategory = arg
	return sqlc.Category{ID: 1, Code: arg.Code, Name: arg.Name, Description: arg.Description, IsActive: true}, nil
}

func (f *fakeRepo) ListCategories(context.Context, bool) ([]sqlc.Category, error) {
	return f.listCategoriesResult, nil
}

func (f *fakeRepo) GetCategoryById(context.Context, int64) (sqlc.Category, error) {
	if f.categoryByID.ID == 0 {
		return sqlc.Category{}, sql.ErrNoRows
	}
	return f.categoryByID, nil
}

func (f *fakeRepo) GetActiveCategoryById(context.Context, int64) (sqlc.Category, error) {
	if f.categoryByID.ID == 0 || !f.categoryByID.IsActive {
		return sqlc.Category{}, sql.ErrNoRows
	}
	return f.categoryByID, nil
}

func (f *fakeRepo) GetCategoryByCode(context.Context, string) (sqlc.Category, error) {
	if f.categoryByCode.ID == 0 {
		return sqlc.Category{}, sql.ErrNoRows
	}
	return f.categoryByCode, nil
}

func (f *fakeRepo) GetActiveCategoryByCode(context.Context, string) (sqlc.Category, error) {
	if f.categoryByCode.ID == 0 || !f.categoryByCode.IsActive {
		return sqlc.Category{}, sql.ErrNoRows
	}
	return f.categoryByCode, nil
}

func (f *fakeRepo) UpdateCategory(_ context.Context, arg sqlc.UpdateCategoryParams) (sqlc.Category, error) {
	f.lastUpdateCategory = arg
	name := arg.Name
	if !arg.SetName {
		name = f.categoryByID.Name
	}
	return sqlc.Category{ID: arg.ID, Code: f.categoryByID.Code, Name: name, Description: arg.Description, IsActive: true}, nil
}

func (f *fakeRepo) DeactivateCategory(context.Context, string) (sqlc.Category, error) {
	if f.categoryByCode.ID == 0 {
		return sqlc.Category{}, sql.ErrNoRows
	}
	f.categoryByCode.IsActive = false
	return f.categoryByCode, nil
}
```

- [ ] **Step 6: Run category service tests**

Run:

```bash
go test ./internal/app -run 'TestCreateCategory|TestListCategories' -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit app category service**

```bash
git add internal/app/categories.go internal/app/categories_test.go internal/app/service.go internal/app/test_fakes_test.go
git commit -m "feat: add category app service"
```

---

### Task 3: Transaction Category Assignment

**Files:**
- Modify: `internal/app/transactions.go`
- Modify: `internal/app/transactions_test.go`
- Modify: `internal/app/test_fakes_test.go`
- Modify: `internal/transaction/types.go`
- Modify: `internal/transaction/transaction_service.go`
- Modify: `internal/transaction/hashing.go`

- [ ] **Step 1: Write app tests for transaction category assignment**

Add tests to `internal/app/transactions_test.go`:

```go
func TestCreateTransactionResolvesCategoryCode(t *testing.T) {
	telegramID := "123456"
	repo := &fakeRepo{
		userByTelegram: sqlc.User{ID: 7, Name: "Rafael", TelegramID: &telegramID},
		categoryByCode: sqlc.Category{ID: 3, Code: "groceries", Name: "Groceries", IsActive: true},
	}
	txSvc := &fakeTransactionService{saveResult: transaction.TransactionResult{Success: map[int64]*sqlc.Transaction{101: {ID: 101}}}}
	svc := NewService(repo, txSvc)

	result := svc.CreateTransaction(context.Background(), CreateTransactionRequest{
		Amount:          42.5,
		TransactionDate: time.Date(2026, 5, 5, 14, 30, 0, 0, time.UTC),
		Author:          IdentitySelector{TelegramID: &telegramID},
		HouseholdID:     intPtr(1),
		CategoryCode:    strPtr("groceries"),
	})

	if len(result.Errors) != 0 {
		t.Fatalf("Errors = %v, want none", result.Errors)
	}
	if txSvc.saved[0].CategoryID == nil || *txSvc.saved[0].CategoryID != 3 {
		t.Fatalf("category id = %v, want 3", txSvc.saved[0].CategoryID)
	}
}

func TestCreateTransactionRejectsConflictingCategorySelectors(t *testing.T) {
	telegramID := "123456"
	repo := &fakeRepo{
		userByTelegram: sqlc.User{ID: 7, Name: "Rafael", TelegramID: &telegramID},
		categoryByCode: sqlc.Category{ID: 3, Code: "groceries", Name: "Groceries", IsActive: true},
	}
	svc := NewService(repo, &fakeTransactionService{})

	result := svc.CreateTransaction(context.Background(), CreateTransactionRequest{
		Amount:          42.5,
		TransactionDate: time.Date(2026, 5, 5, 14, 30, 0, 0, time.UTC),
		Author:          IdentitySelector{TelegramID: &telegramID},
		HouseholdID:     intPtr(1),
		CategoryID:      intPtr(9),
		CategoryCode:    strPtr("groceries"),
	})

	if len(result.Errors) != 1 || result.Errors[0].Code != CodeValidationError {
		t.Fatalf("Errors = %+v, want validation error", result.Errors)
	}
}

func TestGetTransactionsIncludesCategoryDetails(t *testing.T) {
	householdID := int64(1)
	householdName := "Voltr"
	categoryID := int64(3)
	categoryCode := "groceries"
	categoryName := "Groceries"
	repo := &fakeRepo{
		transactionDetails: []sqlc.GetTransactionsByIdWithDetailsRow{
			{
				Transaction:   sqlc.Transaction{ID: 101, AuthorID: 9, HouseholdID: &householdID, CategoryID: &categoryID},
				AuthorName:    "CLI Tester",
				HouseholdID:   &householdID,
				HouseholdName: &householdName,
				CategoryID:    &categoryID,
				CategoryCode:  &categoryCode,
				CategoryName:  &categoryName,
			},
		},
	}
	svc := NewService(repo, &fakeTransactionService{})

	txs, err := svc.GetTransactions(context.Background(), []int64{101}, false)
	if err != nil {
		t.Fatalf("GetTransactions returned error: %v", err)
	}
	if txs[0].Category == nil || txs[0].Category.Code != "groceries" {
		t.Fatalf("category = %+v, want groceries", txs[0].Category)
	}
}
```

- [ ] **Step 2: Run transaction app tests and confirm failures**

Run:

```bash
go test ./internal/app -run 'TestCreateTransactionResolvesCategoryCode|TestCreateTransactionRejectsConflictingCategorySelectors|TestGetTransactionsIncludesCategoryDetails' -count=1
```

Expected: FAIL because transaction request/DTO fields and sqlc params are not wired.

- [ ] **Step 3: Replace transaction request and DTO fields**

In `internal/app/transactions.go`, update structs:

```go
type TransactionDTO struct {
	ID              int64           `json:"id"`
	Amount          float32         `json:"amount"`
	TransactionDate time.Time       `json:"transactionDate"`
	AuthorID        int64           `json:"authorId"`
	AuthorName      string          `json:"authorName,omitempty"`
	HouseholdID     *int64          `json:"householdId,omitempty"`
	HouseholdName   *string         `json:"householdName,omitempty"`
	Category        *CategoryRefDTO `json:"category,omitempty"`
	Description     *string         `json:"description,omitempty"`
	Notes           *string         `json:"notes,omitempty"`
	CreatedAt       *time.Time      `json:"createdAt,omitempty"`
	UpdatedAt       *time.Time      `json:"updatedAt,omitempty"`
	DeletedAt       *time.Time      `json:"deletedAt,omitempty"`
	DeleteReason    *string         `json:"deleteReason,omitempty"`
}
```

In `CreateTransactionRequest`, replace `BudgetCategoryID` with:

```go
CategoryID   *int64  `json:"categoryId,omitempty"`
CategoryCode *string `json:"categoryCode,omitempty"`
```

In `UpdateTransactionRequest`, replace budget category fields with:

```go
CategoryID      *int64  `json:"categoryId,omitempty"`
CategoryCode    *string `json:"categoryCode,omitempty"`
ClearCategoryID bool    `json:"clearCategoryId,omitempty"`
```

- [ ] **Step 4: Add category resolution helper**

Add to `internal/app/transactions.go`:

```go
func (s *Service) resolveCategoryID(ctx context.Context, id *int64, code *string) (*int64, error) {
	if id == nil && code == nil {
		return nil, nil
	}
	var byID *sqlc.Category
	if id != nil {
		category, err := s.repo.GetActiveCategoryById(ctx, *id)
		if err != nil {
			return nil, mapCategoryError(err)
		}
		byID = &category
	}
	if code != nil {
		category, err := s.repo.GetActiveCategoryByCode(ctx, *code)
		if err != nil {
			return nil, mapCategoryError(err)
		}
		if byID != nil && byID.ID != category.ID {
			return nil, NewError(CodeValidationError, "category id and code refer to different categories", nil)
		}
		return &category.ID, nil
	}
	return id, nil
}
```

- [ ] **Step 5: Wire create and update params**

In `createParams`, resolve category before returning params:

```go
categoryID, err := s.resolveCategoryID(ctx, req.CategoryID, req.CategoryCode)
if err != nil {
	return sqlc.CreateTransactionParams{}, err
}
```

Return:

```go
return sqlc.CreateTransactionParams{
	Amount:          req.Amount,
	CategoryID:      categoryID,
	Description:     req.Description,
	TransactionDate: pgtype.Timestamptz{Time: req.TransactionDate, Valid: true},
	AuthorID:        author.ID,
	HouseholdID:     req.HouseholdID,
	Notes:           req.Notes,
}, nil
```

In `updateParams`, replace budget category handling:

```go
if req.CategoryID != nil || req.CategoryCode != nil || req.ClearCategoryID {
	categoryID, err := s.resolveCategoryID(ctx, req.CategoryID, req.CategoryCode)
	if err != nil {
		return transaction.UpdateTransactionById{}, err
	}
	updates.CategoryID = utils.NewOptional(categoryID)
}
```

- [ ] **Step 6: Update DTO mapping**

Change `transactionDTO` signature:

```go
func transactionDTO(tx sqlc.Transaction, authorName string, householdName *string, categoryID *int64, categoryCode *string, categoryName *string) TransactionDTO {
	var category *CategoryRefDTO
	if categoryID != nil && categoryCode != nil && categoryName != nil {
		category = &CategoryRefDTO{ID: *categoryID, Code: *categoryCode, Name: *categoryName}
	}
	return TransactionDTO{
		ID:              tx.ID,
		Amount:          tx.Amount,
		TransactionDate: tx.TransactionDate.Time,
		AuthorID:        tx.AuthorID,
		AuthorName:      authorName,
		HouseholdID:     tx.HouseholdID,
		HouseholdName:   householdName,
		Category:        category,
		Description:     tx.Description,
		Notes:           tx.Notes,
		CreatedAt:       validTime(tx.CreatedAt.Time, tx.CreatedAt.Valid),
		UpdatedAt:       validTime(tx.UpdatedAt.Time, tx.UpdatedAt.Valid),
		DeletedAt:       validTime(tx.DeletedAt.Time, tx.DeletedAt.Valid),
		DeleteReason:    tx.DeleteReason,
	}
}
```

Update call sites:

```go
transactionDTO(row.Transaction, row.AuthorName, row.HouseholdName, row.CategoryID, row.CategoryCode, row.CategoryName)
```

- [ ] **Step 7: Update transaction package fields**

In `internal/transaction/types.go`, replace:

```go
BudgetCategoryID utils.Optional[*int64]
```

with:

```go
CategoryID utils.Optional[*int64]
```

In `internal/transaction/transaction_service.go`, update params:

```go
SetCategoryID: trans.Updates.CategoryID.Set,
CategoryID:    trans.Updates.CategoryID.Value,
```

Remove `SetBudgetCategoryID` and `BudgetCategoryID` references.

In `internal/transaction/hashing.go`, replace references to budget category with category ID so duplicate detection includes category assignment consistently.

- [ ] **Step 8: Run app and transaction tests**

Run:

```bash
go test ./internal/app ./internal/transaction -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit transaction category flow**

```bash
git add internal/app/transactions.go internal/app/transactions_test.go internal/app/test_fakes_test.go internal/transaction
git commit -m "feat: assign categories to transactions"
```

---

### Task 4: CLI Category Commands and Transaction Flags

**Files:**
- Modify: `internal/cli/commands.go`
- Modify: `internal/cli/commands_test.go`
- Modify: `internal/cli/render.go`

- [ ] **Step 1: Write CLI tests**

Add to `internal/cli/commands_test.go`:

```go
func TestKongCategoryCreate(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"categories", "create",
		"Restaurants & Takeout",
		"--code", "restaurants",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if svc.createCategory.Name != "Restaurants & Takeout" || svc.createCategory.Code == nil || *svc.createCategory.Code != "restaurants" {
		t.Fatalf("createCategory = %+v, want name and explicit code", svc.createCategory)
	}
}

func TestKongTransactionsCreateCategoryFlag(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"transactions", "create",
		"--amount", "42.50",
		"--transaction-date", "2026-05-05T14:30:00-04:00",
		"--author-telegram-id", "123456",
		"--household-id", "1",
		"--category", "groceries",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if svc.createTransaction.CategoryCode == nil || *svc.createTransaction.CategoryCode != "groceries" {
		t.Fatalf("category = %v, want groceries", svc.createTransaction.CategoryCode)
	}
}

func TestKongTransactionsUpdateClearCategory(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"transactions", "update",
		"--id", "101",
		"--clear-category",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !svc.updateTransaction.ClearCategoryID {
		t.Fatalf("ClearCategoryID = false, want true")
	}
}
```

- [ ] **Step 2: Run CLI tests and confirm failures**

Run:

```bash
go test ./internal/cli -run 'TestKongCategoryCreate|TestKongTransactionsCreateCategoryFlag|TestKongTransactionsUpdateClearCategory' -count=1
```

Expected: FAIL because category CLI service methods and flags are not defined.

- [ ] **Step 3: Extend CLI service interface**

In `internal/cli/commands.go`, add to `AppService`:

```go
CreateCategory(context.Context, app.CreateCategoryRequest) (app.CategoryDTO, error)
ListCategories(context.Context, app.ListCategoriesRequest) ([]app.CategoryDTO, error)
UpdateCategory(context.Context, app.UpdateCategoryRequest) (app.CategoryDTO, error)
DeactivateCategory(context.Context, string) (app.CategoryDTO, error)
```

- [ ] **Step 4: Add categories command group**

In `CLI`, add:

```go
Categories CategoriesCmd `cmd:"" help:"Manage transaction categories."`
```

Add command structs:

```go
type CategoriesCmd struct {
	Create     CategoryCreateCmd     `cmd:"" help:"Create a category."`
	List       CategoryListCmd       `cmd:"" help:"List categories."`
	Rename     CategoryRenameCmd     `cmd:"" help:"Rename a category by code."`
	Deactivate CategoryDeactivateCmd `cmd:"" help:"Deactivate a category by code."`
}

type CategoryCreateCmd struct {
	Name        string  `arg:"" required:"" help:"Category display name."`
	Code        *string `help:"Stable category code. Defaults to a slug generated from name."`
	Description *string `help:"Optional category description."`
}

func (c *CategoryCreateCmd) Run(ctx *runContext) error {
	category, err := ctx.svc.CreateCategory(ctx.Context, app.CreateCategoryRequest{
		Name:        c.Name,
		Code:        c.Code,
		Description: c.Description,
	})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, category)
}

type CategoryListCmd struct {
	IncludeInactive bool `help:"Include inactive categories."`
}

func (c *CategoryListCmd) Run(ctx *runContext) error {
	categories, err := ctx.svc.ListCategories(ctx.Context, app.ListCategoriesRequest{IncludeInactive: c.IncludeInactive})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, categories)
}

type CategoryRenameCmd struct {
	Code string `arg:"" required:"" help:"Existing category code."`
	Name string `arg:"" required:"" help:"New category display name."`
}

func (c *CategoryRenameCmd) Run(ctx *runContext) error {
	existing, err := ctx.svc.GetCategoryByCode(ctx.Context, c.Code)
	if err != nil {
		return err
	}
	category, err := ctx.svc.UpdateCategory(ctx.Context, app.UpdateCategoryRequest{
		ID:   existing.ID,
		Name: &c.Name,
	})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, category)
}

type CategoryDeactivateCmd struct {
	Code string `arg:"" required:"" help:"Category code to deactivate."`
}

func (c *CategoryDeactivateCmd) Run(ctx *runContext) error {
	category, err := ctx.svc.DeactivateCategory(ctx.Context, c.Code)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, category)
}
```

Also add `GetCategoryByCode(context.Context, string) (app.CategoryDTO, error)` to `AppService`; it is needed by `rename`.

- [ ] **Step 5: Replace transaction flags**

In `TransactionCreateCmd`, remove `BudgetCategoryID` and add:

```go
Category *string `help:"Category code."`
```

In `Run`, pass:

```go
CategoryCode: c.Category,
```

In `TransactionUpdateCmd`, remove `BudgetCategoryID` and `ClearBudgetCategoryID`, then add:

```go
Category      *string `help:"Replacement category code."`
ClearCategory bool    `help:"Clear the transaction category."`
```

In `Run`, pass:

```go
CategoryCode:    c.Category,
ClearCategoryID: c.ClearCategory,
```

- [ ] **Step 6: Update renderers**

In `RenderTransactionCompact`, insert a `Category` line:

```go
{"Category", categoryValue(tx.Category)},
```

Add helper:

```go
func categoryValue(category *app.CategoryRefDTO) string {
	if category == nil {
		return ""
	}
	return category.Name
}
```

In `RenderTransactionsCSV`, add headers:

```go
"category_code",
"category_name",
```

Add row values after household columns:

```go
categoryCode(tx.Category),
categoryValue(tx.Category),
```

Add:

```go
func categoryCode(category *app.CategoryRefDTO) string {
	if category == nil {
		return ""
	}
	return category.Code
}
```

- [ ] **Step 7: Extend CLI fake service**

In `internal/cli/commands_test.go`, add fields:

```go
createCategory    app.CreateCategoryRequest
updateCategory    app.UpdateCategoryRequest
updateTransaction app.UpdateTransactionRequest
```

Update fake methods:

```go
func (f *fakeAppService) UpdateTransaction(_ context.Context, req app.UpdateTransactionRequest) app.WriteResult {
	f.updateTransaction = req
	return app.WriteResult{}
}

func (f *fakeAppService) CreateCategory(_ context.Context, req app.CreateCategoryRequest) (app.CategoryDTO, error) {
	f.createCategory = req
	code := ""
	if req.Code != nil {
		code = *req.Code
	}
	return app.CategoryDTO{ID: 1, Code: code, Name: req.Name, IsActive: true}, nil
}

func (f *fakeAppService) ListCategories(context.Context, app.ListCategoriesRequest) ([]app.CategoryDTO, error) {
	return []app.CategoryDTO{}, nil
}

func (f *fakeAppService) GetCategoryByCode(context.Context, string) (app.CategoryDTO, error) {
	return app.CategoryDTO{ID: 1, Code: "groceries", Name: "Groceries", IsActive: true}, nil
}

func (f *fakeAppService) UpdateCategory(_ context.Context, req app.UpdateCategoryRequest) (app.CategoryDTO, error) {
	f.updateCategory = req
	return app.CategoryDTO{ID: req.ID, Code: "groceries", Name: *req.Name, IsActive: true}, nil
}

func (f *fakeAppService) DeactivateCategory(context.Context, string) (app.CategoryDTO, error) {
	return app.CategoryDTO{ID: 1, Code: "groceries", Name: "Groceries", IsActive: false}, nil
}
```

- [ ] **Step 8: Run CLI tests**

Run:

```bash
go test ./internal/cli -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit CLI category support**

```bash
git add internal/cli/commands.go internal/cli/commands_test.go internal/cli/render.go
git commit -m "feat: add category CLI commands"
```

---

### Task 5: Final Verification

**Files:**
- Review: `docs/superpowers/specs/2026-05-08-transaction-categories-design.md`
- Review: all changed files

- [ ] **Step 1: Run full test suite**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 2: Verify migration shape**

Run:

```bash
grep -RIn "budget_category_id\\|BudgetCategoryID\\|budget_category" internal db/migrations/20260508000000_transaction_categories.sql
```

Expected: only the new migration down section may contain `budget_category` or `budget_category_id`. No `internal/` files should contain `BudgetCategoryID`.

- [ ] **Step 3: Verify CLI help includes category commands**

Run:

```bash
go test ./internal/cli -run TestKongHelpDocumentsFlagSemantics -count=1
```

Expected: PASS after adding category help assertions or leaving existing assertions unaffected.

- [ ] **Step 4: Review git diff**

Run:

```bash
git diff --stat HEAD
git diff --name-only HEAD
```

Expected: changed files match this plan: migration, query/sqlc generated files, app category/transaction files, transaction package files, CLI files, tests.

- [ ] **Step 5: Commit final cleanups if any**

If Step 4 shows formatting-only or small missed wiring fixes, stage only those files and commit:

```bash
git add <specific-files>
git commit -m "chore: finish category wiring"
```

If there are no changes after Step 4, skip this commit.
