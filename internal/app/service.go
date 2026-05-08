package app

import (
	"context"
	"database/sql"
	"errors"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/transaction"
)

type Repository interface {
	UserRepository
	HouseholdRepository
	TransactionRepository
	CategoryRepository
}

type UserRepository interface {
	CreateUser(context.Context, sqlc.CreateUserParams) (sqlc.User, error)
	UpdateUser(context.Context, sqlc.UpdateUserParams) (sqlc.User, error)
	GetUserById(context.Context, int64) (sqlc.User, error)
	GetUserByDiscordId(context.Context, *string) (sqlc.User, error)
	GetUserByTelegramId(context.Context, *string) (sqlc.User, error)
	GetUserByPhoneNumber(context.Context, *string) (sqlc.User, error)
	GetUserByWhatsappId(context.Context, *string) (sqlc.User, error)
	ListUsers(context.Context) ([]sqlc.User, error)
}

type HouseholdRepository interface {
	GetHouseholdById(context.Context, int64) (sqlc.Household, error)
	GetHouseholdByGuildId(context.Context, string) (sqlc.Household, error)
	GetHouseholdByName(context.Context, string) (sqlc.Household, error)
	ListHouseholds(context.Context) ([]sqlc.Household, error)
	GetHouseholdUsers(context.Context, int64) ([]sqlc.User, error)
}

type TransactionRepository interface {
	GetTransactionsByIdWithDetails(context.Context, sqlc.GetTransactionsByIdWithDetailsParams) ([]sqlc.GetTransactionsByIdWithDetailsRow, error)
	ListTransactions(context.Context, sqlc.ListTransactionsParams) ([]sqlc.ListTransactionsRow, error)
}

type CategoryRepository interface {
	CreateCategory(context.Context, sqlc.CreateCategoryParams) (sqlc.Category, error)
	ListCategories(context.Context, bool) ([]sqlc.Category, error)
	GetCategoryById(context.Context, int64) (sqlc.Category, error)
	GetActiveCategoryById(context.Context, int64) (sqlc.Category, error)
	GetCategoryByCode(context.Context, string) (sqlc.Category, error)
	GetActiveCategoryByCode(context.Context, string) (sqlc.Category, error)
	UpdateCategory(context.Context, sqlc.UpdateCategoryParams) (sqlc.Category, error)
	DeactivateCategory(context.Context, string) (sqlc.Category, error)
}

type TransactionService interface {
	GetTransactionsById(context.Context, []int64) (map[int64]sqlc.Transaction, error)
	SaveTransactions(context.Context, []sqlc.CreateTransactionParams) transaction.TransactionResult
	UpdateTransactionsById(context.Context, []transaction.UpdateTransactionById) transaction.TransactionResult
	SoftDeleteTransactionsById(context.Context, []int64, int64, *string) transaction.TransactionResult
	RestoreTransactionsById(context.Context, []int64, int64) transaction.TransactionResult
}

type Service struct {
	repo         Repository
	transactions TransactionService
}

func NewService(repo Repository, transactions TransactionService) *Service {
	return &Service{repo: repo, transactions: transactions}
}

func mapUserError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return NewError(CodeUserNotFound, "user not found", err)
	}
	return NewError(CodeDatabaseError, "database error", err)
}

func mapTransactionError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, transaction.ErrTransactionNotFound) || errors.Is(err, sql.ErrNoRows) {
		return NewError(CodeTransactionNotFound, "transaction not found", err)
	}
	if errors.Is(err, transaction.ErrDuplicateTransaction) {
		return NewError(CodeDuplicateTransaction, "duplicate transaction", err)
	}
	if errors.Is(err, transaction.ErrTransactionValidation) {
		return NewError(CodeValidationError, "transaction validation failed", err)
	}
	return NewError(CodeDatabaseError, "database error", err)
}
