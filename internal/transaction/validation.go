package transaction

import (
	"errors"
	"fmt"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"reflect"
)

func validateTransactionCreate(transaction sqlc.CreateTransactionParams) error {
	var errs []error

	if reflect.ValueOf(transaction.AuthorID).IsZero() {
		errs = append(errs, fmt.Errorf("author id is required"))
	}

	if reflect.ValueOf(transaction.Amount).IsZero() {
		errs = append(errs, fmt.Errorf("amount is required"))
	}

	if reflect.ValueOf(transaction.TransactionDate).IsZero() {
		errs = append(errs, fmt.Errorf("transaction date is required"))
	}

	return errors.Join(errs...)
}
