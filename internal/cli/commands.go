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
}

type CLI struct {
	Transactions TransactionsCmd `cmd:"" help:"Manage transactions."`
	Users        UsersCmd        `cmd:"" help:"Manage users."`
	Households   HouseholdsCmd   `cmd:"" help:"Read households."`
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
	if err := kctx.Run(&runContext{Context: ctx, stdin: stdin, stdout: stdout, stderr: stderr, svc: svc}); err != nil {
		fmt.Fprintln(stderr, err)
		if isExpectedError(err) {
			return 2
		}
		return 1
	}
	return 0
}

type TransactionsCmd struct {
	Create     TransactionCreateCmd     `cmd:"" help:"Create a transaction."`
	CreateBulk TransactionCreateBulkCmd `cmd:"create-bulk" help:"Create transactions from JSON."`
	Update     TransactionUpdateCmd     `cmd:"" help:"Update a transaction."`
	UpdateBulk TransactionUpdateBulkCmd `cmd:"update-bulk" help:"Update transactions from JSON."`
	Get        TransactionGetCmd        `cmd:"" help:"Get transactions."`
	List       TransactionListCmd       `cmd:"" help:"List transactions."`
	Delete     TransactionDeleteCmd     `cmd:"" help:"Soft-delete transactions."`
	Restore    TransactionRestoreCmd    `cmd:"" help:"Restore transactions."`
}

type TransactionCreateCmd struct {
	Amount            float32   `required:"" help:"Transaction amount."`
	TransactionDate   time.Time `required:"" help:"Transaction date/time."`
	Description       *string
	Notes             *string
	BudgetCategoryID  *int64
	HouseholdID       *int64 `required:""`
	AuthorID          *int64
	AuthorDiscordID   *string
	AuthorTelegramID  *string
	AuthorPhoneNumber *string
	AuthorWhatsappID  *string
}

func (c *TransactionCreateCmd) Run(ctx *runContext) error {
	result := ctx.svc.CreateTransaction(ctx.Context, app.CreateTransactionRequest{
		Amount:           c.Amount,
		TransactionDate:  c.TransactionDate,
		Description:      c.Description,
		Notes:            c.Notes,
		BudgetCategoryID: c.BudgetCategoryID,
		HouseholdID:      c.HouseholdID,
		Author:           identity(c.AuthorID, c.AuthorDiscordID, c.AuthorTelegramID, c.AuthorPhoneNumber, c.AuthorWhatsappID),
	})
	return renderWriteResult(ctx.stdout, result)
}

type TransactionCreateBulkCmd struct {
	Input *string `help:"JSON input file. Defaults to stdin."`
}

func (c *TransactionCreateBulkCmd) Run(ctx *runContext) error {
	var req app.BulkCreateTransactionsRequest
	if err := decodeJSONInput(ctx.stdin, c.Input, &req); err != nil {
		return err
	}
	return renderWriteResult(ctx.stdout, ctx.svc.CreateTransactions(ctx.Context, req))
}

type TransactionUpdateCmd struct {
	ID                    int64 `required:""`
	Amount                *float32
	TransactionDate       *time.Time
	Description           *string
	Notes                 *string
	BudgetCategoryID      *int64
	HouseholdID           *int64
	AuthorID              *int64
	AuthorDiscordID       *string
	AuthorTelegramID      *string
	AuthorPhoneNumber     *string
	AuthorWhatsappID      *string
	ClearDescription      bool
	ClearNotes            bool
	ClearBudgetCategoryID bool
	ClearHouseholdID      bool
}

func (c *TransactionUpdateCmd) Run(ctx *runContext) error {
	selector := identity(c.AuthorID, c.AuthorDiscordID, c.AuthorTelegramID, c.AuthorPhoneNumber, c.AuthorWhatsappID)
	req := app.UpdateTransactionRequest{
		ID:                    c.ID,
		Amount:                c.Amount,
		TransactionDate:       c.TransactionDate,
		Description:           c.Description,
		Notes:                 c.Notes,
		BudgetCategoryID:      c.BudgetCategoryID,
		HouseholdID:           c.HouseholdID,
		ClearDescription:      c.ClearDescription,
		ClearNotes:            c.ClearNotes,
		ClearBudgetCategoryID: c.ClearBudgetCategoryID,
		ClearHouseholdID:      c.ClearHouseholdID,
	}
	if selector != (app.IdentitySelector{}) {
		req.Author = &selector
	}
	return renderWriteResult(ctx.stdout, ctx.svc.UpdateTransaction(ctx.Context, req))
}

type TransactionUpdateBulkCmd struct {
	Input *string `help:"JSON input file. Defaults to stdin."`
}

func (c *TransactionUpdateBulkCmd) Run(ctx *runContext) error {
	var req app.BulkUpdateTransactionsRequest
	if err := decodeJSONInput(ctx.stdin, c.Input, &req); err != nil {
		return err
	}
	return renderWriteResult(ctx.stdout, ctx.svc.UpdateTransactions(ctx.Context, req))
}

type TransactionGetCmd struct {
	IDs            string `name:"ids" required:"" help:"Comma-separated transaction IDs."`
	IncludeDeleted bool
	Format         string `default:"json" enum:"json,compact"`
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
	Format         string `default:"json" enum:"json,csv"`
	AuthorID       *int64
	HouseholdID    *int64
	FromDate       *time.Time
	ToDate         *time.Time
	Search         *string
	Sort           string
	Order          string `name:"order"`
	Limit          int32  `default:"100"`
	Offset         int32
	IncludeDeleted bool
	OnlyDeleted    bool
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
	IDs             string `name:"ids" required:"" help:"Comma-separated transaction IDs."`
	Reason          *string
	DeletedByUserID int64 `required:""`
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
	IDs              string `name:"ids" required:"" help:"Comma-separated transaction IDs."`
	RestoredByUserID int64  `required:""`
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

type UsersCmd struct {
	Create  UserCreateCmd  `cmd:"" help:"Create a user."`
	Update  UserUpdateCmd  `cmd:"" help:"Update a user."`
	Get     UserGetCmd     `cmd:"" help:"Get a user."`
	Resolve UserResolveCmd `cmd:"" help:"Resolve a user identity."`
	List    UserListCmd    `cmd:"" help:"List users."`
}

type UserCreateCmd struct {
	Name        string `required:""`
	DiscordID   *string
	TelegramID  *string
	PhoneNumber *string
	WhatsappID  *string
}

func (c *UserCreateCmd) Run(ctx *runContext) error {
	user, err := ctx.svc.CreateUser(ctx.Context, app.CreateUserRequest{Name: c.Name, DiscordID: c.DiscordID, TelegramID: c.TelegramID, PhoneNumber: c.PhoneNumber, WhatsappID: c.WhatsappID})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, user)
}

type UserUpdateCmd struct {
	ID               int64 `required:""`
	Name             *string
	DiscordID        *string
	TelegramID       *string
	PhoneNumber      *string
	WhatsappID       *string
	ClearDiscordID   bool
	ClearTelegramID  bool
	ClearPhoneNumber bool
	ClearWhatsappID  bool
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
	ID int64 `required:""`
}

func (c *UserGetCmd) Run(ctx *runContext) error {
	user, err := ctx.svc.GetUser(ctx.Context, c.ID)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, user)
}

type UserResolveCmd struct {
	AuthorID    *int64
	DiscordID   *string
	TelegramID  *string
	PhoneNumber *string
	WhatsappID  *string
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
	Get   HouseholdGetCmd   `cmd:"" help:"Get a household."`
	List  HouseholdListCmd  `cmd:"" help:"List households."`
	Users HouseholdUsersCmd `cmd:"" help:"List household users."`
}

type HouseholdGetCmd struct {
	ID      *int64
	GuildID *string
	Name    *string
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
	HouseholdID int64 `required:""`
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
