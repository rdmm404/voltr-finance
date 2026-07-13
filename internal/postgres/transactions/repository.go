package transactions

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
	apptransactions "rdmm404/voltr-finance/internal/app/transactions"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/postgres"
)

type queries interface {
	CreateTransaction(context.Context, sqlc.CreateTransactionParams) (sqlc.Transaction, error)
	GetTransactionsByIdWithDetails(context.Context, sqlc.GetTransactionsByIdWithDetailsParams) ([]sqlc.GetTransactionsByIdWithDetailsRow, error)
	ListTransactions(context.Context, sqlc.ListTransactionsParams) ([]sqlc.ListTransactionsRow, error)
	UpdateTransactionById(context.Context, sqlc.UpdateTransactionByIdParams) (sqlc.Transaction, error)
	SoftDeleteTransactionsById(context.Context, sqlc.SoftDeleteTransactionsByIdParams) ([]sqlc.Transaction, error)
	RestoreTransactionsById(context.Context, []int64) ([]sqlc.Transaction, error)
}

type Repository struct{ queries queries }

func NewRepository(queries queries) *Repository { return &Repository{queries: queries} }

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
	_, err := r.queries.UpdateTransactionById(ctx, sqlc.UpdateTransactionByIdParams{ID: id, TransactionID: input.Hash, SetAmount: input.SetAmount, Amount: input.Amount, SetAuthorID: input.SetAuthorID, AuthorID: input.AuthorID, SetCategoryID: input.SetCategoryID, CategoryID: input.CategoryID, SetDescription: input.SetDescription, Description: input.Description, SetTransactionDate: input.SetTransactionDate, TransactionDate: optionalTimestamp(input.TransactionDate, input.SetTransactionDate), SetNotes: input.SetNotes, Notes: input.Notes, SetHouseholdID: input.SetHouseholdID, HouseholdID: input.HouseholdID})
	if err != nil {
		return apptransactions.Transaction{}, mapError(err)
	}
	return r.getDetails(ctx, id, true)
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
	rows, err := r.queries.GetTransactionsByIdWithDetails(ctx, sqlc.GetTransactionsByIdWithDetailsParams{Ids: []int64{id}, IncludeDeleted: includeDeleted})
	if err != nil {
		return apptransactions.Transaction{}, mapError(err)
	}
	if len(rows) == 0 {
		return apptransactions.Transaction{}, notFound(nil)
	}
	row := rows[0]
	return mapDetailed(row.Transaction, row.AuthorName, row.HouseholdID, row.HouseholdName, row.CategoryID, row.CategoryCode, row.CategoryName), nil
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
func optionalTimestamp(value time.Time, valid bool) pgtype.Timestamp {
	return pgtype.Timestamp{Time: value, Valid: valid}
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
