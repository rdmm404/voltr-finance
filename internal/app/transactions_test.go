package app

import (
	"context"
	"testing"
	"time"

	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/transaction"
)

func TestCreateTransactionResolvesAuthorAndRequiresHousehold(t *testing.T) {
	telegramID := "123456"
	repo := &fakeRepo{userByTelegram: sqlc.User{ID: 7, Name: "Rafael", TelegramID: &telegramID}}
	txSvc := &fakeTransactionService{saveResult: transaction.TransactionResult{Success: map[int64]*sqlc.Transaction{101: {ID: 101}}}}
	svc := NewService(repo, txSvc)

	result := svc.CreateTransaction(context.Background(), CreateTransactionRequest{
		Amount:          42.5,
		TransactionDate: time.Date(2026, 5, 5, 14, 30, 0, 0, time.UTC),
		Description:     strPtr("Groceries"),
		Author:          IdentitySelector{TelegramID: &telegramID},
		HouseholdID:     intPtr(1),
	})

	if len(result.Errors) != 0 {
		t.Fatalf("Errors = %v, want none", result.Errors)
	}
	if len(result.CreatedIDs) != 1 || result.CreatedIDs[0] != 101 {
		t.Fatalf("CreatedIDs = %v, want [101]", result.CreatedIDs)
	}
	if len(txSvc.saved) != 1 || txSvc.saved[0].AuthorID != 7 {
		t.Fatalf("saved params = %+v, want author 7", txSvc.saved)
	}
	if txSvc.saved[0].HouseholdID == nil || *txSvc.saved[0].HouseholdID != 1 {
		t.Fatalf("household = %v, want 1", txSvc.saved[0].HouseholdID)
	}

	missingHousehold := svc.CreateTransaction(context.Background(), CreateTransactionRequest{
		Amount:          42.5,
		TransactionDate: time.Date(2026, 5, 5, 14, 30, 0, 0, time.UTC),
		Author:          IdentitySelector{TelegramID: &telegramID},
	})
	if len(missingHousehold.Errors) != 1 || missingHousehold.Errors[0].Code != CodeValidationError {
		t.Fatalf("missing household errors = %+v, want validation_error", missingHousehold.Errors)
	}
}

func TestCreateTransactionResolvesCategoryCode(t *testing.T) {
	repo := &fakeRepo{
		userByID:           sqlc.User{ID: 7, Name: "Rafael"},
		categoryByCode:     sqlc.Category{ID: 42, Code: "groceries", Name: "Groceries", IsActive: true},
		categoryByID:       sqlc.Category{ID: 42, Code: "groceries", Name: "Groceries", IsActive: true},
		transactionDetails: nil,
	}
	txSvc := &fakeTransactionService{saveResult: transaction.TransactionResult{Success: map[int64]*sqlc.Transaction{101: {ID: 101}}}}
	svc := NewService(repo, txSvc)

	result := svc.CreateTransaction(context.Background(), CreateTransactionRequest{
		Amount:          42.5,
		TransactionDate: time.Date(2026, 5, 5, 14, 30, 0, 0, time.UTC),
		CategoryCode:    strPtr("groceries"),
		Author:          IdentitySelector{AuthorID: intPtr(7)},
		HouseholdID:     intPtr(1),
	})

	if len(result.Errors) != 0 {
		t.Fatalf("Errors = %v, want none", result.Errors)
	}
	if len(txSvc.saved) != 1 || txSvc.saved[0].CategoryID == nil || *txSvc.saved[0].CategoryID != 42 {
		t.Fatalf("saved category = %+v, want category id 42", txSvc.saved)
	}
}

func TestCreateTransactionRejectsConflictingCategorySelectors(t *testing.T) {
	repo := &fakeRepo{
		userByID:       sqlc.User{ID: 7, Name: "Rafael"},
		categoryByID:   sqlc.Category{ID: 41, Code: "dining", Name: "Dining", IsActive: true},
		categoryByCode: sqlc.Category{ID: 42, Code: "groceries", Name: "Groceries", IsActive: true},
	}
	txSvc := &fakeTransactionService{}
	svc := NewService(repo, txSvc)

	result := svc.CreateTransaction(context.Background(), CreateTransactionRequest{
		Amount:          42.5,
		TransactionDate: time.Date(2026, 5, 5, 14, 30, 0, 0, time.UTC),
		CategoryID:      intPtr(41),
		CategoryCode:    strPtr("groceries"),
		Author:          IdentitySelector{AuthorID: intPtr(7)},
		HouseholdID:     intPtr(1),
	})

	if len(result.Errors) != 1 || result.Errors[0].Code != CodeValidationError {
		t.Fatalf("Errors = %+v, want validation error", result.Errors)
	}
	if result.Errors[0].Message != "category id and code refer to different categories" {
		t.Fatalf("Message = %q, want category selector conflict", result.Errors[0].Message)
	}
	if len(txSvc.saved) != 0 {
		t.Fatalf("saved = %+v, want no transaction write", txSvc.saved)
	}
}

func TestUpdateTransactionAssignsCategory(t *testing.T) {
	repo := &fakeRepo{
		categoryByCode: sqlc.Category{ID: 42, Code: "groceries", Name: "Groceries", IsActive: true},
	}
	txSvc := &fakeTransactionService{updateResult: transaction.TransactionResult{Success: map[int64]*sqlc.Transaction{101: {ID: 101}}}}
	svc := NewService(repo, txSvc)

	result := svc.UpdateTransaction(context.Background(), UpdateTransactionRequest{
		ID:           101,
		CategoryCode: strPtr("groceries"),
	})

	if len(result.Errors) != 0 {
		t.Fatalf("Errors = %v, want none", result.Errors)
	}
	if len(txSvc.updated) != 1 {
		t.Fatalf("updated = %+v, want one transaction update", txSvc.updated)
	}
	categoryUpdate := txSvc.updated[0].Updates.CategoryID
	if !categoryUpdate.Set {
		t.Fatalf("CategoryID.Set = false, want true")
	}
	if categoryUpdate.Value == nil || *categoryUpdate.Value != 42 {
		t.Fatalf("CategoryID.Value = %v, want 42", categoryUpdate.Value)
	}
}

func TestUpdateTransactionClearsCategory(t *testing.T) {
	txSvc := &fakeTransactionService{updateResult: transaction.TransactionResult{Success: map[int64]*sqlc.Transaction{101: {ID: 101}}}}
	svc := NewService(&fakeRepo{}, txSvc)

	result := svc.UpdateTransaction(context.Background(), UpdateTransactionRequest{
		ID:              101,
		ClearCategoryID: true,
	})

	if len(result.Errors) != 0 {
		t.Fatalf("Errors = %v, want none", result.Errors)
	}
	if len(txSvc.updated) != 1 {
		t.Fatalf("updated = %+v, want one transaction update", txSvc.updated)
	}
	categoryUpdate := txSvc.updated[0].Updates.CategoryID
	if !categoryUpdate.Set {
		t.Fatalf("CategoryID.Set = false, want true")
	}
	if categoryUpdate.Value != nil {
		t.Fatalf("CategoryID.Value = %v, want nil", categoryUpdate.Value)
	}
}

func TestBulkCreateReturnsPartialFailure(t *testing.T) {
	repo := &fakeRepo{userByID: sqlc.User{ID: 7, Name: "Rafael"}}
	txSvc := &fakeTransactionService{saveResult: transaction.TransactionResult{
		Success: map[int64]*sqlc.Transaction{101: {ID: 101}},
		Errors:  []transaction.TransactionError{{Index: 1, Err: transaction.ErrDuplicateTransaction}},
	}}
	svc := NewService(repo, txSvc)

	result := svc.CreateTransactions(context.Background(), BulkCreateTransactionsRequest{
		Transactions: []CreateTransactionRequest{
			{Amount: 10, TransactionDate: time.Now(), Author: IdentitySelector{AuthorID: intPtr(7)}, HouseholdID: intPtr(1)},
			{Amount: 10, TransactionDate: time.Now(), Author: IdentitySelector{AuthorID: intPtr(7)}, HouseholdID: intPtr(1)},
		},
	})

	if len(result.CreatedIDs) != 1 || result.CreatedIDs[0] != 101 {
		t.Fatalf("CreatedIDs = %v, want [101]", result.CreatedIDs)
	}
	if len(result.Errors) != 1 || result.Errors[0].Index != 1 || result.Errors[0].Code != CodeDuplicateTransaction {
		t.Fatalf("Errors = %+v, want duplicate at index 1", result.Errors)
	}
}

func TestListTransactionsDefaultsToDateDesc(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeTransactionService{})

	_, err := svc.ListTransactions(context.Background(), ListTransactionsRequest{})
	if err != nil {
		t.Fatalf("ListTransactions returned error: %v", err)
	}
	if repo.lastListTransactions.Sort != "transaction_date" || repo.lastListTransactions.SortOrder != "desc" {
		t.Fatalf("sort = %q/%q, want transaction_date/desc", repo.lastListTransactions.Sort, repo.lastListTransactions.SortOrder)
	}
}

func TestListTransactionsIncludesCategoryDetails(t *testing.T) {
	categoryID := int64(42)
	categoryCode := "groceries"
	categoryName := "Groceries"
	repo := &fakeRepo{
		listTransactionRows: []sqlc.ListTransactionsRow{
			{
				Transaction:  sqlc.Transaction{ID: 101, AuthorID: 9, CategoryID: &categoryID},
				AuthorName:   "CLI Tester",
				CategoryID:   &categoryID,
				CategoryCode: &categoryCode,
				CategoryName: &categoryName,
			},
		},
	}
	svc := NewService(repo, &fakeTransactionService{})

	txs, err := svc.ListTransactions(context.Background(), ListTransactionsRequest{})
	if err != nil {
		t.Fatalf("ListTransactions returned error: %v", err)
	}
	if len(txs) != 1 || txs[0].Category == nil {
		t.Fatalf("transactions = %+v, want category details", txs)
	}
	if txs[0].Category.ID != 42 || txs[0].Category.Code != "groceries" || txs[0].Category.Name != "Groceries" {
		t.Fatalf("category = %+v, want groceries ref", txs[0].Category)
	}
}

func TestGetTransactionsIncludesAuthorAndHouseholdNames(t *testing.T) {
	householdID := int64(1)
	householdName := "Voltr"
	repo := &fakeRepo{
		transactionDetails: []sqlc.GetTransactionsByIdWithDetailsRow{
			{
				Transaction:   sqlc.Transaction{ID: 101, AuthorID: 9, HouseholdID: &householdID},
				AuthorName:    "CLI Tester",
				HouseholdID:   &householdID,
				HouseholdName: &householdName,
			},
		},
	}
	svc := NewService(repo, &fakeTransactionService{})

	txs, err := svc.GetTransactions(context.Background(), []int64{101}, false)
	if err != nil {
		t.Fatalf("GetTransactions returned error: %v", err)
	}
	if len(txs) != 1 || txs[0].AuthorName != "CLI Tester" || txs[0].HouseholdName == nil || *txs[0].HouseholdName != "Voltr" {
		t.Fatalf("transactions = %+v, want author and household names", txs)
	}
}

func TestGetTransactionsIncludesCategoryDetails(t *testing.T) {
	householdID := int64(1)
	householdName := "Voltr"
	categoryID := int64(42)
	categoryCode := "groceries"
	categoryName := "Groceries"
	repo := &fakeRepo{
		transactionDetails: []sqlc.GetTransactionsByIdWithDetailsRow{
			{
				Transaction:   sqlc.Transaction{ID: 101, AuthorID: 9, HouseholdID: &householdID, CategoryID: &categoryID},
				AuthorName:    "CLI Tester",
				HouseholdID:   &householdID,
				HouseholdName: &householdName,
				CategoryID:    &categoryID,
				CategoryCode:  &categoryCode,
				CategoryName:  &categoryName,
			},
		},
	}
	svc := NewService(repo, &fakeTransactionService{})

	txs, err := svc.GetTransactions(context.Background(), []int64{101}, false)
	if err != nil {
		t.Fatalf("GetTransactions returned error: %v", err)
	}
	if len(txs) != 1 || txs[0].Category == nil {
		t.Fatalf("transactions = %+v, want category details", txs)
	}
	if txs[0].Category.ID != 42 || txs[0].Category.Code != "groceries" || txs[0].Category.Name != "Groceries" {
		t.Fatalf("category = %+v, want groceries ref", txs[0].Category)
	}
}

func TestDeleteTransactionsReturnsDeletedIDs(t *testing.T) {
	txSvc := &fakeTransactionService{deleteResult: transaction.TransactionResult{Success: map[int64]*sqlc.Transaction{101: {ID: 101}}}}
	svc := NewService(&fakeRepo{}, txSvc)

	result := svc.DeleteTransactions(context.Background(), DeleteTransactionsRequest{
		IDs:             []int64{101},
		DeletedByUserID: 7,
	})

	if len(result.Errors) != 0 {
		t.Fatalf("Errors = %v, want none", result.Errors)
	}
	if len(result.DeletedIDs) != 1 || result.DeletedIDs[0] != 101 {
		t.Fatalf("DeletedIDs = %v, want [101]", result.DeletedIDs)
	}
}

func TestBulkCreatePreservesOriginalErrorIndexesAndDeterministicIDs(t *testing.T) {
	repo := &fakeRepo{userByID: sqlc.User{ID: 7, Name: "Rafael"}}
	txSvc := &fakeTransactionService{saveResult: transaction.TransactionResult{
		Success: map[int64]*sqlc.Transaction{103: {ID: 103}, 101: {ID: 101}},
		Errors:  []transaction.TransactionError{{Index: 1, Err: transaction.ErrDuplicateTransaction}},
	}}
	result := NewService(repo, txSvc).CreateTransactions(context.Background(), BulkCreateTransactionsRequest{
		Transactions: []CreateTransactionRequest{
			{Amount: 10, TransactionDate: time.Now(), Author: IdentitySelector{AuthorID: intPtr(7)}, HouseholdID: intPtr(1)},
			{Amount: 20, TransactionDate: time.Now(), HouseholdID: intPtr(1)},
			{Amount: 30, TransactionDate: time.Now(), Author: IdentitySelector{AuthorID: intPtr(7)}, HouseholdID: intPtr(1)},
			{Amount: 40, TransactionDate: time.Now(), Author: IdentitySelector{AuthorID: intPtr(7)}, HouseholdID: intPtr(1)},
		},
	})

	if len(result.Errors) != 2 || result.Errors[0].Index != 1 || result.Errors[1].Index != 2 {
		t.Fatalf("Errors = %+v, want indexes 1 and 2 in ascending order", result.Errors)
	}
	if got, want := result.CreatedIDs, []int64{101, 103}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("CreatedIDs = %v, want %v", got, want)
	}
}

func TestDeleteTransactionsReportsMissingIDsByOriginalIndex(t *testing.T) {
	txSvc := &fakeTransactionService{deleteResult: transaction.TransactionResult{
		Success: map[int64]*sqlc.Transaction{103: {ID: 103}, 101: {ID: 101}},
		Errors:  []transaction.TransactionError{{Index: 1, ID: 999, Err: transaction.ErrTransactionNotFound}},
	}}
	result := NewService(&fakeRepo{}, txSvc).DeleteTransactions(context.Background(), DeleteTransactionsRequest{
		IDs: []int64{103, 999, 101}, DeletedByUserID: 7,
	})

	if got, want := result.DeletedIDs, []int64{103, 101}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("DeletedIDs = %v, want input order %v", got, want)
	}
	if len(result.Errors) != 1 || result.Errors[0].Index != 1 || result.Errors[0].ID != 999 || result.Errors[0].Code != CodeTransactionNotFound {
		t.Fatalf("Errors = %+v, want missing id 999 at index 1", result.Errors)
	}
}

func TestRestoreTransactionsReportsMissingIDsAndOrdersErrors(t *testing.T) {
	txSvc := &fakeTransactionService{restoreResult: transaction.TransactionResult{
		Success: map[int64]*sqlc.Transaction{202: {ID: 202}},
		Errors: []transaction.TransactionError{
			{Index: 2, ID: 404, Err: transaction.ErrTransactionNotFound},
			{Index: 0, ID: 303, Err: transaction.ErrTransactionNotFound},
		},
	}}
	result := NewService(&fakeRepo{}, txSvc).RestoreTransactions(context.Background(), RestoreTransactionsRequest{
		IDs: []int64{303, 202, 404}, RestoredByUserID: 7,
	})

	if len(result.RestoredIDs) != 1 || result.RestoredIDs[0] != 202 {
		t.Fatalf("RestoredIDs = %v, want [202]", result.RestoredIDs)
	}
	if len(result.Errors) != 2 || result.Errors[0].Index != 0 || result.Errors[1].Index != 2 {
		t.Fatalf("Errors = %+v, want indexes 0 and 2 in ascending order", result.Errors)
	}
}

func intPtr(value int64) *int64 {
	return &value
}
