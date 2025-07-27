package database

import (
	"context"
	"fmt"
	"rdmm404/voltr-finance/internal/config"

	"github.com/jackc/pgx/v5"
)

func Init() *pgx.Conn {
	ctx := context.Background()
	connString := fmt.Sprintf("postgres://%v:%v@%v:%v/%v", config.DB_USER, config.DB_PASSWORD, config.DB_HOST, config.DB_PORT, config.DB_NAME)
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		fmt.Println("Error while connecting to database. Panicking!!!")
		panic(err)
	}
	return conn
}
