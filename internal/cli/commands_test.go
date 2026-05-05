package cli

import (
	"bytes"
	"context"
	"testing"

	"rdmm404/voltr-finance/internal/app"
)

func TestKongTransactionsCreate(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"transactions", "create",
		"--amount", "42.50",
		"--transaction-date", "2026-05-05T14:30:00-04:00",
		"--description", "Groceries",
		"--author-telegram-id", "123456",
		"--household-id", "1",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if svc.createTransaction.Amount != 42.5 || svc.createTransaction.Author.TelegramID == nil || *svc.createTransaction.Author.TelegramID != "123456" {
		t.Fatalf("request = %+v, want amount and telegram author", svc.createTransaction)
	}
	if svc.createTransaction.HouseholdID == nil || *svc.createTransaction.HouseholdID != 1 {
		t.Fatalf("household id = %v, want 1", svc.createTransaction.HouseholdID)
	}
}

func TestKongTransactionsListCSV(t *testing.T) {
	svc := &fakeAppService{}
	var out bytes.Buffer
	code := Run(context.Background(), []string{
		"transactions", "list",
		"--format", "csv",
		"--search", "Manual",
	}, nil, &out, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if svc.listTransactions.Search == nil || *svc.listTransactions.Search != "Manual" {
		t.Fatalf("search = %v, want Manual", svc.listTransactions.Search)
	}
	if got := out.String(); got == "" || got[:len("id,amount")] != "id,amount" {
		t.Fatalf("csv output = %q", got)
	}
}

func TestKongUsersResolveTelegram(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"users", "resolve",
		"--telegram-id", "123456",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if svc.resolveUser.TelegramID == nil || *svc.resolveUser.TelegramID != "123456" {
		t.Fatalf("selector = %+v, want telegram id", svc.resolveUser)
	}
}

func TestKongHouseholdsGetByName(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"households", "get",
		"--name", "Home",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if svc.getHousehold.Name == nil || *svc.getHousehold.Name != "Home" {
		t.Fatalf("request = %+v, want name Home", svc.getHousehold)
	}
}

type fakeAppService struct {
	createTransaction app.CreateTransactionRequest
	listTransactions  app.ListTransactionsRequest
	resolveUser       app.IdentitySelector
	getHousehold      app.GetHouseholdRequest
}

func (f *fakeAppService) CreateTransaction(_ context.Context, req app.CreateTransactionRequest) app.WriteResult {
	f.createTransaction = req
	return app.WriteResult{CreatedIDs: []int64{101}}
}

func (f *fakeAppService) CreateTransactions(context.Context, app.BulkCreateTransactionsRequest) app.WriteResult {
	return app.WriteResult{}
}

func (f *fakeAppService) UpdateTransaction(context.Context, app.UpdateTransactionRequest) app.WriteResult {
	return app.WriteResult{}
}

func (f *fakeAppService) UpdateTransactions(context.Context, app.BulkUpdateTransactionsRequest) app.WriteResult {
	return app.WriteResult{}
}

func (f *fakeAppService) GetTransactions(context.Context, []int64, bool) ([]app.TransactionDTO, error) {
	return nil, nil
}

func (f *fakeAppService) ListTransactions(_ context.Context, req app.ListTransactionsRequest) ([]app.TransactionDTO, error) {
	f.listTransactions = req
	return []app.TransactionDTO{}, nil
}

func (f *fakeAppService) DeleteTransactions(context.Context, app.DeleteTransactionsRequest) app.WriteResult {
	return app.WriteResult{}
}

func (f *fakeAppService) RestoreTransactions(context.Context, app.RestoreTransactionsRequest) app.WriteResult {
	return app.WriteResult{}
}

func (f *fakeAppService) CreateUser(context.Context, app.CreateUserRequest) (app.UserDTO, error) {
	return app.UserDTO{ID: 1}, nil
}

func (f *fakeAppService) UpdateUser(context.Context, app.UpdateUserRequest) (app.UserDTO, error) {
	return app.UserDTO{ID: 1}, nil
}

func (f *fakeAppService) GetUser(context.Context, int64) (app.UserDTO, error) {
	return app.UserDTO{ID: 1}, nil
}

func (f *fakeAppService) ResolveUser(_ context.Context, selector app.IdentitySelector) (app.UserDTO, error) {
	f.resolveUser = selector
	return app.UserDTO{ID: 1}, nil
}

func (f *fakeAppService) ListUsers(context.Context) ([]app.UserDTO, error) {
	return nil, nil
}

func (f *fakeAppService) GetHousehold(_ context.Context, req app.GetHouseholdRequest) (app.HouseholdDTO, error) {
	f.getHousehold = req
	return app.HouseholdDTO{ID: 1, Name: "Home"}, nil
}

func (f *fakeAppService) ListHouseholds(context.Context) ([]app.HouseholdDTO, error) {
	return nil, nil
}

func (f *fakeAppService) GetHouseholdUsers(context.Context, int64) ([]app.UserDTO, error) {
	return nil, nil
}
