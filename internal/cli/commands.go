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

	"rdmm404/voltr-finance/internal/app"

	"github.com/alecthomas/kong"
)

type AppService interface {
	CreateTransaction(context.Context, app.CreateTransactionRequest) app.WriteResult
	CreateTransactions(context.Context, app.BulkCreateTransactionsRequest) app.WriteResult
	UpdateTransaction(context.Context, app.UpdateTransactionRequest) app.WriteResult
	UpdateTransactions(context.Context, app.BulkUpdateTransactionsRequest) app.WriteResult
	GetTransactions(context.Context, []int64, bool) ([]app.TransactionDTO, error)
	ListTransactions(context.Context, app.ListTransactionsRequest) ([]app.TransactionDTO, error)
	DeleteTransactions(context.Context, app.DeleteTransactionsRequest) app.WriteResult
	RestoreTransactions(context.Context, app.RestoreTransactionsRequest) app.WriteResult

	CreateUser(context.Context, app.CreateUserRequest) (app.UserDTO, error)
	UpdateUser(context.Context, app.UpdateUserRequest) (app.UserDTO, error)
	GetUser(context.Context, int64) (app.UserDTO, error)
	ResolveUser(context.Context, app.IdentitySelector) (app.UserDTO, error)
	ListUsers(context.Context) ([]app.UserDTO, error)

	GetHousehold(context.Context, app.GetHouseholdRequest) (app.HouseholdDTO, error)
	ListHouseholds(context.Context) ([]app.HouseholdDTO, error)
	GetHouseholdUsers(context.Context, int64) ([]app.UserDTO, error)

	CreateCategory(context.Context, app.CreateCategoryRequest) (app.CategoryDTO, error)
	ListCategories(context.Context, app.ListCategoriesRequest) ([]app.CategoryDTO, error)
	GetCategoryByCode(context.Context, string) (app.CategoryDTO, error)
	UpdateCategory(context.Context, app.UpdateCategoryRequest) (app.CategoryDTO, error)
	DeactivateCategory(context.Context, string) (app.CategoryDTO, error)
}

type CLI struct {
	Transactions TransactionsCmd `cmd:"" help:"Manage transactions."`
	Users        UsersCmd        `cmd:"" help:"Manage users."`
	Households   HouseholdsCmd   `cmd:"" help:"Read households."`
	Categories   CategoriesCmd   `cmd:"" help:"Manage transaction categories."`
}

type runContext struct {
	context.Context
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	svc    AppService
}

func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, svc AppService) int {
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
	if err := kctx.Run(&runContext{Context: ctx, stdin: stdin, stdout: stdout, stderr: stderr, svc: svc}); err != nil {
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
	result := ctx.svc.CreateTransaction(ctx.Context, app.CreateTransactionRequest{
		Amount:          c.Amount,
		TransactionDate: c.TransactionDate,
		Description:     c.Description,
		Notes:           c.Notes,
		CategoryCode:    c.Category,
		HouseholdID:     c.HouseholdID,
		Author:          identity(c.AuthorID, c.AuthorDiscordID, c.AuthorTelegramID, c.AuthorPhoneNumber, c.AuthorWhatsappID),
	})
	return renderWriteResult(ctx.stdout, result)
}

type TransactionCreateBulkCmd struct {
	Input *string `help:"Path to a JSON file containing a bulk create request. Reads stdin when omitted. Expected shape: {\"transactions\":[...]}."`
}

func (c *TransactionCreateBulkCmd) Run(ctx *runContext) error {
	var req app.BulkCreateTransactionsRequest
	if err := decodeJSONInput(ctx.stdin, c.Input, &req); err != nil {
		return err
	}
	return renderWriteResult(ctx.stdout, ctx.svc.CreateTransactions(ctx.Context, req))
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
	req := app.UpdateTransactionRequest{
		ID:               c.ID,
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
	if selector != (app.IdentitySelector{}) {
		req.Author = &selector
	}
	return renderWriteResult(ctx.stdout, ctx.svc.UpdateTransaction(ctx.Context, req))
}

type TransactionUpdateBulkCmd struct {
	Input *string `help:"Path to a JSON file containing a bulk update request. Reads stdin when omitted. Expected shape: {\"transactions\":[...]}."`
}

func (c *TransactionUpdateBulkCmd) Run(ctx *runContext) error {
	var req app.BulkUpdateTransactionsRequest
	if err := decodeJSONInput(ctx.stdin, c.Input, &req); err != nil {
		return err
	}
	return renderWriteResult(ctx.stdout, ctx.svc.UpdateTransactions(ctx.Context, req))
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
	txs, err := ctx.svc.GetTransactions(ctx.Context, ids, c.IncludeDeleted)
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
	txs, err := ctx.svc.ListTransactions(ctx.Context, app.ListTransactionsRequest{
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
	return renderWriteResult(ctx.stdout, ctx.svc.DeleteTransactions(ctx.Context, app.DeleteTransactionsRequest{
		IDs:             ids,
		DeletedByUserID: c.DeletedByUserID,
		Reason:          c.Reason,
	}))
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
	return renderWriteResult(ctx.stdout, ctx.svc.RestoreTransactions(ctx.Context, app.RestoreTransactionsRequest{
		IDs:              ids,
		RestoredByUserID: c.RestoredByUserID,
	}))
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
	category, err := ctx.svc.CreateCategory(ctx.Context, app.CreateCategoryRequest{
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
	categories, err := ctx.svc.ListCategories(ctx.Context, app.ListCategoriesRequest{IncludeInactive: c.IncludeInactive})
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
	existing, err := ctx.svc.GetCategoryByCode(ctx.Context, c.Code)
	if err != nil {
		return err
	}
	category, err := ctx.svc.UpdateCategory(ctx.Context, app.UpdateCategoryRequest{
		ID:   existing.ID,
		Name: &c.Name,
	})
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
	user, err := ctx.svc.CreateUser(ctx.Context, app.CreateUserRequest{Name: c.Name, DiscordID: c.DiscordID, TelegramID: c.TelegramID, PhoneNumber: c.PhoneNumber, WhatsappID: c.WhatsappID})
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
	user, err := ctx.svc.UpdateUser(ctx.Context, app.UpdateUserRequest{
		ID:               c.ID,
		Name:             c.Name,
		DiscordID:        c.DiscordID,
		TelegramID:       c.TelegramID,
		PhoneNumber:      c.PhoneNumber,
		WhatsappID:       c.WhatsappID,
		ClearDiscordID:   c.ClearDiscordID,
		ClearTelegramID:  c.ClearTelegramID,
		ClearPhoneNumber: c.ClearPhoneNumber,
		ClearWhatsappID:  c.ClearWhatsappID,
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
	household, err := ctx.svc.GetHousehold(ctx.Context, app.GetHouseholdRequest{ID: c.ID, GuildID: c.GuildID, Name: c.Name})
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
	users, err := ctx.svc.GetHouseholdUsers(ctx.Context, c.HouseholdID)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, users)
}

func identity(authorID *int64, discordID, telegramID, phoneNumber, whatsappID *string) app.IdentitySelector {
	return app.IdentitySelector{AuthorID: authorID, DiscordID: discordID, TelegramID: telegramID, PhoneNumber: phoneNumber, WhatsappID: whatsappID}
}

func renderWriteResult(w io.Writer, result app.WriteResult) error {
	if err := RenderJSON(w, result); err != nil {
		return err
	}
	if len(result.Errors) > 0 {
		return NewCLIError("write completed with errors")
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
	var appErr *app.AppError
	var cliErr CLIError
	return errors.As(err, &appErr) || errors.As(err, &cliErr)
}

func expectedErrorMessage(err error) string {
	var appErr *app.AppError
	if errors.As(err, &appErr) {
		if cause := appErr.Unwrap(); cause != nil {
			return fmt.Sprintf("%s: %v", appErr.Message, cause)
		}
		return appErr.Message
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
	return json.NewDecoder(reader).Decode(value)
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

func isHelpArgs(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" || arg == "help" {
			return true
		}
	}
	return false
}
