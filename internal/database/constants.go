package database

type PgErrorCode string

const (
	ErrorCodeUniqueViolation PgErrorCode = "23505"
)
