package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"rdmm404/voltr-finance/internal/api"
	"rdmm404/voltr-finance/internal/restclient"

	"github.com/alecthomas/kong"
)

type APIClient interface {
	CreateTransaction(context.Context, api.CreateTransactionRequest) (api.Transaction, error)
	CreateTransactions(context.Context, api.BulkCreateTransactionsRequest) (api.BulkResult, error)
	GetTransaction(context.Context, int64, bool) (api.Transaction, error)
	ListTransactions(context.Context, api.ListTransactionsQuery) ([]api.Transaction, error)
	UpdateTransaction(context.Context, int64, api.UpdateTransactionRequest) (api.Transaction, error)
	UpdateTransactions(context.Context, api.BulkUpdateTransactionsRequest) (api.BulkResult, error)
	DeleteTransactions(context.Context, api.DeleteTransactionsRequest) (api.BulkResult, error)
	RestoreTransactions(context.Context, api.RestoreTransactionsRequest) (api.BulkResult, error)

	CreateUser(context.Context, api.CreateUserRequest) (api.User, error)
	UpdateUser(context.Context, int64, api.UpdateUserRequest) (api.User, error)
	GetUser(context.Context, int64) (api.User, error)
	ResolveUser(context.Context, api.IdentitySelector) (api.User, error)
	ListUsers(context.Context) ([]api.User, error)

	GetHousehold(context.Context, int64) (api.Household, error)
	ResolveHousehold(context.Context, api.ResolveHouseholdQuery) (api.Household, error)
	ListHouseholds(context.Context) ([]api.Household, error)
	ListHouseholdUsers(context.Context, int64) ([]api.User, error)

	CreateCategory(context.Context, api.CreateCategoryRequest) (api.Category, error)
	ListCategories(context.Context, api.ListCategoriesQuery) ([]api.Category, error)
	GetCategory(context.Context, string) (api.Category, error)
	UpdateCategory(context.Context, int64, api.UpdateCategoryRequest) (api.Category, error)
	DeactivateCategory(context.Context, string) (api.Category, error)

	GetMonthlyBudget(context.Context, api.MonthlyBudgetParams) (api.Budget, error)
	EnsureMonthlyBudget(context.Context, api.MonthlyBudgetParams) (api.Budget, error)
	CreateBudgetLine(context.Context, int64, api.CreateBudgetLineRequest) (api.BudgetLine, error)
	UpdateBudgetLine(context.Context, int64, api.UpdateBudgetLineRequest) (api.BudgetLine, error)
	DeleteBudgetLine(context.Context, int64) error
	GetBudgetReport(context.Context, int64) (api.BudgetReport, error)
}

var _ APIClient = (*restclient.Client)(nil)

type CLI struct {
	Transactions TransactionsCmd `cmd:"" help:"Manage transactions."`
	Users        UsersCmd        `cmd:"" help:"Manage users."`
	Households   HouseholdsCmd   `cmd:"" help:"Read households."`
	Categories   CategoriesCmd   `cmd:"" help:"Manage transaction categories."`
	Budgets      BudgetsCmd      `cmd:"" help:"Manage budgets."`
}

type runContext struct {
	context.Context
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	svc    APIClient
}

func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, client APIClient) int {
	if stdin == nil {
		stdin = os.Stdin
	}
	var cli CLI
	parser, err := kong.New(&cli, kong.Name("voltr-finance"), kong.Exit(func(int) {}), kong.Writers(stdout, stderr))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	kctx, err := parser.Parse(args)
	if err != nil {
		if isHelpArgs(args) {
			return 0
		}
		fmt.Fprintln(stderr, err)
		return 2
	}
	if isHelpArgs(args) {
		return 0
	}
	if err := kctx.Run(&runContext{Context: ctx, stdin: stdin, stdout: stdout, stderr: stderr, svc: client}); err != nil {
		if isExpectedError(err) {
			fmt.Fprintln(stderr, expectedErrorMessage(err))
			return 2
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

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
	transaction, err := ctx.svc.CreateTransaction(ctx.Context, api.CreateTransactionRequest{
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
	result, err := ctx.svc.CreateTransactions(ctx.Context, req)
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
	transaction, err := ctx.svc.UpdateTransaction(ctx.Context, c.ID, req)
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
	result, err := ctx.svc.UpdateTransactions(ctx.Context, req)
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
	txs, err := ctx.svc.ListTransactions(ctx.Context, api.ListTransactionsQuery{IDs: ids, IncludeDeleted: c.IncludeDeleted})
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
	txs, err := ctx.svc.ListTransactions(ctx.Context, api.ListTransactionsQuery{
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
	result, err := ctx.svc.DeleteTransactions(ctx.Context, api.DeleteTransactionsRequest{IDs: ids, DeletedByUserID: c.DeletedByUserID, Reason: c.Reason})
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
	result, err := ctx.svc.RestoreTransactions(ctx.Context, api.RestoreTransactionsRequest{IDs: ids, RestoredByUserID: c.RestoredByUserID})
	if err != nil {
		return err
	}
	return renderBulkResult(ctx.stdout, result)
}

type CategoriesCmd struct {
	Create     CategoryCreateCmd     `cmd:"" help:"Create a category."`
	List       CategoryListCmd       `cmd:"" help:"List categories."`
	Rename     CategoryRenameCmd     `cmd:"" help:"Rename a category by code."`
	Deactivate CategoryDeactivateCmd `cmd:"" help:"Deactivate a category by code."`
}

type CategoryCreateCmd struct {
	Name        string  `arg:"" required:"" help:"Category display name."`
	Code        *string `help:"Stable category code. Defaults to a slug generated from name."`
	Description *string `help:"Optional category description."`
}

func (c *CategoryCreateCmd) Run(ctx *runContext) error {
	category, err := ctx.svc.CreateCategory(ctx.Context, api.CreateCategoryRequest{
		Name:        c.Name,
		Code:        c.Code,
		Description: c.Description,
	})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, category)
}

type CategoryListCmd struct {
	IncludeInactive bool `help:"Include inactive categories."`
}

func (c *CategoryListCmd) Run(ctx *runContext) error {
	categories, err := ctx.svc.ListCategories(ctx.Context, api.ListCategoriesQuery{IncludeInactive: c.IncludeInactive})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, categories)
}

type CategoryRenameCmd struct {
	Code string `arg:"" required:"" help:"Existing category code."`
	Name string `arg:"" required:"" help:"New category display name."`
}

func (c *CategoryRenameCmd) Run(ctx *runContext) error {
	existing, err := ctx.svc.GetCategory(ctx.Context, c.Code)
	if err != nil {
		return err
	}
	category, err := ctx.svc.UpdateCategory(ctx.Context, existing.ID, api.UpdateCategoryRequest{Name: &c.Name})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, category)
}

type CategoryDeactivateCmd struct {
	Code string `arg:"" required:"" help:"Category code to deactivate."`
}

func (c *CategoryDeactivateCmd) Run(ctx *runContext) error {
	category, err := ctx.svc.DeactivateCategory(ctx.Context, c.Code)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, category)
}

type UsersCmd struct {
	Create  UserCreateCmd  `cmd:"" help:"Create a user with optional external identities."`
	Update  UserUpdateCmd  `cmd:"" help:"Update a user and optional external identities."`
	Get     UserGetCmd     `cmd:"" help:"Get a user by internal ID."`
	Resolve UserResolveCmd `cmd:"" help:"Resolve a user from exactly one identity selector."`
	List    UserListCmd    `cmd:"" help:"List all users."`
}

type UserCreateCmd struct {
	Name        string  `required:"" help:"Display name for the user."`
	DiscordID   *string `help:"Discord user ID."`
	TelegramID  *string `help:"Telegram user ID."`
	PhoneNumber *string `help:"Phone number."`
	WhatsappID  *string `help:"WhatsApp ID."`
}

func (c *UserCreateCmd) Run(ctx *runContext) error {
	user, err := ctx.svc.CreateUser(ctx.Context, api.CreateUserRequest{Name: c.Name, DiscordID: c.DiscordID, TelegramID: c.TelegramID, PhoneNumber: c.PhoneNumber, WhatsAppID: c.WhatsappID})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, user)
}

type UserUpdateCmd struct {
	ID               int64   `required:"" help:"Internal user ID."`
	Name             *string `help:"Replacement display name."`
	DiscordID        *string `help:"Replacement Discord user ID."`
	TelegramID       *string `help:"Replacement Telegram user ID."`
	PhoneNumber      *string `help:"Replacement phone number."`
	WhatsappID       *string `help:"Replacement WhatsApp ID."`
	ClearDiscordID   bool    `help:"Clear the Discord ID."`
	ClearTelegramID  bool    `help:"Clear the Telegram ID."`
	ClearPhoneNumber bool    `help:"Clear the phone number."`
	ClearWhatsappID  bool    `help:"Clear the WhatsApp ID."`
}

func (c *UserUpdateCmd) Run(ctx *runContext) error {
	user, err := ctx.svc.UpdateUser(ctx.Context, c.ID, api.UpdateUserRequest{
		Name: c.Name, DiscordID: c.DiscordID, TelegramID: c.TelegramID, PhoneNumber: c.PhoneNumber, WhatsAppID: c.WhatsappID,
		ClearDiscordID: c.ClearDiscordID, ClearTelegramID: c.ClearTelegramID,
		ClearPhoneNumber: c.ClearPhoneNumber, ClearWhatsAppID: c.ClearWhatsappID,
	})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, user)
}

type UserGetCmd struct {
	ID int64 `required:"" help:"Internal user ID."`
}

func (c *UserGetCmd) Run(ctx *runContext) error {
	user, err := ctx.svc.GetUser(ctx.Context, c.ID)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, user)
}

type UserResolveCmd struct {
	AuthorID    *int64  `placeholder:"INT-64" help:"Internal user ID. Exactly one identity selector is required."`
	DiscordID   *string `help:"Discord user ID. Exactly one identity selector is required."`
	TelegramID  *string `help:"Telegram user ID. Exactly one identity selector is required."`
	PhoneNumber *string `help:"Phone number. Exactly one identity selector is required."`
	WhatsappID  *string `help:"WhatsApp ID. Exactly one identity selector is required."`
}

func (c *UserResolveCmd) Run(ctx *runContext) error {
	user, err := ctx.svc.ResolveUser(ctx.Context, identity(c.AuthorID, c.DiscordID, c.TelegramID, c.PhoneNumber, c.WhatsappID))
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, user)
}

type UserListCmd struct{}

func (c *UserListCmd) Run(ctx *runContext) error {
	users, err := ctx.svc.ListUsers(ctx.Context)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, users)
}

type HouseholdsCmd struct {
	Get   HouseholdGetCmd   `cmd:"" help:"Get a household from exactly one selector."`
	List  HouseholdListCmd  `cmd:"" help:"List all households."`
	Users HouseholdUsersCmd `cmd:"" help:"List users in a household."`
}

type HouseholdGetCmd struct {
	ID      *int64  `placeholder:"INT-64" help:"Internal household ID. Exactly one household selector is required."`
	GuildID *string `help:"Discord guild/server ID. Exactly one household selector is required."`
	Name    *string `help:"Household name. Exactly one household selector is required."`
}

func (c *HouseholdGetCmd) Run(ctx *runContext) error {
	var household api.Household
	var err error
	if c.ID != nil {
		household, err = ctx.svc.GetHousehold(ctx.Context, *c.ID)
	} else {
		household, err = ctx.svc.ResolveHousehold(ctx.Context, api.ResolveHouseholdQuery{Name: c.Name, GuildID: c.GuildID})
	}
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, household)
}

type HouseholdListCmd struct{}

func (c *HouseholdListCmd) Run(ctx *runContext) error {
	households, err := ctx.svc.ListHouseholds(ctx.Context)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, households)
}

type HouseholdUsersCmd struct {
	HouseholdID int64 `required:"" help:"Internal household ID."`
}

func (c *HouseholdUsersCmd) Run(ctx *runContext) error {
	users, err := ctx.svc.ListHouseholdUsers(ctx.Context, c.HouseholdID)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, users)
}

type BudgetsCmd struct {
	Get    BudgetGetCmd    `cmd:"" help:"Get a monthly budget."`
	Report BudgetReportCmd `cmd:"" help:"Show a budget report."`
	Lines  BudgetLinesCmd  `cmd:"" help:"Manage budget lines."`
}

type BudgetGetCmd struct {
	HouseholdID *int64 `placeholder:"INT-64" help:"Household budget owner."`
	UserID      *int64 `placeholder:"INT-64" help:"Personal budget owner."`
	Month       string `required:"" help:"Budget month in YYYY-MM format."`
	Create      bool   `help:"Create the monthly budget if missing."`
}

func (c *BudgetGetCmd) Run(ctx *runContext) error {
	year, month, err := parseBudgetMonth(c.Month)
	if err != nil {
		return err
	}
	params := api.MonthlyBudgetParams{HouseholdID: c.HouseholdID, UserID: c.UserID, Year: year, Month: month}
	var budget api.Budget
	if c.Create {
		budget, err = ctx.svc.EnsureMonthlyBudget(ctx.Context, params)
	} else {
		budget, err = ctx.svc.GetMonthlyBudget(ctx.Context, params)
	}
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, budget)
}

type BudgetReportCmd struct {
	ID int64 `arg:"" required:"" help:"Budget ID."`
}

func (c *BudgetReportCmd) Run(ctx *runContext) error {
	report, err := ctx.svc.GetBudgetReport(ctx.Context, c.ID)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, report)
}

type BudgetLinesCmd struct {
	Add    BudgetLineAddCmd    `cmd:"" help:"Add a budget line."`
	Update BudgetLineUpdateCmd `cmd:"" help:"Update a budget line."`
	Delete BudgetLineDeleteCmd `cmd:"" help:"Delete a budget line."`
}

type BudgetLineAddCmd struct {
	BudgetID   int64   `required:"" placeholder:"INT-64" help:"Budget ID."`
	Name       string  `required:"" help:"Budget line name."`
	Amount     string  `required:"" help:"Allocation amount."`
	Categories *string `help:"Comma-separated category codes."`
	SortOrder  *int32  `help:"Display sort order."`
}

func (c *BudgetLineAddCmd) Run(ctx *runContext) error {
	line, err := ctx.svc.CreateBudgetLine(ctx.Context, c.BudgetID, api.CreateBudgetLineRequest{
		Name:             c.Name,
		AllocationAmount: c.Amount,
		CategoryCodes:    parseOptionalCSV(c.Categories),
		SortOrder:        c.SortOrder,
	})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, line)
}

type BudgetLineUpdateCmd struct {
	ID         int64   `arg:"" required:"" help:"Budget line ID."`
	Name       *string `help:"Replacement budget line name."`
	Amount     *string `help:"Replacement allocation amount."`
	Categories *string `help:"Replacement comma-separated category codes."`
	SortOrder  *int32  `help:"Replacement display sort order."`
}

func (c *BudgetLineUpdateCmd) Run(ctx *runContext) error {
	var categoryCodes *[]string
	if c.Categories != nil {
		parsed := parseOptionalCSV(c.Categories)
		categoryCodes = &parsed
	}
	line, err := ctx.svc.UpdateBudgetLine(ctx.Context, c.ID, api.UpdateBudgetLineRequest{
		Name:             c.Name,
		AllocationAmount: c.Amount,
		CategoryCodes:    categoryCodes,
		SortOrder:        c.SortOrder,
	})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, line)
}

type BudgetLineDeleteCmd struct {
	ID int64 `arg:"" required:"" help:"Budget line ID."`
}

func (c *BudgetLineDeleteCmd) Run(ctx *runContext) error {
	return ctx.svc.DeleteBudgetLine(ctx.Context, c.ID)
}

func identity(userID *int64, discordID, telegramID, phoneNumber, whatsappID *string) api.IdentitySelector {
	return api.IdentitySelector{UserID: userID, DiscordID: discordID, TelegramID: telegramID, PhoneNumber: phoneNumber, WhatsAppID: whatsappID}
}

func renderBulkResult(w io.Writer, result api.BulkResult) error {
	if err := RenderJSON(w, result); err != nil {
		return err
	}
	if len(result.Failed) > 0 {
		return NewCLIError("write completed with item failures")
	}
	return nil
}

type CLIError struct {
	Message string
}

func NewCLIError(message string) error {
	return CLIError{Message: message}
}

func (e CLIError) Error() string {
	return e.Message
}

func isExpectedError(err error) bool {
	var cliErr CLIError
	if errors.As(err, &cliErr) {
		return true
	}
	var apiErr *restclient.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 && apiErr.StatusCode != 401 && apiErr.StatusCode != 403
}

func expectedErrorMessage(err error) string {
	var apiErr *restclient.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Message
	}
	return err.Error()
}

func decodeJSONInput(stdin io.Reader, input *string, value any) error {
	reader := stdin
	if input != nil {
		file, err := os.Open(*input)
		if err != nil {
			return err
		}
		defer file.Close()
		reader = file
	}
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("input contains multiple JSON values")
		}
		return err
	}
	return nil
}

func parseIDs(raw string) ([]int64, error) {
	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func parseBudgetMonth(value string) (int, int, error) {
	parsed, err := time.Parse("2006-01", value)
	if err != nil {
		return 0, 0, fmt.Errorf("month must be in YYYY-MM format")
	}
	return parsed.Year(), int(parsed.Month()), nil
}

func parseOptionalCSV(value *string) []string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	parts := strings.Split(*value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func isHelpArgs(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" || arg == "help" {
			return true
		}
	}
	return false
}
