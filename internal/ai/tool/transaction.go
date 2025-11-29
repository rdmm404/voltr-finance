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
	AuthorID        int64    `json:"authorId" jsonschema_description:"The ID of the user who originated this transaction. Can be indicated by the human, otherwise you can assume that it's the message sender."`
	TransactionDate DateTime `json:"transactionDate" jsonschema_description:"The date and time of the transaction. Only set if can be inferred by the data provided. IMPORTANT! You must format this date in the format YYYY-MM-DD HH:MM:SS."`
	// not required
	HouseholdId *int64  `json:"householdId,omitempty" jsonschema_description:"ID of the household the user belongs to. Only set if the transaction is of type household."`
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

func formatResultsForLLM(result transaction.SaveTransactionsResult) string {
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


type updateTransactionsByIdTool struct {
	deps *ToolDependencies
}

type UpdateTransactionsByIdInput struct {
	TransactionUpdates []TransactionUpdateById
}

type TransactionUpdateById struct {
	ID string
	Updates TransactionUpdate
}

type TransactionUpdate struct {
	Amount          utils.Partial[float32]  `json:"amount,omitempty" jsonschema_description:"The amount of the transaction."`
	AuthorID        utils.Partial[int64]    `json:"authorId,omitempty" jsonschema_description:"The ID of the user who originated this transaction. Can be indicated by the human, otherwise you can assume that it's the message sender."`
	TransactionDate utils.Partial[DateTime] `json:"transactionDate,omitempty" jsonschema_description:"The date and time of the transaction. Only set if can be inferred by the data provided. IMPORTANT! You must format this date in the format YYYY-MM-DD HH:MM:SS."`
	HouseholdId utils.Partial[*int64]  `json:"householdId,omitempty" jsonschema_extras:"nullable=true" jsonschema_description:"ID of the household the user belongs to. Only set if the transaction is of type household."`
	Notes       utils.Partial[*string] `json:"notes,omitempty" jsonschema_extras:"nullable=true" jsonschema_description:"Notes for this transaction. Add here any relevant information shared BY THE HUMAN regarding this transaction."`
	Description utils.Partial[*string] `json:"description,omitempty" jsonschema_extras:"nullable=true" jsonschema_description:"Description of the transaction."`
}

func NewUpdateTransactionsByIdTool(deps *ToolDependencies) (Tool, error) {
	if deps.Ts == nil {
		return nil, fmt.Errorf("transaction service not present in dependencies")
	}

	return &updateTransactionsByIdTool{deps: deps}, nil
}

func (ut *updateTransactionsByIdTool) Name() string {
	return "UpdateTransactionsById"
}

func (ut *updateTransactionsByIdTool) Description() string {
	return "Use this function to set the specified data to the transactions with the provided IDs. IMPORTANT: This is a partial update so you only need to provide the fields you want to update. Do not include any unnecessary fields."
}

func (ut *updateTransactionsByIdTool) Create(g *genkit.Genkit, tp *ToolProvider) ai.Tool {
	return DefineTool(tp, g, ut, ut.execute)
}

func (ut *updateTransactionsByIdTool) execute(ctx *ai.ToolContext, input UpdateTransactionsByIdInput) (string, error) {
	slog.Info("update transaction tools called", "input", utils.JsonMarshalIgnore(input))
	return "", nil
}
