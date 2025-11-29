package transaction

import (
	"errors"
	"rdmm404/voltr-finance/internal/database/sqlc"
)

var (
	ErrTransactionValidation = errors.New("transaction validation failed")
	ErrDatabaseUnkown        = errors.New("unknown database error")
	ErrDuplicateTransaction  = errors.New("transaction already exists")
	ErrHashCreation          = errors.New("error while creating hash")
	ErrTransactionNotFound   = errors.New("transaction with provided id not found")
)

type TransactionError struct {
	ID    string
	Index int
	Err   error
}

type SaveTransactionsResult struct {
	Created map[string]*sqlc.Transaction
	Errors  []TransactionError
}
