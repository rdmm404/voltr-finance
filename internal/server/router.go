// Package server is the HTTP composition root.
package server

import (
	"log/slog"
	"net/http"
	"strings"

	"rdmm404/voltr-finance/internal/httpapi"
	budgethttp "rdmm404/voltr-finance/internal/httpapi/budgets"
	categoryhttp "rdmm404/voltr-finance/internal/httpapi/categories"
	householdhttp "rdmm404/voltr-finance/internal/httpapi/households"
	transactionhttp "rdmm404/voltr-finance/internal/httpapi/transactions"
	userhttp "rdmm404/voltr-finance/internal/httpapi/users"
	"rdmm404/voltr-finance/internal/webui"
)

// New wires feature handlers into the shared authenticated HTTP server.
func New(
	config httpapi.Config,
	uiConfig webui.Config,
	transactionService transactionhttp.Service,
	userService userhttp.Service,
	householdService householdhttp.Service,
	categoryService categoryhttp.Service,
	budgetService interface {
		budgethttp.Service
		webui.BudgetReader
	},
) (*http.Server, error) {
	support := httpapi.NewHandlerSupport(slog.Default())
	apiServer, err := httpapi.NewServer(config, func(router *httpapi.Router) {
		transactionhttp.New(transactionService, support).Register(router)
		userhttp.New(userService, support).Register(router)
		householdhttp.New(householdService, support).Register(router)
		categoryhttp.New(categoryService, support).Register(router)
		budgethttp.New(budgetService, support).Register(router)
	})
	if err != nil {
		return nil, err
	}
	ui, err := webui.New(uiConfig, webui.Services{Budgets: budgetService, Users: userService, Households: householdService}, slog.Default())
	if err != nil {
		return nil, err
	}
	uiMux := http.NewServeMux()
	ui.Register(uiMux)
	apiHandler := apiServer.Handler
	apiServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/live", r.URL.Path == "/v1", strings.HasPrefix(r.URL.Path, "/v1/"):
			apiHandler.ServeHTTP(w, r)
		default:
			uiMux.ServeHTTP(w, r)
		}
	})
	return apiServer, nil
}
