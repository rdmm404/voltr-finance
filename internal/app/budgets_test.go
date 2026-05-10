package app

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"rdmm404/voltr-finance/internal/database/sqlc"
)

func TestMonthlyBudgetPeriodReturnsUTCMonthBounds(t *testing.T) {
	start, end, err := monthlyBudgetPeriod(2026, 5)

	if err != nil {
		t.Fatalf("monthlyBudgetPeriod returned error: %v", err)
	}
	wantStart := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) {
		t.Fatalf("start = %s, want %s", start, wantStart)
	}
	if !end.Equal(wantEnd) {
		t.Fatalf("end = %s, want %s", end, wantEnd)
	}
	if start.Location() != time.UTC || end.Location() != time.UTC {
		t.Fatalf("locations = %s/%s, want UTC", start.Location(), end.Location())
	}
}

func TestMonthlyBudgetPeriodRejectsInvalidMonth(t *testing.T) {
	_, _, err := monthlyBudgetPeriod(2026, 13)

	if appErr, ok := err.(*AppError); !ok || appErr.Code != CodeValidationError {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestGetMonthlyBudgetRejectsMissingOwner(t *testing.T) {
	svc := NewService(&fakeRepo{}, &fakeTransactionService{})

	_, err := svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		Year:  2026,
		Month: 5,
	})

	if appErr, ok := err.(*AppError); !ok || appErr.Code != CodeValidationError {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestGetMonthlyBudgetRejectsHouseholdAndUserOwners(t *testing.T) {
	svc := NewService(&fakeRepo{}, &fakeTransactionService{})

	_, err := svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		HouseholdID: int64Ptr(1),
		UserID:      int64Ptr(2),
		Year:        2026,
		Month:       5,
	})

	if appErr, ok := err.(*AppError); !ok || appErr.Code != CodeValidationError {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestGetMonthlyBudgetReturnsExistingHouseholdBudget(t *testing.T) {
	householdID := int64(7)
	start := pgtype.Date{Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	end := pgtype.Date{Time: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC), Valid: true}
	repo := &fakeRepo{
		householdBudgetByPeriod: sqlc.Budget{
			ID:          11,
			HouseholdID: &householdID,
			PeriodStart: start,
			PeriodEnd:   end,
		},
		budgetLines: []sqlc.BudgetLine{
			{
				ID:               21,
				BudgetID:         11,
				Name:             "Groceries",
				AllocationAmount: pgtype.Numeric{Int: big.NewInt(12550), Exp: -2, Valid: true},
				SortOrder:        1,
			},
		},
		budgetLineCategories: []sqlc.ListBudgetLineCategoriesRow{
			{BudgetID: 11, BudgetLineID: 21, CategoryID: 31, CategoryCode: "groceries", CategoryName: "Groceries"},
		},
	}
	svc := NewService(repo, &fakeTransactionService{})

	budget, err := svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		HouseholdID: &householdID,
		Year:        2026,
		Month:       5,
	})

	if err != nil {
		t.Fatalf("GetMonthlyBudget returned error: %v", err)
	}
	if budget.ID != 11 || budget.HouseholdID == nil || *budget.HouseholdID != householdID || budget.UserID != nil {
		t.Fatalf("budget owner = %+v, want household budget", budget)
	}
	if !budget.PeriodStart.Equal(start.Time) || !budget.PeriodEnd.Equal(end.Time) {
		t.Fatalf("period = %s-%s, want %s-%s", budget.PeriodStart, budget.PeriodEnd, start.Time, end.Time)
	}
	if repo.lastHouseholdBudgetPeriodStart != start.Time {
		t.Fatalf("period start passed to repo = %s, want %s", repo.lastHouseholdBudgetPeriodStart, start.Time)
	}
	if len(budget.Lines) != 1 || budget.Lines[0].AllocationAmount != "125.50" {
		t.Fatalf("lines = %+v, want mapped line with allocation amount string", budget.Lines)
	}
	if len(budget.Lines[0].Categories) != 1 || budget.Lines[0].Categories[0].Code != "groceries" {
		t.Fatalf("categories = %+v, want groceries category", budget.Lines[0].Categories)
	}
}

func TestGetMonthlyBudgetCreatesEmptyHouseholdBudgetWhenMissing(t *testing.T) {
	householdID := int64(7)
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeTransactionService{})

	budget, err := svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		HouseholdID:     &householdID,
		Year:            2026,
		Month:           5,
		CreateIfMissing: true,
	})

	if err != nil {
		t.Fatalf("GetMonthlyBudget returned error: %v", err)
	}
	if budget.ID == 0 || budget.HouseholdID == nil || *budget.HouseholdID != householdID {
		t.Fatalf("budget = %+v, want created household budget", budget)
	}
	wantStart := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	if repo.lastCreateHouseholdBudget.HouseholdID != householdID {
		t.Fatalf("created household id = %d, want %d", repo.lastCreateHouseholdBudget.HouseholdID, householdID)
	}
	if repo.lastCreateHouseholdBudget.PeriodStart.Time != wantStart || repo.lastCreateHouseholdBudget.PeriodEnd.Time != wantEnd {
		t.Fatalf("created period = %s-%s, want %s-%s", repo.lastCreateHouseholdBudget.PeriodStart.Time, repo.lastCreateHouseholdBudget.PeriodEnd.Time, wantStart, wantEnd)
	}
	if repo.lastCreateHouseholdBudget.SourceBudgetID != nil {
		t.Fatalf("source budget id = %v, want nil", repo.lastCreateHouseholdBudget.SourceBudgetID)
	}
	if len(budget.Lines) != 0 {
		t.Fatalf("lines = %+v, want empty budget", budget.Lines)
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
