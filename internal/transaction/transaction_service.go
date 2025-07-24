package transaction

import (
	"context"
	"fmt"
	"rdmm404/voltr-finance/internal/config"
	database "rdmm404/voltr-finance/internal/database/repository"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/pgtype"
)

func SaveTransactions(transactions []*Transaction) error {

	connString := fmt.Sprintf("postgres://%v:%v@%v:%v/%v", config.DB_USER, config.DB_PASSWORD, config.DB_HOST, config.DB_PORT, config.DB_NAME)

	fmt.Println("Connecting with DSN " + connString)
	conn, err := pgx.Connect(context.TODO(), connString)


	if err != nil {
		panic(err)
	}

	db := database.New(conn)
	fmt.Println("received transactions")
	for _, trans := range transactions {
		fmt.Printf("%+v\n", *trans)

		dbTrans := database.CreateTransactionParams{
			Amount: trans.Amount,
			// Description: sql.NullString{String: trans.Description, Valid: true},
			PaidBy: 1,
		}

		res, err := db.CreateTransaction(context.TODO(), dbTrans)

		if err != nil {
			panic(err)
		}

		fmt.Printf("db result: %+v\n", res)
	}
	return nil
}
