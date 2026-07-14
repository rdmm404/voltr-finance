package budgets

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	appbudgets "rdmm404/voltr-finance/internal/app/budgets"
	apperrors "rdmm404/voltr-finance/internal/app/errors"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/postgres"
)

// Repository owns all PostgreSQL mechanics needed by the cohesive budget port.
type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) FindMonthly(ctx context.Context, owner appbudgets.Owner, start, end time.Time) (appbudgets.Budget, error) {
	return withTransaction(ctx, r.pool, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}, func(q *sqlc.Queries) (appbudgets.Budget, error) {
		budget, err := findMonthly(ctx, q, owner, start, end)
		if err != nil {
			return appbudgets.Budget{}, err
		}
		return loadBudget(ctx, q, budget)
	})
}

func (r *Repository) CreateMonthlyFromTemplate(ctx context.Context, input appbudgets.CreateMonthlyFromTemplateInput) (appbudgets.Budget, error) {
	return withTransaction(ctx, r.pool, pgx.TxOptions{}, func(q *sqlc.Queries) (appbudgets.Budget, error) {
		prior, err := findLatestPrior(ctx, q, input.Owner, input.PeriodStart)
		if err != nil && !apperrors.IsKind(err, apperrors.KindNotFound) {
			return appbudgets.Budget{}, err
		}
		var sourceID *int64
		if err == nil {
			sourceID = &prior.ID
		}
		created, err := createBudget(ctx, q, input.Owner, input.PeriodStart, input.PeriodEnd, sourceID)
		if err != nil {
			return appbudgets.Budget{}, err
		}
		if sourceID != nil {
			if err := copyStructure(ctx, q, prior.ID, created.ID); err != nil {
				return appbudgets.Budget{}, err
			}
		}
		return loadBudget(ctx, q, created)
	})
}

func (r *Repository) CreateLineWithCategories(ctx context.Context, input appbudgets.CreateLineInput) (appbudgets.Line, error) {
	return withTransaction(ctx, r.pool, pgx.TxOptions{}, func(q *sqlc.Queries) (appbudgets.Line, error) {
		if _, err := q.GetBudgetById(ctx, input.BudgetID); err != nil {
			return appbudgets.Line{}, mapBudgetError(err)
		}
		categoryIDs, err := resolveCategoryIDs(ctx, q, input.CategoryIDs, input.CategoryCodes)
		if err != nil {
			return appbudgets.Line{}, err
		}
		sortOrder := input.SortOrder
		if sortOrder == nil {
			if _, err := q.LockBudgetForUpdate(ctx, input.BudgetID); err != nil {
				return appbudgets.Line{}, mapBudgetError(err)
			}
			max, err := q.GetMaxBudgetLineSortOrder(ctx, input.BudgetID)
			if err != nil {
				return appbudgets.Line{}, mapBudgetError(err)
			}
			next := max + 1
			sortOrder = &next
		}
		created, err := createLine(ctx, q, input.BudgetID, input.Name, input.AllocationAmount, *sortOrder)
		if err != nil {
			return appbudgets.Line{}, err
		}
		if err := replaceCategories(ctx, q, input.BudgetID, created.ID, categoryIDs); err != nil {
			return appbudgets.Line{}, err
		}
		return loadLine(ctx, q, created)
	})
}

func (r *Repository) UpdateLineWithCategories(ctx context.Context, input appbudgets.UpdateLineInput) (appbudgets.Line, error) {
	return withTransaction(ctx, r.pool, pgx.TxOptions{}, func(q *sqlc.Queries) (appbudgets.Line, error) {
		row, err := q.GetBudgetLineById(ctx, input.LineID)
		if err != nil {
			return appbudgets.Line{}, mapLineError(err)
		}
		existing, err := mapLine(row)
		if err != nil {
			return appbudgets.Line{}, err
		}
		changeCategories := input.CategoryIDs != nil || input.CategoryCodes != nil
		var categoryIDs []int64
		if changeCategories {
			var ids []int64
			var codes []string
			if input.CategoryIDs != nil {
				ids = *input.CategoryIDs
			}
			if input.CategoryCodes != nil {
				codes = *input.CategoryCodes
			}
			categoryIDs, err = resolveCategoryIDs(ctx, q, ids, codes)
			if err != nil {
				return appbudgets.Line{}, err
			}
		}
		updated, err := updateLine(ctx, q, input)
		if err != nil {
			return appbudgets.Line{}, err
		}
		if changeCategories {
			if err := replaceCategories(ctx, q, existing.BudgetID, existing.ID, categoryIDs); err != nil {
				return appbudgets.Line{}, err
			}
		}
		return loadLine(ctx, q, updated)
	})
}

func (r *Repository) DeleteLine(ctx context.Context, id int64) error {
	q := sqlc.New(r.pool)
	if _, err := q.GetBudgetLineById(ctx, id); err != nil {
		return mapLineError(err)
	}
	return mapLineError(q.DeleteBudgetLine(ctx, id))
}

func (r *Repository) LoadReportSnapshot(ctx context.Context, budgetID int64) (appbudgets.ReportSnapshot, error) {
	return withTransaction(ctx, r.pool, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}, func(q *sqlc.Queries) (appbudgets.ReportSnapshot, error) {
		budgetRow, err := q.GetBudgetById(ctx, budgetID)
		if err != nil {
			return appbudgets.ReportSnapshot{}, mapBudgetError(err)
		}
		rows, err := listReportLines(ctx, q, budgetID)
		if err != nil {
			return appbudgets.ReportSnapshot{}, err
		}
		mappings, err := listLineCategories(ctx, q, budgetID)
		if err != nil {
			return appbudgets.ReportSnapshot{}, err
		}
		categories := make(map[int64][]appbudgets.Category)
		for _, mapping := range mappings {
			categories[mapping.lineID] = append(categories[mapping.lineID], mapping.category)
		}
		for i := range rows {
			rows[i].Categories = nonNilCategories(categories[rows[i].ID])
		}
		uncategorized, err := sumUncategorized(ctx, q, budgetID)
		if err != nil {
			return appbudgets.ReportSnapshot{}, err
		}
		unmapped, err := listUnmappedTransactions(ctx, q, budgetID)
		if err != nil {
			return appbudgets.ReportSnapshot{}, err
		}
		return appbudgets.ReportSnapshot{Budget: mapBudget(budgetRow), Lines: rows, UnmappedTransactions: unmapped, UncategorizedAmount: uncategorized}, nil
	})
}

func findMonthly(ctx context.Context, q *sqlc.Queries, owner appbudgets.Owner, start, end time.Time) (appbudgets.Budget, error) {
	var row sqlc.Budget
	var err error
	if owner.HouseholdID != nil {
		row, err = q.GetHouseholdBudgetByPeriod(ctx, sqlc.GetHouseholdBudgetByPeriodParams{HouseholdID: *owner.HouseholdID, PeriodStart: date(start), PeriodEnd: date(end)})
	} else if owner.UserID != nil {
		row, err = q.GetUserBudgetByPeriod(ctx, sqlc.GetUserBudgetByPeriodParams{UserID: *owner.UserID, PeriodStart: date(start), PeriodEnd: date(end)})
	} else {
		return appbudgets.Budget{}, apperrors.Validation("budget owner is required")
	}
	return mapBudget(row), mapBudgetError(err)
}

func findLatestPrior(ctx context.Context, q *sqlc.Queries, owner appbudgets.Owner, start time.Time) (appbudgets.Budget, error) {
	var row sqlc.Budget
	var err error
	if owner.HouseholdID != nil {
		row, err = q.GetLatestPriorHouseholdBudget(ctx, sqlc.GetLatestPriorHouseholdBudgetParams{HouseholdID: *owner.HouseholdID, PeriodStart: date(start)})
	} else if owner.UserID != nil {
		row, err = q.GetLatestPriorUserBudget(ctx, sqlc.GetLatestPriorUserBudgetParams{UserID: *owner.UserID, PeriodStart: date(start)})
	} else {
		return appbudgets.Budget{}, apperrors.Validation("budget owner is required")
	}
	return mapBudget(row), mapBudgetError(err)
}

func createBudget(ctx context.Context, q *sqlc.Queries, owner appbudgets.Owner, start, end time.Time, sourceID *int64) (appbudgets.Budget, error) {
	var row sqlc.Budget
	var err error
	if owner.HouseholdID != nil {
		row, err = q.CreateHouseholdBudget(ctx, sqlc.CreateHouseholdBudgetParams{HouseholdID: *owner.HouseholdID, PeriodStart: date(start), PeriodEnd: date(end), SourceBudgetID: sourceID})
	} else if owner.UserID != nil {
		row, err = q.CreateUserBudget(ctx, sqlc.CreateUserBudgetParams{UserID: *owner.UserID, PeriodStart: date(start), PeriodEnd: date(end), SourceBudgetID: sourceID})
	} else {
		return appbudgets.Budget{}, apperrors.Validation("budget owner is required")
	}
	return mapBudget(row), mapBudgetError(err)
}

func copyStructure(ctx context.Context, q *sqlc.Queries, sourceID, targetID int64) error {
	lines, err := listLines(ctx, q, sourceID)
	if err != nil {
		return err
	}
	mappings, err := listLineCategories(ctx, q, sourceID)
	if err != nil {
		return err
	}
	byLine := make(map[int64][]int64)
	for _, mapping := range mappings {
		byLine[mapping.lineID] = append(byLine[mapping.lineID], mapping.category.ID)
	}
	for _, source := range lines {
		created, err := createLine(ctx, q, targetID, source.Name, source.AllocationAmount, source.SortOrder)
		if err != nil {
			return err
		}
		for _, categoryID := range byLine[source.ID] {
			if err := createLineCategory(ctx, q, targetID, created.ID, categoryID); err != nil {
				return err
			}
		}
	}
	return nil
}

func loadBudget(ctx context.Context, q *sqlc.Queries, budget appbudgets.Budget) (appbudgets.Budget, error) {
	lines, err := listLines(ctx, q, budget.ID)
	if err != nil {
		return appbudgets.Budget{}, err
	}
	mappings, err := listLineCategories(ctx, q, budget.ID)
	if err != nil {
		return appbudgets.Budget{}, err
	}
	categories := make(map[int64][]appbudgets.Category)
	for _, mapping := range mappings {
		categories[mapping.lineID] = append(categories[mapping.lineID], mapping.category)
	}
	for i := range lines {
		lines[i].Categories = nonNilCategories(categories[lines[i].ID])
	}
	budget.Lines = nonNilLines(lines)
	return budget, nil
}

func loadLine(ctx context.Context, q *sqlc.Queries, line appbudgets.Line) (appbudgets.Line, error) {
	mappings, err := listLineCategories(ctx, q, line.BudgetID)
	if err != nil {
		return appbudgets.Line{}, err
	}
	line.Categories = []appbudgets.Category{}
	for _, mapping := range mappings {
		if mapping.lineID == line.ID {
			line.Categories = append(line.Categories, mapping.category)
		}
	}
	return line, nil
}

func listLines(ctx context.Context, q *sqlc.Queries, budgetID int64) ([]appbudgets.Line, error) {
	rows, err := q.ListBudgetLines(ctx, budgetID)
	if err != nil {
		return nil, mapBudgetError(err)
	}
	items := make([]appbudgets.Line, 0, len(rows))
	for _, row := range rows {
		item, err := mapLine(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

type lineCategory struct {
	lineID   int64
	category appbudgets.Category
}

func listLineCategories(ctx context.Context, q *sqlc.Queries, budgetID int64) ([]lineCategory, error) {
	rows, err := q.ListBudgetLineCategories(ctx, budgetID)
	if err != nil {
		return nil, mapBudgetError(err)
	}
	items := make([]lineCategory, 0, len(rows))
	for _, row := range rows {
		items = append(items, lineCategory{lineID: row.BudgetLineID, category: appbudgets.Category{ID: row.CategoryID, Code: row.CategoryCode, Name: row.CategoryName}})
	}
	return items, nil
}

func createLine(ctx context.Context, q *sqlc.Queries, budgetID int64, name, amount string, sortOrder int32) (appbudgets.Line, error) {
	numericAmount, err := numeric(amount)
	if err != nil {
		return appbudgets.Line{}, apperrors.Internal(err)
	}
	row, err := q.CreateBudgetLine(ctx, sqlc.CreateBudgetLineParams{BudgetID: budgetID, Name: name, AllocationAmount: numericAmount, SortOrder: sortOrder})
	if err != nil {
		return appbudgets.Line{}, mapLineError(err)
	}
	return mapLine(row)
}

func updateLine(ctx context.Context, q *sqlc.Queries, input appbudgets.UpdateLineInput) (appbudgets.Line, error) {
	name := ""
	if input.Name != nil {
		name = *input.Name
	}
	var amount pgtype.Numeric
	var err error
	if input.AllocationAmount != nil {
		amount, err = numeric(*input.AllocationAmount)
		if err != nil {
			return appbudgets.Line{}, apperrors.Internal(err)
		}
	}
	order := int32(0)
	if input.SortOrder != nil {
		order = *input.SortOrder
	}
	row, err := q.UpdateBudgetLine(ctx, sqlc.UpdateBudgetLineParams{SetName: input.Name != nil, Name: name, SetAllocationAmount: input.AllocationAmount != nil, AllocationAmount: amount, SetSortOrder: input.SortOrder != nil, SortOrder: order, ID: input.LineID})
	if err != nil {
		return appbudgets.Line{}, mapLineError(err)
	}
	return mapLine(row)
}

func resolveCategoryIDs(ctx context.Context, q *sqlc.Queries, ids []int64, codes []string) ([]int64, error) {
	seen := make(map[int64]struct{})
	resolved := make([]int64, 0, len(ids)+len(codes))
	appendCategory := func(row sqlc.Category, err error) error {
		if err != nil {
			return mapCategoryError(err)
		}
		if _, ok := seen[row.ID]; !ok {
			seen[row.ID] = struct{}{}
			resolved = append(resolved, row.ID)
		}
		return nil
	}
	for _, id := range ids {
		if err := appendCategory(q.GetActiveCategoryById(ctx, id)); err != nil {
			return nil, err
		}
	}
	for _, code := range codes {
		if err := appendCategory(q.GetActiveCategoryByCode(ctx, strings.TrimSpace(code))); err != nil {
			return nil, err
		}
	}
	return resolved, nil
}

func replaceCategories(ctx context.Context, q *sqlc.Queries, budgetID, lineID int64, categoryIDs []int64) error {
	if err := q.DeleteBudgetLineCategories(ctx, lineID); err != nil {
		return mapLineError(err)
	}
	for _, categoryID := range categoryIDs {
		if err := createLineCategory(ctx, q, budgetID, lineID, categoryID); err != nil {
			return err
		}
	}
	return nil
}

func createLineCategory(ctx context.Context, q *sqlc.Queries, budgetID, lineID, categoryID int64) error {
	return mapLineCategoryError(q.CreateBudgetLineCategory(ctx, sqlc.CreateBudgetLineCategoryParams{BudgetID: budgetID, BudgetLineID: lineID, CategoryID: categoryID}))
}

func listReportLines(ctx context.Context, q *sqlc.Queries, budgetID int64) ([]appbudgets.ReportLineData, error) {
	rows, err := q.ListBudgetReportLines(ctx, budgetID)
	if err != nil {
		return nil, mapBudgetError(err)
	}
	items := make([]appbudgets.ReportLineData, 0, len(rows))
	for _, row := range rows {
		allocation, err := numericString(row.AllocationAmount)
		if err != nil {
			return nil, apperrors.Internal(err)
		}
		actual, err := numericString(row.ActualAmount)
		if err != nil {
			return nil, apperrors.Internal(err)
		}
		items = append(items, appbudgets.ReportLineData{Line: appbudgets.Line{ID: row.ID, BudgetID: row.BudgetID, Name: row.Name, AllocationAmount: allocation, SortOrder: row.SortOrder}, ActualAmount: actual})
	}
	return items, nil
}

func listUnmappedTransactions(ctx context.Context, q *sqlc.Queries, budgetID int64) ([]appbudgets.UnmappedTransaction, error) {
	rows, err := q.ListUnmappedBudgetTransactions(ctx, budgetID)
	if err != nil {
		return nil, mapBudgetError(err)
	}
	items := make([]appbudgets.UnmappedTransaction, 0, len(rows))
	for _, row := range rows {
		amount, err := numericString(row.Amount)
		if err != nil {
			return nil, apperrors.Internal(err)
		}
		item := appbudgets.UnmappedTransaction{ID: row.ID, TransactionDate: row.TransactionDate.Time, Description: row.Description, Amount: amount}
		if row.CategoryID != nil {
			code, name := "", ""
			if row.CategoryCode != nil {
				code = *row.CategoryCode
			}
			if row.CategoryName != nil {
				name = *row.CategoryName
			}
			item.Category = &appbudgets.Category{ID: *row.CategoryID, Code: code, Name: name}
		}
		items = append(items, item)
	}
	return items, nil
}

func sumUncategorized(ctx context.Context, q *sqlc.Queries, budgetID int64) (string, error) {
	value, err := q.SumUncategorizedBudgetTransactions(ctx, budgetID)
	if err != nil {
		return "", mapBudgetError(err)
	}
	result, err := numericString(value)
	if err != nil {
		return "", apperrors.Internal(err)
	}
	return result, nil
}

func withTransaction[T any](ctx context.Context, pool *pgxpool.Pool, options pgx.TxOptions, operation func(*sqlc.Queries) (T, error)) (T, error) {
	var zero T
	tx, err := pool.BeginTx(ctx, options)
	if err != nil {
		return zero, mapBudgetError(err)
	}
	defer tx.Rollback(ctx)
	result, err := operation(sqlc.New(tx))
	if err != nil {
		return zero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return zero, mapBudgetError(err)
	}
	return result, nil
}

func mapBudget(row sqlc.Budget) appbudgets.Budget {
	return appbudgets.Budget{ID: row.ID, Owner: appbudgets.Owner{HouseholdID: row.HouseholdID, UserID: row.UserID}, PeriodStart: row.PeriodStart.Time, PeriodEnd: row.PeriodEnd.Time, SourceBudgetID: row.SourceBudgetID}
}
func mapLine(row sqlc.BudgetLine) (appbudgets.Line, error) {
	amount, err := numericString(row.AllocationAmount)
	if err != nil {
		return appbudgets.Line{}, apperrors.Internal(err)
	}
	return appbudgets.Line{ID: row.ID, BudgetID: row.BudgetID, Name: row.Name, AllocationAmount: amount, SortOrder: row.SortOrder}, nil
}
func date(value time.Time) pgtype.Date { return pgtype.Date{Time: value, Valid: true} }
func numeric(value string) (pgtype.Numeric, error) {
	var result pgtype.Numeric
	if err := result.Scan(value); err != nil {
		return pgtype.Numeric{}, fmt.Errorf("parse numeric: %w", err)
	}
	return result, nil
}
func numericString(value pgtype.Numeric) (string, error) {
	raw, err := value.Value()
	if err != nil {
		return "", fmt.Errorf("format numeric: %w", err)
	}
	if raw == nil {
		return "0", nil
	}
	result, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("unexpected numeric value %T", raw)
	}
	return result, nil
}
func mapBudgetError(err error) error {
	return postgres.MapError(err, postgres.ErrorMapping{NotFoundCode: apperrors.CodeBudgetNotFound, NotFoundMessage: "budget not found", ConflictCode: apperrors.CodeBudgetConflict, ConflictMessage: "budget already exists or violates an invariant"})
}
func mapLineError(err error) error {
	return postgres.MapError(err, postgres.ErrorMapping{NotFoundCode: apperrors.CodeBudgetLineNotFound, NotFoundMessage: "budget line not found", ConflictCode: apperrors.CodeBudgetConflict, ConflictMessage: "budget line violates an invariant"})
}
func mapLineCategoryError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "budget_line_category_budget_id_category_id_key" {
		return apperrors.Conflict(apperrors.CodeBudgetConflict, "category already mapped to another budget line", err)
	}
	return mapLineError(err)
}
func mapCategoryError(err error) error {
	return postgres.MapError(err, postgres.ErrorMapping{NotFoundCode: apperrors.CodeCategoryNotFound, NotFoundMessage: "category not found", ConflictCode: apperrors.CodeBudgetConflict, ConflictMessage: "category violates a budget invariant"})
}
func nonNilCategories(items []appbudgets.Category) []appbudgets.Category {
	if items == nil {
		return []appbudgets.Category{}
	}
	return items
}
func nonNilLines(items []appbudgets.Line) []appbudgets.Line {
	if items == nil {
		return []appbudgets.Line{}
	}
	return items
}

var _ appbudgets.Repository = (*Repository)(nil)
