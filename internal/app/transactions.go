package app

import (
	"context"
	"time"

	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/transaction"
	"rdmm404/voltr-finance/internal/utils"

	"github.com/jackc/pgx/v5/pgtype"
)

type TransactionDTO struct {
	ID              int64      `json:"id"`
	Amount          float32    `json:"amount"`
	TransactionDate time.Time  `json:"transactionDate"`
	AuthorID        int64      `json:"authorId"`
	AuthorName      string     `json:"authorName,omitempty"`
	HouseholdID     *int64     `json:"householdId,omitempty"`
	HouseholdName   *string    `json:"householdName,omitempty"`
	Description     *string    `json:"description,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
	CreatedAt       *time.Time `json:"createdAt,omitempty"`
	UpdatedAt       *time.Time `json:"updatedAt,omitempty"`
	DeletedAt       *time.Time `json:"deletedAt,omitempty"`
	DeleteReason    *string    `json:"deleteReason,omitempty"`
}

type CreateTransactionRequest struct {
	Amount           float32          `json:"amount"`
	TransactionDate  time.Time        `json:"transactionDate"`
	Description      *string          `json:"description,omitempty"`
	Notes            *string          `json:"notes,omitempty"`
	BudgetCategoryID *int64           `json:"budgetCategoryId,omitempty"`
	HouseholdID      *int64           `json:"householdId,omitempty"`
	Author           IdentitySelector `json:"author"`
}

type BulkCreateTransactionsRequest struct {
	Transactions []CreateTransactionRequest `json:"transactions"`
}

type UpdateTransactionRequest struct {
	ID int64 `json:"id"`

	Amount           *float32          `json:"amount,omitempty"`
	TransactionDate  *time.Time        `json:"transactionDate,omitempty"`
	Description      *string           `json:"description,omitempty"`
	Notes            *string           `json:"notes,omitempty"`
	BudgetCategoryID *int64            `json:"budgetCategoryId,omitempty"`
	HouseholdID      *int64            `json:"householdId,omitempty"`
	Author           *IdentitySelector `json:"author,omitempty"`

	ClearDescription      bool `json:"clearDescription,omitempty"`
	ClearNotes            bool `json:"clearNotes,omitempty"`
	ClearBudgetCategoryID bool `json:"clearBudgetCategoryId,omitempty"`
	ClearHouseholdID      bool `json:"clearHouseholdId,omitempty"`
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
	for id := range txResult.Success {
		result.CreatedIDs = append(result.CreatedIDs, id)
	}
	result.Errors = append(result.Errors, mapTransactionErrors(txResult.Errors, indexMap)...)
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
	for id := range txResult.Success {
		result.UpdatedIDs = append(result.UpdatedIDs, id)
	}
	result.Errors = append(result.Errors, mapTransactionErrors(txResult.Errors, indexMap)...)
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
		dtos = append(dtos, transactionDTO(row.Transaction, row.AuthorName, row.HouseholdName))
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
		dtos = append(dtos, transactionDTO(row.Transaction, row.AuthorName, row.HouseholdName))
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
	result := WriteResult{}
	for id := range txResult.Success {
		result.DeletedIDs = append(result.DeletedIDs, id)
	}
	result.Errors = append(result.Errors, mapTransactionErrors(txResult.Errors, nil)...)
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
	result := WriteResult{}
	for id := range txResult.Success {
		result.RestoredIDs = append(result.RestoredIDs, id)
	}
	result.Errors = append(result.Errors, mapTransactionErrors(txResult.Errors, nil)...)
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
	return sqlc.CreateTransactionParams{
		Amount:           req.Amount,
		BudgetCategoryID: req.BudgetCategoryID,
		Description:      req.Description,
		TransactionDate:  pgtype.Timestamptz{Time: req.TransactionDate, Valid: true},
		AuthorID:         author.ID,
		HouseholdID:      req.HouseholdID,
		Notes:            req.Notes,
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
	if req.BudgetCategoryID != nil || req.ClearBudgetCategoryID {
		updates.BudgetCategoryID = utils.NewOptional(req.BudgetCategoryID)
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

func transactionDTO(tx sqlc.Transaction, authorName string, householdName *string) TransactionDTO {
	return TransactionDTO{
		ID:              tx.ID,
		Amount:          tx.Amount,
		TransactionDate: tx.TransactionDate.Time,
		AuthorID:        tx.AuthorID,
		AuthorName:      authorName,
		HouseholdID:     tx.HouseholdID,
		HouseholdName:   householdName,
		Description:     tx.Description,
		Notes:           tx.Notes,
		CreatedAt:       validTime(tx.CreatedAt.Time, tx.CreatedAt.Valid),
		UpdatedAt:       validTime(tx.UpdatedAt.Time, tx.UpdatedAt.Valid),
		DeletedAt:       validTime(tx.DeletedAt.Time, tx.DeletedAt.Valid),
		DeleteReason:    tx.DeleteReason,
	}
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

func writeError(index int, id int64, err error) WriteError {
	appErr, ok := err.(*AppError)
	if !ok {
		appErr = &AppError{Code: CodeDatabaseError, Message: err.Error(), Err: err}
	}
	return WriteError{Index: index, ID: id, Code: appErr.Code, Message: appErr.Message}
}
