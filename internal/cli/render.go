package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"rdmm404/voltr-finance/internal/app"
)

func RenderJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func RenderTransactionCompact(w io.Writer, tx app.TransactionDTO) error {
	_, err := fmt.Fprintf(w, "Transaction #%d\n", tx.ID)
	if err != nil {
		return err
	}
	lines := []struct {
		label string
		value string
	}{
		{"Amount", fmt.Sprintf("%.2f", tx.Amount)},
		{"Date", tx.TransactionDate.Format("2006-01-02 15:04")},
		{"Author", tx.AuthorName},
		{"Household", stringValue(tx.HouseholdName)},
		{"Description", stringValue(tx.Description)},
		{"Notes", stringValue(tx.Notes)},
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "%s: %s\n", line.label, line.value); err != nil {
			return err
		}
	}
	return nil
}

func RenderTransactionsCSV(w io.Writer, txs []app.TransactionDTO) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{
		"id",
		"amount",
		"transaction_date",
		"author_id",
		"author_name",
		"household_id",
		"household_name",
		"description",
		"notes",
		"created_at",
		"deleted_at",
	}); err != nil {
		return err
	}

	for _, tx := range txs {
		if err := writer.Write([]string{
			strconv.FormatInt(tx.ID, 10),
			fmt.Sprintf("%.2f", tx.Amount),
			formatTime(&tx.TransactionDate),
			strconv.FormatInt(tx.AuthorID, 10),
			tx.AuthorName,
			formatInt(tx.HouseholdID),
			stringValue(tx.HouseholdName),
			stringValue(tx.Description),
			stringValue(tx.Notes),
			formatTime(tx.CreatedAt),
			formatTime(tx.DeletedAt),
		}); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func formatInt(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func formatTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}
