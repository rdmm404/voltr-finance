package transaction

import (
	"context"
	"database/sql"
	"fmt"
	"rdmm404/voltr-finance/internal/config"
	database "rdmm404/voltr-finance/internal/database/repository"

	_ "github.com/go-sql-driver/mysql"
)

func SaveTransactions(transactions []*Transaction) error {

	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?parseTime=true", config.DB_USER, config.DB_PASSWORD, config.DB_HOST, config.DB_PORT, config.DB_NAME)

	fmt.Println("Connecting with DSN " + dsn)
	conn, err := sql.Open("mysql", dsn)

	if err != nil {
		panic(err)
	}

	db := database.New(conn)
	fmt.Println("received transactions")
	for _, trans := range transactions {
		fmt.Printf("%+v\n", *trans)

		dbTrans := database.CreateTransactionParams{
			Amount: float64(trans.Amount),
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
