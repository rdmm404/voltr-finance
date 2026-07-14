package restclient

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"rdmm404/voltr-finance/internal/api"
)

func (c *Client) CreateUser(ctx context.Context, request api.CreateUserRequest) (api.User, error) {
	var response api.User
	err := c.do(ctx, http.MethodPost, api.UsersPath, nil, request, &response)
	return response, err
}
func (c *Client) ListUsers(ctx context.Context) ([]api.User, error) {
	var response []api.User
	err := c.do(ctx, http.MethodGet, api.UsersPath, nil, nil, &response)
	return response, err
}
func (c *Client) GetUser(ctx context.Context, id int64) (api.User, error) {
	var response api.User
	err := c.do(ctx, http.MethodGet, replace(api.UserPath, "{id}", id), nil, nil, &response)
	return response, err
}
func (c *Client) UpdateUser(ctx context.Context, id int64, request api.UpdateUserRequest) (api.User, error) {
	var response api.User
	err := c.do(ctx, http.MethodPatch, replace(api.UserPath, "{id}", id), nil, request, &response)
	return response, err
}
func (c *Client) ResolveUser(ctx context.Context, selector api.IdentitySelector) (api.User, error) {
	var response api.User
	err := c.do(ctx, http.MethodPost, api.UserResolvePath, nil, api.ResolveUserRequest{IdentitySelector: selector}, &response)
	return response, err
}

func (c *Client) ListHouseholds(ctx context.Context) ([]api.Household, error) {
	var response []api.Household
	err := c.do(ctx, http.MethodGet, api.HouseholdsPath, nil, nil, &response)
	return response, err
}
func (c *Client) GetHousehold(ctx context.Context, id int64) (api.Household, error) {
	var response api.Household
	err := c.do(ctx, http.MethodGet, replace(api.HouseholdPath, "{id}", id), nil, nil, &response)
	return response, err
}
func (c *Client) ResolveHousehold(ctx context.Context, selector api.ResolveHouseholdQuery) (api.Household, error) {
	query := url.Values{}
	if selector.Name != nil {
		query.Set("name", *selector.Name)
	}
	if selector.GuildID != nil {
		query.Set("guildId", *selector.GuildID)
	}
	var response api.Household
	err := c.do(ctx, http.MethodGet, api.HouseholdResolvePath, query, nil, &response)
	return response, err
}
func (c *Client) ListHouseholdUsers(ctx context.Context, id int64) ([]api.User, error) {
	var response []api.User
	err := c.do(ctx, http.MethodGet, replace(api.HouseholdUsersPath, "{id}", id), nil, nil, &response)
	return response, err
}

func (c *Client) CreateCategory(ctx context.Context, request api.CreateCategoryRequest) (api.Category, error) {
	var response api.Category
	err := c.do(ctx, http.MethodPost, api.CategoriesPath, nil, request, &response)
	return response, err
}
func (c *Client) ListCategories(ctx context.Context, input api.ListCategoriesQuery) ([]api.Category, error) {
	query := url.Values{}
	if input.IncludeInactive {
		query.Set("includeInactive", "true")
	}
	var response []api.Category
	err := c.do(ctx, http.MethodGet, api.CategoriesPath, query, nil, &response)
	return response, err
}
func (c *Client) GetCategory(ctx context.Context, code string) (api.Category, error) {
	var response api.Category
	err := c.do(ctx, http.MethodGet, strings.Replace(api.CategoryPath, "{code}", url.PathEscape(code), 1), nil, nil, &response)
	return response, err
}
func (c *Client) UpdateCategory(ctx context.Context, code string, request api.UpdateCategoryRequest) (api.Category, error) {
	var response api.Category
	err := c.do(ctx, http.MethodPatch, strings.Replace(api.CategoryPath, "{code}", url.PathEscape(code), 1), nil, request, &response)
	return response, err
}
func (c *Client) DeactivateCategory(ctx context.Context, code string) (api.Category, error) {
	var response api.Category
	err := c.do(ctx, http.MethodDelete, strings.Replace(api.CategoryPath, "{code}", url.PathEscape(code), 1), nil, nil, &response)
	return response, err
}
