package transaction

import (
	"context"
	"fmt"
	database "rdmm404/voltr-finance/internal/database/repository"

	"github.com/jackc/pgx/v5"
)

type TransactionService struct {
	db         *pgx.Conn
	repository *database.Queries
}

func NewTransactionService(db *pgx.Conn, repository *database.Queries) *TransactionService {
	return &TransactionService{db: db, repository: repository}
}

func (ts *TransactionService) SaveTransactions(ctx context.Context, transactions []database.CreateTransactionParams) (map[int32]*database.Transaction, error) {
	tx, err := ts.db.Begin(ctx)

	if err != nil {
		return nil, fmt.Errorf("error while creating DB transaction %w", err)
	}
	defer tx.Rollback(ctx)

	// TODO fix this
	ts.repository.WithTx(tx)

	createdTransactions := make(map[int32]*database.Transaction, len(transactions))

	for _, trans := range transactions {
		hashParams := transactionHashParams{
			AuthorId:        trans.AuthorID,
			Amount:          trans.Amount,
			TransactionDate: trans.TransactionDate.Time,
		}

		if trans.Description != nil {
			hashParams.Description = *trans.Description
		}

		if trans.HouseholdID != nil {
			hashParams.HouseholdId = *trans.HouseholdID
		}
		transHash, err := generateTransactionHash(hashParams)

		if err != nil {
			return nil, fmt.Errorf("error creating transaction hash %w", err)
		}
		trans.TransactionID = &transHash

		createdTrans, err := ts.repository.CreateTransaction(ctx, trans)

		if err != nil {
			return nil, fmt.Errorf("error while storing transaction %v - %w", trans.Description, err)
		}

		createdTransactions[createdTrans.ID] = &createdTrans
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("error while committing db transaction - %w", err)
	}

	return createdTransactions, nil
}
