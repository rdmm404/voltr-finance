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

type transactionClient interface {
	CreateTransaction(context.Context, api.CreateTransactionRequest) (api.Transaction, error)
	CreateTransactions(context.Context, api.BulkCreateTransactionsRequest) (api.BulkResult, error)
	ListTransactions(context.Context, api.ListTransactionsQuery) ([]api.Transaction, error)
	UpdateTransaction(context.Context, int64, api.UpdateTransactionRequest) (api.Transaction, error)
	UpdateTransactions(context.Context, api.BulkUpdateTransactionsRequest) (api.BulkResult, error)
	DeleteTransactions(context.Context, api.DeleteTransactionsRequest) (api.BulkResult, error)
	RestoreTransactions(context.Context, api.RestoreTransactionsRequest) (api.BulkResult, error)
}

type userClient interface {
	CreateUser(context.Context, api.CreateUserRequest) (api.User, error)
	UpdateUser(context.Context, int64, api.UpdateUserRequest) (api.User, error)
	GetUser(context.Context, int64) (api.User, error)
	ResolveUser(context.Context, api.IdentitySelector) (api.User, error)
	ListUsers(context.Context) ([]api.User, error)
}

type householdClient interface {
	GetHousehold(context.Context, int64) (api.Household, error)
	ResolveHousehold(context.Context, api.ResolveHouseholdQuery) (api.Household, error)
	ListHouseholds(context.Context) ([]api.Household, error)
	ListHouseholdUsers(context.Context, int64) ([]api.User, error)
}

type categoryClient interface {
	CreateCategory(context.Context, api.CreateCategoryRequest) (api.Category, error)
	ListCategories(context.Context, api.ListCategoriesQuery) ([]api.Category, error)
	UpdateCategory(context.Context, string, api.UpdateCategoryRequest) (api.Category, error)
	DeactivateCategory(context.Context, string) (api.Category, error)
}

type budgetClient interface {
	GetMonthlyBudget(context.Context, api.MonthlyBudgetQuery) (api.Budget, error)
	EnsureMonthlyBudget(context.Context, api.EnsureMonthlyBudgetRequest) (api.Budget, error)
	CreateBudgetLine(context.Context, int64, api.CreateBudgetLineRequest) (api.BudgetLine, error)
	UpdateBudgetLine(context.Context, int64, api.UpdateBudgetLineRequest) (api.BudgetLine, error)
	DeleteBudgetLine(context.Context, int64) error
	GetBudgetReport(context.Context, int64) (api.BudgetReport, error)
}

type APIClient interface {
	transactionClient
	userClient
	householdClient
	categoryClient
	budgetClient
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
	stdin        io.Reader
	stdout       io.Writer
	stderr       io.Writer
	transactions transactionClient
	users        userClient
	households   householdClient
	categories   categoryClient
	budgets      budgetClient
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
	if err := kctx.Run(&runContext{Context: ctx, stdin: stdin, stdout: stdout, stderr: stderr, transactions: client, users: client, households: client, categories: client, budgets: client}); err != nil {
		if isExpectedError(err) {
			fmt.Fprintln(stderr, expectedErrorMessage(err))
			return 2
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
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
