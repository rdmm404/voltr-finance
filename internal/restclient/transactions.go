package restclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"rdmm404/voltr-finance/internal/api"
)

func (c *Client) CreateTransaction(ctx context.Context, request api.CreateTransactionRequest) (api.Transaction, error) {
	var response api.Transaction
	err := c.do(ctx, http.MethodPost, api.TransactionsPath, nil, request, &response)
	return response, err
}

func (c *Client) CreateTransactions(ctx context.Context, request api.BulkCreateTransactionsRequest) (api.BulkResult, error) {
	var response api.BulkResult
	err := c.do(ctx, http.MethodPost, api.TransactionsBulkPath, nil, request, &response)
	return response, err
}

func (c *Client) GetTransaction(ctx context.Context, id int64, includeDeleted bool) (api.Transaction, error) {
	query := url.Values{}
	if includeDeleted {
		query.Set("includeDeleted", "true")
	}
	var response api.Transaction
	err := c.do(ctx, http.MethodGet, replace(api.TransactionPath, "{id}", id), query, nil, &response)
	return response, err
}

func (c *Client) ListTransactions(ctx context.Context, input api.ListTransactionsQuery) ([]api.Transaction, error) {
	query := url.Values{}
	for _, id := range input.IDs {
		query.Add("ids", strconv.FormatInt(id, 10))
	}
	setInt64(query, "authorId", input.AuthorID)
	setInt64(query, "householdId", input.HouseholdID)
	setTime(query, "fromDate", input.FromDate)
	setTime(query, "toDate", input.ToDate)
	if input.Search != nil {
		query.Set("search", *input.Search)
	}
	if input.Sort != "" {
		query.Set("sort", input.Sort)
	}
	if input.SortOrder != "" {
		query.Set("sortOrder", input.SortOrder)
	}
	if input.Limit != 0 {
		query.Set("limit", strconv.FormatInt(int64(input.Limit), 10))
	}
	if input.Offset != 0 {
		query.Set("offset", strconv.FormatInt(int64(input.Offset), 10))
	}
	if input.IncludeDeleted {
		query.Set("includeDeleted", "true")
	}
	if input.OnlyDeleted {
		query.Set("onlyDeleted", "true")
	}
	var response []api.Transaction
	err := c.do(ctx, http.MethodGet, api.TransactionsPath, query, nil, &response)
	return response, err
}

func (c *Client) UpdateTransaction(ctx context.Context, id int64, request api.UpdateTransactionRequest) (api.Transaction, error) {
	var response api.Transaction
	err := c.do(ctx, http.MethodPatch, replace(api.TransactionPath, "{id}", id), nil, request, &response)
	return response, err
}

func (c *Client) UpdateTransactions(ctx context.Context, request api.BulkUpdateTransactionsRequest) (api.BulkResult, error) {
	var response api.BulkResult
	err := c.do(ctx, http.MethodPatch, api.TransactionsBulkPath, nil, request, &response)
	return response, err
}

func (c *Client) DeleteTransactions(ctx context.Context, request api.DeleteTransactionsRequest) (api.BulkResult, error) {
	var response api.BulkResult
	err := c.do(ctx, http.MethodDelete, api.TransactionsPath, nil, request, &response)
	return response, err
}

func (c *Client) RestoreTransactions(ctx context.Context, request api.RestoreTransactionsRequest) (api.BulkResult, error) {
	var response api.BulkResult
	err := c.do(ctx, http.MethodPost, api.TransactionsRestorePath, nil, request, &response)
	return response, err
}

func replace(pattern, placeholder string, id int64) string {
	return strings.Replace(pattern, placeholder, strconv.FormatInt(id, 10), 1)
}
func setInt64(query url.Values, name string, value *int64) {
	if value != nil {
		query.Set(name, strconv.FormatInt(*value, 10))
	}
}
func setTime(query url.Values, name string, value *time.Time) {
	if value != nil {
		query.Set(name, value.Format(time.RFC3339))
	}
}
