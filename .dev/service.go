package main

import (
	"context"
	database "rdmm404/voltr-finance/internal/database/repository"
	"rdmm404/voltr-finance/internal/transaction"
)

func main() {
	ctx := context.Background()
	db := database.Init()
	defer db.Close(ctx)

	repository := database.New(db)

	ts := transaction.NewTransactionService(db, repository)

	err := ts.SaveTransactions(ctx, []*transaction.Transaction{
		{Name: "Foo", Amount: 11},
	})

	if err != nil {
		panic(err)
	}
}
