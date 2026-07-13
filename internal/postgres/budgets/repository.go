package budgets

import (
	"context"
	"errors"
	"fmt"
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

type Repository struct{ queries *sqlc.Queries }

func NewRepository(queries *sqlc.Queries) *Repository { return &Repository{queries: queries} }

func (r *Repository) FindMonthly(ctx context.Context, owner appbudgets.Owner, start, end time.Time) (appbudgets.Budget, error) {
	var row sqlc.Budget
	var err error
	if owner.HouseholdID != nil {
		row, err = r.queries.GetHouseholdBudgetByPeriod(ctx, sqlc.GetHouseholdBudgetByPeriodParams{HouseholdID: *owner.HouseholdID, PeriodStart: date(start), PeriodEnd: date(end)})
	} else if owner.UserID != nil {
		row, err = r.queries.GetUserBudgetByPeriod(ctx, sqlc.GetUserBudgetByPeriodParams{UserID: *owner.UserID, PeriodStart: date(start), PeriodEnd: date(end)})
	} else {
		return appbudgets.Budget{}, apperrors.Validation("budget owner is required")
	}
	return mapBudget(row), mapBudgetError(err)
}

func (r *Repository) FindLatestPrior(ctx context.Context, owner appbudgets.Owner, start time.Time) (appbudgets.Budget, error) {
	var row sqlc.Budget
	var err error
	if owner.HouseholdID != nil {
		row, err = r.queries.GetLatestPriorHouseholdBudget(ctx, sqlc.GetLatestPriorHouseholdBudgetParams{HouseholdID: *owner.HouseholdID, PeriodStart: date(start)})
	} else if owner.UserID != nil {
		row, err = r.queries.GetLatestPriorUserBudget(ctx, sqlc.GetLatestPriorUserBudgetParams{UserID: *owner.UserID, PeriodStart: date(start)})
	} else {
		return appbudgets.Budget{}, apperrors.Validation("budget owner is required")
	}
	return mapBudget(row), mapBudgetError(err)
}

func (r *Repository) GetBudget(ctx context.Context, id int64) (appbudgets.Budget, error) {
	row, err := r.queries.GetBudgetById(ctx, id)
	return mapBudget(row), mapBudgetError(err)
}
func (r *Repository) LockBudget(ctx context.Context, id int64) error {
	_, err := r.queries.LockBudgetForUpdate(ctx, id)
	return mapBudgetError(err)
}
func (r *Repository) CreateBudget(ctx context.Context, input appbudgets.CreateBudget) (appbudgets.Budget, error) {
	var row sqlc.Budget
	var err error
	if input.Owner.HouseholdID != nil {
		row, err = r.queries.CreateHouseholdBudget(ctx, sqlc.CreateHouseholdBudgetParams{HouseholdID: *input.Owner.HouseholdID, PeriodStart: date(input.PeriodStart), PeriodEnd: date(input.PeriodEnd), SourceBudgetID: input.SourceBudgetID})
	} else if input.Owner.UserID != nil {
		row, err = r.queries.CreateUserBudget(ctx, sqlc.CreateUserBudgetParams{UserID: *input.Owner.UserID, PeriodStart: date(input.PeriodStart), PeriodEnd: date(input.PeriodEnd), SourceBudgetID: input.SourceBudgetID})
	} else {
		return appbudgets.Budget{}, apperrors.Validation("budget owner is required")
	}
	return mapBudget(row), mapBudgetError(err)
}

func (r *Repository) ListLines(ctx context.Context, budgetID int64) ([]appbudgets.Line, error) {
	rows, err := r.queries.ListBudgetLines(ctx, budgetID)
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
func (r *Repository) ListLineCategories(ctx context.Context, budgetID int64) ([]appbudgets.LineCategory, error) {
	rows, err := r.queries.ListBudgetLineCategories(ctx, budgetID)
	if err != nil {
		return nil, mapBudgetError(err)
	}
	items := make([]appbudgets.LineCategory, 0, len(rows))
	for _, row := range rows {
		items = append(items, appbudgets.LineCategory{BudgetID: row.BudgetID, LineID: row.BudgetLineID, Category: appbudgets.Category{ID: row.CategoryID, Code: row.CategoryCode, Name: row.CategoryName}})
	}
	return items, nil
}
func (r *Repository) GetLine(ctx context.Context, id int64) (appbudgets.Line, error) {
	row, err := r.queries.GetBudgetLineById(ctx, id)
	if err != nil {
		return appbudgets.Line{}, mapLineError(err)
	}
	return mapLine(row)
}
func (r *Repository) MaxSortOrder(ctx context.Context, budgetID int64) (int32, error) {
	value, err := r.queries.GetMaxBudgetLineSortOrder(ctx, budgetID)
	return value, mapBudgetError(err)
}
func (r *Repository) CreateLine(ctx context.Context, input appbudgets.CreateLineInput) (appbudgets.Line, error) {
	amount, err := numeric(input.AllocationAmount)
	if err != nil {
		return appbudgets.Line{}, apperrors.Internal(err)
	}
	order := int32(0)
	if input.SortOrder != nil {
		order = *input.SortOrder
	}
	row, err := r.queries.CreateBudgetLine(ctx, sqlc.CreateBudgetLineParams{BudgetID: input.BudgetID, Name: input.Name, AllocationAmount: amount, SortOrder: order})
	if err != nil {
		return appbudgets.Line{}, mapLineError(err)
	}
	return mapLine(row)
}
func (r *Repository) UpdateLine(ctx context.Context, id int64, input appbudgets.LineUpdate) (appbudgets.Line, error) {
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
	row, err := r.queries.UpdateBudgetLine(ctx, sqlc.UpdateBudgetLineParams{SetName: input.Name != nil, Name: name, SetAllocationAmount: input.AllocationAmount != nil, AllocationAmount: amount, SetSortOrder: input.SortOrder != nil, SortOrder: order, ID: id})
	if err != nil {
		return appbudgets.Line{}, mapLineError(err)
	}
	return mapLine(row)
}
func (r *Repository) DeleteLine(ctx context.Context, id int64) error {
	if _, err := r.queries.GetBudgetLineById(ctx, id); err != nil {
		return mapLineError(err)
	}
	return mapLineError(r.queries.DeleteBudgetLine(ctx, id))
}
func (r *Repository) DeleteLineCategories(ctx context.Context, lineID int64) error {
	return mapLineError(r.queries.DeleteBudgetLineCategories(ctx, lineID))
}
func (r *Repository) CreateLineCategory(ctx context.Context, budgetID, lineID, categoryID int64) error {
	return mapLineCategoryError(r.queries.CreateBudgetLineCategory(ctx, sqlc.CreateBudgetLineCategoryParams{BudgetID: budgetID, BudgetLineID: lineID, CategoryID: categoryID}))
}
func (r *Repository) GetActiveCategoryByID(ctx context.Context, id int64) (appbudgets.Category, error) {
	row, err := r.queries.GetActiveCategoryById(ctx, id)
	return mapCategory(row), mapCategoryError(err)
}
func (r *Repository) GetActiveCategoryByCode(ctx context.Context, code string) (appbudgets.Category, error) {
	row, err := r.queries.GetActiveCategoryByCode(ctx, code)
	return mapCategory(row), mapCategoryError(err)
}

func (r *Repository) ListReportLines(ctx context.Context, budgetID int64) ([]appbudgets.ReportLineData, error) {
	rows, err := r.queries.ListBudgetReportLines(ctx, budgetID)
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
func (r *Repository) ListUnmappedTransactions(ctx context.Context, budgetID int64) ([]appbudgets.UnmappedTransaction, error) {
	rows, err := r.queries.ListUnmappedBudgetTransactions(ctx, budgetID)
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
func (r *Repository) SumUncategorized(ctx context.Context, budgetID int64) (string, error) {
	value, err := r.queries.SumUncategorizedBudgetTransactions(ctx, budgetID)
	if err != nil {
		return "", mapBudgetError(err)
	}
	result, err := numericString(value)
	if err != nil {
		return "", apperrors.Internal(err)
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
func mapCategory(row sqlc.Category) appbudgets.Category {
	return appbudgets.Category{ID: row.ID, Code: row.Code, Name: row.Name}
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

// Transactor ensures budget creation/copy and line/category replacement use the
// same sqlc query set on one PostgreSQL transaction.
type Transactor struct{ pool *pgxpool.Pool }

func NewTransactor(pool *pgxpool.Pool) *Transactor { return &Transactor{pool: pool} }
func (t *Transactor) WithinTransaction(ctx context.Context, callback func(appbudgets.Repository) error) error {
	return t.withOptions(ctx, pgx.TxOptions{}, callback)
}
func (t *Transactor) WithinSnapshot(ctx context.Context, callback func(appbudgets.Repository) error) error {
	return t.withOptions(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}, callback)
}
func (t *Transactor) withOptions(ctx context.Context, options pgx.TxOptions, callback func(appbudgets.Repository) error) error {
	tx, err := t.pool.BeginTx(ctx, options)
	if err != nil {
		return mapBudgetError(err)
	}
	defer tx.Rollback(ctx)
	if err := callback(NewRepository(sqlc.New(tx))); err != nil {
		return err
	}
	return mapBudgetError(tx.Commit(ctx))
}

var _ appbudgets.Repository = (*Repository)(nil)
var _ appbudgets.Transactor = (*Transactor)(nil)
