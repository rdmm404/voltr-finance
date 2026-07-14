package budgets

import (
	"context"
	"net/http"

	"rdmm404/voltr-finance/internal/api"
	appbudgets "rdmm404/voltr-finance/internal/app/budgets"
	"rdmm404/voltr-finance/internal/httpapi"
)

type Service interface {
	GetMonthly(context.Context, appbudgets.MonthlyInput) (appbudgets.Budget, error)
	EnsureMonthly(context.Context, appbudgets.MonthlyInput) (appbudgets.EnsureResult, error)
	CreateLine(context.Context, appbudgets.CreateLineInput) (appbudgets.Line, error)
	UpdateLine(context.Context, appbudgets.UpdateLineInput) (appbudgets.Line, error)
	DeleteLine(context.Context, int64) error
	Report(context.Context, int64) (appbudgets.Report, error)
}

type Handler struct {
	service Service
	support *httpapi.HandlerSupport
}

func New(service Service, support ...*httpapi.HandlerSupport) *Handler {
	return &Handler{service: service, support: httpapi.HandlerSupportOrDefault(support...)}
}

func (h *Handler) Register(router *httpapi.Router) {
	router.HandleFunc(http.MethodGet, api.MonthlyBudgetsPath, h.getMonthly)
	router.HandleFunc(http.MethodPost, api.MonthlyBudgetsPath, h.ensureMonthly)
	router.HandleFunc(http.MethodGet, api.BudgetReportPath, h.report)
	router.HandleFunc(http.MethodPost, api.BudgetLinesPath, h.createLine)
	router.HandleFunc(http.MethodPatch, api.BudgetLinePath, h.updateLine)
	router.HandleFunc(http.MethodDelete, api.BudgetLinePath, h.deleteLine)
}

func (h *Handler) getMonthly(w http.ResponseWriter, request *http.Request) {
	query, err := monthlyQuery(request)
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	item, err := h.service.GetMonthly(request.Context(), monthlyQueryInput(query))
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, budget(item))
}

func (h *Handler) ensureMonthly(w http.ResponseWriter, request *http.Request) {
	var body api.EnsureMonthlyBudgetRequest
	if !h.support.Decode(w, request, &body) {
		return
	}
	result, err := h.service.EnsureMonthly(request.Context(), monthlyInput(body))
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	httpapi.WriteJSON(w, status, budget(result.Budget))
}

func (h *Handler) createLine(w http.ResponseWriter, request *http.Request) {
	budgetID, err := httpapi.ParsePathID(request, "id")
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	var body api.CreateBudgetLineRequest
	if !h.support.Decode(w, request, &body) {
		return
	}
	item, err := h.service.CreateLine(request.Context(), appbudgets.CreateLineInput{
		BudgetID: budgetID, Name: body.Name, AllocationAmount: body.AllocationAmount,
		CategoryIDs: body.CategoryIDs, CategoryCodes: body.CategoryCodes, SortOrder: body.SortOrder,
	})
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, line(item))
}

func (h *Handler) updateLine(w http.ResponseWriter, request *http.Request) {
	lineID, err := httpapi.ParsePathID(request, "id")
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	var body api.UpdateBudgetLineRequest
	if !h.support.Decode(w, request, &body) {
		return
	}
	item, err := h.service.UpdateLine(request.Context(), appbudgets.UpdateLineInput{
		LineID: lineID, Name: body.Name, AllocationAmount: body.AllocationAmount,
		CategoryIDs: body.CategoryIDs, CategoryCodes: body.CategoryCodes, SortOrder: body.SortOrder,
	})
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, line(item))
}

func (h *Handler) deleteLine(w http.ResponseWriter, request *http.Request) {
	lineID, err := httpapi.ParsePathID(request, "id")
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	if err := h.service.DeleteLine(request.Context(), lineID); err != nil {
		h.support.Fail(w, request, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) report(w http.ResponseWriter, request *http.Request) {
	budgetID, err := httpapi.ParsePathID(request, "id")
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	item, err := h.service.Report(request.Context(), budgetID)
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, report(item))
}

func monthlyQuery(request *http.Request) (api.MonthlyBudgetQuery, error) {
	householdID, err := httpapi.QueryInt64(request, "householdId")
	if err != nil {
		return api.MonthlyBudgetQuery{}, err
	}
	userID, err := httpapi.QueryInt64(request, "userId")
	if err != nil {
		return api.MonthlyBudgetQuery{}, err
	}
	year, err := httpapi.QueryInt(request, "year", 0)
	if err != nil {
		return api.MonthlyBudgetQuery{}, err
	}
	month, err := httpapi.QueryInt(request, "month", 0)
	if err != nil {
		return api.MonthlyBudgetQuery{}, err
	}
	return api.MonthlyBudgetQuery{HouseholdID: householdID, UserID: userID, Year: year, Month: month}, nil
}

func monthlyQueryInput(value api.MonthlyBudgetQuery) appbudgets.MonthlyInput {
	return appbudgets.MonthlyInput{Owner: appbudgets.Owner{HouseholdID: value.HouseholdID, UserID: value.UserID}, Year: value.Year, Month: value.Month}
}

func monthlyInput(value api.EnsureMonthlyBudgetRequest) appbudgets.MonthlyInput {
	return appbudgets.MonthlyInput{Owner: appbudgets.Owner{HouseholdID: value.HouseholdID, UserID: value.UserID}, Year: value.Year, Month: value.Month}
}

func budget(item appbudgets.Budget) api.Budget {
	result := api.Budget{
		ID: item.ID, HouseholdID: item.Owner.HouseholdID, UserID: item.Owner.UserID,
		PeriodStart: item.PeriodStart, PeriodEnd: item.PeriodEnd, SourceBudgetID: item.SourceBudgetID,
		Lines: make([]api.BudgetLine, 0, len(item.Lines)),
	}
	for _, value := range item.Lines {
		result.Lines = append(result.Lines, line(value))
	}
	return result
}

func line(item appbudgets.Line) api.BudgetLine {
	result := api.BudgetLine{
		ID: item.ID, BudgetID: item.BudgetID, Name: item.Name, AllocationAmount: item.AllocationAmount,
		SortOrder: item.SortOrder, Categories: make([]api.CategoryRef, 0, len(item.Categories)),
	}
	for _, value := range item.Categories {
		result.Categories = append(result.Categories, api.CategoryRef{ID: value.ID, Code: value.Code, Name: value.Name})
	}
	return result
}

func report(item appbudgets.Report) api.BudgetReport {
	result := api.BudgetReport{
		Budget: api.BudgetSummary{
			ID: item.Budget.ID, HouseholdID: item.Budget.Owner.HouseholdID, UserID: item.Budget.Owner.UserID,
			PeriodStart: item.Budget.PeriodStart, PeriodEnd: item.Budget.PeriodEnd, SourceBudgetID: item.Budget.SourceBudgetID,
		},
		Lines:                make([]api.BudgetReportLine, 0, len(item.Lines)),
		UnmappedTransactions: make([]api.BudgetUnmappedTransaction, 0, len(item.UnmappedTransactions)),
		Totals: api.BudgetReportTotals{
			AllocationAmount: item.Totals.AllocationAmount, ActualAmount: item.Totals.ActualAmount,
			RemainingAmount: item.Totals.RemainingAmount, UnmappedActualAmount: item.Totals.UnmappedActualAmount,
			UncategorizedActualAmount: item.Totals.UncategorizedActualAmount,
		},
	}
	for _, value := range item.Lines {
		mapped := line(value.Line)
		result.Lines = append(result.Lines, api.BudgetReportLine{
			ID: mapped.ID, BudgetID: mapped.BudgetID, Name: mapped.Name, AllocationAmount: mapped.AllocationAmount,
			ActualAmount: value.ActualAmount, RemainingAmount: value.RemainingAmount,
			SortOrder: mapped.SortOrder, Categories: mapped.Categories,
		})
	}
	for _, value := range item.UnmappedTransactions {
		mapped := api.BudgetUnmappedTransaction{ID: value.ID, TransactionDate: value.TransactionDate, Description: value.Description, Amount: value.Amount}
		if value.Category != nil {
			mapped.Category = &api.CategoryRef{ID: value.Category.ID, Code: value.Category.Code, Name: value.Category.Name}
		}
		result.UnmappedTransactions = append(result.UnmappedTransactions, mapped)
	}
	return result
}
