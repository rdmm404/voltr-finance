package restclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"rdmm404/voltr-finance/internal/api"
)

func TestBudgetMethods(t *testing.T) {
	householdID := int64(3)
	tests := []struct {
		name, method, path, response string
		status                       int
		call                         func(*Client) error
	}{
		{"get monthly", http.MethodGet, "/v1/budgets/monthly?householdId=3&month=7&year=2026", `{}`, http.StatusOK, func(c *Client) error {
			_, err := c.GetMonthlyBudget(context.Background(), api.MonthlyBudgetQuery{HouseholdID: &householdID, Year: 2026, Month: 7})
			return err
		}},
		{"ensure monthly", http.MethodPost, "/v1/budgets/monthly", `{}`, http.StatusCreated, func(c *Client) error {
			_, err := c.EnsureMonthlyBudget(context.Background(), api.EnsureMonthlyBudgetRequest{HouseholdID: &householdID, Year: 2026, Month: 7})
			return err
		}},
		{"create line", http.MethodPost, "/v1/budgets/5/lines", `{}`, http.StatusCreated, func(c *Client) error {
			_, err := c.CreateBudgetLine(context.Background(), 5, api.CreateBudgetLineRequest{})
			return err
		}},
		{"update line", http.MethodPatch, "/v1/budget-lines/6", `{}`, http.StatusOK, func(c *Client) error {
			_, err := c.UpdateBudgetLine(context.Background(), 6, api.UpdateBudgetLineRequest{})
			return err
		}},
		{"delete line", http.MethodDelete, "/v1/budget-lines/6", ``, http.StatusNoContent, func(c *Client) error { return c.DeleteBudgetLine(context.Background(), 6) }},
		{"report", http.MethodGet, "/v1/budgets/5/report", `{"budget":{},"lines":[],"unmappedTransactions":[],"totals":{}}`, http.StatusOK, func(c *Client) error { _, err := c.GetBudgetReport(context.Background(), 5); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.Method != test.method || request.URL.RequestURI() != test.path {
					t.Errorf("request = %s %s", request.Method, request.URL.RequestURI())
				}
				w.WriteHeader(test.status)
				if test.response != "" {
					_, _ = w.Write([]byte(test.response))
				}
			}))
			defer server.Close()
			client, _ := New(Config{BaseURL: server.URL, APIKey: "key"})
			if err := test.call(client); err != nil {
				t.Fatal(err)
			}
		})
	}
}
