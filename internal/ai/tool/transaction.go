package tool

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/transaction"
	"rdmm404/voltr-finance/internal/utils"
	"strings"
	"time"

	"github.com/firebase/genkit/go/ai"
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
	HouseholdID *int64  `json:"householdId,omitempty" jsonschema_description:"ID of the household the user belongs to. Set to null if transaction is personal."`
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

func (st *saveTransactionsTool) Create(tp *ToolProvider) ai.Tool {
	return DefineTool(tp, st, st.execute)
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
			HouseholdID: transaction.HouseholdID,
			Notes:       transaction.Notes,
			Description: transaction.Description,
		})
	}

	result := st.deps.Ts.SaveTransactions(ctx, mappedTransactions)

	return formatTransactionResultForLLM(result, "transactions have been created successfully"), nil
}

type updateTransactionsByIdTool struct {
	deps *ToolDependencies
}

type UpdateTransactionsByIdInput struct {
	TransactionUpdates []TransactionUpdateById
}

type TransactionUpdateById struct {
	ID      int64
	Updates TransactionSave
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
	return "Use this function to set the specified data to the transactions with the provided IDs. You must provide complete data regarding the transaction, if you don't have it please use the GetTransactions tool to get it."
}

func (ut *updateTransactionsByIdTool) Create(tp *ToolProvider) ai.Tool {
	return DefineTool(tp, ut, ut.execute)
}

func (ut *updateTransactionsByIdTool) execute(ctx *ai.ToolContext, input UpdateTransactionsByIdInput) (string, error) {
	params := make([]transaction.UpdateTransactionById, 0, len(input.TransactionUpdates))

	for _, update := range input.TransactionUpdates {
		params = append(params, transaction.UpdateTransactionById{
			ID: update.ID,
			Updates: &transaction.TransactionUpdate{
				Amount:          utils.NewOptional(update.Updates.Amount),
				AuthorID:        utils.NewOptional(update.Updates.AuthorID),
				Description:     utils.NewOptional(update.Updates.Description),
				TransactionDate: utils.NewOptional(time.Time(update.Updates.TransactionDate.Time)),
				Notes:           utils.NewOptional(update.Updates.Notes),
				HouseholdID:     utils.NewOptional(update.Updates.HouseholdID),
			},
		})
	}

	result := ut.deps.Ts.UpdateTransactionsById(ctx, params)

	return formatTransactionResultForLLM(result, "transactions have been updated successfully"), nil
}

type getTransactionsTool struct {
	deps *ToolDependencies
}

type GetTransactionsInput struct {
	TransactionIds []int64
}

func NewGetTransactionsTool(deps *ToolDependencies) (Tool, error) {
	if deps.Ts == nil {
		return nil, fmt.Errorf("transaction service not present in dependencies")
	}

	return &getTransactionsTool{deps: deps}, nil
}

func (gt *getTransactionsTool) Name() string {
	return "GetTransactions"
}

func (gt *getTransactionsTool) Description() string {
	return "Get the details for the transactions with the provided IDs. If you have multiple transactions you want to get the details for, call this tool ONCE with a list of IDs."
}

func (gt *getTransactionsTool) Create(tp *ToolProvider) ai.Tool {
	return DefineTool(tp, gt, gt.execute)
}

func (gt *getTransactionsTool) execute(ctx *ai.ToolContext, input GetTransactionsInput) (string, error) {
	trans, err := gt.deps.Ts.GetTransactionsById(ctx, input.TransactionIds)

	if errors.Is(err, transaction.ErrTransactionNotFound) {
		return fmt.Sprintf("Transactions with ids %q not found", input.TransactionIds), nil
	}

	if err != nil {
		slog.Error("GetTransaction: db error", "error", err)
		return fmt.Sprintf("an error ocurred: %v", err), nil
	}

	var output strings.Builder

	output.WriteString("The following transactions were found:\n")
	foundTrans, err := json.Marshal(trans)

	if err != nil {
		return "", fmt.Errorf("error while marshaling transactions: %w", err)
	}

	output.Write(foundTrans)

	if len(input.TransactionIds) > len(trans) {
		output.WriteString("\n No transactions were found for IDs: ")
		for _, id := range input.TransactionIds {
			if _, ok := trans[id]; ok {
				continue
			}

			output.WriteString(fmt.Sprintf("%q,", id))
		}
	}

	return output.String(), nil
}

func formatTransactionResultForLLM(result transaction.TransactionResult, successMsg string) string {
	var sb strings.Builder

	if len(result.Success) > 0 {
		sb.WriteString(fmt.Sprintf(
			"%v %s, with ids: %v\n",
			len(result.Success),
			successMsg,
			utils.MapKeys(result.Success)),
		)
	}

	if len(result.Errors) > 0 {
		slog.Error(fmt.Sprintf("formatResultsForLLM: received errors %+v", result.Errors))

		sb.WriteString(fmt.Sprintf("%v transactions had errors:\n", len(result.Errors)))
		for _, err := range result.Errors {
			if errors.Is(err.Err, transaction.ErrTransactionValidation) {
				sb.WriteString(fmt.Sprintf("- Transaction #%v: validation failed - %v\n", err.Index, err.Err))
			} else if errors.Is(err.Err, transaction.ErrDuplicateTransaction) {
				sb.WriteString(fmt.Sprintf("- Transaction #%v: already exists with id %v\n", err.Index, err.ID))
			} else if errors.Is(err.Err, transaction.ErrTransactionNotFound) {
				sb.WriteString(fmt.Sprintf("- Transaction #%v: transaction with id %v was not found\n", err.Index, err.ID))
			} else {
				sb.WriteString(fmt.Sprintf("- Transaction #%v: %v\n", err.Index, err.Err))
			}
		}
	}

	return sb.String()
}
