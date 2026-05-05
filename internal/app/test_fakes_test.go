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

func (f *fakeRepo) ListTransactions(_ context.Context, arg sqlc.ListTransactionsParams) ([]sqlc.ListTransactionsRow, error) {
	f.lastListTransactions = arg
	return nil, nil
}

type fakeTransactionService struct {
	saved         []sqlc.CreateTransactionParams
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

func (f *fakeTransactionService) UpdateTransactionsById(context.Context, []transaction.UpdateTransactionById) transaction.TransactionResult {
	return f.updateResult
}

func (f *fakeTransactionService) SoftDeleteTransactionsById(context.Context, []int64, int64, *string) transaction.TransactionResult {
	return f.deleteResult
}

func (f *fakeTransactionService) RestoreTransactionsById(context.Context, []int64, int64) transaction.TransactionResult {
	return f.restoreResult
}
