package transaction

import (
	"context"
	"fmt"
	database "rdmm404/voltr-finance/internal/database/repository"
)

type TransactionService struct {
	db *database.Queries
}

func (ts *TransactionService) SaveTransactions(transactions []*Transaction) error {
	for _, trans := range transactions {
		fmt.Printf("%+v\n", *trans)

		dbTrans := database.CreateTransactionParams{
			Amount: trans.Amount,
			// Description: sql.NullString{String: trans.Description, Valid: true},
			PaidBy: 1,
		}

		res, err := ts.db.CreateTransaction(context.TODO(), dbTrans)

		if err != nil {
			panic(err)
		}

		fmt.Printf("db result: %+v\n", res)
	}
	return nil
}

func NewTransactionService(db *database.Queries) *TransactionService {
	return &TransactionService{db: db}
}