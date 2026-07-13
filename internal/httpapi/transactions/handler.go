package transactions

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"rdmm404/voltr-finance/internal/api"
	apperrors "rdmm404/voltr-finance/internal/app/errors"
	apppatch "rdmm404/voltr-finance/internal/app/patch"
	apptransactions "rdmm404/voltr-finance/internal/app/transactions"
	"rdmm404/voltr-finance/internal/httpapi"
)

type Service interface {
	Create(context.Context, apptransactions.CreateInput) (apptransactions.Transaction, error)
	CreateBatch(context.Context, []apptransactions.CreateInput) apptransactions.BulkResult
	Get(context.Context, int64, bool) (apptransactions.Transaction, error)
	GetMany(context.Context, []int64, bool) ([]apptransactions.Transaction, error)
	List(context.Context, apptransactions.ListFilter) ([]apptransactions.Transaction, error)
	Update(context.Context, apptransactions.UpdateInput) (apptransactions.Transaction, error)
	UpdateBatch(context.Context, []apptransactions.UpdateInput) apptransactions.BulkResult
	DeleteBatch(context.Context, []int64, int64, *string) apptransactions.BulkResult
	RestoreBatch(context.Context, []int64, int64) apptransactions.BulkResult
}

type Handler struct {
	service Service
	support *httpapi.HandlerSupport
}

func New(service Service, support ...*httpapi.HandlerSupport) *Handler {
	return &Handler{service: service, support: httpapi.HandlerSupportOrDefault(support...)}
}

func (h *Handler) Register(router *httpapi.Router) {
	router.HandleFunc(http.MethodPost, api.TransactionsPath, h.create)
	router.HandleFunc(http.MethodGet, api.TransactionsPath, h.list)
	router.HandleFunc(http.MethodDelete, api.TransactionsPath, h.deleteBatch)
	router.HandleFunc(http.MethodPost, api.TransactionsBulkPath, h.createBatch)
	router.HandleFunc(http.MethodPatch, api.TransactionsBulkPath, h.updateBatch)
	router.HandleFunc(http.MethodPost, api.TransactionsRestorePath, h.restoreBatch)
	router.HandleFunc(http.MethodGet, api.TransactionPath, h.get)
	router.HandleFunc(http.MethodPatch, api.TransactionPath, h.update)
}

func (h *Handler) create(w http.ResponseWriter, request *http.Request) {
	var body api.CreateTransactionRequest
	if !h.support.Decode(w, request, &body) {
		return
	}
	item, err := h.service.Create(request.Context(), createInput(body))
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusCreated, transaction(item))
}

func (h *Handler) createBatch(w http.ResponseWriter, request *http.Request) {
	var body api.BulkCreateTransactionsRequest
	if !h.support.Decode(w, request, &body) {
		return
	}
	inputs := make([]apptransactions.CreateInput, 0, len(body.Transactions))
	for _, item := range body.Transactions {
		inputs = append(inputs, createInput(item))
	}
	httpapi.WriteJSON(w, http.StatusOK, bulkResult(h.service.CreateBatch(request.Context(), inputs)))
}

func (h *Handler) get(w http.ResponseWriter, request *http.Request) {
	id, err := httpapi.ParsePathID(request, "id")
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	includeDeleted, err := httpapi.QueryBool(request, "includeDeleted", false)
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	item, err := h.service.Get(request.Context(), id, includeDeleted)
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, transaction(item))
}

func (h *Handler) list(w http.ResponseWriter, request *http.Request) {
	filter, ids, err := listInput(request)
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	var items []apptransactions.Transaction
	if len(ids) > 0 {
		items, err = h.service.GetMany(request.Context(), ids, filter.IncludeDeleted)
	} else {
		items, err = h.service.List(request.Context(), filter)
	}
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	response := make([]api.Transaction, 0, len(items))
	for _, item := range items {
		response = append(response, transaction(item))
	}
	httpapi.WriteJSON(w, http.StatusOK, response)
}

func (h *Handler) update(w http.ResponseWriter, request *http.Request) {
	id, err := httpapi.ParsePathID(request, "id")
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	var body api.UpdateTransactionRequest
	if !h.support.Decode(w, request, &body) {
		return
	}
	input, err := updateInput(id, body)
	if err != nil {
		httpapi.WriteValidationError(w, err.Error())
		return
	}
	item, err := h.service.Update(request.Context(), input)
	if err != nil {
		h.support.Fail(w, request, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, transaction(item))
}

func (h *Handler) updateBatch(w http.ResponseWriter, request *http.Request) {
	var body api.BulkUpdateTransactionsRequest
	if !h.support.Decode(w, request, &body) {
		return
	}
	inputs := make([]apptransactions.UpdateInput, 0, len(body.Transactions))
	for _, item := range body.Transactions {
		input, err := updateInput(item.ID, item.UpdateTransactionRequest)
		if err != nil {
			httpapi.WriteValidationError(w, err.Error())
			return
		}
		inputs = append(inputs, input)
	}
	httpapi.WriteJSON(w, http.StatusOK, bulkResult(h.service.UpdateBatch(request.Context(), inputs)))
}

func (h *Handler) deleteBatch(w http.ResponseWriter, request *http.Request) {
	var body api.DeleteTransactionsRequest
	if !h.support.Decode(w, request, &body) {
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, bulkResult(h.service.DeleteBatch(request.Context(), body.IDs, body.DeletedByUserID, body.Reason)))
}

func (h *Handler) restoreBatch(w http.ResponseWriter, request *http.Request) {
	var body api.RestoreTransactionsRequest
	if !h.support.Decode(w, request, &body) {
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, bulkResult(h.service.RestoreBatch(request.Context(), body.IDs, body.RestoredByUserID)))
}

func createInput(body api.CreateTransactionRequest) apptransactions.CreateInput {
	return apptransactions.CreateInput{Amount: body.Amount, TransactionDate: body.TransactionDate, Description: body.Description, Notes: body.Notes, CategoryID: body.CategoryID, CategoryCode: body.CategoryCode, HouseholdID: body.HouseholdID, Author: identity(body.Author)}
}
func updateInput(id int64, body api.UpdateTransactionRequest) (apptransactions.UpdateInput, error) {
	description, err := httpapi.NullablePatch(body.Description, body.ClearDescription, "description")
	if err != nil {
		return apptransactions.UpdateInput{}, err
	}
	notes, err := httpapi.NullablePatch(body.Notes, body.ClearNotes, "notes")
	if err != nil {
		return apptransactions.UpdateInput{}, err
	}
	householdID, err := httpapi.NullablePatch(body.HouseholdID, body.ClearHouseholdID, "householdId")
	if err != nil {
		return apptransactions.UpdateInput{}, err
	}
	if body.ClearCategoryID && (body.CategoryID != nil || body.CategoryCode != nil) {
		return apptransactions.UpdateInput{}, fmt.Errorf("categoryId/categoryCode and clearCategoryId are mutually exclusive")
	}
	category := apppatch.Unchanged[apptransactions.CategorySelector]()
	if body.ClearCategoryID {
		category = apppatch.Clear[apptransactions.CategorySelector]()
	} else if body.CategoryID != nil || body.CategoryCode != nil {
		category = apppatch.Set(apptransactions.CategorySelector{ID: body.CategoryID, Code: body.CategoryCode})
	}
	input := apptransactions.UpdateInput{ID: id, Amount: body.Amount, TransactionDate: body.TransactionDate, Description: description, Notes: notes, Category: category, HouseholdID: householdID}
	if body.Author != nil {
		value := identity(*body.Author)
		input.Author = &value
	}
	return input, nil
}
func identity(value api.IdentitySelector) apptransactions.IdentitySelector {
	return apptransactions.IdentitySelector{UserID: value.UserID, DiscordID: value.DiscordID, TelegramID: value.TelegramID, PhoneNumber: value.PhoneNumber, WhatsAppID: value.WhatsAppID}
}

func transaction(item apptransactions.Transaction) api.Transaction {
	result := api.Transaction{ID: item.ID, Amount: item.Amount, TransactionDate: item.TransactionDate, AuthorID: item.AuthorID, AuthorName: item.AuthorName, HouseholdID: item.HouseholdID, HouseholdName: item.HouseholdName, Description: item.Description, Notes: item.Notes, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, DeletedAt: item.DeletedAt, DeleteReason: item.DeleteReason}
	if item.Category != nil {
		result.Category = &api.CategoryRef{ID: item.Category.ID, Code: item.Category.Code, Name: item.Category.Name}
	}
	return result
}
func bulkResult(result apptransactions.BulkResult) api.BulkResult {
	response := api.BulkResult{Succeeded: make([]api.BulkSucceeded, 0, len(result.Succeeded)), Failed: make([]api.BulkFailed, 0, len(result.Failed))}
	for _, item := range result.Succeeded {
		response.Succeeded = append(response.Succeeded, api.BulkSucceeded{Index: item.Index, ID: item.ID})
	}
	for _, item := range result.Failed {
		response.Failed = append(response.Failed, api.BulkFailed{Index: item.Index, ID: item.ID, Error: api.Error{Code: string(apperrors.CodeOf(item.Error)), Message: apperrors.MessageOf(item.Error)}})
	}
	return response
}

func listInput(request *http.Request) (apptransactions.ListFilter, []int64, error) {
	query := request.URL.Query()
	ids, err := parseIDs(query["ids"])
	if err != nil {
		return apptransactions.ListFilter{}, nil, err
	}
	authorID, err := httpapi.QueryInt64(request, "authorId")
	if err != nil {
		return apptransactions.ListFilter{}, nil, err
	}
	householdID, err := httpapi.QueryInt64(request, "householdId")
	if err != nil {
		return apptransactions.ListFilter{}, nil, err
	}
	from, err := parseTime(query.Get("fromDate"), "fromDate")
	if err != nil {
		return apptransactions.ListFilter{}, nil, err
	}
	to, err := parseTime(query.Get("toDate"), "toDate")
	if err != nil {
		return apptransactions.ListFilter{}, nil, err
	}
	limit, err := httpapi.QueryInt(request, "limit", 100)
	if err != nil || limit < 1 || limit > 1000 {
		return apptransactions.ListFilter{}, nil, fmt.Errorf("limit must be between 1 and 1000")
	}
	offset, err := httpapi.QueryInt(request, "offset", 0)
	if err != nil || offset < 0 {
		return apptransactions.ListFilter{}, nil, fmt.Errorf("offset must be a non-negative integer")
	}
	includeDeleted, err := httpapi.QueryBool(request, "includeDeleted", false)
	if err != nil {
		return apptransactions.ListFilter{}, nil, err
	}
	onlyDeleted, err := httpapi.QueryBool(request, "onlyDeleted", false)
	if err != nil {
		return apptransactions.ListFilter{}, nil, err
	}
	sortBy, order := query.Get("sort"), query.Get("sortOrder")
	if sortBy == "" {
		sortBy = "transaction_date"
	}
	if order == "" {
		order = "desc"
	}
	if !oneOf(sortBy, "transaction_date", "created_at", "amount", "id") {
		return apptransactions.ListFilter{}, nil, fmt.Errorf("unsupported sort field")
	}
	if !oneOf(order, "asc", "desc") {
		return apptransactions.ListFilter{}, nil, fmt.Errorf("sortOrder must be asc or desc")
	}
	return apptransactions.ListFilter{AuthorID: authorID, HouseholdID: householdID, FromDate: from, ToDate: to, Search: httpapi.QueryString(request, "search"), Sort: sortBy, SortOrder: order, Limit: int32(limit), Offset: int32(offset), IncludeDeleted: includeDeleted, OnlyDeleted: onlyDeleted}, ids, nil
}
func parseIDs(values []string) ([]int64, error) {
	var ids []int64
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if strings.TrimSpace(part) == "" {
				continue
			}
			id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
			if err != nil || id < 1 {
				return nil, fmt.Errorf("ids must contain positive integers")
			}
			ids = append(ids, id)
		}
	}
	return ids, nil
}
func parseTime(value, name string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("%s must use RFC3339 format", name)
	}
	return &parsed, nil
}
func oneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}
