package transaction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
