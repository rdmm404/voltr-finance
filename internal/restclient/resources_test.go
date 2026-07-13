package restclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"rdmm404/voltr-finance/internal/api"
)

func TestUserHouseholdAndCategoryMethods(t *testing.T) {
	name, guild := "Home", "guild-1"
	tests := []struct {
		name, method, path, response string
		call                         func(*Client) error
	}{
		{"create user", http.MethodPost, "/v1/users", `{}`, func(c *Client) error {
			_, err := c.CreateUser(context.Background(), api.CreateUserRequest{})
			return err
		}},
		{"list users", http.MethodGet, "/v1/users", `[]`, func(c *Client) error { _, err := c.ListUsers(context.Background()); return err }},
		{"get user", http.MethodGet, "/v1/users/2", `{}`, func(c *Client) error { _, err := c.GetUser(context.Background(), 2); return err }},
		{"update user", http.MethodPatch, "/v1/users/2", `{}`, func(c *Client) error {
			_, err := c.UpdateUser(context.Background(), 2, api.UpdateUserRequest{})
			return err
		}},
		{"resolve user", http.MethodPost, "/v1/users/resolve", `{}`, func(c *Client) error {
			_, err := c.ResolveUser(context.Background(), api.IdentitySelector{UserID: pointer64(2)})
			return err
		}},
		{"list households", http.MethodGet, "/v1/households", `[]`, func(c *Client) error { _, err := c.ListHouseholds(context.Background()); return err }},
		{"get household", http.MethodGet, "/v1/households/3", `{}`, func(c *Client) error { _, err := c.GetHousehold(context.Background(), 3); return err }},
		{"resolve household", http.MethodGet, "/v1/households/resolve?guildId=guild-1&name=Home", `{}`, func(c *Client) error {
			_, err := c.ResolveHousehold(context.Background(), api.ResolveHouseholdQuery{Name: &name, GuildID: &guild})
			return err
		}},
		{"household users", http.MethodGet, "/v1/households/3/users", `[]`, func(c *Client) error { _, err := c.ListHouseholdUsers(context.Background(), 3); return err }},
		{"create category", http.MethodPost, "/v1/categories", `{}`, func(c *Client) error {
			_, err := c.CreateCategory(context.Background(), api.CreateCategoryRequest{})
			return err
		}},
		{"list categories", http.MethodGet, "/v1/categories?includeInactive=true", `[]`, func(c *Client) error {
			_, err := c.ListCategories(context.Background(), api.ListCategoriesQuery{IncludeInactive: true})
			return err
		}},
		{"get category", http.MethodGet, "/v1/categories/food", `{}`, func(c *Client) error { _, err := c.GetCategory(context.Background(), "food"); return err }},
		{"update category", http.MethodPatch, "/v1/categories/4", `{}`, func(c *Client) error {
			_, err := c.UpdateCategory(context.Background(), 4, api.UpdateCategoryRequest{})
			return err
		}},
		{"deactivate category", http.MethodDelete, "/v1/categories/food", `{}`, func(c *Client) error { _, err := c.DeactivateCategory(context.Background(), "food"); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.Method != test.method || request.URL.RequestURI() != test.path {
					t.Errorf("request = %s %s", request.Method, request.URL.RequestURI())
				}
				if test.name == "resolve user" {
					body, _ := io.ReadAll(request.Body)
					if string(body) != `{"userId":2}` {
						t.Errorf("selector body = %s", body)
					}
				}
				_, _ = w.Write([]byte(test.response))
			}))
			defer server.Close()
			client, _ := New(Config{BaseURL: server.URL, APIKey: "key"})
			if err := test.call(client); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func pointer64(value int64) *int64 { return &value }
