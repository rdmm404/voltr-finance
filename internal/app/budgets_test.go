package app

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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

func TestGetMonthlyBudgetCreatesEmptyHouseholdBudgetWhenNoPriorBudget(t *testing.T) {
	householdID := int64(7)
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeTransactionService{})

	budget, err := svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		HouseholdID:     &householdID,
		Year:            2026,
		Month:           7,
		CreateIfMissing: true,
	})

	if err != nil {
		t.Fatalf("GetMonthlyBudget returned error: %v", err)
	}
	if budget.ID == 0 || budget.HouseholdID == nil || *budget.HouseholdID != householdID {
		t.Fatalf("budget = %+v, want created household budget", budget)
	}
	if repo.lastLatestPriorHouseholdBudget.HouseholdID != householdID {
		t.Fatalf("latest prior household id = %d, want %d", repo.lastLatestPriorHouseholdBudget.HouseholdID, householdID)
	}
	wantStart := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	if repo.lastLatestPriorHouseholdBudget.PeriodStart.Time != wantStart {
		t.Fatalf("latest prior period start = %s, want %s", repo.lastLatestPriorHouseholdBudget.PeriodStart.Time, wantStart)
	}
	if repo.lastCreateHouseholdBudget.SourceBudgetID != nil {
		t.Fatalf("source budget id = %v, want nil", repo.lastCreateHouseholdBudget.SourceBudgetID)
	}
	if len(repo.createdBudgetLines) != 0 {
		t.Fatalf("created budget lines = %+v, want none", repo.createdBudgetLines)
	}
}

func TestGetMonthlyBudgetReturnsExistingBudgetAfterConcurrentCreate(t *testing.T) {
	householdID := int64(7)
	julyStart := pgtype.Date{Time: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	julyEnd := pgtype.Date{Time: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC), Valid: true}
	repo := &fakeRepo{
		householdBudgetByPeriodMisses: 1,
		householdBudgetByPeriod: sqlc.Budget{
			ID:          55,
			HouseholdID: &householdID,
			PeriodStart: julyStart,
			PeriodEnd:   julyEnd,
		},
		createHouseholdBudgetErr: &pgconn.PgError{Code: "23505"},
	}
	svc := NewService(repo, &fakeTransactionService{})

	budget, err := svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		HouseholdID:     &householdID,
		Year:            2026,
		Month:           7,
		CreateIfMissing: true,
	})

	if err != nil {
		t.Fatalf("GetMonthlyBudget returned error: %v", err)
	}
	if budget.ID != 55 || budget.HouseholdID == nil || *budget.HouseholdID != householdID {
		t.Fatalf("budget = %+v, want existing concurrently-created budget", budget)
	}
	if repo.lastCreateHouseholdBudget.HouseholdID != householdID {
		t.Fatalf("create household id = %d, want %d", repo.lastCreateHouseholdBudget.HouseholdID, householdID)
	}
}

func TestGetMonthlyBudgetCopiesLatestPriorHouseholdBudgetWhenMissing(t *testing.T) {
	householdID := int64(7)
	sourceID := int64(7)
	targetID := int64(12)
	julyStart := pgtype.Date{Time: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	julyEnd := pgtype.Date{Time: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC), Valid: true}
	groceriesAllocation, err := parseBudgetNumeric("250.25")
	if err != nil {
		t.Fatalf("parseBudgetNumeric returned error: %v", err)
	}
	rentAllocation, err := parseBudgetNumeric("1200.00")
	if err != nil {
		t.Fatalf("parseBudgetNumeric returned error: %v", err)
	}
	repo := &fakeRepo{
		latestPriorHousehold: sqlc.Budget{
			ID:          sourceID,
			HouseholdID: &householdID,
		},
		createdHouseholdBudget: sqlc.Budget{
			ID:             targetID,
			HouseholdID:    &householdID,
			PeriodStart:    julyStart,
			PeriodEnd:      julyEnd,
			SourceBudgetID: &sourceID,
		},
		budgetLines: []sqlc.BudgetLine{
			{ID: 101, BudgetID: sourceID, Name: "Groceries", AllocationAmount: groceriesAllocation, SortOrder: 1},
			{ID: 102, BudgetID: sourceID, Name: "Rent", AllocationAmount: rentAllocation, SortOrder: 2},
		},
		budgetLineCategories: []sqlc.ListBudgetLineCategoriesRow{
			{BudgetID: sourceID, BudgetLineID: 101, CategoryID: 3, CategoryCode: "groceries", CategoryName: "Groceries"},
		},
		createdBudgetLineRows: []sqlc.BudgetLine{
			{ID: 201, BudgetID: targetID, Name: "Groceries", AllocationAmount: groceriesAllocation, SortOrder: 1},
			{ID: 202, BudgetID: targetID, Name: "Rent", AllocationAmount: rentAllocation, SortOrder: 2},
		},
	}
	transactor := &fakeTransactor{repo: repo}
	svc := NewServiceWithTransactor(repo, &fakeTransactionService{}, transactor)

	budget, err := svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		HouseholdID:     &householdID,
		Year:            2026,
		Month:           7,
		CreateIfMissing: true,
	})

	if err != nil {
		t.Fatalf("GetMonthlyBudget returned error: %v", err)
	}
	if budget.SourceBudgetID == nil || *budget.SourceBudgetID != sourceID {
		t.Fatalf("source budget id = %v, want %d", budget.SourceBudgetID, sourceID)
	}
	if transactor.calls != 1 {
		t.Fatalf("transaction calls = %d, want 1", transactor.calls)
	}
	if repo.lastCreateHouseholdBudget.SourceBudgetID == nil || *repo.lastCreateHouseholdBudget.SourceBudgetID != sourceID {
		t.Fatalf("created source budget id = %v, want %d", repo.lastCreateHouseholdBudget.SourceBudgetID, sourceID)
	}
	if len(repo.createdBudgetLines) != 2 {
		t.Fatalf("created budget lines = %+v, want 2 lines", repo.createdBudgetLines)
	}
	wantLines := []sqlc.CreateBudgetLineParams{
		{BudgetID: targetID, Name: "Groceries", AllocationAmount: groceriesAllocation, SortOrder: 1},
		{BudgetID: targetID, Name: "Rent", AllocationAmount: rentAllocation, SortOrder: 2},
	}
	for i, want := range wantLines {
		got := repo.createdBudgetLines[i]
		if got.BudgetID != want.BudgetID || got.Name != want.Name || got.SortOrder != want.SortOrder || budgetNumericString(got.AllocationAmount) != budgetNumericString(want.AllocationAmount) {
			t.Fatalf("created budget line %d = %+v, want %+v", i, got, want)
		}
	}
	if len(repo.createdBudgetLineCategories) != 1 {
		t.Fatalf("created budget line categories = %+v, want one mapping", repo.createdBudgetLineCategories)
	}
	wantCategory := sqlc.CreateBudgetLineCategoryParams{BudgetID: targetID, BudgetLineID: 201, CategoryID: 3}
	if repo.createdBudgetLineCategories[0] != wantCategory {
		t.Fatalf("created budget line category = %+v, want %+v", repo.createdBudgetLineCategories[0], wantCategory)
	}
	if len(budget.Lines) != 2 {
		t.Fatalf("returned lines = %+v, want two copied target lines", budget.Lines)
	}
	if budget.Lines[0].ID != 201 || budget.Lines[0].BudgetID != targetID || budget.Lines[0].Name != "Groceries" || budget.Lines[0].AllocationAmount != "250.25" {
		t.Fatalf("returned first line = %+v, want copied groceries target line", budget.Lines[0])
	}
	if len(budget.Lines[0].Categories) != 1 || budget.Lines[0].Categories[0].ID != 3 || budget.Lines[0].Categories[0].Code != "groceries" {
		t.Fatalf("returned first line categories = %+v, want copied groceries category", budget.Lines[0].Categories)
	}
	if budget.Lines[1].ID != 202 || budget.Lines[1].BudgetID != targetID || budget.Lines[1].Name != "Rent" || budget.Lines[1].AllocationAmount != "1200.00" {
		t.Fatalf("returned second line = %+v, want copied rent target line", budget.Lines[1])
	}
}

func TestGetMonthlyBudgetCopiesLatestPriorUserBudgetWhenMissing(t *testing.T) {
	userID := int64(8)
	sourceID := int64(17)
	targetID := int64(18)
	julyStart := pgtype.Date{Time: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	julyEnd := pgtype.Date{Time: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC), Valid: true}
	utilitiesAllocation, err := parseBudgetNumeric("88.40")
	if err != nil {
		t.Fatalf("parseBudgetNumeric returned error: %v", err)
	}
	repo := &fakeRepo{
		latestPriorUser: sqlc.Budget{
			ID:     sourceID,
			UserID: &userID,
		},
		createdUserBudget: sqlc.Budget{
			ID:             targetID,
			UserID:         &userID,
			PeriodStart:    julyStart,
			PeriodEnd:      julyEnd,
			SourceBudgetID: &sourceID,
		},
		budgetLines: []sqlc.BudgetLine{
			{ID: 301, BudgetID: sourceID, Name: "Utilities", AllocationAmount: utilitiesAllocation, SortOrder: 4},
		},
		budgetLineCategories: []sqlc.ListBudgetLineCategoriesRow{
			{BudgetID: sourceID, BudgetLineID: 301, CategoryID: 9, CategoryCode: "utilities", CategoryName: "Utilities"},
		},
		createdBudgetLineRows: []sqlc.BudgetLine{
			{ID: 401, BudgetID: targetID, Name: "Utilities", AllocationAmount: utilitiesAllocation, SortOrder: 4},
		},
	}
	transactor := &fakeTransactor{repo: repo}
	svc := NewServiceWithTransactor(repo, &fakeTransactionService{}, transactor)

	budget, err := svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		UserID:          &userID,
		Year:            2026,
		Month:           7,
		CreateIfMissing: true,
	})

	if err != nil {
		t.Fatalf("GetMonthlyBudget returned error: %v", err)
	}
	if repo.lastLatestPriorUserBudget.UserID != userID {
		t.Fatalf("latest prior user id = %d, want %d", repo.lastLatestPriorUserBudget.UserID, userID)
	}
	if transactor.calls != 1 {
		t.Fatalf("transaction calls = %d, want 1", transactor.calls)
	}
	if budget.SourceBudgetID == nil || *budget.SourceBudgetID != sourceID {
		t.Fatalf("source budget id = %v, want %d", budget.SourceBudgetID, sourceID)
	}
	if repo.lastCreateUserBudget.SourceBudgetID == nil || *repo.lastCreateUserBudget.SourceBudgetID != sourceID {
		t.Fatalf("created source budget id = %v, want %d", repo.lastCreateUserBudget.SourceBudgetID, sourceID)
	}
	if len(repo.createdBudgetLines) != 1 {
		t.Fatalf("created budget lines = %+v, want one line", repo.createdBudgetLines)
	}
	wantLine := sqlc.CreateBudgetLineParams{BudgetID: targetID, Name: "Utilities", AllocationAmount: utilitiesAllocation, SortOrder: 4}
	gotLine := repo.createdBudgetLines[0]
	if gotLine.BudgetID != wantLine.BudgetID || gotLine.Name != wantLine.Name || gotLine.SortOrder != wantLine.SortOrder || budgetNumericString(gotLine.AllocationAmount) != budgetNumericString(wantLine.AllocationAmount) {
		t.Fatalf("created budget line = %+v, want %+v", gotLine, wantLine)
	}
	if len(repo.createdBudgetLineCategories) != 1 {
		t.Fatalf("created budget line categories = %+v, want one mapping", repo.createdBudgetLineCategories)
	}
	wantCategory := sqlc.CreateBudgetLineCategoryParams{BudgetID: targetID, BudgetLineID: 401, CategoryID: 9}
	if repo.createdBudgetLineCategories[0] != wantCategory {
		t.Fatalf("created budget line category = %+v, want %+v", repo.createdBudgetLineCategories[0], wantCategory)
	}
	if len(budget.Lines) != 1 || budget.Lines[0].ID != 401 || budget.Lines[0].BudgetID != targetID || budget.Lines[0].Name != "Utilities" {
		t.Fatalf("returned lines = %+v, want copied user target line", budget.Lines)
	}
	if len(budget.Lines[0].Categories) != 1 || budget.Lines[0].Categories[0].ID != 9 || budget.Lines[0].Categories[0].Code != "utilities" {
		t.Fatalf("returned categories = %+v, want copied utilities category", budget.Lines[0].Categories)
	}
}

func TestGetMonthlyBudgetReturnsDatabaseErrorWhenCopyNeedsMissingTransactor(t *testing.T) {
	householdID := int64(7)
	sourceID := int64(7)
	amount, err := parseBudgetNumeric("10.00")
	if err != nil {
		t.Fatalf("parseBudgetNumeric returned error: %v", err)
	}
	repo := &fakeRepo{
		latestPriorHousehold: sqlc.Budget{
			ID:          sourceID,
			HouseholdID: &householdID,
		},
		budgetLines: []sqlc.BudgetLine{
			{ID: 101, BudgetID: sourceID, Name: "Groceries", AllocationAmount: amount, SortOrder: 1},
		},
	}
	svc := NewService(repo, &fakeTransactionService{})

	_, err = svc.GetMonthlyBudget(context.Background(), GetMonthlyBudgetRequest{
		HouseholdID:     &householdID,
		Year:            2026,
		Month:           7,
		CreateIfMissing: true,
	})

	if appErr, ok := err.(*AppError); !ok || appErr.Code != CodeDatabaseError {
		t.Fatalf("err = %v, want database error", err)
	}
	if len(repo.createdBudgetLines) != 0 {
		t.Fatalf("created budget lines = %+v, want none", repo.createdBudgetLines)
	}
}

func TestCreateBudgetLineResolvesCategoryCodes(t *testing.T) {
	allocation, err := parseBudgetNumeric("800.00")
	if err != nil {
		t.Fatalf("parseBudgetNumeric returned error: %v", err)
	}
	repo := &fakeRepo{
		budgetByID:     sqlc.Budget{ID: 12},
		categoryByCode: sqlc.Category{ID: 3, Code: "groceries", Name: "Groceries", IsActive: true},
		maxSortOrder:   2,
		createdBudgetLineRows: []sqlc.BudgetLine{
			{ID: 44, BudgetID: 12, Name: "Groceries", AllocationAmount: allocation, SortOrder: 3},
		},
	}
	transactor := &fakeTransactor{repo: repo}
	svc := NewServiceWithTransactor(repo, &fakeTransactionService{}, transactor)

	line, err := svc.CreateBudgetLine(context.Background(), CreateBudgetLineRequest{
		BudgetID:         12,
		Name:             " Groceries ",
		AllocationAmount: "800.00",
		CategoryCodes:    []string{"groceries"},
	})

	if err != nil {
		t.Fatalf("CreateBudgetLine returned error: %v", err)
	}
	if line.ID != 44 || line.BudgetID != 12 || line.Name != "Groceries" || line.SortOrder != 3 || line.AllocationAmount != "800.00" {
		t.Fatalf("line = %+v, want created groceries line", line)
	}
	if len(line.Categories) != 1 || line.Categories[0].ID != 3 || line.Categories[0].Code != "groceries" {
		t.Fatalf("line categories = %+v, want groceries", line.Categories)
	}
	if len(repo.createdBudgetLines) != 1 || repo.createdBudgetLines[0].SortOrder != 3 || budgetNumericString(repo.createdBudgetLines[0].AllocationAmount) != "800.00" {
		t.Fatalf("created budget lines = %+v, want next sort order and normalized amount", repo.createdBudgetLines)
	}
	if len(repo.createdBudgetLineCategories) != 1 || repo.createdBudgetLineCategories[0].CategoryID != 3 {
		t.Fatalf("created mappings = %+v, want category 3", repo.createdBudgetLineCategories)
	}
	if transactor.calls != 1 {
		t.Fatalf("transaction calls = %d, want 1", transactor.calls)
	}
}

func TestCreateBudgetLineRejectsReusedCategory(t *testing.T) {
	repo := &fakeRepo{
		budgetByID:     sqlc.Budget{ID: 12},
		categoryByCode: sqlc.Category{ID: 3, Code: "groceries", Name: "Groceries", IsActive: true},
		budgetLineCategories: []sqlc.ListBudgetLineCategoriesRow{
			{BudgetID: 12, BudgetLineID: 40, CategoryID: 3, CategoryCode: "groceries", CategoryName: "Groceries"},
		},
	}
	transactor := &fakeTransactor{repo: repo}
	svc := NewServiceWithTransactor(repo, &fakeTransactionService{}, transactor)

	_, err := svc.CreateBudgetLine(context.Background(), CreateBudgetLineRequest{
		BudgetID:         12,
		Name:             "Food",
		AllocationAmount: "800.00",
		CategoryCodes:    []string{"groceries"},
	})

	if appErr, ok := err.(*AppError); !ok || appErr.Code != CodeValidationError {
		t.Fatalf("err = %v, want validation error", err)
	}
	if len(repo.createdBudgetLines) != 0 {
		t.Fatalf("created budget lines = %+v, want none", repo.createdBudgetLines)
	}
}

func TestUpdateBudgetLineReplacesOnlyThatLineCategories(t *testing.T) {
	oldAllocation, err := parseBudgetNumeric("800.00")
	if err != nil {
		t.Fatalf("parseBudgetNumeric returned error: %v", err)
	}
	newAllocation, err := parseBudgetNumeric("900.00")
	if err != nil {
		t.Fatalf("parseBudgetNumeric returned error: %v", err)
	}
	repo := &fakeRepo{
		budgetLineByID:    sqlc.BudgetLine{ID: 44, BudgetID: 12, Name: "Groceries", AllocationAmount: oldAllocation, SortOrder: 1},
		categoryByCode:    sqlc.Category{ID: 3, Code: "groceries", Name: "Groceries", IsActive: true},
		updatedBudgetLine: sqlc.BudgetLine{ID: 44, BudgetID: 12, Name: "Groceries", AllocationAmount: newAllocation, SortOrder: 1},
	}
	transactor := &fakeTransactor{repo: repo}
	svc := NewServiceWithTransactor(repo, &fakeTransactionService{}, transactor)

	line, err := svc.UpdateBudgetLine(context.Background(), UpdateBudgetLineRequest{
		LineID:           44,
		AllocationAmount: strPtr("900.00"),
		CategoryCodes:    &[]string{"groceries"},
	})

	if err != nil {
		t.Fatalf("UpdateBudgetLine returned error: %v", err)
	}
	if line.ID != 44 || line.AllocationAmount != "900.00" {
		t.Fatalf("line = %+v, want updated line amount", line)
	}
	if !repo.lastUpdateBudgetLine.SetAllocationAmount || budgetNumericString(repo.lastUpdateBudgetLine.AllocationAmount) != "900.00" {
		t.Fatalf("lastUpdateBudgetLine = %+v, want amount update", repo.lastUpdateBudgetLine)
	}
	if repo.deletedBudgetLineCategoryID != 44 {
		t.Fatalf("deletedBudgetLineCategoryID = %d, want 44", repo.deletedBudgetLineCategoryID)
	}
	if len(repo.createdBudgetLineCategories) != 1 || repo.createdBudgetLineCategories[0].BudgetLineID != 44 {
		t.Fatalf("created mappings = %+v, want mapping for line 44", repo.createdBudgetLineCategories)
	}
	if transactor.calls != 1 {
		t.Fatalf("transaction calls = %d, want 1", transactor.calls)
	}
}

func TestUpdateBudgetLineCanClearCategories(t *testing.T) {
	allocation, err := parseBudgetNumeric("100.00")
	if err != nil {
		t.Fatalf("parseBudgetNumeric returned error: %v", err)
	}
	repo := &fakeRepo{
		budgetLineByID:    sqlc.BudgetLine{ID: 44, BudgetID: 12, Name: "Savings", AllocationAmount: allocation, SortOrder: 1},
		updatedBudgetLine: sqlc.BudgetLine{ID: 44, BudgetID: 12, Name: "Savings", AllocationAmount: allocation, SortOrder: 1},
		budgetLineCategories: []sqlc.ListBudgetLineCategoriesRow{
			{BudgetID: 12, BudgetLineID: 44, CategoryID: 3, CategoryCode: "groceries", CategoryName: "Groceries"},
		},
	}
	transactor := &fakeTransactor{repo: repo}
	svc := NewServiceWithTransactor(repo, &fakeTransactionService{}, transactor)
	emptyCodes := []string{}

	line, err := svc.UpdateBudgetLine(context.Background(), UpdateBudgetLineRequest{
		LineID:        44,
		CategoryCodes: &emptyCodes,
	})

	if err != nil {
		t.Fatalf("UpdateBudgetLine returned error: %v", err)
	}
	if repo.deletedBudgetLineCategoryID != 44 {
		t.Fatalf("deletedBudgetLineCategoryID = %d, want 44", repo.deletedBudgetLineCategoryID)
	}
	if len(repo.createdBudgetLineCategories) != 0 {
		t.Fatalf("created mappings = %+v, want none", repo.createdBudgetLineCategories)
	}
	if len(line.Categories) != 0 {
		t.Fatalf("line categories = %+v, want none", line.Categories)
	}
}

func TestDeleteBudgetLineDeletesLine(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeTransactionService{})

	err := svc.DeleteBudgetLine(context.Background(), 44)

	if err != nil {
		t.Fatalf("DeleteBudgetLine returned error: %v", err)
	}
	if repo.deletedBudgetLineID != 44 {
		t.Fatalf("deletedBudgetLineID = %d, want 44", repo.deletedBudgetLineID)
	}
}

func TestGetBudgetReportDerivesActualsFromCategories(t *testing.T) {
	householdID := int64(1)
	groceriesAllocation, err := parseBudgetNumeric("800.00")
	if err != nil {
		t.Fatalf("parseBudgetNumeric returned error: %v", err)
	}
	savingsAllocation, err := parseBudgetNumeric("500.00")
	if err != nil {
		t.Fatalf("parseBudgetNumeric returned error: %v", err)
	}
	groceriesActual, err := parseBudgetNumeric("570.25")
	if err != nil {
		t.Fatalf("parseBudgetNumeric returned error: %v", err)
	}
	zeroActual, err := parseBudgetNumeric("0.00")
	if err != nil {
		t.Fatalf("parseBudgetNumeric returned error: %v", err)
	}
	uncategorizedActual, err := parseBudgetNumeric("123.45")
	if err != nil {
		t.Fatalf("parseBudgetNumeric returned error: %v", err)
	}
	repo := &fakeRepo{
		budgetByID: sqlc.Budget{
			ID:          12,
			HouseholdID: &householdID,
			PeriodStart: pgtype.Date{Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			PeriodEnd:   pgtype.Date{Time: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC), Valid: true},
		},
		budgetLines: []sqlc.BudgetLine{
			{ID: 44, BudgetID: 12, Name: "Groceries", AllocationAmount: groceriesAllocation, SortOrder: 1},
			{ID: 45, BudgetID: 12, Name: "Savings", AllocationAmount: savingsAllocation, SortOrder: 2},
		},
		budgetReportLines: []sqlc.ListBudgetReportLinesRow{
			{ID: 44, BudgetID: 12, Name: "Groceries", AllocationAmount: groceriesAllocation, ActualAmount: groceriesActual, SortOrder: 1},
			{ID: 45, BudgetID: 12, Name: "Savings", AllocationAmount: savingsAllocation, ActualAmount: zeroActual, SortOrder: 2},
		},
		budgetLineCategories: []sqlc.ListBudgetLineCategoriesRow{
			{BudgetID: 12, BudgetLineID: 44, CategoryID: 3, CategoryCode: "groceries", CategoryName: "Groceries"},
		},
		uncategorizedBudgetTransactions: uncategorizedActual,
	}
	svc := NewService(repo, &fakeTransactionService{})

	report, err := svc.GetBudgetReport(context.Background(), 12)

	if err != nil {
		t.Fatalf("GetBudgetReport returned error: %v", err)
	}
	if len(report.Lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(report.Lines))
	}
	if report.Lines[0].ActualAmount != "570.25" || report.Lines[0].RemainingAmount != "229.75" {
		t.Fatalf("groceries line = %+v, want actual 570.25 remaining 229.75", report.Lines[0])
	}
	if report.Lines[1].ActualAmount != "0.00" || report.Lines[1].RemainingAmount != "500.00" {
		t.Fatalf("savings line = %+v, want zero actual and full remaining", report.Lines[1])
	}
	if report.Totals.AllocationAmount != "1300.00" || report.Totals.ActualAmount != "570.25" || report.Totals.RemainingAmount != "729.75" {
		t.Fatalf("totals = %+v, want allocation 1300 actual 570.25 remaining 729.75", report.Totals)
	}
	if report.Totals.UncategorizedActualAmount != "123.45" {
		t.Fatalf("uncategorized = %q, want 123.45", report.Totals.UncategorizedActualAmount)
	}
	if repo.lastListBudgetReportLinesBudgetID != 12 {
		t.Fatalf("report lines budget id = %d, want 12", repo.lastListBudgetReportLinesBudgetID)
	}
	if repo.lastSumUncategorizedBudgetTransactions != 12 {
		t.Fatalf("uncategorized budget id = %d, want 12", repo.lastSumUncategorizedBudgetTransactions)
	}
}

func TestGetBudgetReportNegativeTransactionsReduceActuals(t *testing.T) {
	householdID := int64(1)
	allocation, err := parseBudgetNumeric("100.00")
	if err != nil {
		t.Fatalf("parseBudgetNumeric returned error: %v", err)
	}
	actual, err := parseBudgetNumeric("60.00")
	if err != nil {
		t.Fatalf("parseBudgetNumeric returned error: %v", err)
	}
	repo := &fakeRepo{
		budgetByID: sqlc.Budget{
			ID:          12,
			HouseholdID: &householdID,
			PeriodStart: pgtype.Date{Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			PeriodEnd:   pgtype.Date{Time: time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC), Valid: true},
		},
		budgetLines: []sqlc.BudgetLine{
			{ID: 44, BudgetID: 12, Name: "Groceries", AllocationAmount: allocation, SortOrder: 1},
		},
		budgetReportLines: []sqlc.ListBudgetReportLinesRow{
			{ID: 44, BudgetID: 12, Name: "Groceries", AllocationAmount: allocation, ActualAmount: actual, SortOrder: 1},
		},
		budgetLineCategories: []sqlc.ListBudgetLineCategoriesRow{
			{BudgetID: 12, BudgetLineID: 44, CategoryID: 3, CategoryCode: "groceries", CategoryName: "Groceries"},
		},
	}
	svc := NewService(repo, &fakeTransactionService{})

	report, err := svc.GetBudgetReport(context.Background(), 12)

	if err != nil {
		t.Fatalf("GetBudgetReport returned error: %v", err)
	}
	if report.Lines[0].ActualAmount != "60.00" || report.Lines[0].RemainingAmount != "40.00" {
		t.Fatalf("line = %+v, want net actual 60.00 remaining 40.00", report.Lines[0])
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
