package app

import (
	"context"
	"database/sql"
	"time"

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

	householdBudgetByPeriod     sqlc.Budget
	userBudgetByPeriod          sqlc.Budget
	budgetByID                  sqlc.Budget
	latestPriorHousehold        sqlc.Budget
	latestPriorUser             sqlc.Budget
	createdHouseholdBudget      sqlc.Budget
	createdUserBudget           sqlc.Budget
	budgetLines                 []sqlc.BudgetLine
	budgetLineCategories        []sqlc.ListBudgetLineCategoriesRow
	createdBudgetLines          []sqlc.CreateBudgetLineParams
	createdBudgetLineRows       []sqlc.BudgetLine
	createdBudgetLineCategories []sqlc.CreateBudgetLineCategoryParams
	budgetLineByID              sqlc.BudgetLine
	updatedBudgetLine           sqlc.BudgetLine
	lastUpdateBudgetLine        sqlc.UpdateBudgetLineParams
	maxSortOrder                int32
	deletedBudgetLineID         int64
	deletedBudgetLineCategoryID int64

	lastHouseholdBudgetPeriodStart       time.Time
	lastUserBudgetPeriodStart            time.Time
	lastCreateHouseholdBudget            sqlc.CreateHouseholdBudgetParams
	lastCreateUserBudget                 sqlc.CreateUserBudgetParams
	lastLatestPriorHouseholdBudget       sqlc.GetLatestPriorHouseholdBudgetParams
	lastLatestPriorUserBudget            sqlc.GetLatestPriorUserBudgetParams
	lastListBudgetLinesBudgetID          int64
	lastListBudgetLineCategoriesBudgetID int64
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

func (f *fakeRepo) GetHouseholdBudgetByPeriod(_ context.Context, arg sqlc.GetHouseholdBudgetByPeriodParams) (sqlc.Budget, error) {
	f.lastHouseholdBudgetPeriodStart = arg.PeriodStart.Time
	if f.householdBudgetByPeriod.ID == 0 {
		return sqlc.Budget{}, sql.ErrNoRows
	}
	return f.householdBudgetByPeriod, nil
}

func (f *fakeRepo) GetUserBudgetByPeriod(_ context.Context, arg sqlc.GetUserBudgetByPeriodParams) (sqlc.Budget, error) {
	f.lastUserBudgetPeriodStart = arg.PeriodStart.Time
	if f.userBudgetByPeriod.ID == 0 {
		return sqlc.Budget{}, sql.ErrNoRows
	}
	return f.userBudgetByPeriod, nil
}

func (f *fakeRepo) GetBudgetById(context.Context, int64) (sqlc.Budget, error) {
	if f.budgetByID.ID == 0 {
		return sqlc.Budget{}, sql.ErrNoRows
	}
	return f.budgetByID, nil
}

func (f *fakeRepo) GetLatestPriorHouseholdBudget(_ context.Context, arg sqlc.GetLatestPriorHouseholdBudgetParams) (sqlc.Budget, error) {
	f.lastLatestPriorHouseholdBudget = arg
	if f.latestPriorHousehold.ID == 0 {
		return sqlc.Budget{}, sql.ErrNoRows
	}
	return f.latestPriorHousehold, nil
}

func (f *fakeRepo) GetLatestPriorUserBudget(_ context.Context, arg sqlc.GetLatestPriorUserBudgetParams) (sqlc.Budget, error) {
	f.lastLatestPriorUserBudget = arg
	if f.latestPriorUser.ID == 0 {
		return sqlc.Budget{}, sql.ErrNoRows
	}
	return f.latestPriorUser, nil
}

func (f *fakeRepo) ListBudgetLines(_ context.Context, budgetID int64) ([]sqlc.BudgetLine, error) {
	f.lastListBudgetLinesBudgetID = budgetID
	lines := make([]sqlc.BudgetLine, 0, len(f.budgetLines))
	for _, line := range f.budgetLines {
		if line.BudgetID == budgetID {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func (f *fakeRepo) ListBudgetLineCategories(_ context.Context, budgetID int64) ([]sqlc.ListBudgetLineCategoriesRow, error) {
	f.lastListBudgetLineCategoriesBudgetID = budgetID
	categories := make([]sqlc.ListBudgetLineCategoriesRow, 0, len(f.budgetLineCategories))
	for _, category := range f.budgetLineCategories {
		if category.BudgetID == budgetID {
			categories = append(categories, category)
		}
	}
	return categories, nil
}

func (f *fakeRepo) GetBudgetLineById(_ context.Context, id int64) (sqlc.BudgetLine, error) {
	if f.budgetLineByID.ID == 0 || f.budgetLineByID.ID != id {
		return sqlc.BudgetLine{}, sql.ErrNoRows
	}
	return f.budgetLineByID, nil
}

func (f *fakeRepo) GetMaxBudgetLineSortOrder(context.Context, int64) (int32, error) {
	return f.maxSortOrder, nil
}

func (f *fakeRepo) CreateHouseholdBudget(_ context.Context, arg sqlc.CreateHouseholdBudgetParams) (sqlc.Budget, error) {
	f.lastCreateHouseholdBudget = arg
	if f.createdHouseholdBudget.ID != 0 {
		return f.createdHouseholdBudget, nil
	}
	return sqlc.Budget{
		ID:             1,
		HouseholdID:    &arg.HouseholdID,
		PeriodStart:    arg.PeriodStart,
		PeriodEnd:      arg.PeriodEnd,
		SourceBudgetID: arg.SourceBudgetID,
	}, nil
}

func (f *fakeRepo) CreateUserBudget(_ context.Context, arg sqlc.CreateUserBudgetParams) (sqlc.Budget, error) {
	f.lastCreateUserBudget = arg
	if f.createdUserBudget.ID != 0 {
		return f.createdUserBudget, nil
	}
	return sqlc.Budget{
		ID:             1,
		UserID:         &arg.UserID,
		PeriodStart:    arg.PeriodStart,
		PeriodEnd:      arg.PeriodEnd,
		SourceBudgetID: arg.SourceBudgetID,
	}, nil
}

func (f *fakeRepo) CreateBudgetLine(_ context.Context, arg sqlc.CreateBudgetLineParams) (sqlc.BudgetLine, error) {
	f.createdBudgetLines = append(f.createdBudgetLines, arg)
	if len(f.createdBudgetLineRows) >= len(f.createdBudgetLines) {
		row := f.createdBudgetLineRows[len(f.createdBudgetLines)-1]
		f.budgetLines = append(f.budgetLines, row)
		return row, nil
	}
	row := sqlc.BudgetLine{
		ID:               int64(len(f.createdBudgetLines)),
		BudgetID:         arg.BudgetID,
		Name:             arg.Name,
		AllocationAmount: arg.AllocationAmount,
		SortOrder:        arg.SortOrder,
	}
	f.budgetLines = append(f.budgetLines, row)
	return row, nil
}

func (f *fakeRepo) UpdateBudgetLine(_ context.Context, arg sqlc.UpdateBudgetLineParams) (sqlc.BudgetLine, error) {
	f.lastUpdateBudgetLine = arg
	if f.updatedBudgetLine.ID != 0 {
		return f.updatedBudgetLine, nil
	}
	if f.budgetLineByID.ID == 0 {
		return sqlc.BudgetLine{}, sql.ErrNoRows
	}
	line := f.budgetLineByID
	if arg.SetName {
		line.Name = arg.Name
	}
	if arg.SetAllocationAmount {
		line.AllocationAmount = arg.AllocationAmount
	}
	if arg.SetSortOrder {
		line.SortOrder = arg.SortOrder
	}
	return line, nil
}

func (f *fakeRepo) DeleteBudgetLine(_ context.Context, id int64) error {
	f.deletedBudgetLineID = id
	return nil
}

func (f *fakeRepo) DeleteBudgetLineCategories(_ context.Context, id int64) error {
	f.deletedBudgetLineCategoryID = id
	filtered := f.budgetLineCategories[:0]
	for _, category := range f.budgetLineCategories {
		if category.BudgetLineID != id {
			filtered = append(filtered, category)
		}
	}
	f.budgetLineCategories = filtered
	return nil
}

func (f *fakeRepo) CreateBudgetLineCategory(_ context.Context, arg sqlc.CreateBudgetLineCategoryParams) error {
	f.createdBudgetLineCategories = append(f.createdBudgetLineCategories, arg)
	categoryCode := ""
	categoryName := ""
	if f.categoryByID.ID == arg.CategoryID {
		categoryCode = f.categoryByID.Code
		categoryName = f.categoryByID.Name
	}
	if f.categoryByCode.ID == arg.CategoryID {
		categoryCode = f.categoryByCode.Code
		categoryName = f.categoryByCode.Name
	}
	for _, category := range f.budgetLineCategories {
		if category.CategoryID == arg.CategoryID {
			categoryCode = category.CategoryCode
			categoryName = category.CategoryName
			break
		}
	}
	f.budgetLineCategories = append(f.budgetLineCategories, sqlc.ListBudgetLineCategoriesRow{
		BudgetID:     arg.BudgetID,
		BudgetLineID: arg.BudgetLineID,
		CategoryID:   arg.CategoryID,
		CategoryCode: categoryCode,
		CategoryName: categoryName,
	})
	return nil
}

func (f *fakeRepo) ListBudgetTransactions(context.Context, sqlc.ListBudgetTransactionsParams) ([]sqlc.ListBudgetTransactionsRow, error) {
	return nil, nil
}

func (f *fakeRepo) SumUncategorizedBudgetTransactions(context.Context, sqlc.SumUncategorizedBudgetTransactionsParams) (float32, error) {
	return 0, nil
}

type fakeTransactor struct {
	repo  Repository
	calls int
}

func (f *fakeTransactor) WithinTx(ctx context.Context, fn func(Repository) error) error {
	f.calls++
	return fn(f.repo)
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
