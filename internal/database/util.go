package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"rdmm404/voltr-finance/internal/database/sqlc"

	"github.com/jackc/pgx/v5"
)

func RunRawQuery(ctx context.Context, db sqlc.DBTX, query string, args ...any) ([]map[string]any, error) {
	rows, err := db.Query(ctx, query, args...)

	if err != nil {
		return nil, fmt.Errorf("error while executing sql: %w", err)
	}

	results, err := pgx.CollectRows(rows, pgx.RowToMap)
	if err != nil {
		return nil, fmt.Errorf("error collecting rows into map: %w", err)
	}

	return results, nil
}

func RollbackTx(ctx context.Context, tx pgx.Tx) {
	err := tx.Rollback(ctx)
	if !errors.Is(err, pgx.ErrTxClosed) {
		slog.Error("error while rolling back tx", "error", err)
	}
}