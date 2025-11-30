package database

import (
	"context"
	"fmt"
	"log/slog"
	"rdmm404/voltr-finance/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Init(ctx context.Context) *pgxpool.Pool {
	connString := fmt.Sprintf(
		"postgres://%v:%v@%v:%v/%v?pool_max_conns=%v",
		config.DB_USER,
		config.DB_PASSWORD,
		config.DB_HOST,
		config.DB_PORT,
		config.DB_NAME,
		config.DB_POOL_SIZE,
	)

	conn, err := pgxpool.New(ctx, connString)
	if err != nil {
		slog.Error("Error while connecting to database. Panicking!!!", "error", err)
		panic(err)
	}

	return conn
}
