package transaction

import (
	"errors"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/utils"
	"time"
)

var (
	ErrTransactionValidation = errors.New("transaction validation failed")
	ErrDatabaseUnkown        = errors.New("unknown database error")
	ErrDuplicateTransaction  = errors.New("transaction already exists")
	ErrHashCreation          = errors.New("error while creating hash")
	ErrTransactionNotFound   = errors.New("transaction with provided id not found")
)

type TransactionError struct {
	ID    int64
	Index int
	Err   error
}

type TransactionResult struct {
	Success map[int64]*sqlc.Transaction
	Errors  []TransactionError
}

type TransactionUpdate struct {
	Amount           utils.Optional[float32]
	AuthorID         utils.Optional[int64]
	BudgetCategoryID utils.Optional[*int64]
	Description      utils.Optional[*string]
	TransactionDate  utils.Optional[time.Time]
	Notes            utils.Optional[*string]
	HouseholdID      utils.Optional[*int64]
}

type UpdateTransactionById struct {
	ID      int64
	Updates *TransactionUpdate
}
