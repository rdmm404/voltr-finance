package transactions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apptransactions "rdmm404/voltr-finance/internal/app/transactions"
	"rdmm404/voltr-finance/internal/httpapi"
)

type transactionServiceStub struct {
	create func(context.Context, apptransactions.CreateInput) (apptransactions.Transaction, error)
}

func (s transactionServiceStub) Create(ctx context.Context, input apptransactions.CreateInput) (apptransactions.Transaction, error) {
	return s.create(ctx, input)
}
func (transactionServiceStub) Get(context.Context, int64, bool) (apptransactions.Transaction, error) {
	return apptransactions.Transaction{ID: 1}, nil
}
func (transactionServiceStub) List(context.Context, apptransactions.ListFilter) ([]apptransactions.Transaction, error) {
	return []apptransactions.Transaction{}, nil
}
func (transactionServiceStub) GetMany(context.Context, []int64, bool) ([]apptransactions.Transaction, error) {
	return []apptransactions.Transaction{}, nil
}
func (transactionServiceStub) Update(_ context.Context, input apptransactions.UpdateInput) (apptransactions.Transaction, error) {
	return apptransactions.Transaction{ID: input.ID}, nil
}
func (transactionServiceStub) CreateBatch(context.Context, []apptransactions.CreateInput) apptransactions.BulkResult {
	return apptransactions.BulkResult{Succeeded: []apptransactions.Succeeded{{Index: 0, ID: 1}}}
}
func (transactionServiceStub) UpdateBatch(context.Context, []apptransactions.UpdateInput) apptransactions.BulkResult {
	return apptransactions.BulkResult{Succeeded: []apptransactions.Succeeded{{Index: 0, ID: 1}}}
}
func (transactionServiceStub) DeleteBatch(context.Context, []int64, int64, *string) apptransactions.BulkResult {
	return apptransactions.BulkResult{Succeeded: []apptransactions.Succeeded{{Index: 0, ID: 1}}}
}
func (transactionServiceStub) RestoreBatch(context.Context, []int64, int64) apptransactions.BulkResult {
	return apptransactions.BulkResult{Succeeded: []apptransactions.Succeeded{{Index: 0, ID: 1}}}
}
func TestCreateRoute(t *testing.T) {
	stub := transactionServiceStub{create: func(_ context.Context, input apptransactions.CreateInput) (apptransactions.Transaction, error) {
		if input.Amount != 12.34 || input.Author.UserID == nil || *input.Author.UserID != 7 {
			t.Fatalf("unexpected input: %#v", input)
		}
		return apptransactions.Transaction{ID: 9, Amount: input.Amount, AuthorID: 7}, nil
	}}
	router := httpapi.NewRouter()
	New(stub).Register(router)
	request := httptest.NewRequest(http.MethodPost, "/v1/transactions", strings.NewReader(`{"amount":12.34,"transactionDate":"2026-07-13T00:00:00Z","author":{"userId":7}}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"id":9`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestLifecycleRoutes(t *testing.T) {
	stub := transactionServiceStub{create: func(context.Context, apptransactions.CreateInput) (apptransactions.Transaction, error) {
		return apptransactions.Transaction{ID: 1}, nil
	}}
	router := httpapi.NewRouter()
	New(stub).Register(router)
	tests := []struct{ method, path, body string }{
		{http.MethodGet, "/v1/transactions/1", ""},
		{http.MethodGet, "/v1/transactions?limit=10&sort=id&sortOrder=asc", ""},
		{http.MethodPatch, "/v1/transactions/1", `{}`},
		{http.MethodPost, "/v1/transactions/bulk", `{"transactions":[]}`},
		{http.MethodPatch, "/v1/transactions/bulk", `{"transactions":[]}`},
		{http.MethodDelete, "/v1/transactions", `{"ids":[1],"deletedByUserId":2}`},
		{http.MethodPost, "/v1/transactions/restore", `{"ids":[1],"restoredByUserId":2}`},
	}
	for _, test := range tests {
		request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		if test.body != "" {
			request.Header.Set("Content-Type", "application/json")
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Errorf("%s %s = %d: %s", test.method, test.path, response.Code, response.Body.String())
		}
	}
}

func TestUpdateRejectsContradictoryNullableFields(t *testing.T) {
	router := httpapi.NewRouter()
	New(transactionServiceStub{}).Register(router)
	for _, body := range []string{
		`{"description":"set","clearDescription":true}`,
		`{"notes":"set","clearNotes":true}`,
		`{"categoryId":1,"clearCategoryId":true}`,
		`{"householdId":1,"clearHouseholdId":true}`,
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPatch, "/v1/transactions/1", strings.NewReader(body)))
		if response.Code != http.StatusBadRequest {
			t.Errorf("body %s = %d: %s", body, response.Code, response.Body.String())
		}
	}
}

func TestCreateRejectsUnknownFields(t *testing.T) {
	router := httpapi.NewRouter()
	New(transactionServiceStub{}).Register(router)
	request := httptest.NewRequest(http.MethodPost, "/v1/transactions", strings.NewReader(`{"unexpected":true}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
