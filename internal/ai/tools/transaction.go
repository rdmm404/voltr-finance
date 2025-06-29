package tools

import (
	"rdmm404/voltr-finance/internal/transaction"
	"rdmm404/voltr-finance/internal/utils"

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

func (st SaveTransactionsTool) Call(functionCall *genai.FunctionCall) *genai.FunctionResponse {
	mappedTransactions := make([]transaction.Transaction, 0)
	err := transaction.SaveTransactions(mappedTransactions)
	response := genai.FunctionResponse{
		ID:   functionCall.ID,
		Name: st.Name(),
	}

	if err != nil {
		response.Response["error"] = "Something bad happened :("
	} else {
		response.Response["output"] = "Transactions saved successfully"
	}

	return &response

}
