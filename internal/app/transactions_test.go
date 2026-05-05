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

func intPtr(value int64) *int64 {
	return &value
}
