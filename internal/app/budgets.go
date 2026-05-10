package app

import (
	"context"
	"database/sql"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"rdmm404/voltr-finance/internal/database/sqlc"
)

type GetMonthlyBudgetRequest struct {
	HouseholdID     *int64 `json:"householdId,omitempty"`
	UserID          *int64 `json:"userId,omitempty"`
	Year            int    `json:"year"`
	Month           int    `json:"month"`
	CreateIfMissing bool   `json:"createIfMissing,omitempty"`
}

type BudgetDTO struct {
	ID             int64           `json:"id"`
	HouseholdID    *int64          `json:"householdId,omitempty"`
	UserID         *int64          `json:"userId,omitempty"`
	PeriodStart    time.Time       `json:"periodStart"`
	PeriodEnd      time.Time       `json:"periodEnd"`
	SourceBudgetID *int64          `json:"sourceBudgetId,omitempty"`
	Lines          []BudgetLineDTO `json:"lines"`
}

type BudgetLineDTO struct {
	ID               int64            `json:"id"`
	BudgetID         int64            `json:"budgetId"`
	Name             string           `json:"name"`
	AllocationAmount string           `json:"allocationAmount"`
	SortOrder        int32            `json:"sortOrder"`
	Categories       []CategoryRefDTO `json:"categories"`
}

func (s *Service) GetMonthlyBudget(ctx context.Context, req GetMonthlyBudgetRequest) (BudgetDTO, error) {
	if err := validateBudgetOwner(req.HouseholdID, req.UserID); err != nil {
		return BudgetDTO{}, err
	}

	start, end, err := monthlyBudgetPeriod(req.Year, req.Month)
	if err != nil {
		return BudgetDTO{}, err
	}

	periodStart := pgtype.Date{Time: start, Valid: true}
	periodEnd := pgtype.Date{Time: end, Valid: true}
	budget, err := s.findBudgetByPeriod(ctx, req.HouseholdID, req.UserID, periodStart, periodEnd)
	if err == nil {
		return s.budgetDTO(ctx, budget)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return BudgetDTO{}, mapBudgetError(err)
	}
	if !req.CreateIfMissing {
		return BudgetDTO{}, mapBudgetError(err)
	}

	budget, err = s.createMonthlyBudget(ctx, req.HouseholdID, req.UserID, periodStart, periodEnd)
	if err != nil {
		return BudgetDTO{}, err
	}
	return s.budgetDTO(ctx, budget)
}

func validateBudgetOwner(householdID, userID *int64) error {
	if (householdID == nil) == (userID == nil) {
		return NewError(CodeValidationError, "exactly one budget owner is required", nil)
	}
	return nil
}

func monthlyBudgetPeriod(year int, month int) (time.Time, time.Time, error) {
	if year < 1 {
		return time.Time{}, time.Time{}, NewError(CodeValidationError, "year must be greater than 0", nil)
	}
	if month < 1 || month > 12 {
		return time.Time{}, time.Time{}, NewError(CodeValidationError, "month must be between 1 and 12", nil)
	}
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, -1)
	return start, end, nil
}

func (s *Service) findBudgetByPeriod(ctx context.Context, householdID, userID *int64, periodStart, periodEnd pgtype.Date) (sqlc.Budget, error) {
	if householdID != nil {
		return s.repo.GetHouseholdBudgetByPeriod(ctx, sqlc.GetHouseholdBudgetByPeriodParams{
			HouseholdID: *householdID,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
		})
	}
	return s.repo.GetUserBudgetByPeriod(ctx, sqlc.GetUserBudgetByPeriodParams{
		UserID:      *userID,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	})
}

func (s *Service) createMonthlyBudget(ctx context.Context, householdID, userID *int64, periodStart, periodEnd pgtype.Date) (sqlc.Budget, error) {
	sourceBudget, sourceBudgetID, err := s.latestPriorBudget(ctx, householdID, userID, periodStart)
	if err != nil {
		return sqlc.Budget{}, err
	}

	var budget sqlc.Budget
	if householdID != nil {
		budget, err = s.repo.CreateHouseholdBudget(ctx, sqlc.CreateHouseholdBudgetParams{
			HouseholdID:    *householdID,
			PeriodStart:    periodStart,
			PeriodEnd:      periodEnd,
			SourceBudgetID: sourceBudgetID,
		})
	} else {
		budget, err = s.repo.CreateUserBudget(ctx, sqlc.CreateUserBudgetParams{
			UserID:         *userID,
			PeriodStart:    periodStart,
			PeriodEnd:      periodEnd,
			SourceBudgetID: sourceBudgetID,
		})
	}
	if err != nil {
		return sqlc.Budget{}, mapBudgetError(err)
	}

	if sourceBudgetID != nil {
		if err := s.copyBudgetLines(ctx, sourceBudget.ID, budget.ID); err != nil {
			return sqlc.Budget{}, err
		}
	}
	return budget, nil
}

func (s *Service) latestPriorBudget(ctx context.Context, householdID, userID *int64, periodStart pgtype.Date) (sqlc.Budget, *int64, error) {
	var (
		budget sqlc.Budget
		err    error
	)
	if householdID != nil {
		budget, err = s.repo.GetLatestPriorHouseholdBudget(ctx, sqlc.GetLatestPriorHouseholdBudgetParams{
			HouseholdID: *householdID,
			PeriodStart: periodStart,
		})
	} else {
		budget, err = s.repo.GetLatestPriorUserBudget(ctx, sqlc.GetLatestPriorUserBudgetParams{
			UserID:      *userID,
			PeriodStart: periodStart,
		})
	}
	if errors.Is(err, sql.ErrNoRows) {
		return sqlc.Budget{}, nil, nil
	}
	if err != nil {
		return sqlc.Budget{}, nil, mapBudgetError(err)
	}
	sourceBudgetID := budget.ID
	return budget, &sourceBudgetID, nil
}

func (s *Service) copyBudgetLines(ctx context.Context, sourceBudgetID, targetBudgetID int64) error {
	sourceLines, err := s.repo.ListBudgetLines(ctx, sourceBudgetID)
	if err != nil {
		return mapBudgetError(err)
	}
	sourceMappings, err := s.repo.ListBudgetLineCategories(ctx, sourceBudgetID)
	if err != nil {
		return mapBudgetError(err)
	}

	mappingsByLineID := make(map[int64][]sqlc.ListBudgetLineCategoriesRow)
	for _, mapping := range sourceMappings {
		mappingsByLineID[mapping.BudgetLineID] = append(mappingsByLineID[mapping.BudgetLineID], mapping)
	}

	for _, sourceLine := range sourceLines {
		targetLine, err := s.repo.CreateBudgetLine(ctx, sqlc.CreateBudgetLineParams{
			BudgetID:         targetBudgetID,
			Name:             sourceLine.Name,
			AllocationAmount: sourceLine.AllocationAmount,
			SortOrder:        sourceLine.SortOrder,
		})
		if err != nil {
			return mapBudgetError(err)
		}
		for _, mapping := range mappingsByLineID[sourceLine.ID] {
			err := s.repo.CreateBudgetLineCategory(ctx, sqlc.CreateBudgetLineCategoryParams{
				BudgetID:     targetBudgetID,
				BudgetLineID: targetLine.ID,
				CategoryID:   mapping.CategoryID,
			})
			if err != nil {
				return mapBudgetError(err)
			}
		}
	}
	return nil
}

func (s *Service) budgetDTO(ctx context.Context, budget sqlc.Budget) (BudgetDTO, error) {
	lines, err := s.repo.ListBudgetLines(ctx, budget.ID)
	if err != nil {
		return BudgetDTO{}, mapBudgetError(err)
	}
	categoryRows, err := s.repo.ListBudgetLineCategories(ctx, budget.ID)
	if err != nil {
		return BudgetDTO{}, mapBudgetError(err)
	}

	categoriesByLine := make(map[int64][]CategoryRefDTO)
	for _, row := range categoryRows {
		categoriesByLine[row.BudgetLineID] = append(categoriesByLine[row.BudgetLineID], CategoryRefDTO{
			ID:   row.CategoryID,
			Code: row.CategoryCode,
			Name: row.CategoryName,
		})
	}

	lineDTOs := make([]BudgetLineDTO, 0, len(lines))
	for _, line := range lines {
		lineDTOs = append(lineDTOs, budgetLineDTO(line, categoriesByLine[line.ID]))
	}

	return BudgetDTO{
		ID:             budget.ID,
		HouseholdID:    budget.HouseholdID,
		UserID:         budget.UserID,
		PeriodStart:    budget.PeriodStart.Time,
		PeriodEnd:      budget.PeriodEnd.Time,
		SourceBudgetID: budget.SourceBudgetID,
		Lines:          lineDTOs,
	}, nil
}

func budgetLineDTO(line sqlc.BudgetLine, categories []CategoryRefDTO) BudgetLineDTO {
	if categories == nil {
		categories = []CategoryRefDTO{}
	}
	return BudgetLineDTO{
		ID:               line.ID,
		BudgetID:         line.BudgetID,
		Name:             line.Name,
		AllocationAmount: budgetNumericString(line.AllocationAmount),
		SortOrder:        line.SortOrder,
		Categories:       categories,
	}
}

func parseBudgetNumeric(value string) (pgtype.Numeric, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return pgtype.Numeric{}, NewError(CodeValidationError, "allocation amount is required", nil)
	}
	if strings.HasPrefix(value, "+") {
		value = strings.TrimPrefix(value, "+")
	}
	if strings.HasPrefix(value, "-") {
		return pgtype.Numeric{}, NewError(CodeValidationError, "allocation amount must be a decimal string", nil)
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return pgtype.Numeric{}, NewError(CodeValidationError, "allocation amount must be a decimal string", nil)
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if len(fraction) > 2 {
			return pgtype.Numeric{}, NewError(CodeValidationError, "allocation amount supports at most two decimal places", nil)
		}
	}
	for _, char := range parts[0] + fraction {
		if char < '0' || char > '9' {
			return pgtype.Numeric{}, NewError(CodeValidationError, "allocation amount must be a decimal string", nil)
		}
	}

	for len(fraction) < 2 {
		fraction += "0"
	}
	digits := parts[0] + fraction
	intValue := new(big.Int)
	if _, ok := intValue.SetString(digits, 10); !ok {
		return pgtype.Numeric{}, NewError(CodeValidationError, "allocation amount must be a decimal string", nil)
	}
	return pgtype.Numeric{Int: intValue, Exp: -2, Valid: true}, nil
}

func budgetNumericString(value pgtype.Numeric) string {
	if !value.Valid || value.Int == nil {
		return "0.00"
	}

	ratio := new(big.Rat).SetInt(value.Int)
	if value.Exp > 0 {
		ratio.Mul(ratio, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(value.Exp)), nil)))
	}
	if value.Exp < 0 {
		ratio.Quo(ratio, new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-value.Exp)), nil)))
	}
	return ratio.FloatString(2)
}

func mapBudgetError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return NewError(CodeValidationError, "budget not found", err)
	}
	return NewError(CodeDatabaseError, "database error", err)
}
