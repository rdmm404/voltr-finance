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

type userRepositoryStub struct{ appusers.Repository }

func (userRepositoryStub) List(context.Context) ([]appusers.User, error) {
	return []appusers.User{}, nil
}

func TestCompositionRegistersEveryEndpointClass(t *testing.T) {
	userService := appusers.NewService(userRepositoryStub{})
	server, err := New(
		httpapi.Config{APIKey: "secret"},
		(*apptransactions.Service)(nil),
		userService,
		(*apphouseholds.Service)(nil),
		(*appcategories.Service)(nil),
		(*appbudgets.Service)(nil),
	)
	if err != nil {
		t.Fatal(err)
	}

	paths := []string{
		"/v1/transactions/1",
		"/v1/users/1",
		"/v1/households/1/users",
		"/v1/categories/food",
		"/v1/budgets/1/report",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodOptions, path, nil)
			request.Header.Set("Authorization", "Bearer secret")
			response := httptest.NewRecorder()
			server.Handler.ServeHTTP(response, request)
			if response.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
		})
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	server.Handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "[]\n" {
		t.Fatalf("authenticated feature response = %d %q", response.Code, response.Body.String())
	}
}
