package errors

import (
	stderrors "errors"
	"fmt"
)

type Kind string

const (
	KindValidation Kind = "validation"
	KindNotFound   Kind = "not_found"
	KindConflict   Kind = "conflict"
	KindInternal   Kind = "internal"
)

type Code string

const (
	CodeValidation           Code = "validation_error"
	CodeUserNotFound         Code = "user_not_found"
	CodeUserConflict         Code = "user_conflict"
	CodeHouseholdNotFound    Code = "household_not_found"
	CodeHouseholdConflict    Code = "household_conflict"
	CodeCategoryNotFound     Code = "category_not_found"
	CodeCategoryConflict     Code = "category_conflict"
	CodeTransactionNotFound  Code = "transaction_not_found"
	CodeDuplicateTransaction Code = "duplicate_transaction"
	CodeBudgetNotFound       Code = "budget_not_found"
	CodeBudgetLineNotFound   Code = "budget_line_not_found"
	CodeBudgetConflict       Code = "budget_conflict"
	CodeInternal             Code = "internal_error"
)

type Error struct {
	Kind    Kind
	Code    Code
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func New(kind Kind, code Code, message string, cause error) error {
	return &Error{Kind: kind, Code: code, Message: message, Cause: cause}
}

func Validation(message string) error { return New(KindValidation, CodeValidation, message, nil) }
func NotFound(code Code, message string, cause error) error {
	return New(KindNotFound, code, message, cause)
}
func Conflict(code Code, message string, cause error) error {
	return New(KindConflict, code, message, cause)
}
func Internal(cause error) error {
	return New(KindInternal, CodeInternal, "internal error", cause)
}

func As(err error) (*Error, bool) {
	var target *Error
	ok := stderrors.As(err, &target)
	return target, ok
}

func IsKind(err error, kind Kind) bool {
	appErr, ok := As(err)
	return ok && appErr.Kind == kind
}

// Normalize preserves an application error returned by a port and safely maps
// every other failure to an internal error.
func Normalize(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := As(err); ok {
		return err
	}
	return Internal(err)
}

func CodeOf(err error) Code {
	if appErr, ok := As(err); ok {
		return appErr.Code
	}
	return CodeInternal
}

func MessageOf(err error) string {
	if appErr, ok := As(err); ok {
		return appErr.Message
	}
	return "internal error"
}

func WrapInternal(operation string, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := As(err); ok {
		return err
	}
	return Internal(fmt.Errorf("%s: %w", operation, err))
}
