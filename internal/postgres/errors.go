package postgres

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
)

type ErrorMapping struct {
	NotFoundCode    apperrors.Code
	NotFoundMessage string
	ConflictCode    apperrors.Code
	ConflictMessage string
}

func MapError(err error, mapping ErrorMapping) error {
	if err == nil {
		return nil
	}
	if _, ok := apperrors.As(err); ok {
		return err
	}
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return apperrors.NotFound(mapping.NotFoundCode, mapping.NotFoundMessage, err)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "23503", "23514", "23P01":
			return apperrors.Conflict(mapping.ConflictCode, mapping.ConflictMessage, err)
		}
	}
	return apperrors.Internal(err)
}
