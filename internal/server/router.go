// Package server is the HTTP composition root.
package server

import (
	"log/slog"
	"net/http"

	appbudgets "rdmm404/voltr-finance/internal/app/budgets"
	appcategories "rdmm404/voltr-finance/internal/app/categories"
	apphouseholds "rdmm404/voltr-finance/internal/app/households"
	apptransactions "rdmm404/voltr-finance/internal/app/transactions"
	appusers "rdmm404/voltr-finance/internal/app/users"
	"rdmm404/voltr-finance/internal/httpapi"
	budgethttp "rdmm404/voltr-finance/internal/httpapi/budgets"
	categoryhttp "rdmm404/voltr-finance/internal/httpapi/categories"
	householdhttp "rdmm404/voltr-finance/internal/httpapi/households"
	transactionhttp "rdmm404/voltr-finance/internal/httpapi/transactions"
	userhttp "rdmm404/voltr-finance/internal/httpapi/users"
)

// New wires feature handlers into the shared authenticated HTTP server.
func New(
	config httpapi.Config,
	transactionService *apptransactions.Service,
	userService *appusers.Service,
	householdService *apphouseholds.Service,
	categoryService *appcategories.Service,
	budgetService *appbudgets.Service,
) (*http.Server, error) {
	support := httpapi.NewHandlerSupport(slog.Default())
	return httpapi.NewServer(config, func(router *httpapi.Router) {
		transactionhttp.New(transactionService, support).Register(router)
		userhttp.New(userService, support).Register(router)
		householdhttp.New(householdService, support).Register(router)
		categoryhttp.New(categoryService, support).Register(router)
		budgethttp.New(budgetService, support).Register(router)
	})
}
