package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"rdmm404/voltr-finance/internal/api"
)

func TestRenderSingleTransactionCompact(t *testing.T) {
	description := "Groceries"
	notes := "Costco"
	householdID := int64(1)
	tx := api.Transaction{
		ID:              101,
		Amount:          42.5,
		TransactionDate: time.Date(2026, 5, 5, 14, 30, 0, 0, time.UTC),
		AuthorName:      "Rafael",
		HouseholdID:     &householdID,
		HouseholdName:   strPtr("Home"),
		Description:     &description,
		Notes:           &notes,
	}

	var out bytes.Buffer
	if err := RenderTransactionCompact(&out, tx); err != nil {
		t.Fatalf("RenderTransactionCompact returned error: %v", err)
	}

	want := strings.Join([]string{
		"Transaction #101",
		"Amount: 42.50",
		"Date: 2026-05-05 14:30",
		"Author: Rafael",
		"Household: Home",
		"Category: ",
		"Description: Groceries",
		"Notes: Costco",
		"",
	}, "\n")
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestRenderTransactionsCSVUsesStableColumns(t *testing.T) {
	tx := api.Transaction{
		ID:              101,
		Amount:          42.5,
		TransactionDate: time.Date(2026, 5, 5, 14, 30, 0, 0, time.UTC),
		AuthorID:        7,
		AuthorName:      "Rafael",
		HouseholdID:     intPtr(1),
		HouseholdName:   strPtr("Home"),
		Description:     strPtr("Groceries"),
		Notes:           strPtr("Costco"),
		CreatedAt:       timePtr(time.Date(2026, 5, 5, 15, 0, 0, 0, time.UTC)),
	}

	var out bytes.Buffer
	if err := RenderTransactionsCSV(&out, []api.Transaction{tx}); err != nil {
		t.Fatalf("RenderTransactionsCSV returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	wantHeader := "id,amount,transaction_date,author_id,author_name,household_id,household_name,category_code,category_name,description,notes,created_at,deleted_at"
	if lines[0] != wantHeader {
		t.Fatalf("header = %q, want %q", lines[0], wantHeader)
	}
}

func TestRenderBulkResultJSON(t *testing.T) {
	result := api.BulkResult{Succeeded: []api.BulkSucceeded{{Index: 0, ID: 101}}, Failed: []api.BulkFailed{}}
	var out bytes.Buffer
	if err := RenderJSON(&out, result); err != nil {
		t.Fatal(err)
	}
	var decoded api.BulkResult
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Succeeded) != 1 || decoded.Succeeded[0].ID != 101 {
		t.Fatalf("decoded=%+v", decoded)
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func strPtr(value string) *string {
	return &value
}

func intPtr(value int64) *int64 {
	return &value
}
