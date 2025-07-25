package tool

import (
	"fmt"
	"rdmm404/voltr-finance/internal/transaction"
	"rdmm404/voltr-finance/internal/utils"

	"github.com/mitchellh/mapstructure"
	"google.golang.org/genai"
)

type SaveTransactionsTool struct{}

func (st SaveTransactionsTool) Name() string {
	return "SaveTransactions"
}
func (st SaveTransactionsTool) Description() string {
	return "This function will store the specified transactions in database."
}
func (st SaveTransactionsTool) Parameters() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"transactions": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"name": {
							Type:        genai.TypeString,
							Description: "Name of the transaction.",
						},
						"description": {
							Type:        genai.TypeString,
							Description: "Description of the transaction. Not required. Should only be set if inferrable from the image.",
							Nullable:    utils.BoolPtr(true),
						},
						"amount":          {Type: genai.TypeNumber},
						"transactionType": {Type: genai.TypeString, Enum: []string{"credit", "debit"}},
					},
				},
			},
		},
	}
}

func (st SaveTransactionsTool) Call(functionCall *genai.FunctionCall, deps *ToolDependencies) *genai.FunctionResponse {
	mappedTransactions := make([]*transaction.Transaction, 0)
	response := genai.FunctionResponse{
		ID:       functionCall.ID,
		Name:     st.Name(),
		Response: make(map[string]any, 0),
	}

	err := st.validateDependencies(deps)

	if err != nil {
		fmt.Printf("SaveTransactions called with invalid deps %v\n", err)
		response.Response["error"] = "Internal error"
		return &response
	}

	transactionsAny, ok := functionCall.Args["transactions"]

	if !ok {
		response.Response["error"] = "Missing argument 'transactions'"
		return &response
	}

	transactions, ok := transactionsAny.([]any)
	if !ok {
		response.Response["error"] = "Invalid format for argument 'transactions'"
		return &response
	}

	for i, trans := range transactions {
		mappedTransaction := transaction.Transaction{}
		err := mapstructure.Decode(trans, &mappedTransaction)

		if err != nil {
			response.Response["error"] = fmt.Sprintf("Invalid format for transaction at index %v", i)
			return &response
		}

		mappedTransactions = append(mappedTransactions, &mappedTransaction)
	}

	err = deps.Ts.SaveTransactions(mappedTransactions)

	if err != nil {
		response.Response["error"] = "Something bad happened :("
	} else {
		response.Response["output"] = "Transactions saved successfully"
	}

	return &response

}

func (st SaveTransactionsTool) validateDependencies(deps *ToolDependencies) error {
	if deps.Ts == nil {
		return fmt.Errorf("transaction service not present in dependencies")
	}

	return nil
}