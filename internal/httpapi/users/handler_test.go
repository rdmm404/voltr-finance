package users

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
	appusers "rdmm404/voltr-finance/internal/app/users"
	"rdmm404/voltr-finance/internal/httpapi"
)

type userServiceStub struct{ service }

func (userServiceStub) Create(context.Context, appusers.CreateInput) (appusers.User, error) {
	return appusers.User{ID: 1}, nil
}
func (userServiceStub) Update(_ context.Context, input appusers.UpdateInput) (appusers.User, error) {
	return appusers.User{ID: input.ID}, nil
}
func (userServiceStub) Get(_ context.Context, id int64) (appusers.User, error) {
	return appusers.User{ID: id}, nil
}
func (userServiceStub) Resolve(context.Context, appusers.Selector) (appusers.User, error) {
	return appusers.User{ID: 1}, nil
}
func (userServiceStub) List(context.Context) ([]appusers.User, error) { return nil, nil }
func TestListNormalizesEmptyArray(t *testing.T) {
	router := httpapi.NewRouter()
	New(userServiceStub{}).Register(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/users", nil))
	if response.Code != http.StatusOK || response.Body.String() != "[]\n" {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
}

type missingUserService struct{ service }

func (missingUserService) Get(context.Context, int64) (appusers.User, error) {
	return appusers.User{}, apperrors.NotFound(apperrors.CodeUserNotFound, "user not found", nil)
}
func TestMissingUserMapsToNotFound(t *testing.T) {
	router := httpapi.NewRouter()
	New(missingUserService{}).Register(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/users/9", nil))
	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "user_not_found") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
func TestUserMutationAndResolutionRoutes(t *testing.T) {
	router := httpapi.NewRouter()
	New(userServiceStub{}).Register(router)
	tests := []struct {
		method, path, body string
		status             int
	}{
		{http.MethodPost, "/v1/users", `{"name":"Ada"}`, http.StatusCreated},
		{http.MethodGet, "/v1/users/1", "", http.StatusOK},
		{http.MethodPatch, "/v1/users/1", `{"name":"Ada"}`, http.StatusOK},
		{http.MethodPost, "/v1/users/resolve", `{"userId":1}`, http.StatusOK},
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
