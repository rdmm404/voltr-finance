package cli

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"rdmm404/voltr-finance/internal/restclient"
)

func TestHelpNeedsNoClient(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run(context.Background(), []string{"--help"}, nil, &stdout, &stderr, nil); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
}

func TestEveryCommandUsesOneAuthenticatedFeatureRequest(t *testing.T) {
	tests := []struct {
		name, method, path string
		args               []string
		input, response    string
		status             int
	}{
		{"transaction create", http.MethodPost, "/v1/transactions", []string{"transactions", "create", "--amount=12.5", "--transaction-date=2026-07-01T00:00:00Z", "--household-id=2", "--author-id=1"}, "", `{}`, 200},
		{"transaction create bulk", http.MethodPost, "/v1/transactions/bulk", []string{"transactions", "create-bulk"}, `{"transactions":[]}`, `{"succeeded":[],"failed":[]}`, 200},
		{"transaction update", http.MethodPatch, "/v1/transactions/1", []string{"transactions", "update", "--id=1", "--amount=13"}, "", `{}`, 200},
		{"transaction update bulk", http.MethodPatch, "/v1/transactions/bulk", []string{"transactions", "update-bulk"}, `{"transactions":[]}`, `{"succeeded":[],"failed":[]}`, 200},
		{"transaction get", http.MethodGet, "/v1/transactions", []string{"transactions", "get", "--ids=1"}, "", `[]`, 200},
		{"transaction list", http.MethodGet, "/v1/transactions", []string{"transactions", "list"}, "", `[]`, 200},
		{"transaction delete", http.MethodDelete, "/v1/transactions", []string{"transactions", "delete", "--ids=1", "--deleted-by-user-id=2"}, "", `{"succeeded":[],"failed":[]}`, 200},
		{"transaction restore", http.MethodPost, "/v1/transactions/restore", []string{"transactions", "restore", "--ids=1", "--restored-by-user-id=2"}, "", `{"succeeded":[],"failed":[]}`, 200},
		{"user create", http.MethodPost, "/v1/users", []string{"users", "create", "--name=Alice"}, "", `{}`, 200},
		{"user update", http.MethodPatch, "/v1/users/1", []string{"users", "update", "--id=1", "--name=Bob"}, "", `{}`, 200},
		{"user get", http.MethodGet, "/v1/users/1", []string{"users", "get", "--id=1"}, "", `{}`, 200},
		{"user resolve", http.MethodPost, "/v1/users/resolve", []string{"users", "resolve", "--author-id=1"}, "", `{}`, 200},
		{"user list", http.MethodGet, "/v1/users", []string{"users", "list"}, "", `[]`, 200},
		{"household get", http.MethodGet, "/v1/households/1", []string{"households", "get", "--id=1"}, "", `{}`, 200},
		{"household list", http.MethodGet, "/v1/households", []string{"households", "list"}, "", `[]`, 200},
		{"household users", http.MethodGet, "/v1/households/1/users", []string{"households", "users", "--household-id=1"}, "", `[]`, 200},
		{"category create", http.MethodPost, "/v1/categories", []string{"categories", "create", "Food"}, "", `{}`, 200},
		{"category list", http.MethodGet, "/v1/categories", []string{"categories", "list"}, "", `[]`, 200},
		{"category rename", http.MethodPatch, "/v1/categories/food", []string{"categories", "rename", "food", "Groceries"}, "", `{}`, 200},
		{"category deactivate", http.MethodDelete, "/v1/categories/food", []string{"categories", "deactivate", "food"}, "", `{}`, 200},
		{"budget get", http.MethodGet, "/v1/budgets/monthly", []string{"budgets", "get", "--household-id=1", "--month=2026-07"}, "", `{"lines":[]}`, 200},
		{"budget report", http.MethodGet, "/v1/budgets/1/report", []string{"budgets", "report", "1"}, "", `{}`, 200},
		{"budget line add", http.MethodPost, "/v1/budgets/1/lines", []string{"budgets", "lines", "add", "--budget-id=1", "--name=Food", "--amount=100"}, "", `{"categories":[]}`, 200},
		{"budget line update", http.MethodPatch, "/v1/budget-lines/1", []string{"budgets", "lines", "update", "1", "--name=Food"}, "", `{"categories":[]}`, 200},
		{"budget line delete", http.MethodDelete, "/v1/budget-lines/1", []string{"budgets", "lines", "delete", "1"}, "", "", http.StatusNoContent},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				calls++
				if request.Method != test.method || request.URL.Path != test.path {
					t.Errorf("request=%s %s, want %s %s", request.Method, request.URL.Path, test.method, test.path)
				}
				if request.Header.Get("Authorization") != "Bearer test-key" {
					t.Errorf("authorization=%q", request.Header.Get("Authorization"))
				}
				status := test.status
				if status == 0 {
					status = http.StatusOK
				}
				w.WriteHeader(status)
				_, _ = w.Write([]byte(test.response))
			}))
			defer server.Close()
			client, err := restclient.New(restclient.Config{BaseURL: server.URL, APIKey: "test-key"})
			if err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			code := Run(context.Background(), test.args, strings.NewReader(test.input), &stdout, &stderr, client)
			if code != 0 || calls != 1 {
				t.Fatalf("code=%d calls=%d stdout=%s stderr=%s", code, calls, stdout.String(), stderr.String())
			}
		})
	}
}

func TestBudgetCreateAndBulkPartialResultSemantics(t *testing.T) {
	for _, test := range []struct {
		name, path, response string
		args                 []string
		input                string
		wantCode             int
	}{
		{"budget ensure", "/v1/budgets/monthly", `{"lines":[]}`, []string{"budgets", "get", "--household-id=2", "--month=2026-07", "--create"}, "", 0},
		{"bulk partial", "/v1/transactions/bulk", `{"succeeded":[{"index":0,"id":7}],"failed":[{"index":1,"error":{"code":"validation_error","message":"bad input"}}]}`, []string{"transactions", "create-bulk"}, `{"transactions":[]}`, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.URL.Path != test.path || (test.name == "budget ensure" && request.Method != http.MethodPost) {
					t.Errorf("request=%s %s", request.Method, request.URL.Path)
				}
				_, _ = w.Write([]byte(test.response))
			}))
			defer server.Close()
			client, _ := restclient.New(restclient.Config{BaseURL: server.URL, APIKey: "key"})
			var stdout, stderr bytes.Buffer
			if code := Run(context.Background(), test.args, strings.NewReader(test.input), &stdout, &stderr, client); code != test.wantCode {
				t.Fatalf("code=%d want=%d out=%s err=%s", code, test.wantCode, stdout.String(), stderr.String())
			}
			if test.name == "bulk partial" && (!strings.Contains(stdout.String(), `"succeeded"`) || !strings.Contains(stdout.String(), `"failed"`)) {
				t.Fatalf("partial output=%s", stdout.String())
			}
		})
	}
}

func TestExitClassesAPIValidationAndTransport(t *testing.T) {
	for name, err := range map[string]error{
		"validation": &restclient.APIError{StatusCode: 400, Code: "validation_error", Message: "bad input"},
		"transport":  &restclient.TransportError{Operation: "send request", Err: errors.New("offline")},
	} {
		t.Run(name, func(t *testing.T) {
			if got := isExpectedError(err); got != (name == "validation") {
				t.Fatalf("isExpectedError=%v", got)
			}
		})
	}
}
