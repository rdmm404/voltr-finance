package restclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"rdmm404/voltr-finance/internal/api"
)

func TestTransactionMethodsAndQueries(t *testing.T) {
	from := time.Date(2026, 7, 1, 2, 3, 4, 0, time.UTC)
	authorID := int64(8)
	search := "food"
	tests := []struct {
		name, method, path string
		call               func(*Client) error
	}{
		{"create", http.MethodPost, "/v1/transactions", func(c *Client) error {
			_, err := c.CreateTransaction(context.Background(), api.CreateTransactionRequest{})
			return err
		}},
		{"bulk create", http.MethodPost, "/v1/transactions/bulk", func(c *Client) error {
			_, err := c.CreateTransactions(context.Background(), api.BulkCreateTransactionsRequest{})
			return err
		}},
		{"get", http.MethodGet, "/v1/transactions/4?includeDeleted=true", func(c *Client) error { _, err := c.GetTransaction(context.Background(), 4, true); return err }},
		{"list", http.MethodGet, "/v1/transactions?authorId=8&fromDate=2026-07-01T02%3A03%3A04Z&ids=1&ids=2&includeDeleted=true&limit=25&offset=3&search=food&sort=amount&sortOrder=asc", func(c *Client) error {
			_, err := c.ListTransactions(context.Background(), api.ListTransactionsQuery{IDs: []int64{1, 2}, AuthorID: &authorID, FromDate: &from, Search: &search, Sort: "amount", SortOrder: "asc", Limit: 25, Offset: 3, IncludeDeleted: true})
			return err
		}},
		{"update", http.MethodPatch, "/v1/transactions/4", func(c *Client) error {
			_, err := c.UpdateTransaction(context.Background(), 4, api.UpdateTransactionRequest{})
			return err
		}},
		{"bulk update", http.MethodPatch, "/v1/transactions/bulk", func(c *Client) error {
			_, err := c.UpdateTransactions(context.Background(), api.BulkUpdateTransactionsRequest{})
			return err
		}},
		{"delete", http.MethodDelete, "/v1/transactions", func(c *Client) error {
			_, err := c.DeleteTransactions(context.Background(), api.DeleteTransactionsRequest{})
			return err
		}},
		{"restore", http.MethodPost, "/v1/transactions/restore", func(c *Client) error {
			_, err := c.RestoreTransactions(context.Background(), api.RestoreTransactionsRequest{})
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.Method != test.method || request.URL.RequestURI() != test.path {
					t.Errorf("request = %s %s", request.Method, request.URL.RequestURI())
				}
				if test.name == "bulk create" || test.name == "bulk update" || test.name == "delete" || test.name == "restore" {
					_, _ = w.Write([]byte(`{"succeeded":[],"failed":[]}`))
					return
				}
				if test.name == "list" {
					_, _ = w.Write([]byte(`[]`))
					return
				}
				_, _ = w.Write([]byte(`{"id":` + strconv.Itoa(4) + `}`))
			}))
			defer server.Close()
			client, _ := New(Config{BaseURL: server.URL, APIKey: "key"})
			if err := test.call(client); err != nil {
				t.Fatal(err)
			}
		})
	}
}
