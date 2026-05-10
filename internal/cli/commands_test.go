package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
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

func TestKongTransactionsCreateCategoryFlag(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"transactions", "create",
		"--amount", "42.50",
		"--transaction-date", "2026-05-05T14:30:00-04:00",
		"--author-telegram-id", "123456",
		"--household-id", "1",
		"--category", "groceries",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if svc.createTransaction.CategoryCode == nil || *svc.createTransaction.CategoryCode != "groceries" {
		t.Fatalf("category = %v, want groceries", svc.createTransaction.CategoryCode)
	}
}

func TestKongTransactionsUpdateClearCategory(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"transactions", "update",
		"--id", "101",
		"--clear-category",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !svc.updateTransaction.ClearCategoryID {
		t.Fatalf("ClearCategoryID = false, want true")
	}
}

func TestKongCategoryCreate(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"categories", "create",
		"Restaurants & Takeout",
		"--code", "restaurants",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if svc.createCategory.Name != "Restaurants & Takeout" || svc.createCategory.Code == nil || *svc.createCategory.Code != "restaurants" {
		t.Fatalf("createCategory = %+v, want name and explicit code", svc.createCategory)
	}
}

func TestKongCategoryListIncludesInactive(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"categories", "list",
		"--include-inactive",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !svc.listCategories.IncludeInactive {
		t.Fatalf("IncludeInactive = false, want true")
	}
}

func TestKongCategoryDeactivate(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"categories", "deactivate",
		"restaurants",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if svc.deactivateCategoryCode != "restaurants" {
		t.Fatalf("deactivate code = %q, want restaurants", svc.deactivateCategoryCode)
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

func TestKongTransactionsDeleteUsesIdsFlag(t *testing.T) {
	svc := &fakeAppService{}
	code := Run(context.Background(), []string{
		"transactions", "delete",
		"--ids", "101,102",
		"--deleted-by-user-id", "7",
	}, nil, &bytes.Buffer{}, &bytes.Buffer{}, svc)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if len(svc.deleteTransactions.IDs) != 2 || svc.deleteTransactions.IDs[0] != 101 || svc.deleteTransactions.IDs[1] != 102 {
		t.Fatalf("ids = %v, want [101 102]", svc.deleteTransactions.IDs)
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

func TestKongSubcommandHelpDoesNotPanicWithoutService(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run(context.Background(), []string{
		"transactions", "list", "--help",
	}, nil, &stdout, &stderr, nil)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if got := stdout.String(); got == "" {
		t.Fatalf("stdout was empty, want help output")
	}
}

func TestKongExpectedErrorIncludesCause(t *testing.T) {
	svc := &fakeAppService{
		listTransactionsErr: app.NewError(app.CodeDatabaseError, "transaction list failed", errors.New("relation transactions.transactions does not exist")),
	}
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{
		"transactions", "list",
	}, nil, &bytes.Buffer{}, &stderr, svc)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if got := stderr.String(); got != "transaction list failed: relation transactions.transactions does not exist\n" {
		t.Fatalf("stderr = %q", got)
	}
}

func TestKongHelpDocumentsFlagSemantics(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "transaction create",
			args: []string{"transactions", "create", "--help"},
			want: []string{
				"Transaction amount, in dollars.",
				"Transaction timestamp in RFC3339 format, for example 2026-05-05T14:30:00-04:00.",
				"Exactly one author selector may be provided.",
				"Internal household ID.",
			},
		},
		{
			name: "transaction list",
			args: []string{"transactions", "list", "--help"},
			want: []string{
				`--format="json"`,
				"Output format: json or csv.",
				"Sort field: transaction_date, created_at, amount, or id.",
				"Sort order: asc or desc.",
				"Include soft-deleted transactions.",
			},
		},
		{
			name: "transaction bulk",
			args: []string{"transactions", "create-bulk", "--help"},
			want: []string{
				"Path to a JSON file containing a bulk create request. Reads stdin when omitted.",
				"Expected shape: {\"transactions\":[...]}",
			},
		},
		{
			name: "user update",
			args: []string{"users", "update", "--help"},
			want: []string{
				"Internal user ID.",
				"Clear the Discord ID.",
				"Telegram user ID.",
			},
		},
		{
			name: "household get",
			args: []string{"households", "get", "--help"},
			want: []string{
				"Exactly one household selector is required.",
				"Internal household ID.",
				"Discord guild/server ID.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := Run(context.Background(), tt.args, nil, &stdout, &stderr, nil)

			if code != 0 {
				t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
			}
			got := normalizeHelp(stdout.String())
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Fatalf("help output missing %q\noutput:\n%s", want, stdout.String())
				}
			}
		})
	}
}

func normalizeHelp(help string) string {
	return strings.Join(strings.Fields(help), " ")
}

type fakeAppService struct {
	createTransaction      app.CreateTransactionRequest
	updateTransaction      app.UpdateTransactionRequest
	listTransactions       app.ListTransactionsRequest
	listTransactionsErr    error
	deleteTransactions     app.DeleteTransactionsRequest
	resolveUser            app.IdentitySelector
	getHousehold           app.GetHouseholdRequest
	createCategory         app.CreateCategoryRequest
	listCategories         app.ListCategoriesRequest
	updateCategory         app.UpdateCategoryRequest
	deactivateCategoryCode string
}

func (f *fakeAppService) CreateTransaction(_ context.Context, req app.CreateTransactionRequest) app.WriteResult {
	f.createTransaction = req
	return app.WriteResult{CreatedIDs: []int64{101}}
}

func (f *fakeAppService) CreateTransactions(context.Context, app.BulkCreateTransactionsRequest) app.WriteResult {
	return app.WriteResult{}
}

func (f *fakeAppService) UpdateTransaction(_ context.Context, req app.UpdateTransactionRequest) app.WriteResult {
	f.updateTransaction = req
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
	if f.listTransactionsErr != nil {
		return nil, f.listTransactionsErr
	}
	return []app.TransactionDTO{}, nil
}

func (f *fakeAppService) DeleteTransactions(_ context.Context, req app.DeleteTransactionsRequest) app.WriteResult {
	f.deleteTransactions = req
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

func (f *fakeAppService) CreateCategory(_ context.Context, req app.CreateCategoryRequest) (app.CategoryDTO, error) {
	f.createCategory = req
	code := ""
	if req.Code != nil {
		code = *req.Code
	}
	return app.CategoryDTO{ID: 1, Code: code, Name: req.Name, IsActive: true}, nil
}

func (f *fakeAppService) ListCategories(_ context.Context, req app.ListCategoriesRequest) ([]app.CategoryDTO, error) {
	f.listCategories = req
	return []app.CategoryDTO{}, nil
}

func (f *fakeAppService) GetCategoryByCode(context.Context, string) (app.CategoryDTO, error) {
	return app.CategoryDTO{ID: 1, Code: "groceries", Name: "Groceries", IsActive: true}, nil
}

func (f *fakeAppService) UpdateCategory(_ context.Context, req app.UpdateCategoryRequest) (app.CategoryDTO, error) {
	f.updateCategory = req
	name := ""
	if req.Name != nil {
		name = *req.Name
	}
	return app.CategoryDTO{ID: req.ID, Code: "groceries", Name: name, IsActive: true}, nil
}

func (f *fakeAppService) DeactivateCategory(_ context.Context, code string) (app.CategoryDTO, error) {
	f.deactivateCategoryCode = code
	return app.CategoryDTO{ID: 1, Code: code, Name: "Groceries", IsActive: false}, nil
}
