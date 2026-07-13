package cli

import (
	"time"

	"rdmm404/voltr-finance/internal/api"
)

type TransactionsCmd struct {
	Create     TransactionCreateCmd     `cmd:"" help:"Create one transaction."`
	CreateBulk TransactionCreateBulkCmd `cmd:"create-bulk" help:"Create multiple transactions from JSON."`
	Update     TransactionUpdateCmd     `cmd:"" help:"Update one transaction by internal ID."`
	UpdateBulk TransactionUpdateBulkCmd `cmd:"update-bulk" help:"Update multiple transactions from JSON."`
	Get        TransactionGetCmd        `cmd:"" help:"Get transactions by internal ID."`
	List       TransactionListCmd       `cmd:"" help:"List transactions with filters, sorting, and pagination."`
	Delete     TransactionDeleteCmd     `cmd:"" help:"Soft-delete transactions by internal ID."`
	Restore    TransactionRestoreCmd    `cmd:"" help:"Restore soft-deleted transactions by internal ID."`
}

type TransactionCreateCmd struct {
	Amount            float32   `required:"" help:"Transaction amount, in dollars."`
	TransactionDate   time.Time `required:"" placeholder:"RFC3339" help:"Transaction timestamp in RFC3339 format, for example 2026-05-05T14:30:00-04:00."`
	Description       *string   `help:"Short transaction description."`
	Notes             *string   `help:"Longer transaction notes."`
	Category          *string   `help:"Category code."`
	HouseholdID       *int64    `required:"" placeholder:"INT-64" help:"Internal household ID."`
	AuthorID          *int64    `placeholder:"INT-64" help:"Internal author user ID. Exactly one author selector may be provided."`
	AuthorDiscordID   *string   `help:"Author Discord user ID. Exactly one author selector may be provided."`
	AuthorTelegramID  *string   `help:"Author Telegram user ID. Exactly one author selector may be provided."`
	AuthorPhoneNumber *string   `help:"Author phone number. Exactly one author selector may be provided."`
	AuthorWhatsappID  *string   `help:"Author WhatsApp ID. Exactly one author selector may be provided."`
}

func (c *TransactionCreateCmd) Run(ctx *runContext) error {
	transaction, err := ctx.transactions.CreateTransaction(ctx.Context, api.CreateTransactionRequest{
		Amount: c.Amount, TransactionDate: c.TransactionDate, Description: c.Description, Notes: c.Notes,
		CategoryCode: c.Category, HouseholdID: c.HouseholdID,
		Author: identity(c.AuthorID, c.AuthorDiscordID, c.AuthorTelegramID, c.AuthorPhoneNumber, c.AuthorWhatsappID),
	})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, transaction)
}

type TransactionCreateBulkCmd struct {
	Input *string `help:"Path to a JSON file containing a bulk create request. Reads stdin when omitted. Expected shape: {\"transactions\":[...]}."`
}

func (c *TransactionCreateBulkCmd) Run(ctx *runContext) error {
	var req api.BulkCreateTransactionsRequest
	if err := decodeJSONInput(ctx.stdin, c.Input, &req); err != nil {
		return err
	}
	result, err := ctx.transactions.CreateTransactions(ctx.Context, req)
	if err != nil {
		return err
	}
	return renderBulkResult(ctx.stdout, result)
}

type TransactionUpdateCmd struct {
	ID                int64      `required:"" help:"Internal transaction ID."`
	Amount            *float32   `placeholder:"FLOAT-32" help:"Replacement transaction amount, in dollars."`
	TransactionDate   *time.Time `placeholder:"RFC3339" help:"Replacement transaction timestamp in RFC3339 format, for example 2026-05-05T14:30:00-04:00."`
	Description       *string    `help:"Replacement short transaction description."`
	Notes             *string    `help:"Replacement longer transaction notes."`
	Category          *string    `help:"Replacement category code."`
	HouseholdID       *int64     `placeholder:"INT-64" help:"Replacement internal household ID."`
	AuthorID          *int64     `placeholder:"INT-64" help:"Replacement internal author user ID. Exactly one author selector may be provided."`
	AuthorDiscordID   *string    `help:"Replacement author Discord user ID. Exactly one author selector may be provided."`
	AuthorTelegramID  *string    `help:"Replacement author Telegram user ID. Exactly one author selector may be provided."`
	AuthorPhoneNumber *string    `help:"Replacement author phone number. Exactly one author selector may be provided."`
	AuthorWhatsappID  *string    `help:"Replacement author WhatsApp ID. Exactly one author selector may be provided."`
	ClearDescription  bool       `help:"Clear the transaction description."`
	ClearNotes        bool       `help:"Clear the transaction notes."`
	ClearCategory     bool       `help:"Clear the transaction category."`
	ClearHouseholdID  bool       `help:"Clear the household ID."`
}

func (c *TransactionUpdateCmd) Run(ctx *runContext) error {
	selector := identity(c.AuthorID, c.AuthorDiscordID, c.AuthorTelegramID, c.AuthorPhoneNumber, c.AuthorWhatsappID)
	req := api.UpdateTransactionRequest{
		Amount:           c.Amount,
		TransactionDate:  c.TransactionDate,
		Description:      c.Description,
		Notes:            c.Notes,
		CategoryCode:     c.Category,
		HouseholdID:      c.HouseholdID,
		ClearDescription: c.ClearDescription,
		ClearNotes:       c.ClearNotes,
		ClearCategoryID:  c.ClearCategory,
		ClearHouseholdID: c.ClearHouseholdID,
	}
	if selector != (api.IdentitySelector{}) {
		req.Author = &selector
	}
	transaction, err := ctx.transactions.UpdateTransaction(ctx.Context, c.ID, req)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, transaction)
}

type TransactionUpdateBulkCmd struct {
	Input *string `help:"Path to a JSON file containing a bulk update request. Reads stdin when omitted. Expected shape: {\"transactions\":[...]}."`
}

func (c *TransactionUpdateBulkCmd) Run(ctx *runContext) error {
	var req api.BulkUpdateTransactionsRequest
	if err := decodeJSONInput(ctx.stdin, c.Input, &req); err != nil {
		return err
	}
	result, err := ctx.transactions.UpdateTransactions(ctx.Context, req)
	if err != nil {
		return err
	}
	return renderBulkResult(ctx.stdout, result)
}

type TransactionGetCmd struct {
	IDs            string `name:"ids" required:"" help:"Comma-separated internal transaction IDs, for example 101,102,103."`
	IncludeDeleted bool   `help:"Include soft-deleted transactions in the lookup."`
	Format         string `default:"json" enum:"json,compact" help:"Output format: json or compact. Compact is only used for a single transaction."`
}

func (c *TransactionGetCmd) Run(ctx *runContext) error {
	ids, err := parseIDs(c.IDs)
	if err != nil {
		return err
	}
	txs, err := ctx.transactions.ListTransactions(ctx.Context, api.ListTransactionsQuery{IDs: ids, IncludeDeleted: c.IncludeDeleted})
	if err != nil {
		return err
	}
	if c.Format == "compact" && len(txs) == 1 {
		return RenderTransactionCompact(ctx.stdout, txs[0])
	}
	return RenderJSON(ctx.stdout, txs)
}

type TransactionListCmd struct {
	Format         string     `default:"json" enum:"json,csv" help:"Output format: json or csv."`
	AuthorID       *int64     `placeholder:"INT-64" help:"Filter by internal author user ID."`
	HouseholdID    *int64     `placeholder:"INT-64" help:"Filter by internal household ID."`
	FromDate       *time.Time `placeholder:"RFC3339" help:"Include transactions on or after this RFC3339 timestamp."`
	ToDate         *time.Time `placeholder:"RFC3339" help:"Include transactions on or before this RFC3339 timestamp."`
	Search         *string    `help:"Case-insensitive search across description and notes."`
	Sort           string     `help:"Sort field: transaction_date, created_at, amount, or id. Defaults to transaction_date."`
	Order          string     `name:"order" help:"Sort order: asc or desc. Defaults to desc."`
	Limit          int32      `default:"100" help:"Maximum number of transactions to return."`
	Offset         int32      `help:"Number of matching transactions to skip before returning results."`
	IncludeDeleted bool       `help:"Include soft-deleted transactions."`
	OnlyDeleted    bool       `help:"Return only soft-deleted transactions."`
}

func (c *TransactionListCmd) Run(ctx *runContext) error {
	txs, err := ctx.transactions.ListTransactions(ctx.Context, api.ListTransactionsQuery{
		AuthorID:       c.AuthorID,
		HouseholdID:    c.HouseholdID,
		FromDate:       c.FromDate,
		ToDate:         c.ToDate,
		Search:         c.Search,
		Sort:           c.Sort,
		SortOrder:      c.Order,
		Limit:          c.Limit,
		Offset:         c.Offset,
		IncludeDeleted: c.IncludeDeleted,
		OnlyDeleted:    c.OnlyDeleted,
	})
	if err != nil {
		return err
	}
	if c.Format == "csv" {
		return RenderTransactionsCSV(ctx.stdout, txs)
	}
	return RenderJSON(ctx.stdout, txs)
}

type TransactionDeleteCmd struct {
	IDs             string  `name:"ids" required:"" help:"Comma-separated internal transaction IDs, for example 101,102,103."`
	Reason          *string `help:"Optional reason stored with the soft delete."`
	DeletedByUserID int64   `required:"" help:"Internal user ID of the person performing the delete."`
}

func (c *TransactionDeleteCmd) Run(ctx *runContext) error {
	ids, err := parseIDs(c.IDs)
	if err != nil {
		return err
	}
	result, err := ctx.transactions.DeleteTransactions(ctx.Context, api.DeleteTransactionsRequest{IDs: ids, DeletedByUserID: c.DeletedByUserID, Reason: c.Reason})
	if err != nil {
		return err
	}
	return renderBulkResult(ctx.stdout, result)
}

type TransactionRestoreCmd struct {
	IDs              string `name:"ids" required:"" help:"Comma-separated internal transaction IDs, for example 101,102,103."`
	RestoredByUserID int64  `required:"" help:"Internal user ID of the person performing the restore."`
}

func (c *TransactionRestoreCmd) Run(ctx *runContext) error {
	ids, err := parseIDs(c.IDs)
	if err != nil {
		return err
	}
	result, err := ctx.transactions.RestoreTransactions(ctx.Context, api.RestoreTransactionsRequest{IDs: ids, RestoredByUserID: c.RestoredByUserID})
	if err != nil {
		return err
	}
	return renderBulkResult(ctx.stdout, result)
}
