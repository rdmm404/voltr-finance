package database

import (
	"context"
	"fmt"
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
