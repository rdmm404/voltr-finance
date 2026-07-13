package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	appbudgets "rdmm404/voltr-finance/internal/app/budgets"
	appcategories "rdmm404/voltr-finance/internal/app/categories"
	apphouseholds "rdmm404/voltr-finance/internal/app/households"
	apptransactions "rdmm404/voltr-finance/internal/app/transactions"
	appusers "rdmm404/voltr-finance/internal/app/users"
	"rdmm404/voltr-finance/internal/httpapi"
)

type transactionServiceStub struct{ calls *int }

func (transactionServiceStub) Create(context.Context, apptransactions.CreateInput) (apptransactions.Transaction, error) {
	panic("unexpected Create")
}
func (transactionServiceStub) CreateBatch(context.Context, []apptransactions.CreateInput) apptransactions.BulkResult {
	panic("unexpected CreateBatch")
}
func (transactionServiceStub) Get(context.Context, int64, bool) (apptransactions.Transaction, error) {
	panic("unexpected Get")
}
func (transactionServiceStub) GetMany(context.Context, []int64, bool) ([]apptransactions.Transaction, error) {
	panic("unexpected GetMany")
}
func (s transactionServiceStub) List(context.Context, apptransactions.ListFilter) ([]apptransactions.Transaction, error) {
	(*s.calls)++
	return []apptransactions.Transaction{}, nil
}
func (transactionServiceStub) Update(context.Context, apptransactions.UpdateInput) (apptransactions.Transaction, error) {
	panic("unexpected Update")
}
func (transactionServiceStub) UpdateBatch(context.Context, []apptransactions.UpdateInput) apptransactions.BulkResult {
	panic("unexpected UpdateBatch")
}
func (transactionServiceStub) DeleteBatch(context.Context, []int64, int64, *string) apptransactions.BulkResult {
	panic("unexpected DeleteBatch")
}
func (transactionServiceStub) RestoreBatch(context.Context, []int64, int64) apptransactions.BulkResult {
	panic("unexpected RestoreBatch")
}

type userServiceStub struct{ calls *int }

func (userServiceStub) Create(context.Context, appusers.CreateInput) (appusers.User, error) {
	panic("unexpected Create")
}
func (userServiceStub) Update(context.Context, appusers.UpdateInput) (appusers.User, error) {
	panic("unexpected Update")
}
func (userServiceStub) Get(context.Context, int64) (appusers.User, error) { panic("unexpected Get") }
func (userServiceStub) Resolve(context.Context, appusers.Selector) (appusers.User, error) {
	panic("unexpected Resolve")
}
func (s userServiceStub) List(context.Context) ([]appusers.User, error) {
	(*s.calls)++
	return []appusers.User{}, nil
}

type householdServiceStub struct{ calls *int }

func (s householdServiceStub) List(context.Context) ([]apphouseholds.Household, error) {
	(*s.calls)++
	return []apphouseholds.Household{}, nil
}
func (householdServiceStub) Get(context.Context, int64) (apphouseholds.Household, error) {
	panic("unexpected Get")
}
func (householdServiceStub) Resolve(context.Context, apphouseholds.Selector) (apphouseholds.Household, error) {
	panic("unexpected Resolve")
}
func (householdServiceStub) ListUsers(context.Context, int64) ([]apphouseholds.User, error) {
	panic("unexpected ListUsers")
}

type categoryServiceStub struct{ calls *int }

func (categoryServiceStub) Create(context.Context, appcategories.CreateInput) (appcategories.Category, error) {
	panic("unexpected Create")
}
func (s categoryServiceStub) List(context.Context, bool) ([]appcategories.Category, error) {
	(*s.calls)++
	return []appcategories.Category{}, nil
}
func (categoryServiceStub) GetByCode(context.Context, string) (appcategories.Category, error) {
	panic("unexpected GetByCode")
}
func (categoryServiceStub) Update(context.Context, appcategories.UpdateInput) (appcategories.Category, error) {
	panic("unexpected Update")
}
func (categoryServiceStub) Deactivate(context.Context, string) (appcategories.Category, error) {
	panic("unexpected Deactivate")
}

type budgetServiceStub struct{ calls *int }

func (s budgetServiceStub) GetMonthly(_ context.Context, input appbudgets.MonthlyInput) (appbudgets.Budget, error) {
	(*s.calls)++
	return appbudgets.Budget{ID: 1, Owner: input.Owner, Lines: []appbudgets.Line{}}, nil
}
func (budgetServiceStub) EnsureMonthly(context.Context, appbudgets.MonthlyInput) (appbudgets.EnsureResult, error) {
	panic("unexpected EnsureMonthly")
}
func (budgetServiceStub) CreateLine(context.Context, appbudgets.CreateLineInput) (appbudgets.Line, error) {
	panic("unexpected CreateLine")
}
func (budgetServiceStub) UpdateLine(context.Context, appbudgets.UpdateLineInput) (appbudgets.Line, error) {
	panic("unexpected UpdateLine")
}
func (budgetServiceStub) DeleteLine(context.Context, int64) error { panic("unexpected DeleteLine") }
func (budgetServiceStub) Report(context.Context, int64) (appbudgets.Report, error) {
	panic("unexpected Report")
}

func TestCompositionExecutesEveryFeatureFlow(t *testing.T) {
	transactionCalls, userCalls, householdCalls, categoryCalls, budgetCalls := 0, 0, 0, 0, 0
	server, err := New(
		httpapi.Config{APIKey: "secret"},
		transactionServiceStub{calls: &transactionCalls},
		userServiceStub{calls: &userCalls},
		householdServiceStub{calls: &householdCalls},
		categoryServiceStub{calls: &categoryCalls},
		budgetServiceStub{calls: &budgetCalls},
	)
	if err != nil {
		t.Fatal(err)
	}

	requests := []struct{ feature, path string }{
		{"transactions", "/v1/transactions"},
		{"users", "/v1/users"},
		{"households", "/v1/households"},
		{"categories", "/v1/categories"},
		{"budgets", "/v1/budgets/monthly?householdId=1&year=2026&month=7"},
	}
	for _, test := range requests {
		t.Run(test.feature, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			request.Header.Set("Authorization", "Bearer secret")
			response := httptest.NewRecorder()
			server.Handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
	for feature, count := range map[string]int{
		"transactions": transactionCalls, "users": userCalls, "households": householdCalls,
		"categories": categoryCalls, "budgets": budgetCalls,
	} {
		if count != 1 {
			t.Errorf("%s service calls=%d, want 1", feature, count)
		}
	}
}
