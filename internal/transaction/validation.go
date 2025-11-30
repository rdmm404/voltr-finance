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

func validateTransactionUpdate(transactionUpdate UpdateTransactionById) error {
	var errs []error
	if reflect.ValueOf(transactionUpdate.ID).IsZero() {
		errs = append(errs, fmt.Errorf("id for update must be provided"))
	}

	updates := transactionUpdate.Updates
	if updates.AuthorID.Set && reflect.ValueOf(updates.AuthorID.Value).IsZero() {
		errs = append(errs, fmt.Errorf("author id is required"))
	}

	if updates.Amount.Set && reflect.ValueOf(updates.Amount.Value).IsZero() {
		errs = append(errs, fmt.Errorf("amount is required"))
	}

	if updates.TransactionDate.Set && reflect.ValueOf(updates.TransactionDate.Value).IsZero() {
		errs = append(errs, fmt.Errorf("transaction date is required"))
	}

	return errors.Join(errs...)
}
