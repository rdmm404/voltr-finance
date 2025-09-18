package tool

import (
	"context"
	"encoding/json"
	"fmt"
	database "rdmm404/voltr-finance/internal/database/repository"
	"strings"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/jackc/pgx/v5/pgtype"
)

type SaveTransactionsTool struct{}

type SaveTransactionsInput struct {
	Transactions []TransactionSave
}

type TransactionSave struct {
	// required
	Amount float32 `json:"amount" jsonschema_description:"The amount of the transaction."`
	TransactionType int32 `json:"transactionType" jsonschema_description:"The type of the transaction. For personal transactions use 1, For household transactions use 2."`
	PaidBy int32 `json:"paidBy" jsonschema_description:"The ID of the user who originated this transaction. Can be indicated by the human, otherwise you can assume that it's the message sender."`
	TransactionDate time.Time `json:"transactionDate" jsonschema_description:"The date and time of the transaction. Only set if can be inferred by the data provided. IMPORTANT! MUST be in the format YYYY-MM-DDTHH:MM:SS.sTZD."`
	// not required
	HouseholdId *int32 `json:"householdId,omitempty" jsonschema_description:"ID of the household the user belongs to. Only set if the transaction is of type household."`
	Notes *string `json:"notes,omitempty" jsonschema_description:"Notes for this transaction. Add here any relevant information shared BY THE HUMAN regarding this transaction."`
	Description *string `json:"description,omitempty" jsonschema_description:"Description of the transaction."`
	// TODO: owedBy, amountOwed, paymentDate, isPaid
}

func (st SaveTransactionsTool) Name() string {
	return "SaveTransactions"
}
func (st SaveTransactionsTool) Description() string {
	return "This function will store the specified transactions in database."
}

func (st SaveTransactionsTool) Create(g *genkit.Genkit, deps *ToolDependencies) ai.Tool {
	return genkit.DefineTool(
		g,
		st.Name(),
		st.Description(),
		func(ctx *ai.ToolContext, input SaveTransactionsInput) (string, error) {
			err := st.validateDependencies(deps)
			if err != nil {
				fmt.Printf("SaveTransactions called with invalid deps %v\n", err)
				return "", fmt.Errorf("invalid dependencies received %w", err)
			}
			return st.execute(&input, deps)
		},
	)
}

func (st SaveTransactionsTool) execute(input *SaveTransactionsInput, deps *ToolDependencies) (string, error) {
	mappedTransactions := make([]database.CreateTransactionParams, 0)

	for _, transaction := range input.Transactions {
		mappedTransactions = append(mappedTransactions, database.CreateTransactionParams{
			Amount: transaction.Amount,
			// TransactionType: &transaction.TransactionType,
			PaidBy: transaction.PaidBy,
			TransactionDate: pgtype.Timestamptz{
				Time: transaction.TransactionDate,
				Valid: true,
			},
			HouseholdID: transaction.HouseholdId,
			Notes: transaction.Notes,
			// Description: transaction.Description,
		})
	}

	createdTrans, err := deps.Ts.SaveTransactions(context.TODO(), mappedTransactions)

	if err != nil {
		return "", fmt.Errorf("unknown error while saving transactions - %w", err)
	}
	// consider formatting transactions to MD instead
	// TODO: look into returning transaction structs directly instead of formatting
	formattedTrans, err := formatTransactionsForLLM(createdTrans)
	if err != nil {
		fmt.Printf("SaveTransactionsTool: Error received when formatting transactions - %v", err)
		return "", fmt.Errorf("unknown error while reading created transactions. insert was successful %w", err)
	}

	return "The following transactions were successfully stored:\n" + formattedTrans, nil

}

func (st SaveTransactionsTool) validateDependencies(deps *ToolDependencies) error {
	if deps.Ts == nil {
		return fmt.Errorf("transaction service not present in dependencies")
	}

	return nil
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


func formatTransactionsForLLM(transactions map[int32]*database.Transaction) (string, error) {
	var sb strings.Builder
	sb.WriteString("[\n")
	count := 0
	for transId, trans := range transactions {
		count++

		transJson, err := json.MarshalIndent(trans, "  ", "  ")

		if err != nil {
			return "", fmt.Errorf("invalid JSON received for trans with id %v - %w", transId, err)
		}

		sb.WriteString(string(transJson))
		if count != len(transactions) {
			sb.WriteString(",\n")
		}
	}
	sb.WriteString("]")

	return sb.String(), nil
}