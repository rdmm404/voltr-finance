package app

type ErrorCode string

const (
	CodeValidationError      ErrorCode = "validation_error"
	CodeUserNotFound         ErrorCode = "user_not_found"
	CodeTransactionNotFound  ErrorCode = "transaction_not_found"
	CodeDuplicateTransaction ErrorCode = "duplicate_transaction"
	CodeDatabaseError        ErrorCode = "database_error"
)

type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func NewError(code ErrorCode, message string, err error) error {
	return &AppError{Code: code, Message: message, Err: err}
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
