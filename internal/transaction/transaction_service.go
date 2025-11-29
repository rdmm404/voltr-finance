package transaction

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"rdmm404/voltr-finance/internal/database"
	"rdmm404/voltr-finance/internal/database/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type TransactionService struct {
	db         *pgx.Conn
	repository *sqlc.Queries
}

func NewTransactionService(db *pgx.Conn, repository *sqlc.Queries) *TransactionService {
	return &TransactionService{db: db, repository: repository}
}

func (ts *TransactionService) GetTransactionsByTransactionId(ctx context.Context, ids []string) (map[string]sqlc.Transaction, error) {
	transactions, err := ts.repository.GetTransactionsByTransactionId(ctx, ids)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return map[string]sqlc.Transaction{}, errors.Join(err, ErrTransactionNotFound)
		}

		return map[string]sqlc.Transaction{}, errors.Join(err, ErrDatabaseUnkown)
	}

	transactionMap := make(map[string]sqlc.Transaction, len(transactions))

	for _, trans := range transactions {
		transactionMap[trans.TransactionID] = trans
	}

	return transactionMap, nil
}

func (ts *TransactionService) SaveTransactions(ctx context.Context, transactions []sqlc.CreateTransactionParams) SaveTransactionsResult {
	result := SaveTransactionsResult{}
	result.Created = make(map[string]*sqlc.Transaction)

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

		trans.TransactionID = transHash
		createdTrans, err := ts.repository.CreateTransaction(ctx, trans)

		if err != nil {
			result.Errors = append(result.Errors, TransactionError{Index: i, ID: transHash, Err: handleCreateTransactionDbError(err)})
			continue
		}

		result.Created[createdTrans.TransactionID] = &createdTrans
	}

	return result
}

func handleCreateTransactionDbError(err error) error {
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

func (ts *TransactionService) UpdateTransactionsById(ctx context.Context, transactions []sqlc.UpdateTransactionByIdParams) {
	slog.Info("UpdateTransactionsById called", "transactions", transactions)
}
