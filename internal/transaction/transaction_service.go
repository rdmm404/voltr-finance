package transaction

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"rdmm404/voltr-finance/internal/database"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/utils"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionService struct {
	db         *pgxpool.Pool
	repository *sqlc.Queries
}

func NewTransactionService(db *pgxpool.Pool, repository *sqlc.Queries) *TransactionService {
	return &TransactionService{db: db, repository: repository}
}

func (ts *TransactionService) GetTransactionsById(ctx context.Context, ids []int64) (map[int64]sqlc.Transaction, error) {
	transactions, err := ts.repository.GetTransactionsById(ctx, ids)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return map[int64]sqlc.Transaction{}, errors.Join(err, ErrTransactionNotFound)
		}

		return map[int64]sqlc.Transaction{}, errors.Join(err, ErrDatabaseUnkown)
	}

	transactionMap := make(map[int64]sqlc.Transaction, len(transactions))

	for _, trans := range transactions {
		transactionMap[trans.ID] = trans
	}

	return transactionMap, nil
}

func (ts *TransactionService) SaveTransactions(ctx context.Context, transactions []sqlc.CreateTransactionParams) TransactionResult {
	result := TransactionResult{}
	result.Success = make(map[int64]*sqlc.Transaction)

	for i, trans := range transactions {
		if err := validateTransactionCreate(trans); err != nil {
			result.Errors = append(result.Errors, TransactionError{Index: i, Err: errors.Join(err, ErrTransactionValidation)})
			continue
		}

		transHash, err := generateHashForTransactionCreate(trans)

		if err != nil {
			result.Errors = append(result.Errors, TransactionError{Index: i, Err: errors.Join(err, ErrHashCreation)})
			continue
		}

		if existing, err := ts.repository.GetIdByTransactionId(ctx, transHash); err == nil {
			result.Errors = append(result.Errors, TransactionError{Index: i, ID: existing, Err: ErrDuplicateTransaction})
			continue
		}

		trans.TransactionID = transHash
		createdTrans, err := ts.repository.CreateTransaction(ctx, trans)

		if err != nil {
			result.Errors = append(result.Errors, TransactionError{Index: i, Err: handleTransactionDbError(err)})
			continue
		}

		result.Success[createdTrans.ID] = &createdTrans
	}

	return result
}

func (ts *TransactionService) UpdateTransactionsById(ctx context.Context, transactionUpdates []UpdateTransactionById) TransactionResult {
	slog.Debug("UpdateTransactionsById called", "transactions", transactionUpdates)

	result := TransactionResult{}
	result.Success = make(map[int64]*sqlc.Transaction)

	for i, trans := range transactionUpdates {
		// TODO maybe validate that author corresponds to the one sending the message or a member of the household
		if err := validateTransactionUpdate(trans); err != nil {
			result.Errors = append(result.Errors, TransactionError{Index: i, Err: errors.Join(err, ErrTransactionValidation)})
			continue
		}

		existing, err := ts.repository.GetTransactionById(ctx, trans.ID)

		if err != nil {
			result.Errors = append(result.Errors, TransactionError{Index: i, ID: trans.ID, Err: handleTransactionDbError(err)})
			continue
		}

		transHash, err := generateHashForTransactionUpdate(existing, trans.Updates)

		if err != nil {
			result.Errors = append(result.Errors, TransactionError{Index: i, Err: errors.Join(err, ErrHashCreation)})
			continue
		}

		if existing, err := ts.repository.GetIdByTransactionId(ctx, transHash); err == nil {
			result.Errors = append(result.Errors, TransactionError{Index: i, ID: existing, Err: ErrDuplicateTransaction})
			continue
		}

		params := sqlc.UpdateTransactionByIdParams{
			ID:                  trans.ID,
			TransactionID:       transHash,
			SetAmount:           trans.Updates.Amount.Set,
			Amount:              trans.Updates.Amount.Value,
			SetAuthorID:         trans.Updates.AuthorID.Set,
			AuthorID:            trans.Updates.AuthorID.Value,
			SetBudgetCategoryID: trans.Updates.BudgetCategoryID.Set,
			BudgetCategoryID:    trans.Updates.BudgetCategoryID.Value,
			SetDescription:      trans.Updates.Description.Set,
			Description:         trans.Updates.Description.Value,
			SetTransactionDate:  trans.Updates.TransactionDate.Set,
			TransactionDate:     pgtype.Timestamp{Time: trans.Updates.TransactionDate.Value, Valid: true},
			SetNotes:            trans.Updates.Notes.Set,
			Notes:               trans.Updates.Notes.Value,
			SetHouseholdID:      trans.Updates.HouseholdID.Set,
			HouseholdID:         trans.Updates.HouseholdID.Value,
		}

		updatedTrans, err := ts.repository.UpdateTransactionById(ctx, params)

		if err != nil {
			slog.Error("database error when updating transaction", "error", err, "transactionId", trans.ID, "updates", utils.JsonMarshalIgnore(trans))
			result.Errors = append(result.Errors, TransactionError{Index: i, Err: handleTransactionDbError(err)})
			continue
		}

		result.Success[updatedTrans.ID] = &updatedTrans
	}

	return result
}

func handleTransactionDbError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return errors.Join(err, ErrTransactionNotFound)
	}

	var pgErr *pgconn.PgError

	if !errors.As(err, &pgErr) {
		return errors.Join(err, ErrDatabaseUnkown)
	}

	switch database.PgErrorCode(pgErr.Code) {
	case database.ErrorCodeUniqueViolation:
		return errors.Join(err, ErrDuplicateTransaction)
	default:
		return errors.Join(err, ErrDatabaseUnkown)
	}
}
