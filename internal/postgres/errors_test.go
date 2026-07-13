package postgres

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
)

func TestMapErrorClassifiesPersistenceFailures(t *testing.T) {
	mapping := ErrorMapping{NotFoundCode: apperrors.CodeUserNotFound, NotFoundMessage: "user not found", ConflictCode: apperrors.CodeUserConflict, ConflictMessage: "user exists"}
	if err := MapError(sql.ErrNoRows, mapping); !apperrors.IsKind(err, apperrors.KindNotFound) || apperrors.CodeOf(err) != apperrors.CodeUserNotFound { t.Fatalf("not found=%v", err) }
	if err := MapError(&pgconn.PgError{Code: "23505", Detail: "sensitive"}, mapping); !apperrors.IsKind(err, apperrors.KindConflict) || apperrors.MessageOf(err) != "user exists" { t.Fatalf("conflict=%v", err) }
	cause := errors.New("connection secret")
	if err := MapError(cause, mapping); !apperrors.IsKind(err, apperrors.KindInternal) || apperrors.MessageOf(err) != "internal error" || !errors.Is(err, cause) { t.Fatalf("internal=%v", err) }
}
