package transactions

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
	apptransactions "rdmm404/voltr-finance/internal/app/transactions"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/postgres"
)

type queries interface {
	CreateTransaction(context.Context, sqlc.CreateTransactionParams) (sqlc.Transaction, error)
	GetTransactionByIdForUpdate(context.Context, int64) (sqlc.Transaction, error)
	GetTransactionsByIdWithDetails(context.Context, sqlc.GetTransactionsByIdWithDetailsParams) ([]sqlc.GetTransactionsByIdWithDetailsRow, error)
	ListTransactions(context.Context, sqlc.ListTransactionsParams) ([]sqlc.ListTransactionsRow, error)
	UpdateTransactionById(context.Context, sqlc.UpdateTransactionByIdParams) (sqlc.Transaction, error)
	SoftDeleteTransactionsById(context.Context, sqlc.SoftDeleteTransactionsByIdParams) ([]sqlc.Transaction, error)
	RestoreTransactionsById(context.Context, []int64) ([]sqlc.Transaction, error)
}

type Repository struct {
	pool    *pgxpool.Pool
	queries queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, queries: sqlc.New(pool)}
}

func (r *Repository) Create(ctx context.Context, input apptransactions.NewTransaction) (apptransactions.Transaction, error) {
	row, err := r.queries.CreateTransaction(ctx, sqlc.CreateTransactionParams{Amount: input.Amount, CategoryID: input.CategoryID, Description: input.Description, TransactionDate: timestamptz(input.TransactionDate), TransactionID: input.Hash, AuthorID: input.AuthorID, HouseholdID: input.HouseholdID, Notes: input.Notes})
	if err != nil {
		return apptransactions.Transaction{}, mapError(err)
	}
	return r.getDetails(ctx, row.ID, true)
}

func (r *Repository) Get(ctx context.Context, id int64, includeDeleted bool) (apptransactions.Transaction, error) {
	return r.getDetails(ctx, id, includeDeleted)
}

func (r *Repository) List(ctx context.Context, filter apptransactions.ListFilter) ([]apptransactions.Transaction, error) {
	rows, err := r.queries.ListTransactions(ctx, sqlc.ListTransactionsParams{OnlyDeleted: filter.OnlyDeleted, IncludeDeleted: filter.IncludeDeleted, AuthorID: filter.AuthorID, HouseholdID: filter.HouseholdID, FromDate: optionalTimestamptz(filter.FromDate), ToDate: optionalTimestamptz(filter.ToDate), Search: filter.Search, Sort: filter.Sort, SortOrder: filter.SortOrder, ResultOffset: filter.Offset, ResultLimit: filter.Limit})
	if err != nil {
		return nil, mapError(err)
	}
	items := make([]apptransactions.Transaction, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapDetailed(row.Transaction, row.AuthorName, row.HouseholdID, row.HouseholdName, row.CategoryID, row.CategoryCode, row.CategoryName))
	}
	return items, nil
}

func (r *Repository) Update(ctx context.Context, id int64, input apptransactions.Mutation) (apptransactions.Transaction, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return apptransactions.Transaction{}, mapError(err)
	}
	defer tx.Rollback(ctx)
	q := sqlc.New(tx)

	row, err := q.GetTransactionByIdForUpdate(ctx, id)
	if err != nil {
		return apptransactions.Transaction{}, mapError(err)
	}
	existing := mapTransaction(row)
	merged := input.Apply(existing)
	hash, err := apptransactions.Hash(merged.Description, merged.TransactionDate, merged.AuthorID, merged.HouseholdID, merged.CategoryID, merged.Amount)
	if err != nil {
		return apptransactions.Transaction{}, err
	}
	_, err = q.UpdateTransactionById(ctx, sqlc.UpdateTransactionByIdParams{
		ID: id, TransactionID: hash,
		SetAmount: input.Amount != nil, Amount: valueOrZero(input.Amount),
		SetAuthorID: input.AuthorID != nil, AuthorID: valueOrZero(input.AuthorID),
		SetCategoryID: input.CategoryID.Present(), CategoryID: input.CategoryID.Value(),
		SetDescription: input.Description.Present(), Description: input.Description.Value(),
		SetTransactionDate: input.TransactionDate != nil, TransactionDate: optionalTimestamptz(input.TransactionDate),
		SetNotes: input.Notes.Present(), Notes: input.Notes.Value(),
		SetHouseholdID: input.HouseholdID.Present(), HouseholdID: input.HouseholdID.Value(),
	})
	if err != nil {
		return apptransactions.Transaction{}, mapError(err)
	}
	item, err := getDetails(ctx, q, id, true)
	if err != nil {
		return apptransactions.Transaction{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return apptransactions.Transaction{}, mapError(err)
	}
	return item, nil
}

func (r *Repository) SoftDelete(ctx context.Context, input apptransactions.DeleteInput) (apptransactions.Transaction, error) {
	rows, err := r.queries.SoftDeleteTransactionsById(ctx, sqlc.SoftDeleteTransactionsByIdParams{DeletedByUserID: input.DeletedByUserID, DeleteReason: input.Reason, Ids: []int64{input.ID}})
	if err != nil {
		return apptransactions.Transaction{}, mapError(err)
	}
	if len(rows) == 0 {
		return apptransactions.Transaction{}, notFound(nil)
	}
	return r.getDetails(ctx, input.ID, true)
}

func (r *Repository) Restore(ctx context.Context, input apptransactions.RestoreInput) (apptransactions.Transaction, error) {
	rows, err := r.queries.RestoreTransactionsById(ctx, []int64{input.ID})
	if err != nil {
		return apptransactions.Transaction{}, mapError(err)
	}
	if len(rows) == 0 {
		return apptransactions.Transaction{}, notFound(nil)
	}
	return r.getDetails(ctx, input.ID, false)
}

func (r *Repository) getDetails(ctx context.Context, id int64, includeDeleted bool) (apptransactions.Transaction, error) {
	return getDetails(ctx, r.queries, id, includeDeleted)
}

func getDetails(ctx context.Context, q queries, id int64, includeDeleted bool) (apptransactions.Transaction, error) {
	rows, err := q.GetTransactionsByIdWithDetails(ctx, sqlc.GetTransactionsByIdWithDetailsParams{Ids: []int64{id}, IncludeDeleted: includeDeleted})
	if err != nil {
		return apptransactions.Transaction{}, mapError(err)
	}
	if len(rows) == 0 {
		return apptransactions.Transaction{}, notFound(nil)
	}
	row := rows[0]
	return mapDetailed(row.Transaction, row.AuthorName, row.HouseholdID, row.HouseholdName, row.CategoryID, row.CategoryCode, row.CategoryName), nil
}

func mapTransaction(row sqlc.Transaction) apptransactions.Transaction {
	return apptransactions.Transaction{ID: row.ID, Hash: row.TransactionID, Amount: row.Amount, TransactionDate: row.TransactionDate.Time, AuthorID: row.AuthorID, HouseholdID: row.HouseholdID, CategoryID: row.CategoryID, Description: row.Description, Notes: row.Notes}
}

func mapDetailed(row sqlc.Transaction, authorName string, householdID *int64, householdName *string, categoryID *int64, categoryCode, categoryName *string) apptransactions.Transaction {
	item := apptransactions.Transaction{ID: row.ID, Hash: row.TransactionID, Amount: row.Amount, TransactionDate: row.TransactionDate.Time, AuthorID: row.AuthorID, AuthorName: authorName, HouseholdID: householdID, HouseholdName: householdName, CategoryID: categoryID, Description: row.Description, Notes: row.Notes, CreatedAt: timestamp(row.CreatedAt), UpdatedAt: timestamp(row.UpdatedAt), DeletedAt: timestamp(row.DeletedAt), DeletedByUserID: row.DeletedByUserID, DeleteReason: row.DeleteReason}
	if categoryID != nil {
		code, name := "", ""
		if categoryCode != nil {
			code = *categoryCode
		}
		if categoryName != nil {
			name = *categoryName
		}
		item.Category = &apptransactions.CategoryRef{ID: *categoryID, Code: code, Name: name}
	}
	return item
}

func timestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}
func optionalTimestamptz(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return timestamptz(*value)
}
func valueOrZero[T any](value *T) T {
	if value == nil {
		var zero T
		return zero
	}
	return *value
}
func timestamp(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
func notFound(cause error) error {
	return apperrors.NotFound(apperrors.CodeTransactionNotFound, "transaction not found", cause)
}
func mapError(err error) error {
	return postgres.MapError(err, postgres.ErrorMapping{NotFoundCode: apperrors.CodeTransactionNotFound, NotFoundMessage: "transaction not found", ConflictCode: apperrors.CodeDuplicateTransaction, ConflictMessage: "duplicate transaction"})
}

var _ apptransactions.Repository = (*Repository)(nil)
