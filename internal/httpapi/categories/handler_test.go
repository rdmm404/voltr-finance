package categories

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appcategories "rdmm404/voltr-finance/internal/app/categories"
	apperrors "rdmm404/voltr-finance/internal/app/errors"
	"rdmm404/voltr-finance/internal/httpapi"
)

type categoryServiceStub struct{ service }

func (categoryServiceStub) Update(_ context.Context, input appcategories.UpdateInput) (appcategories.Category, error) {
	return appcategories.Category{ID: input.ID, Code: "food", Name: *input.Name, IsActive: true}, nil
}
func (categoryServiceStub) Create(context.Context, appcategories.CreateInput) (appcategories.Category, error) {
	return appcategories.Category{ID: 1}, nil
}
func (categoryServiceStub) List(context.Context, bool) ([]appcategories.Category, error) {
	return []appcategories.Category{}, nil
}
func (categoryServiceStub) GetByCode(context.Context, string) (appcategories.Category, error) {
	return appcategories.Category{ID: 1}, nil
}
func (categoryServiceStub) Deactivate(context.Context, string) (appcategories.Category, error) {
	return appcategories.Category{ID: 1, IsActive: false}, nil
}
func TestUpdateRouteUsesNumericID(t *testing.T) {
	router := httpapi.NewRouter()
	New(categoryServiceStub{}).Register(router)
	request := httptest.NewRequest(http.MethodPatch, "/v1/categories/4", strings.NewReader(`{"name":"Food"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"id":4`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

type conflictingCategoryService struct{ service }

func (conflictingCategoryService) Create(context.Context, appcategories.CreateInput) (appcategories.Category, error) {
	return appcategories.Category{}, apperrors.Conflict(apperrors.CodeCategoryConflict, "category conflict", nil)
}
func TestCategoryConflictMapsToConflict(t *testing.T) {
	router := httpapi.NewRouter()
	New(conflictingCategoryService{}).Register(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/categories", strings.NewReader(`{"name":"Food"}`)))
	if response.Code != http.StatusConflict {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
func TestCategoryLifecycleRoutes(t *testing.T) {
	router := httpapi.NewRouter()
	New(categoryServiceStub{}).Register(router)
	tests := []struct {
		method, path, body string
		status             int
	}{
		{http.MethodPost, "/v1/categories", `{"name":"Food"}`, http.StatusCreated},
		{http.MethodGet, "/v1/categories?includeInactive=true", "", http.StatusOK},
		{http.MethodGet, "/v1/categories/food", "", http.StatusOK},
		{http.MethodDelete, "/v1/categories/food", "", http.StatusOK},
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
