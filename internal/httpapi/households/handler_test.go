package households

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	apphouseholds "rdmm404/voltr-finance/internal/app/households"
	"rdmm404/voltr-finance/internal/httpapi"
)

type householdServiceStub struct{}

func (householdServiceStub) Resolve(_ context.Context, selector apphouseholds.Selector) (apphouseholds.Household, error) {
	return apphouseholds.Household{ID: 3, Name: *selector.Name}, nil
}
func (householdServiceStub) List(context.Context) ([]apphouseholds.Household, error) {
	return []apphouseholds.Household{}, nil
}
func (householdServiceStub) Get(_ context.Context, id int64) (apphouseholds.Household, error) {
	return apphouseholds.Household{ID: id}, nil
}
func (householdServiceStub) ListUsers(context.Context, int64) ([]apphouseholds.User, error) {
	return []apphouseholds.User{}, nil
}
func TestResolveRoute(t *testing.T) {
	router := httpapi.NewRouter()
	New(householdServiceStub{}).Register(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/households/resolve?name=Home", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
func TestHouseholdReadRoutes(t *testing.T) {
	router := httpapi.NewRouter()
	New(householdServiceStub{}).Register(router)
	for _, path := range []string{"/v1/households", "/v1/households/1", "/v1/households/1/users"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Errorf("GET %s = %d: %s", path, response.Code, response.Body.String())
		}
	}
}
