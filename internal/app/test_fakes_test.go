package app

import (
	"context"
	"database/sql"

	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/transaction"
)

type fakeRepo struct {
	createUser sqlc.User
	updateUser sqlc.User

	userByID        sqlc.User
	userByDiscord   sqlc.User
	userByTelegram  sqlc.User
	userByPhone     sqlc.User
	userByWhatsapp  sqlc.User
	listUsersResult []sqlc.User

	lastUpdateUser       sqlc.UpdateUserParams
	lastTelegramID       *string
	lastListTransactions sqlc.ListTransactionsParams
	transactionDetails   []sqlc.GetTransactionsByIdWithDetailsRow
	listTransactionRows  []sqlc.ListTransactionsRow

	lastCreateCategory                sqlc.CreateCategoryParams
	lastUpdateCategory                sqlc.UpdateCategoryParams
	lastListCategoriesIncludeInactive bool
	categoryByID                      sqlc.Category
	categoryByCode                    sqlc.Category
	listCategoriesResult              []sqlc.Category
}

func (f *fakeRepo) CreateUser(_ context.Context, arg sqlc.CreateUserParams) (sqlc.User, error) {
	if f.createUser.ID == 0 {
		return sqlc.User{ID: 1, Name: arg.Name, DiscordID: arg.DiscordID, TelegramID: arg.TelegramID, PhoneNumber: arg.PhoneNumber, WhatsappID: arg.WhatsappID}, nil
	}
	return f.createUser, nil
}

func (f *fakeRepo) UpdateUser(_ context.Context, arg sqlc.UpdateUserParams) (sqlc.User, error) {
	f.lastUpdateUser = arg
	if f.updateUser.ID == 0 {
		return sqlc.User{ID: arg.ID, Name: arg.Name, DiscordID: arg.DiscordID, TelegramID: arg.TelegramID, PhoneNumber: arg.PhoneNumber, WhatsappID: arg.WhatsappID}, nil
	}
	return f.updateUser, nil
}

func (f *fakeRepo) GetUserById(context.Context, int64) (sqlc.User, error) {
	if f.userByID.ID == 0 {
		return sqlc.User{}, sql.ErrNoRows
	}
	return f.userByID, nil
}

func (f *fakeRepo) GetUserByDiscordId(context.Context, *string) (sqlc.User, error) {
	return f.userByDiscord, nil
}

func (f *fakeRepo) GetUserByTelegramId(_ context.Context, telegramID *string) (sqlc.User, error) {
	f.lastTelegramID = telegramID
	if f.userByTelegram.ID == 0 {
		return sqlc.User{}, sql.ErrNoRows
	}
	return f.userByTelegram, nil
}

func (f *fakeRepo) GetUserByPhoneNumber(context.Context, *string) (sqlc.User, error) {
	return f.userByPhone, nil
}

func (f *fakeRepo) GetUserByWhatsappId(context.Context, *string) (sqlc.User, error) {
	return f.userByWhatsapp, nil
}

func (f *fakeRepo) ListUsers(context.Context) ([]sqlc.User, error) {
	return f.listUsersResult, nil
}

func (f *fakeRepo) GetHouseholdById(context.Context, int64) (sqlc.Household, error) {
	return sqlc.Household{}, nil
}

func (f *fakeRepo) GetHouseholdByGuildId(context.Context, string) (sqlc.Household, error) {
	return sqlc.Household{}, nil
}

func (f *fakeRepo) GetHouseholdByName(context.Context, string) (sqlc.Household, error) {
	return sqlc.Household{}, nil
}

func (f *fakeRepo) ListHouseholds(context.Context) ([]sqlc.Household, error) {
	return nil, nil
}

func (f *fakeRepo) GetHouseholdUsers(context.Context, int64) ([]sqlc.User, error) {
	return nil, nil
}

func (f *fakeRepo) GetTransactionsByIdWithDetails(context.Context, sqlc.GetTransactionsByIdWithDetailsParams) ([]sqlc.GetTransactionsByIdWithDetailsRow, error) {
	return f.transactionDetails, nil
}

func (f *fakeRepo) ListTransactions(_ context.Context, arg sqlc.ListTransactionsParams) ([]sqlc.ListTransactionsRow, error) {
	f.lastListTransactions = arg
	return f.listTransactionRows, nil
}

func (f *fakeRepo) CreateCategory(_ context.Context, arg sqlc.CreateCategoryParams) (sqlc.Category, error) {
	f.lastCreateCategory = arg
	return sqlc.Category{ID: 1, Code: arg.Code, Name: arg.Name, Description: arg.Description, IsActive: true}, nil
}

func (f *fakeRepo) ListCategories(_ context.Context, includeInactive bool) ([]sqlc.Category, error) {
	f.lastListCategoriesIncludeInactive = includeInactive
	return f.listCategoriesResult, nil
}

func (f *fakeRepo) GetCategoryById(context.Context, int64) (sqlc.Category, error) {
	if f.categoryByID.ID == 0 {
		return sqlc.Category{}, sql.ErrNoRows
	}
	return f.categoryByID, nil
}

func (f *fakeRepo) GetActiveCategoryById(_ context.Context, id int64) (sqlc.Category, error) {
	if f.categoryByID.ID == 0 || f.categoryByID.ID != id || !f.categoryByID.IsActive {
		return sqlc.Category{}, sql.ErrNoRows
	}
	return f.categoryByID, nil
}

func (f *fakeRepo) GetCategoryByCode(context.Context, string) (sqlc.Category, error) {
	if f.categoryByCode.ID == 0 {
		return sqlc.Category{}, sql.ErrNoRows
	}
	return f.categoryByCode, nil
}

func (f *fakeRepo) GetActiveCategoryByCode(_ context.Context, code string) (sqlc.Category, error) {
	if f.categoryByCode.ID == 0 || f.categoryByCode.Code != code || !f.categoryByCode.IsActive {
		return sqlc.Category{}, sql.ErrNoRows
	}
	return f.categoryByCode, nil
}

func (f *fakeRepo) UpdateCategory(_ context.Context, arg sqlc.UpdateCategoryParams) (sqlc.Category, error) {
	f.lastUpdateCategory = arg
	name := arg.Name
	if !arg.SetName {
		name = f.categoryByID.Name
	}
	description := arg.Description
	if !arg.SetDescription {
		description = f.categoryByID.Description
	}
	return sqlc.Category{ID: arg.ID, Code: f.categoryByID.Code, Name: name, Description: description, IsActive: true}, nil
}

func (f *fakeRepo) DeactivateCategory(context.Context, string) (sqlc.Category, error) {
	if f.categoryByCode.ID == 0 {
		return sqlc.Category{}, sql.ErrNoRows
	}
	f.categoryByCode.IsActive = false
	return f.categoryByCode, nil
}

type fakeTransactionService struct {
	saved         []sqlc.CreateTransactionParams
	updated       []transaction.UpdateTransactionById
	saveResult    transaction.TransactionResult
	updateResult  transaction.TransactionResult
	deleteResult  transaction.TransactionResult
	restoreResult transaction.TransactionResult
}

func (f *fakeTransactionService) GetTransactionsById(context.Context, []int64) (map[int64]sqlc.Transaction, error) {
	return nil, nil
}

func (f *fakeTransactionService) SaveTransactions(_ context.Context, transactions []sqlc.CreateTransactionParams) transaction.TransactionResult {
	f.saved = transactions
	if f.saveResult.Success == nil {
		f.saveResult.Success = map[int64]*sqlc.Transaction{}
	}
	return f.saveResult
}

func (f *fakeTransactionService) UpdateTransactionsById(_ context.Context, transactions []transaction.UpdateTransactionById) transaction.TransactionResult {
	f.updated = transactions
	return f.updateResult
}

func (f *fakeTransactionService) SoftDeleteTransactionsById(context.Context, []int64, int64, *string) transaction.TransactionResult {
	return f.deleteResult
}

func (f *fakeTransactionService) RestoreTransactionsById(context.Context, []int64, int64) transaction.TransactionResult {
	return f.restoreResult
}
