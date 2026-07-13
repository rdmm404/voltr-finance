package budgets

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appbudgets "rdmm404/voltr-finance/internal/app/budgets"
	"rdmm404/voltr-finance/internal/httpapi"
)

type budgetServiceStub struct{ created bool }

func (s budgetServiceStub) EnsureMonthly(_ context.Context, input appbudgets.MonthlyInput) (appbudgets.EnsureResult, error) {
	return appbudgets.EnsureResult{Budget: appbudgets.Budget{ID: 5, Owner: input.Owner, Lines: []appbudgets.Line{}}, Created: s.created}, nil
}
func (budgetServiceStub) GetMonthly(_ context.Context, input appbudgets.MonthlyInput) (appbudgets.Budget, error) {
	return appbudgets.Budget{ID: 5, Owner: input.Owner, Lines: []appbudgets.Line{}}, nil
}
func (budgetServiceStub) CreateLine(_ context.Context, input appbudgets.CreateLineInput) (appbudgets.Line, error) {
	return appbudgets.Line{ID: 2, BudgetID: input.BudgetID, Categories: []appbudgets.Category{}}, nil
}
func (budgetServiceStub) UpdateLine(_ context.Context, input appbudgets.UpdateLineInput) (appbudgets.Line, error) {
	return appbudgets.Line{ID: input.LineID, Categories: []appbudgets.Category{}}, nil
}
func (budgetServiceStub) DeleteLine(context.Context, int64) error { return nil }
func (budgetServiceStub) Report(context.Context, int64) (appbudgets.Report, error) {
	return appbudgets.Report{Lines: []appbudgets.ReportLine{}, UnmappedTransactions: []appbudgets.UnmappedTransaction{}}, nil
}
func TestEnsureMonthlyReturnsCreated(t *testing.T) {
	router := httpapi.NewRouter()
	New(budgetServiceStub{created: true}).Register(router)
	request := httptest.NewRequest(http.MethodPost, "/v1/budgets/monthly", strings.NewReader(`{"householdId":2,"year":2026,"month":7}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"id":5`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
func TestExistingMonthlyBudgetReturnsOK(t *testing.T) {
	router := httpapi.NewRouter()
	New(budgetServiceStub{}).Register(router)
	request := httptest.NewRequest(http.MethodPost, "/v1/budgets/monthly", strings.NewReader(`{"userId":2,"year":2026,"month":7}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

type failingBudgetService struct{ budgetServiceStub }

func (failingBudgetService) Report(context.Context, int64) (appbudgets.Report, error) {
	return appbudgets.Report{}, errors.New("sql secret")
}
func TestBudgetInternalErrorIsSafe(t *testing.T) {
	router := httpapi.NewRouter()
	New(failingBudgetService{}).Register(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/budgets/1/report", nil))
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "sql secret") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
func TestBudgetReadLineAndReportRoutes(t *testing.T) {
	router := httpapi.NewRouter()
	New(budgetServiceStub{}).Register(router)
	tests := []struct {
		method, path, body string
		status             int
	}{
		{http.MethodGet, "/v1/budgets/monthly?householdId=2&year=2026&month=7", "", http.StatusOK},
		{http.MethodPost, "/v1/budgets/1/lines", `{"name":"Food","allocationAmount":"100.00"}`, http.StatusCreated},
		{http.MethodPatch, "/v1/budget-lines/2", `{"name":"Groceries"}`, http.StatusOK},
		{http.MethodDelete, "/v1/budget-lines/2", "", http.StatusNoContent},
		{http.MethodGet, "/v1/budgets/1/report", "", http.StatusOK},
	}
	for _, test := range tests {
		request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != test.status {
			t.Errorf("%s %s = %d: %s", test.method, test.path, response.Code, response.Body.String())
		}
	}
}
