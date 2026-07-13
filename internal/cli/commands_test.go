package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"rdmm404/voltr-finance/internal/api"
	"rdmm404/voltr-finance/internal/restclient"
)

type fakeAPIClient struct {
	APIClient
	createTransaction   func(context.Context, api.CreateTransactionRequest) (api.Transaction, error)
	createTransactions  func(context.Context, api.BulkCreateTransactionsRequest) (api.BulkResult, error)
	listTransactions    func(context.Context, api.ListTransactionsQuery) ([]api.Transaction, error)
	getMonthlyBudget    func(context.Context, api.MonthlyBudgetParams) (api.Budget, error)
	ensureMonthlyBudget func(context.Context, api.MonthlyBudgetParams) (api.Budget, error)
	resolveUser         func(context.Context, api.IdentitySelector) (api.User, error)
}

func (f fakeAPIClient) CreateTransaction(ctx context.Context, input api.CreateTransactionRequest) (api.Transaction, error) {
	return f.createTransaction(ctx, input)
}
func (f fakeAPIClient) CreateTransactions(ctx context.Context, input api.BulkCreateTransactionsRequest) (api.BulkResult, error) {
	return f.createTransactions(ctx, input)
}
func (f fakeAPIClient) ListTransactions(ctx context.Context, input api.ListTransactionsQuery) ([]api.Transaction, error) {
	return f.listTransactions(ctx, input)
}
func (f fakeAPIClient) GetMonthlyBudget(ctx context.Context, input api.MonthlyBudgetParams) (api.Budget, error) {
	return f.getMonthlyBudget(ctx, input)
}
func (f fakeAPIClient) EnsureMonthlyBudget(ctx context.Context, input api.MonthlyBudgetParams) (api.Budget, error) {
	return f.ensureMonthlyBudget(ctx, input)
}
func (f fakeAPIClient) ResolveUser(ctx context.Context, input api.IdentitySelector) (api.User, error) {
	return f.resolveUser(ctx, input)
}

func TestHelpNeedsNoClient(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run(context.Background(), []string{"--help"}, nil, &stdout, &stderr, nil); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
}

func TestTransactionCreateMapsFlags(t *testing.T) {
	client := fakeAPIClient{createTransaction: func(_ context.Context, input api.CreateTransactionRequest) (api.Transaction, error) {
		if input.Amount != 12.5 || input.Author.DiscordID == nil || *input.Author.DiscordID != "42" {
			t.Fatalf("input=%+v", input)
		}
		return api.Transaction{ID: 7}, nil
	}}
	var stdout, stderr bytes.Buffer
	args := []string{"transactions", "create", "--amount=12.5", "--transaction-date=2026-07-01T00:00:00Z", "--household-id=2", "--author-discord-id=42"}
	if code := Run(context.Background(), args, nil, &stdout, &stderr, client); code != 0 || !strings.Contains(stdout.String(), `"id": 7`) {
		t.Fatalf("code=%d out=%s err=%s", code, stdout.String(), stderr.String())
	}
}

func TestBulkPartialSuccessRendersBeforeExitTwo(t *testing.T) {
	client := fakeAPIClient{createTransactions: func(context.Context, api.BulkCreateTransactionsRequest) (api.BulkResult, error) {
		return api.BulkResult{Succeeded: []api.BulkSucceeded{{Index: 0, ID: 7}}, Failed: []api.BulkFailed{{Index: 1, Error: api.Error{Code: "validation_error", Message: "bad input"}}}}, nil
	}}
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"transactions", "create-bulk"}, strings.NewReader(`{"transactions":[]}`), &stdout, &stderr, client)
	if code != 2 || !strings.Contains(stdout.String(), `"succeeded"`) || !strings.Contains(stdout.String(), `"failed"`) {
		t.Fatalf("code=%d out=%s err=%s", code, stdout.String(), stderr.String())
	}
}

func TestTransactionListRendersCSV(t *testing.T) {
	client := fakeAPIClient{listTransactions: func(_ context.Context, input api.ListTransactionsQuery) ([]api.Transaction, error) {
		if input.Limit != 10 {
			t.Fatalf("input=%+v", input)
		}
		return []api.Transaction{{ID: 1, Amount: 2.5, TransactionDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)}}, nil
	}}
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"transactions", "list", "--format=csv", "--limit=10"}, nil, &stdout, &stderr, client)
	if code != 0 || !strings.HasPrefix(stdout.String(), "id,amount") {
		t.Fatalf("code=%d out=%s err=%s", code, stdout.String(), stderr.String())
	}
}

func TestBudgetCreateFlagSelectsEnsureWithoutRead(t *testing.T) {
	ensured, read := 0, 0
	client := fakeAPIClient{
		ensureMonthlyBudget: func(context.Context, api.MonthlyBudgetParams) (api.Budget, error) {
			ensured++
			return api.Budget{ID: 1, Lines: []api.BudgetLine{}}, nil
		},
		getMonthlyBudget: func(context.Context, api.MonthlyBudgetParams) (api.Budget, error) { read++; return api.Budget{}, nil },
	}
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"budgets", "get", "--household-id=2", "--month=2026-07", "--create"}, nil, &stdout, &stderr, client)
	if code != 0 || ensured != 1 || read != 0 {
		t.Fatalf("code=%d ensured=%d read=%d err=%s", code, ensured, read, stderr.String())
	}
}

func TestExitClassesAPIValidationAndTransport(t *testing.T) {
	for name, err := range map[string]error{
		"validation": &restclient.APIError{StatusCode: 400, Code: "validation_error", Message: "bad input"},
		"transport":  &restclient.TransportError{Operation: "send request", Err: errors.New("offline")},
	} {
		t.Run(name, func(t *testing.T) {
			client := fakeAPIClient{resolveUser: func(context.Context, api.IdentitySelector) (api.User, error) { return api.User{}, err }}
			var stdout, stderr bytes.Buffer
			code := Run(context.Background(), []string{"users", "resolve", "--author-id=2"}, nil, &stdout, &stderr, client)
			want := 1
			if name == "validation" {
				want = 2
			}
			if code != want {
				t.Fatalf("code=%d want=%d stderr=%s", code, want, stderr.String())
			}
		})
	}
}
