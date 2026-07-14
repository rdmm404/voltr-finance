package restclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"rdmm404/voltr-finance/internal/api"
)

func (c *Client) GetMonthlyBudget(ctx context.Context, input api.MonthlyBudgetQuery) (api.Budget, error) {
	var response api.Budget
	err := c.do(ctx, http.MethodGet, api.MonthlyBudgetsPath, monthlyQuery(input), nil, &response)
	return response, err
}

func (c *Client) EnsureMonthlyBudget(ctx context.Context, input api.EnsureMonthlyBudgetRequest) (api.Budget, error) {
	var response api.Budget
	err := c.do(ctx, http.MethodPost, api.MonthlyBudgetsPath, nil, input, &response)
	return response, err
}

func (c *Client) CreateBudgetLine(ctx context.Context, budgetID int64, request api.CreateBudgetLineRequest) (api.BudgetLine, error) {
	var response api.BudgetLine
	err := c.do(ctx, http.MethodPost, replace(api.BudgetLinesPath, "{id}", budgetID), nil, request, &response)
	return response, err
}

func (c *Client) UpdateBudgetLine(ctx context.Context, lineID int64, request api.UpdateBudgetLineRequest) (api.BudgetLine, error) {
	var response api.BudgetLine
	err := c.do(ctx, http.MethodPatch, replace(api.BudgetLinePath, "{id}", lineID), nil, request, &response)
	return response, err
}

func (c *Client) DeleteBudgetLine(ctx context.Context, lineID int64) error {
	return c.do(ctx, http.MethodDelete, replace(api.BudgetLinePath, "{id}", lineID), nil, nil, nil)
}

func (c *Client) GetBudgetReport(ctx context.Context, budgetID int64) (api.BudgetReport, error) {
	var response api.BudgetReport
	err := c.do(ctx, http.MethodGet, replace(api.BudgetReportPath, "{id}", budgetID), nil, nil, &response)
	return response, err
}

func monthlyQuery(input api.MonthlyBudgetQuery) url.Values {
	query := url.Values{"year": []string{strconv.Itoa(input.Year)}, "month": []string{strconv.Itoa(input.Month)}}
	setInt64(query, "householdId", input.HouseholdID)
	setInt64(query, "userId", input.UserID)
	return query
}
