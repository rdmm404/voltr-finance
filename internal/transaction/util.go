package transaction

import (
	"fmt"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"time"

	"github.com/cespare/xxhash"
	"github.com/jxskiss/base62"
)

func generateTransactionHash(
	description string,
	transactionDate time.Time,
	authorId, householdId int32,
	amount float32,
) (string, error) {
	if authorId == 0 && householdId == 0 {
		return "", fmt.Errorf("either authorId or householdId must be set")
	}

	h := xxhash.New()

	fmt.Fprintf(h, "%s|%d|%d|%d|%.2f",
		description,
		transactionDate.Unix(),
		authorId,
		householdId,
		amount,
	)

	return base62.EncodeToString(h.Sum(nil)), nil
}

func generateHashForTransactionCreate(transaction sqlc.CreateTransactionParams) (string, error) {
	var description string
	if transaction.Description != nil {
		description = *transaction.Description
	}

	var householdId int32
	if transaction.HouseholdID != nil {
		householdId = *transaction.HouseholdID
	}

	return generateTransactionHash(
		description,
		transaction.TransactionDate.Time,
		transaction.AuthorID,
		householdId,
		transaction.Amount,
	)
}