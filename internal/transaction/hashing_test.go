package transaction

import (
	"fmt"
	"testing"
	"time"

	"rdmm404/voltr-finance/internal/database/sqlc"

	"github.com/cespare/xxhash"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jxskiss/base62"
)

func TestGenerateHashForTransactionCreatePreservesLegacyFormatWhenUncategorized(t *testing.T) {
	transactionDate := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	description := "Coffee"
	householdID := int64(2)
	params := sqlc.CreateTransactionParams{
		Amount:          4.25,
		Description:     &description,
		TransactionDate: pgtype.Timestamptz{Time: transactionDate, Valid: true},
		AuthorID:        7,
		HouseholdID:     &householdID,
	}

	hash, err := generateHashForTransactionCreate(params)
	if err != nil {
		t.Fatalf("generateHashForTransactionCreate returned error: %v", err)
	}

	want := legacyTransactionHash(description, transactionDate, 7, householdID, 4.25)
	if hash != want {
		t.Fatalf("hash = %q, want legacy hash %q", hash, want)
	}
}

func TestGenerateHashForTransactionCreateIncludesCategoryWhenSet(t *testing.T) {
	transactionDate := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	description := "Coffee"
	householdID := int64(2)
	categoryID := int64(42)
	params := sqlc.CreateTransactionParams{
		Amount:          4.25,
		CategoryID:      &categoryID,
		Description:     &description,
		TransactionDate: pgtype.Timestamptz{Time: transactionDate, Valid: true},
		AuthorID:        7,
		HouseholdID:     &householdID,
	}

	hash, err := generateHashForTransactionCreate(params)
	if err != nil {
		t.Fatalf("generateHashForTransactionCreate returned error: %v", err)
	}

	legacy := legacyTransactionHash(description, transactionDate, 7, householdID, 4.25)
	withCategory := categorizedTransactionHash(description, transactionDate, 7, householdID, categoryID, 4.25)
	if hash == legacy {
		t.Fatalf("hash = %q, want category to participate in duplicate detection", hash)
	}
	if hash != withCategory {
		t.Fatalf("hash = %q, want categorized hash %q", hash, withCategory)
	}
}

func TestGenerateHashForTransactionUpdatePreservesLegacyFormatWhenStillUncategorized(t *testing.T) {
	transactionDate := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	description := "Coffee"
	householdID := int64(2)
	existing := sqlc.Transaction{
		Amount:          4.25,
		AuthorID:        7,
		Description:     &description,
		TransactionDate: pgtype.Timestamptz{Time: transactionDate, Valid: true},
		HouseholdID:     &householdID,
	}

	hash, err := generateHashForTransactionUpdate(existing, &TransactionUpdate{})
	if err != nil {
		t.Fatalf("generateHashForTransactionUpdate returned error: %v", err)
	}

	want := legacyTransactionHash(description, transactionDate, 7, householdID, 4.25)
	if hash != want {
		t.Fatalf("hash = %q, want legacy hash %q", hash, want)
	}
}

func legacyTransactionHash(description string, transactionDate time.Time, authorID, householdID int64, amount float32) string {
	h := xxhash.New()
	fmt.Fprintf(h, "%s|%d|%d|%d|%.2f", description, transactionDate.Unix(), authorID, householdID, amount)
	return base62.EncodeToString(h.Sum(nil))
}

func categorizedTransactionHash(description string, transactionDate time.Time, authorID, householdID, categoryID int64, amount float32) string {
	h := xxhash.New()
	fmt.Fprintf(h, "%s|%d|%d|%d|%d|%.2f", description, transactionDate.Unix(), authorID, householdID, categoryID, amount)
	return base62.EncodeToString(h.Sum(nil))
}
