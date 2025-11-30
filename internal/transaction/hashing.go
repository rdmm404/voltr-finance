package transaction

import (
	"fmt"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/utils"
	"time"

	"github.com/cespare/xxhash"
	"github.com/jxskiss/base62"
)

func generateTransactionHash(
	description string,
	transactionDate time.Time,
	authorId, householdId int64,
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
	return generateTransactionHash(
		utils.ValueOrZero(transaction.Description),
		transaction.TransactionDate.Time,
		transaction.AuthorID,
		utils.ValueOrZero(transaction.HouseholdID),
		transaction.Amount,
	)
}

func generateHashForTransactionUpdate(transaction sqlc.Transaction, updates *TransactionUpdate) (string, error) {
	description := transaction.Description
	if updates.Description.Set {
		description = updates.Description.Value
	}

	householdId := transaction.HouseholdID
	if updates.HouseholdID.Set {
		householdId = updates.HouseholdID.Value
	}

	transactionDate := transaction.TransactionDate.Time
	if updates.TransactionDate.Set {
		transactionDate = updates.TransactionDate.Value
	}

	authorId := transaction.AuthorID
	if updates.AuthorID.Set {
		authorId = updates.AuthorID.Value
	}

	amount := transaction.Amount
	if updates.Amount.Set {
		amount = updates.Amount.Value
	}

	return generateTransactionHash(
		utils.ValueOrZero(description),
		transactionDate,
		authorId,
		utils.ValueOrZero(householdId),
		amount,
	)
}
