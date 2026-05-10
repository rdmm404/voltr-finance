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

func TestMonthlyBudgetPeriodRejectsInvalidYear(t *testing.T) {
	_, _, err := monthlyBudgetPeriod(0, 5)

	if appErr, ok := err.(*AppError); !ok || appErr.Code != CodeValidationError {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestParseBudgetNumericNormalizesToCents(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int64
	}{
		{name: "zero", in: "0", want: 0},
		{name: "whole dollars", in: "1", want: 100},
		{name: "one decimal place", in: "1.2", want: 120},
		{name: "two decimal places", in: "1.23", want: 123},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBudgetNumeric(tt.in)
			if err != nil {
				t.Fatalf("parseBudgetNumeric returned error: %v", err)
			}
			if !got.Valid || got.Exp != -2 || got.Int == nil || got.Int.Cmp(big.NewInt(tt.want)) != 0 {
				t.Fatalf("numeric = %+v, want Int=%d Exp=-2 Valid=true", got, tt.want)
			}
		})
	}
}

func TestParseBudgetNumericRejectsMoreThanTwoDecimalPlaces(t *testing.T) {
	_, err := parseBudgetNumeric("1.234")

	if appErr, ok := err.(*AppError); !ok || appErr.Code != CodeValidationError {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestBudgetNumericStringFormatsTwoDecimalPlaces(t *testing.T) {
	tests := []struct {
		name string
		in   pgtype.Numeric
		want string
	}{
		{name: "zero", in: pgtype.Numeric{Int: big.NewInt(0), Exp: -2, Valid: true}, want: "0.00"},
		{name: "whole dollars", in: pgtype.Numeric{Int: big.NewInt(1), Exp: 0, Valid: true}, want: "1.00"},
		{name: "one decimal place", in: pgtype.Numeric{Int: big.NewInt(12), Exp: -1, Valid: true}, want: "1.20"},
		{name: "two decimal places", in: pgtype.Numeric{Int: big.NewInt(123), Exp: -2, Valid: true}, want: "1.23"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := budgetNumericString(tt.in)
			if got != tt.want {
				t.Fatalf("budgetNumericString = %q, want %q", got, tt.want)
			}
		})
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
			{
				ID:               22,
				BudgetID:         99,
				Name:             "Other Budget",
				AllocationAmount: pgtype.Numeric{Int: big.NewInt(99999), Exp: -2, Valid: true},
				SortOrder:        1,
			},
		},
		budgetLineCategories: []sqlc.ListBudgetLineCategoriesRow{
			{BudgetID: 11, BudgetLineID: 21, CategoryID: 31, CategoryCode: "groceries", CategoryName: "Groceries"},
			{BudgetID: 99, BudgetLineID: 22, CategoryID: 32, CategoryCode: "other", CategoryName: "Other"},
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
	if repo.lastListBudgetLinesBudgetID != 11 || repo.lastListBudgetLineCategoriesBudgetID != 11 {
		t.Fatalf("list budget ids = %d/%d, want 11/11", repo.lastListBudgetLinesBudgetID, repo.lastListBudgetLineCategoriesBudgetID)
	}
	if len(budget.Lines) != 1 || budget.Lines[0].AllocationAmount != "125.50" {
		t.Fatalf("lines = %+v, want mapped line with allocation amount string", budget.Lines)
	}
	if len(budget.Lines[0].Categories) != 1 || budget.Lines[0].Categories[0].Code != "groceries" {
		t.Fatalf("categories = %+v, want groceries category", budget.Lines[0].Categories)
	}
}

func TestGetMonthlyBudgetReturnsExistingUserBudget(t *testing.T) {
	userID := int64(8)
	start := pgtype.Date{Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	end := pgtype.Date{Time: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC), Valid: true}
	repo := &fakeRepo{
		userBudgetByPeriod: sqlc.Budget{
			ID:          12,
			UserID:      &userID,
			PeriodStart: start,
			PeriodEnd:   end,
		},
	}
	svc := NewService(repo, &fakeTransactionService{})

	budget, err := svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		UserID: &userID,
		Year:   2026,
		Month:  5,
	})

	if err != nil {
		t.Fatalf("GetMonthlyBudget returned error: %v", err)
	}
	if budget.ID != 12 || budget.UserID == nil || *budget.UserID != userID || budget.HouseholdID != nil {
		t.Fatalf("budget owner = %+v, want user budget", budget)
	}
	if repo.lastUserBudgetPeriodStart != start.Time {
		t.Fatalf("period start passed to repo = %s, want %s", repo.lastUserBudgetPeriodStart, start.Time)
	}
}

func TestGetMonthlyBudgetReturnsValidationErrorWhenMissingWithoutCreate(t *testing.T) {
	householdID := int64(7)
	svc := NewService(&fakeRepo{}, &fakeTransactionService{})

	_, err := svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		HouseholdID: &householdID,
		Year:        2026,
		Month:       5,
	})

	if appErr, ok := err.(*AppError); !ok || appErr.Code != CodeValidationError {
		t.Fatalf("err = %v, want validation error", err)
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

func TestGetMonthlyBudgetCreatesEmptyUserBudgetWhenMissing(t *testing.T) {
	userID := int64(8)
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeTransactionService{})

	budget, err := svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		UserID:          &userID,
		Year:            2026,
		Month:           5,
		CreateIfMissing: true,
	})

	if err != nil {
		t.Fatalf("GetMonthlyBudget returned error: %v", err)
	}
	if budget.ID == 0 || budget.UserID == nil || *budget.UserID != userID {
		t.Fatalf("budget = %+v, want created user budget", budget)
	}
	wantStart := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	if repo.lastCreateUserBudget.UserID != userID {
		t.Fatalf("created user id = %d, want %d", repo.lastCreateUserBudget.UserID, userID)
	}
	if repo.lastCreateUserBudget.PeriodStart.Time != wantStart || repo.lastCreateUserBudget.PeriodEnd.Time != wantEnd {
		t.Fatalf("created period = %s-%s, want %s-%s", repo.lastCreateUserBudget.PeriodStart.Time, repo.lastCreateUserBudget.PeriodEnd.Time, wantStart, wantEnd)
	}
	if repo.lastCreateUserBudget.SourceBudgetID != nil {
		t.Fatalf("source budget id = %v, want nil", repo.lastCreateUserBudget.SourceBudgetID)
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
