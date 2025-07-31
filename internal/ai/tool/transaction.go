package tool

import (
	"context"
	"fmt"
	database "rdmm404/voltr-finance/internal/database/repository"
	"rdmm404/voltr-finance/internal/transaction"
	"rdmm404/voltr-finance/internal/utils"
	"reflect"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
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
					Required: []string{"description", "amount", "transactionType", "paidBy", "transactionDate"},
					Properties: map[string]*genai.Schema{
						"description": {
							Type:        genai.TypeString,
							Description: "Description of the transaction.",
							Nullable:    utils.BoolPtr(true),
						},
						"amount":          {
							Type: genai.TypeNumber,
							Description: "The amount of the transaction.",
						},
						"transactionType": {
						Type: genai.TypeString,
							Enum: []string{
								fmt.Sprintf("%v", transaction.TransactionTypePersonal),
								fmt.Sprintf("%v", transaction.TransactionTypeHousehold),
							},

							Description: "The type of the transaction. For personal transactions use 1, For household transactions use 2.",
						},
						"paidBy": {
							Type: genai.TypeInteger,
							Description: "The ID of the user who originated this transaction. Can be indicated by the human, otherwise you can assume that it's the message sender.",
						},
						"transactionDate": {
							Type: genai.TypeString,
							Description: "The date and time of the transaction in ISO format. Only set if can be inferred by the data provided. Must be in the format YYYY-MM-DDTHH:MM:SS.",
							Nullable: utils.BoolPtr(true),
						},
						"notes": {
							Type: genai.TypeString,
							Description: "Notes for this transaction. Add here any relevant information shared BY THE HUMAN regarding this transaction.",
							Nullable: utils.BoolPtr(true),
						},
						"householdId": {
							Type: genai.TypeInteger,
							Description: "ID of the household the user belongs to. Only set if the transaction is of type household.",
							Nullable: utils.BoolPtr(true),
						},

						// TODO missing fields
						// owedBy, amountOwed, paymentDate, isPaid
					},
				},
			},
		},
	}
}

func (st SaveTransactionsTool) Call(functionCall *genai.FunctionCall, deps *ToolDependencies) *genai.FunctionResponse {
	mappedTransactions := make([]*database.Transaction, 0)
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
		transMap, ok := trans.(map[string]any)

		if !ok {
			response.Response["error"] = fmt.Sprintf("Invalid format for transaction %v", trans)
			return &response
		}

		if !convertFieldStrToInt(&transMap, "transactionType") {
			response.Response["error"] = fmt.Sprintf("Invalid transaction type received %v", trans)
			return &response
		}

		fmt.Printf("Type of paidby %v", reflect.TypeOf(transMap["paidBy"]))
		paidByFloat, ok := transMap["paidBy"].(float64)
		if ok {
			transMap["paidBy"] = int32(paidByFloat)
		} else if !convertFieldStrToInt(&transMap, "paidBy") {
			response.Response["error"] = fmt.Sprintf("Invalid paidBy ID received %v", trans)
			return &response
		}

		if !convertFieldStrToPgDate(&transMap, "transactionDate") {
			response.Response["error"] = fmt.Sprintf("Invalid transactionDate received %v", trans)
			return &response
		}

		mappedTransaction := database.Transaction{}
		err = mapstructure.Decode(trans, &mappedTransaction)

		if err != nil {
			fmt.Println(err)
			response.Response["error"] = fmt.Sprintf("Invalid format for transaction at index %v", i)
			return &response
		}

		mappedTransactions = append(mappedTransactions, &mappedTransaction)
	}

	err = deps.Ts.SaveTransactions(context.TODO(), mappedTransactions)

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

func convertFieldStrToInt(callArgs *map[string]any, fieldName string) bool {
	callArgsMap := *callArgs
	valueAny, ok := callArgsMap[fieldName]
	if !ok {
		fmt.Printf("\nnot found in map\n")
		return ok
	}

	valueStr, ok := valueAny.(string)
	if !ok {
		fmt.Printf("\nnot a string\n")
		return ok
	}

	valueInt, err := strconv.Atoi(valueStr)
	if err != nil {
		fmt.Printf("\nError while converting %v\n", err)
		return false
	}
	callArgsMap[fieldName] = valueInt
	return true
}

func convertFieldStrToPgDate(callArgs *map[string]any, fieldName string) bool {
	callArgsMap := *callArgs
	valueAny, ok := callArgsMap[fieldName]
	if !ok {
		return ok
	}

	valueStr, ok := valueAny.(string)
	if !ok {
		return ok
	}

	valueTime, err := time.Parse("2006-01-02T15:04:05", valueStr)

	if err != nil {
		return false
	}

	ts := pgtype.Timestamptz{
		Time: valueTime,
		Valid: true,
	}

	callArgsMap[fieldName] = ts

	return true
}