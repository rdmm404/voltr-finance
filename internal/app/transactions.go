package app

import (
	"context"
	"sort"
	"time"

	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/transaction"
	"rdmm404/voltr-finance/internal/utils"

	"github.com/jackc/pgx/v5/pgtype"
)

type TransactionDTO struct {
	ID              int64           `json:"id"`
	Amount          float32         `json:"amount"`
	TransactionDate time.Time       `json:"transactionDate"`
	AuthorID        int64           `json:"authorId"`
	AuthorName      string          `json:"authorName,omitempty"`
	HouseholdID     *int64          `json:"householdId,omitempty"`
	HouseholdName   *string         `json:"householdName,omitempty"`
	Category        *CategoryRefDTO `json:"category,omitempty"`
	Description     *string         `json:"description,omitempty"`
	Notes           *string         `json:"notes,omitempty"`
	CreatedAt       *time.Time      `json:"createdAt,omitempty"`
	UpdatedAt       *time.Time      `json:"updatedAt,omitempty"`
	DeletedAt       *time.Time      `json:"deletedAt,omitempty"`
	DeleteReason    *string         `json:"deleteReason,omitempty"`
}

type CreateTransactionRequest struct {
	Amount          float32          `json:"amount"`
	TransactionDate time.Time        `json:"transactionDate"`
	Description     *string          `json:"description,omitempty"`
	Notes           *string          `json:"notes,omitempty"`
	CategoryID      *int64           `json:"categoryId,omitempty"`
	CategoryCode    *string          `json:"categoryCode,omitempty"`
	HouseholdID     *int64           `json:"householdId,omitempty"`
	Author          IdentitySelector `json:"author"`
}

type BulkCreateTransactionsRequest struct {
	Transactions []CreateTransactionRequest `json:"transactions"`
}

type UpdateTransactionRequest struct {
	ID int64 `json:"id"`

	Amount          *float32          `json:"amount,omitempty"`
	TransactionDate *time.Time        `json:"transactionDate,omitempty"`
	Description     *string           `json:"description,omitempty"`
	Notes           *string           `json:"notes,omitempty"`
	CategoryID      *int64            `json:"categoryId,omitempty"`
	CategoryCode    *string           `json:"categoryCode,omitempty"`
	HouseholdID     *int64            `json:"householdId,omitempty"`
	Author          *IdentitySelector `json:"author,omitempty"`

	ClearDescription bool `json:"clearDescription,omitempty"`
	ClearNotes       bool `json:"clearNotes,omitempty"`
	ClearCategoryID  bool `json:"clearCategoryId,omitempty"`
	ClearHouseholdID bool `json:"clearHouseholdId,omitempty"`
}

type BulkUpdateTransactionsRequest struct {
	Transactions []UpdateTransactionRequest `json:"transactions"`
}

type ListTransactionsRequest struct {
	AuthorID       *int64     `json:"authorId,omitempty"`
	HouseholdID    *int64     `json:"householdId,omitempty"`
	FromDate       *time.Time `json:"fromDate,omitempty"`
	ToDate         *time.Time `json:"toDate,omitempty"`
	Search         *string    `json:"search,omitempty"`
	Sort           string     `json:"sort,omitempty"`
	SortOrder      string     `json:"sortOrder,omitempty"`
	Limit          int32      `json:"limit,omitempty"`
	Offset         int32      `json:"offset,omitempty"`
	IncludeDeleted bool       `json:"includeDeleted,omitempty"`
	OnlyDeleted    bool       `json:"onlyDeleted,omitempty"`
}

type DeleteTransactionsRequest struct {
	IDs             []int64 `json:"ids"`
	DeletedByUserID int64   `json:"deletedByUserId"`
	Reason          *string `json:"reason,omitempty"`
}

type RestoreTransactionsRequest struct {
	IDs              []int64 `json:"ids"`
	RestoredByUserID int64   `json:"restoredByUserId"`
}

type WriteError struct {
	Index   int       `json:"index"`
	ID      int64     `json:"id,omitempty"`
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type WriteResult struct {
	CreatedIDs  []int64      `json:"createdIds"`
	UpdatedIDs  []int64      `json:"updatedIds"`
	DeletedIDs  []int64      `json:"deletedIds"`
	RestoredIDs []int64      `json:"restoredIds"`
	Errors      []WriteError `json:"errors"`
}

func (s *Service) CreateTransaction(ctx context.Context, req CreateTransactionRequest) WriteResult {
	return s.CreateTransactions(ctx, BulkCreateTransactionsRequest{Transactions: []CreateTransactionRequest{req}})
}

func (s *Service) CreateTransactions(ctx context.Context, req BulkCreateTransactionsRequest) WriteResult {
	result := WriteResult{}
	params := make([]sqlc.CreateTransactionParams, 0, len(req.Transactions))
	indexMap := make([]int, 0, len(req.Transactions))

	for i, item := range req.Transactions {
		param, err := s.createParams(ctx, item)
		if err != nil {
			result.Errors = append(result.Errors, writeError(i, 0, err))
			continue
		}
		params = append(params, param)
		indexMap = append(indexMap, i)
	}

	txResult := s.transactions.SaveTransactions(ctx, params)
	result.CreatedIDs = sortedTransactionIDs(txResult.Success)
	result.Errors = append(result.Errors, mapTransactionErrors(txResult.Errors, indexMap)...)
	sortWriteErrors(result.Errors)
	return result
}

func (s *Service) UpdateTransaction(ctx context.Context, req UpdateTransactionRequest) WriteResult {
	return s.UpdateTransactions(ctx, BulkUpdateTransactionsRequest{Transactions: []UpdateTransactionRequest{req}})
}

func (s *Service) UpdateTransactions(ctx context.Context, req BulkUpdateTransactionsRequest) WriteResult {
	result := WriteResult{}
	params := make([]transaction.UpdateTransactionById, 0, len(req.Transactions))
	indexMap := make([]int, 0, len(req.Transactions))

	for i, item := range req.Transactions {
		param, err := s.updateParams(ctx, item)
		if err != nil {
			result.Errors = append(result.Errors, writeError(i, item.ID, err))
			continue
		}
		params = append(params, param)
		indexMap = append(indexMap, i)
	}

	txResult := s.transactions.UpdateTransactionsById(ctx, params)
	result.UpdatedIDs = sortedTransactionIDs(txResult.Success)
	result.Errors = append(result.Errors, mapTransactionErrors(txResult.Errors, indexMap)...)
	sortWriteErrors(result.Errors)
	return result
}

func (s *Service) GetTransactions(ctx context.Context, ids []int64, includeDeleted bool) ([]TransactionDTO, error) {
	if len(ids) == 0 {
		return nil, NewError(CodeValidationError, "at least one transaction id is required", nil)
	}
	rows, err := s.repo.GetTransactionsByIdWithDetails(ctx, sqlc.GetTransactionsByIdWithDetailsParams{
		Ids:            ids,
		IncludeDeleted: includeDeleted,
	})
	if err != nil {
		return nil, mapTransactionError(err)
	}
	dtos := make([]TransactionDTO, 0, len(rows))
	for _, row := range rows {
		dtos = append(dtos, transactionDTO(row.Transaction, row.AuthorName, row.HouseholdName, row.CategoryID, row.CategoryCode, row.CategoryName))
	}
	return dtos, nil
}

func (s *Service) ListTransactions(ctx context.Context, req ListTransactionsRequest) ([]TransactionDTO, error) {
	sortValue := req.Sort
	if sortValue == "" {
		sortValue = "transaction_date"
	}
	order := req.SortOrder
	if order == "" {
		order = "desc"
	}
	limit := req.Limit
	if limit == 0 {
		limit = 100
	}

	rows, err := s.repo.ListTransactions(ctx, sqlc.ListTransactionsParams{
		OnlyDeleted:    req.OnlyDeleted,
		IncludeDeleted: req.IncludeDeleted,
		AuthorID:       req.AuthorID,
		HouseholdID:    req.HouseholdID,
		FromDate:       timestamptz(req.FromDate),
		ToDate:         timestamptz(req.ToDate),
		Search:         req.Search,
		Sort:           sortValue,
		SortOrder:      order,
		ResultOffset:   req.Offset,
		ResultLimit:    limit,
	})
	if err != nil {
		return nil, NewError(CodeDatabaseError, "transaction list failed", err)
	}

	dtos := make([]TransactionDTO, 0, len(rows))
	for _, row := range rows {
		dtos = append(dtos, transactionDTO(row.Transaction, row.AuthorName, row.HouseholdName, row.CategoryID, row.CategoryCode, row.CategoryName))
	}
	return dtos, nil
}

func (s *Service) DeleteTransactions(ctx context.Context, req DeleteTransactionsRequest) WriteResult {
	if len(req.IDs) == 0 {
		return WriteResult{Errors: []WriteError{writeError(0, 0, NewError(CodeValidationError, "at least one transaction id is required", nil))}}
	}
	if req.DeletedByUserID == 0 {
		return WriteResult{Errors: []WriteError{writeError(0, 0, NewError(CodeValidationError, "deleted by user id is required", nil))}}
	}
	txResult := s.transactions.SoftDeleteTransactionsById(ctx, req.IDs, req.DeletedByUserID, req.Reason)
	result := WriteResult{DeletedIDs: successfulInputIDs(req.IDs, txResult.Success)}
	result.Errors = append(result.Errors, mapTransactionErrors(txResult.Errors, nil)...)
	sortWriteErrors(result.Errors)
	return result
}

func (s *Service) RestoreTransactions(ctx context.Context, req RestoreTransactionsRequest) WriteResult {
	if len(req.IDs) == 0 {
		return WriteResult{Errors: []WriteError{writeError(0, 0, NewError(CodeValidationError, "at least one transaction id is required", nil))}}
	}
	if req.RestoredByUserID == 0 {
		return WriteResult{Errors: []WriteError{writeError(0, 0, NewError(CodeValidationError, "restored by user id is required", nil))}}
	}
	txResult := s.transactions.RestoreTransactionsById(ctx, req.IDs, req.RestoredByUserID)
	result := WriteResult{RestoredIDs: successfulInputIDs(req.IDs, txResult.Success)}
	result.Errors = append(result.Errors, mapTransactionErrors(txResult.Errors, nil)...)
	sortWriteErrors(result.Errors)
	return result
}

func (s *Service) createParams(ctx context.Context, req CreateTransactionRequest) (sqlc.CreateTransactionParams, error) {
	if req.HouseholdID == nil {
		return sqlc.CreateTransactionParams{}, NewError(CodeValidationError, "household id is required", nil)
	}
	author, err := s.ResolveUser(ctx, req.Author)
	if err != nil {
		return sqlc.CreateTransactionParams{}, err
	}
	categoryID, err := s.resolveCategoryID(ctx, req.CategoryID, req.CategoryCode)
	if err != nil {
		return sqlc.CreateTransactionParams{}, err
	}
	return sqlc.CreateTransactionParams{
		Amount:          req.Amount,
		CategoryID:      categoryID,
		Description:     req.Description,
		TransactionDate: pgtype.Timestamptz{Time: req.TransactionDate, Valid: true},
		AuthorID:        author.ID,
		HouseholdID:     req.HouseholdID,
		Notes:           req.Notes,
	}, nil
}

func (s *Service) updateParams(ctx context.Context, req UpdateTransactionRequest) (transaction.UpdateTransactionById, error) {
	if req.ID == 0 {
		return transaction.UpdateTransactionById{}, NewError(CodeValidationError, "transaction id is required", nil)
	}
	updates := &transaction.TransactionUpdate{}
	if req.Amount != nil {
		updates.Amount = utils.NewOptional(*req.Amount)
	}
	if req.TransactionDate != nil {
		updates.TransactionDate = utils.NewOptional(*req.TransactionDate)
	}
	if req.Description != nil || req.ClearDescription {
		updates.Description = utils.NewOptional(req.Description)
	}
	if req.Notes != nil || req.ClearNotes {
		updates.Notes = utils.NewOptional(req.Notes)
	}
	if req.CategoryID != nil || req.CategoryCode != nil {
		categoryID, err := s.resolveCategoryID(ctx, req.CategoryID, req.CategoryCode)
		if err != nil {
			return transaction.UpdateTransactionById{}, err
		}
		updates.CategoryID = utils.NewOptional(categoryID)
	} else if req.ClearCategoryID {
		updates.CategoryID = utils.NewOptional((*int64)(nil))
	}
	if req.HouseholdID != nil || req.ClearHouseholdID {
		updates.HouseholdID = utils.NewOptional(req.HouseholdID)
	}
	if req.Author != nil {
		author, err := s.ResolveUser(ctx, *req.Author)
		if err != nil {
			return transaction.UpdateTransactionById{}, err
		}
		updates.AuthorID = utils.NewOptional(author.ID)
	}
	return transaction.UpdateTransactionById{ID: req.ID, Updates: updates}, nil
}

func (s *Service) resolveCategoryID(ctx context.Context, id *int64, code *string) (*int64, error) {
	if id == nil && code == nil {
		return nil, nil
	}

	if id != nil && code != nil {
		categoryByID, err := s.repo.GetActiveCategoryById(ctx, *id)
		if err != nil {
			return nil, mapCategoryError(err)
		}
		categoryByCode, err := s.repo.GetActiveCategoryByCode(ctx, *code)
		if err != nil {
			return nil, mapCategoryError(err)
		}
		if categoryByID.ID != categoryByCode.ID {
			return nil, NewError(CodeValidationError, "category id and code refer to different categories", nil)
		}
		return &categoryByID.ID, nil
	}

	if id != nil {
		category, err := s.repo.GetActiveCategoryById(ctx, *id)
		if err != nil {
			return nil, mapCategoryError(err)
		}
		return &category.ID, nil
	}

	category, err := s.repo.GetActiveCategoryByCode(ctx, *code)
	if err != nil {
		return nil, mapCategoryError(err)
	}
	return &category.ID, nil
}

func transactionDTO(tx sqlc.Transaction, authorName string, householdName *string, categoryID *int64, categoryCode, categoryName *string) TransactionDTO {
	return TransactionDTO{
		ID:              tx.ID,
		Amount:          tx.Amount,
		TransactionDate: tx.TransactionDate.Time,
		AuthorID:        tx.AuthorID,
		AuthorName:      authorName,
		HouseholdID:     tx.HouseholdID,
		HouseholdName:   householdName,
		Category:        transactionCategoryDTO(categoryID, categoryCode, categoryName),
		Description:     tx.Description,
		Notes:           tx.Notes,
		CreatedAt:       validTime(tx.CreatedAt.Time, tx.CreatedAt.Valid),
		UpdatedAt:       validTime(tx.UpdatedAt.Time, tx.UpdatedAt.Valid),
		DeletedAt:       validTime(tx.DeletedAt.Time, tx.DeletedAt.Valid),
		DeleteReason:    tx.DeleteReason,
	}
}

func transactionCategoryDTO(id *int64, code, name *string) *CategoryRefDTO {
	if id == nil || code == nil || name == nil {
		return nil
	}
	return &CategoryRefDTO{ID: *id, Code: *code, Name: *name}
}

func timestamptz(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func mapTransactionErrors(errors []transaction.TransactionError, indexMap []int) []WriteError {
	writeErrors := make([]WriteError, 0, len(errors))
	for _, item := range errors {
		index := item.Index
		if indexMap != nil && item.Index >= 0 && item.Index < len(indexMap) {
			index = indexMap[item.Index]
		}
		writeErrors = append(writeErrors, writeError(index, item.ID, mapTransactionError(item.Err)))
	}
	return writeErrors
}

func sortedTransactionIDs(success map[int64]*sqlc.Transaction) []int64 {
	ids := make([]int64, 0, len(success))
	for id := range success {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func successfulInputIDs(ids []int64, success map[int64]*sqlc.Transaction) []int64 {
	result := make([]int64, 0, len(success))
	for _, id := range ids {
		if _, ok := success[id]; ok {
			result = append(result, id)
		}
	}
	return result
}

func sortWriteErrors(items []WriteError) {
	sort.SliceStable(items, func(i, j int) bool { return items[i].Index < items[j].Index })
}

func writeError(index int, id int64, err error) WriteError {
	appErr, ok := err.(*AppError)
	if !ok {
		appErr = &AppError{Code: CodeDatabaseError, Message: err.Error(), Err: err}
	}
	return WriteError{Index: index, ID: id, Code: appErr.Code, Message: appErr.Message}
}
