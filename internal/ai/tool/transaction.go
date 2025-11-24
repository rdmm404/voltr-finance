package tool

import (
	"errors"
	"fmt"
	"log/slog"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/transaction"
	"rdmm404/voltr-finance/internal/utils"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/jackc/pgx/v5/pgtype"
)

type saveTransactionsTool struct {
	deps *ToolDependencies
}

type SaveTransactionsInput struct {
	Transactions []TransactionSave
}

type TransactionSave struct {
	// required
	Amount          float32  `json:"amount" jsonschema_description:"The amount of the transaction."`
	AuthorID        int32    `json:"authorId" jsonschema_description:"The ID of the user who originated this transaction. Can be indicated by the human, otherwise you can assume that it's the message sender."`
	TransactionDate DateTime `json:"transactionDate" jsonschema_description:"The date and time of the transaction. Only set if can be inferred by the data provided. IMPORTANT! You must format this date in the format YYYY-MM-DD HH:MM:SS."`
	// not required
	HouseholdId *int32  `json:"householdId,omitempty" jsonschema_description:"ID of the household the user belongs to. Only set if the transaction is of type household."`
	Notes       *string `json:"notes,omitempty" jsonschema_description:"Notes for this transaction. Add here any relevant information shared BY THE HUMAN regarding this transaction."`
	Description *string `json:"description,omitempty" jsonschema_description:"Description of the transaction."`
}

func NewSaveTransactionsTool(deps *ToolDependencies) (Tool, error) {
	if deps.Ts == nil {
		return nil, fmt.Errorf("transaction service not present in dependencies")
	}

	return &saveTransactionsTool{deps: deps}, nil
}

func (st *saveTransactionsTool) Name() string {
	return "SaveTransactions"
}
func (st *saveTransactionsTool) Description() string {
	return "This function will store the specified transactions in database."
}

func (st *saveTransactionsTool) Create(g *genkit.Genkit, tp *ToolProvider) ai.Tool {
	return DefineTool(tp, g, st, st.execute)
}

func (st *saveTransactionsTool) execute(ctx *ai.ToolContext, input *SaveTransactionsInput) (string, error) {
	mappedTransactions := make([]sqlc.CreateTransactionParams, 0)

	for _, transaction := range input.Transactions {
		mappedTransactions = append(mappedTransactions, sqlc.CreateTransactionParams{
			Amount:   transaction.Amount,
			AuthorID: transaction.AuthorID,
			TransactionDate: pgtype.Timestamptz{
				Time:  transaction.TransactionDate.Time,
				Valid: true,
			},
			HouseholdID: transaction.HouseholdId,
			Notes:       transaction.Notes,
			Description: transaction.Description,
		})
	}

	result := st.deps.Ts.SaveTransactions(ctx, mappedTransactions)

	return formatResultsForLLM(result), nil

}

type UpdateTransactionsByIdTool struct{}

func (ut UpdateTransactionsByIdTool) Name() string {
	return "UpdateTransactionsById"
}

func (ut UpdateTransactionsByIdTool) Description() string {
	return "This function set the specified data to the transactions with the provided IDs."
}

func (ut UpdateTransactionsByIdTool) Create(g *genkit.Genkit, deps *ToolDependencies) ai.Tool {
	return genkit.DefineTool(
		g,
		ut.Name(),
		ut.Description(),
		func(ctx *ai.ToolContext, input SaveTransactionsInput) (string, error) {
			return "hi", nil
		},
	)
}

func formatResultsForLLM(result transaction.SaveTransactionsResult) (string) {
	var sb strings.Builder

	if len(result.Created) > 0 {
		sb.WriteString(fmt.Sprintf(
			"%v transactions have been created successfully, with ids: %v\n",
			len(result.Created),
			strings.Join(utils.MapKeys(result.Created), ","),
		))
	}

	if len(result.Errors) > 0 {
		slog.Error(fmt.Sprintf("SaveTransactionsTool: received errors %+v", result.Errors))

		sb.WriteString(fmt.Sprintf("%v transactions had errors:\n", len(result.Errors)))
		for _, err := range result.Errors {
			if errors.Is(err.Err, transaction.ErrTransactionValidation) {
				sb.WriteString(fmt.Sprintf("- Transaction #%v: validation failed - %v\n", err.Index, err.Err))
			} else if errors.Is(err.Err, transaction.ErrDuplicateTransaction) {
				sb.WriteString(fmt.Sprintf("- Transaction #%v: already exists with id %s\n", err.Index, err.ID))
			} else {
				sb.WriteString(fmt.Sprintf("- Transaction #%v: %v\n", err.Index, err.Err))
			}
		}
	}

	return sb.String()
}
