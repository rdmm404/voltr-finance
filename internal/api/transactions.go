package api

import "time"

type CategoryRef struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type Transaction struct {
	ID              int64        `json:"id"`
	Amount          float32      `json:"amount"`
	TransactionDate time.Time    `json:"transactionDate"`
	AuthorID        int64        `json:"authorId"`
	AuthorName      string       `json:"authorName,omitempty"`
	HouseholdID     *int64       `json:"householdId,omitempty"`
	HouseholdName   *string      `json:"householdName,omitempty"`
	Category        *CategoryRef `json:"category,omitempty"`
	Description     *string      `json:"description,omitempty"`
	Notes           *string      `json:"notes,omitempty"`
	CreatedAt       *time.Time   `json:"createdAt,omitempty"`
	UpdatedAt       *time.Time   `json:"updatedAt,omitempty"`
	DeletedAt       *time.Time   `json:"deletedAt,omitempty"`
	DeleteReason    *string      `json:"deleteReason,omitempty"`
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

type BulkUpdateTransaction struct {
	ID int64 `json:"id"`
	UpdateTransactionRequest
}

type BulkUpdateTransactionsRequest struct {
	Transactions []BulkUpdateTransaction `json:"transactions"`
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

type GetTransactionQuery struct {
	IncludeDeleted bool `query:"includeDeleted"`
}

type ListTransactionsQuery struct {
	IDs            []int64    `query:"ids"`
	AuthorID       *int64     `query:"authorId"`
	HouseholdID    *int64     `query:"householdId"`
	FromDate       *time.Time `query:"fromDate"`
	ToDate         *time.Time `query:"toDate"`
	Search         *string    `query:"search"`
	Sort           string     `query:"sort"`
	SortOrder      string     `query:"sortOrder"`
	Limit          int32      `query:"limit"`
	Offset         int32      `query:"offset"`
	IncludeDeleted bool       `query:"includeDeleted"`
	OnlyDeleted    bool       `query:"onlyDeleted"`
}
