package transaction

import (
	"context"
	"fmt"
	database "rdmm404/voltr-finance/internal/database/repository"

	"github.com/jackc/pgx/v5"
)

type TransactionService struct {
	db *pgx.Conn
	repository *database.Queries
}

func (ts *TransactionService) SaveTransactions(ctx context.Context, transactions []*database.Transaction) error {
	tx, err := ts.db.Begin(ctx)

	if err != nil {
		return fmt.Errorf("error while creating DB transaction %w", err)
	}
	defer tx.Rollback(ctx)

	ts.repository.WithTx(tx)

	for _, trans := range transactions {
		dbTrans := database.CreateTransactionParams{
			Amount: trans.Amount,
			// Description: sql.NullString{String: trans.Description, Valid: true},
			PaidBy: 1,
		}

		_, err := ts.repository.CreateTransaction(ctx, dbTrans)

		if err != nil {
			return fmt.Errorf("error while storing transaction %v - %w", trans.Description, err)
		}

	}
	return tx.Commit(ctx)
}

func NewTransactionService(db *pgx.Conn, repository *database.Queries) *TransactionService {
	return &TransactionService{db: db, repository: repository}
}