package database

import (
	"os"
	"strings"
	"testing"
)

func TestBudgetQueriesPreserveOwnerScopeAndDeterministicPriorCopy(t *testing.T) {
	contents, err := os.ReadFile("query.sql")
	if err != nil {
		t.Fatalf("read query.sql: %v", err)
	}
	query := string(contents)

	required := []string{
		"(b.household_id IS NOT NULL AND t.household_id = b.household_id)",
		"(b.user_id IS NOT NULL AND t.author_id = b.user_id AND t.household_id IS NULL)",
		"t.deleted_at IS NULL",
		"t.transaction_date >= (b.period_start::DATE::TIMESTAMP AT TIME ZONE 'UTC')",
		"t.transaction_date < ((b.period_end::DATE + INTERVAL '1 day')::TIMESTAMP AT TIME ZONE 'UTC')",
		"WHERE blc.budget_id = b.id",
		"AND blc.category_id = t.category_id",
		"ORDER BY period_start DESC, id DESC",
	}
	for _, fragment := range required {
		if !strings.Contains(query, fragment) {
			t.Errorf("query.sql no longer contains required budget behavior %q", fragment)
		}
	}
}
