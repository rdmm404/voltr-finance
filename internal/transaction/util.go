package transaction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"time"
)

type transactionHashParams struct {
	Description     string    `json:",omitempty"`
	TransactionDate time.Time `json:",omitempty"`
	AuthorId        int32     `json:",omitempty"`
	HouseholdId     int32     `json:",omitempty"`
	Amount          float32   `json:",omitempty"`
}

func generateTransactionHash(params transactionHashParams) (string, error) {
	if params.AuthorId == 0 && params.HouseholdId == 0 {
		return "", fmt.Errorf("either authorId or householdId must be set")
	}

	jsonHash, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("error marshalling json %w", err)
	}

	hash := sha256.Sum256(jsonHash)
	return hex.EncodeToString(hash[:]), nil
}

func generateHashForTransactionCreate(transaction sqlc.CreateTransactionParams) (string, error) {
	hashParams := transactionHashParams{
		AuthorId:        transaction.AuthorID,
		Amount:          transaction.Amount,
		TransactionDate: transaction.TransactionDate.Time,
	}

	if transaction.Description != nil {
		hashParams.Description = *transaction.Description
	}

	if transaction.HouseholdID != nil {
		hashParams.HouseholdId = *transaction.HouseholdID
	}

	return generateTransactionHash(hashParams)
}