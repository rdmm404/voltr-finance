package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strconv"
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

type BudgetReportDTO struct {
	Budget BudgetSummaryDTO      `json:"budget"`
	Lines  []BudgetReportLineDTO `json:"lines"`
	Totals BudgetReportTotalsDTO `json:"totals"`
}

type BudgetSummaryDTO struct {
	ID             int64     `json:"id"`
	HouseholdID    *int64    `json:"householdId,omitempty"`
	UserID         *int64    `json:"userId,omitempty"`
	PeriodStart    time.Time `json:"periodStart"`
	PeriodEnd      time.Time `json:"periodEnd"`
	SourceBudgetID *int64    `json:"sourceBudgetId,omitempty"`
}

type BudgetReportLineDTO struct {
	ID               int64            `json:"id"`
	BudgetID         int64            `json:"budgetId"`
	Name             string           `json:"name"`
	AllocationAmount string           `json:"allocationAmount"`
	ActualAmount     string           `json:"actualAmount"`
	RemainingAmount  string           `json:"remainingAmount"`
	SortOrder        int32            `json:"sortOrder"`
	Categories       []CategoryRefDTO `json:"categories"`
}

type BudgetReportTotalsDTO struct {
	AllocationAmount          string `json:"allocationAmount"`
	ActualAmount              string `json:"actualAmount"`
	RemainingAmount           string `json:"remainingAmount"`
	UncategorizedActualAmount string `json:"uncategorizedActualAmount"`
}

type CreateBudgetLineRequest struct {
	BudgetID         int64    `json:"budgetId"`
	Name             string   `json:"name"`
	AllocationAmount string   `json:"allocationAmount"`
	CategoryIDs      []int64  `json:"categoryIds,omitempty"`
	CategoryCodes    []string `json:"categoryCodes,omitempty"`
	SortOrder        *int32   `json:"sortOrder,omitempty"`
}

type UpdateBudgetLineRequest struct {
	LineID           int64     `json:"lineId"`
	Name             *string   `json:"name,omitempty"`
	AllocationAmount *string   `json:"allocationAmount,omitempty"`
	CategoryIDs      *[]int64  `json:"categoryIds,omitempty"`
	CategoryCodes    *[]string `json:"categoryCodes,omitempty"`
	SortOrder        *int32    `json:"sortOrder,omitempty"`
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

	if sourceBudgetID == nil {
		return s.createBudget(ctx, s.repo, householdID, userID, periodStart, periodEnd, nil)
	}
	if s.transactor == nil {
		return sqlc.Budget{}, NewError(CodeDatabaseError, "database error", errors.New("budget copy requires transaction support"))
	}

	var budget sqlc.Budget
	err = s.transactor.WithinTx(ctx, func(repo Repository) error {
		created, err := s.createBudget(ctx, repo, householdID, userID, periodStart, periodEnd, sourceBudgetID)
		if err != nil {
			return err
		}
		if err := s.copyBudgetLines(ctx, repo, sourceBudget.ID, created.ID); err != nil {
			return err
		}
		budget = created
		return nil
	})
	if err != nil {
		return sqlc.Budget{}, err
	}
	return budget, nil
}

func (s *Service) createBudget(ctx context.Context, repo Repository, householdID, userID *int64, periodStart, periodEnd pgtype.Date, sourceBudgetID *int64) (sqlc.Budget, error) {
	if householdID != nil {
		budget, err := repo.CreateHouseholdBudget(ctx, sqlc.CreateHouseholdBudgetParams{
			HouseholdID:    *householdID,
			PeriodStart:    periodStart,
			PeriodEnd:      periodEnd,
			SourceBudgetID: sourceBudgetID,
		})
		if err != nil {
			return sqlc.Budget{}, mapBudgetError(err)
		}
		return budget, nil
	}

	budget, err := repo.CreateUserBudget(ctx, sqlc.CreateUserBudgetParams{
		UserID:         *userID,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		SourceBudgetID: sourceBudgetID,
	})
	if err != nil {
		return sqlc.Budget{}, mapBudgetError(err)
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

func (s *Service) copyBudgetLines(ctx context.Context, repo Repository, sourceBudgetID, targetBudgetID int64) error {
	sourceLines, err := repo.ListBudgetLines(ctx, sourceBudgetID)
	if err != nil {
		return mapBudgetError(err)
	}
	sourceMappings, err := repo.ListBudgetLineCategories(ctx, sourceBudgetID)
	if err != nil {
		return mapBudgetError(err)
	}

	mappingsByLineID := make(map[int64][]sqlc.ListBudgetLineCategoriesRow)
	for _, mapping := range sourceMappings {
		mappingsByLineID[mapping.BudgetLineID] = append(mappingsByLineID[mapping.BudgetLineID], mapping)
	}

	for _, sourceLine := range sourceLines {
		targetLine, err := repo.CreateBudgetLine(ctx, sqlc.CreateBudgetLineParams{
			BudgetID:         targetBudgetID,
			Name:             sourceLine.Name,
			AllocationAmount: sourceLine.AllocationAmount,
			SortOrder:        sourceLine.SortOrder,
		})
		if err != nil {
			return mapBudgetError(err)
		}
		for _, mapping := range mappingsByLineID[sourceLine.ID] {
			err := repo.CreateBudgetLineCategory(ctx, sqlc.CreateBudgetLineCategoryParams{
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

func (s *Service) CreateBudgetLine(ctx context.Context, req CreateBudgetLineRequest) (BudgetLineDTO, error) {
	if req.BudgetID == 0 {
		return BudgetLineDTO{}, NewError(CodeValidationError, "budget id is required", nil)
	}
	if _, err := s.repo.GetBudgetById(ctx, req.BudgetID); err != nil {
		return BudgetLineDTO{}, mapBudgetError(err)
	}
	name, err := validateBudgetLineName(req.Name)
	if err != nil {
		return BudgetLineDTO{}, err
	}
	allocation, err := parseBudgetNumeric(req.AllocationAmount)
	if err != nil {
		return BudgetLineDTO{}, err
	}
	categoryIDs, err := s.resolveLineCategoryIDs(ctx, req.CategoryIDs, req.CategoryCodes)
	if err != nil {
		return BudgetLineDTO{}, err
	}
	if err := s.validateBudgetCategoryAvailability(ctx, req.BudgetID, 0, categoryIDs); err != nil {
		return BudgetLineDTO{}, err
	}
	sortOrder := req.SortOrder
	if sortOrder == nil {
		maxSortOrder, err := s.repo.GetMaxBudgetLineSortOrder(ctx, req.BudgetID)
		if err != nil {
			return BudgetLineDTO{}, mapBudgetError(err)
		}
		nextSortOrder := maxSortOrder + 1
		sortOrder = &nextSortOrder
	}
	line, err := s.repo.CreateBudgetLine(ctx, sqlc.CreateBudgetLineParams{
		BudgetID:         req.BudgetID,
		Name:             name,
		AllocationAmount: allocation,
		SortOrder:        *sortOrder,
	})
	if err != nil {
		return BudgetLineDTO{}, mapBudgetError(err)
	}
	if err := s.replaceBudgetLineCategories(ctx, req.BudgetID, line.ID, categoryIDs); err != nil {
		return BudgetLineDTO{}, err
	}
	return s.budgetLineDTOWithCategories(ctx, line)
}

func (s *Service) UpdateBudgetLine(ctx context.Context, req UpdateBudgetLineRequest) (BudgetLineDTO, error) {
	if req.LineID == 0 {
		return BudgetLineDTO{}, NewError(CodeValidationError, "budget line id is required", nil)
	}
	existing, err := s.repo.GetBudgetLineById(ctx, req.LineID)
	if err != nil {
		return BudgetLineDTO{}, mapBudgetError(err)
	}
	params := sqlc.UpdateBudgetLineParams{ID: req.LineID}
	if req.Name != nil {
		name, err := validateBudgetLineName(*req.Name)
		if err != nil {
			return BudgetLineDTO{}, err
		}
		params.SetName = true
		params.Name = name
	}
	if req.AllocationAmount != nil {
		allocation, err := parseBudgetNumeric(*req.AllocationAmount)
		if err != nil {
			return BudgetLineDTO{}, err
		}
		params.SetAllocationAmount = true
		params.AllocationAmount = allocation
	}
	if req.SortOrder != nil {
		params.SetSortOrder = true
		params.SortOrder = *req.SortOrder
	}
	line, err := s.repo.UpdateBudgetLine(ctx, params)
	if err != nil {
		return BudgetLineDTO{}, mapBudgetError(err)
	}
	if req.CategoryIDs != nil || req.CategoryCodes != nil {
		categoryIDs := []int64(nil)
		categoryCodes := []string(nil)
		if req.CategoryIDs != nil {
			categoryIDs = *req.CategoryIDs
		}
		if req.CategoryCodes != nil {
			categoryCodes = *req.CategoryCodes
		}
		resolvedCategoryIDs, err := s.resolveLineCategoryIDs(ctx, categoryIDs, categoryCodes)
		if err != nil {
			return BudgetLineDTO{}, err
		}
		if err := s.validateBudgetCategoryAvailability(ctx, existing.BudgetID, existing.ID, resolvedCategoryIDs); err != nil {
			return BudgetLineDTO{}, err
		}
		if err := s.replaceBudgetLineCategories(ctx, existing.BudgetID, existing.ID, resolvedCategoryIDs); err != nil {
			return BudgetLineDTO{}, err
		}
	}
	return s.budgetLineDTOWithCategories(ctx, line)
}

func (s *Service) DeleteBudgetLine(ctx context.Context, lineID int64) error {
	if lineID == 0 {
		return NewError(CodeValidationError, "budget line id is required", nil)
	}
	if err := s.repo.DeleteBudgetLine(ctx, lineID); err != nil {
		return mapBudgetError(err)
	}
	return nil
}

func (s *Service) GetBudgetReport(ctx context.Context, budgetID int64) (BudgetReportDTO, error) {
	if budgetID == 0 {
		return BudgetReportDTO{}, NewError(CodeValidationError, "budget id is required", nil)
	}
	budget, err := s.repo.GetBudgetById(ctx, budgetID)
	if err != nil {
		return BudgetReportDTO{}, mapBudgetError(err)
	}
	lines, err := s.repo.ListBudgetLines(ctx, budgetID)
	if err != nil {
		return BudgetReportDTO{}, mapBudgetError(err)
	}
	mappings, err := s.repo.ListBudgetLineCategories(ctx, budgetID)
	if err != nil {
		return BudgetReportDTO{}, mapBudgetError(err)
	}
	actualRows, err := s.repo.ListBudgetTransactions(ctx, sqlc.ListBudgetTransactionsParams{
		PeriodStart: budget.PeriodStart,
		PeriodEnd:   budget.PeriodEnd,
		HouseholdID: budget.HouseholdID,
		UserID:      budget.UserID,
	})
	if err != nil {
		return BudgetReportDTO{}, mapBudgetError(err)
	}
	uncategorized, err := s.repo.SumUncategorizedBudgetTransactions(ctx, sqlc.SumUncategorizedBudgetTransactionsParams{
		PeriodStart: budget.PeriodStart,
		PeriodEnd:   budget.PeriodEnd,
		HouseholdID: budget.HouseholdID,
		UserID:      budget.UserID,
	})
	if err != nil {
		return BudgetReportDTO{}, mapBudgetError(err)
	}

	actualByCategory := make(map[int64]float64, len(actualRows))
	for _, row := range actualRows {
		actualByCategory[row.CategoryID] = float64(row.ActualAmount)
	}
	categoriesByLine := make(map[int64][]CategoryRefDTO)
	categoryIDsByLine := make(map[int64][]int64)
	for _, row := range mappings {
		categoriesByLine[row.BudgetLineID] = append(categoriesByLine[row.BudgetLineID], CategoryRefDTO{
			ID:   row.CategoryID,
			Code: row.CategoryCode,
			Name: row.CategoryName,
		})
		categoryIDsByLine[row.BudgetLineID] = append(categoryIDsByLine[row.BudgetLineID], row.CategoryID)
	}

	reportLines := make([]BudgetReportLineDTO, 0, len(lines))
	totalAllocation := 0.0
	totalActual := 0.0
	for _, line := range lines {
		allocation, err := parseMoney(budgetNumericString(line.AllocationAmount))
		if err != nil {
			return BudgetReportDTO{}, NewError(CodeDatabaseError, "invalid budget allocation amount", err)
		}
		actual := 0.0
		for _, categoryID := range categoryIDsByLine[line.ID] {
			actual += actualByCategory[categoryID]
		}
		totalAllocation += allocation
		totalActual += actual
		reportLines = append(reportLines, BudgetReportLineDTO{
			ID:               line.ID,
			BudgetID:         line.BudgetID,
			Name:             line.Name,
			AllocationAmount: moneyString(allocation),
			ActualAmount:     moneyString(actual),
			RemainingAmount:  moneyString(allocation - actual),
			SortOrder:        line.SortOrder,
			Categories:       categoriesByLine[line.ID],
		})
	}

	return BudgetReportDTO{
		Budget: BudgetSummaryDTO{
			ID:             budget.ID,
			HouseholdID:    budget.HouseholdID,
			UserID:         budget.UserID,
			PeriodStart:    budget.PeriodStart.Time,
			PeriodEnd:      budget.PeriodEnd.Time,
			SourceBudgetID: budget.SourceBudgetID,
		},
		Lines: reportLines,
		Totals: BudgetReportTotalsDTO{
			AllocationAmount:          moneyString(totalAllocation),
			ActualAmount:              moneyString(totalActual),
			RemainingAmount:           moneyString(totalAllocation - totalActual),
			UncategorizedActualAmount: moneyString(float64(uncategorized)),
		},
	}, nil
}

func validateBudgetLineName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", NewError(CodeValidationError, "budget line name is required", nil)
	}
	return name, nil
}

func (s *Service) resolveLineCategoryIDs(ctx context.Context, ids []int64, codes []string) ([]int64, error) {
	seen := make(map[int64]struct{}, len(ids)+len(codes))
	resolved := make([]int64, 0, len(ids)+len(codes))
	for _, id := range ids {
		category, err := s.repo.GetActiveCategoryById(ctx, id)
		if err != nil {
			return nil, mapCategoryError(err)
		}
		if _, ok := seen[category.ID]; ok {
			continue
		}
		seen[category.ID] = struct{}{}
		resolved = append(resolved, category.ID)
	}
	for _, code := range codes {
		category, err := s.repo.GetActiveCategoryByCode(ctx, strings.TrimSpace(code))
		if err != nil {
			return nil, mapCategoryError(err)
		}
		if _, ok := seen[category.ID]; ok {
			continue
		}
		seen[category.ID] = struct{}{}
		resolved = append(resolved, category.ID)
	}
	return resolved, nil
}

func (s *Service) validateBudgetCategoryAvailability(ctx context.Context, budgetID, currentLineID int64, categoryIDs []int64) error {
	existingMappings, err := s.repo.ListBudgetLineCategories(ctx, budgetID)
	if err != nil {
		return mapBudgetError(err)
	}
	wanted := make(map[int64]struct{}, len(categoryIDs))
	for _, categoryID := range categoryIDs {
		wanted[categoryID] = struct{}{}
	}
	for _, mapping := range existingMappings {
		if mapping.BudgetLineID == currentLineID {
			continue
		}
		if _, ok := wanted[mapping.CategoryID]; ok {
			return NewError(CodeValidationError, "category already mapped to another budget line", nil)
		}
	}
	return nil
}

func (s *Service) replaceBudgetLineCategories(ctx context.Context, budgetID, lineID int64, categoryIDs []int64) error {
	if err := s.repo.DeleteBudgetLineCategories(ctx, lineID); err != nil {
		return mapBudgetError(err)
	}
	for _, categoryID := range categoryIDs {
		if err := s.repo.CreateBudgetLineCategory(ctx, sqlc.CreateBudgetLineCategoryParams{
			BudgetID:     budgetID,
			BudgetLineID: lineID,
			CategoryID:   categoryID,
		}); err != nil {
			return mapBudgetError(err)
		}
	}
	return nil
}

func (s *Service) budgetLineDTOWithCategories(ctx context.Context, line sqlc.BudgetLine) (BudgetLineDTO, error) {
	categoryRows, err := s.repo.ListBudgetLineCategories(ctx, line.BudgetID)
	if err != nil {
		return BudgetLineDTO{}, mapBudgetError(err)
	}
	categories := make([]CategoryRefDTO, 0)
	for _, row := range categoryRows {
		if row.BudgetLineID != line.ID {
			continue
		}
		categories = append(categories, CategoryRefDTO{
			ID:   row.CategoryID,
			Code: row.CategoryCode,
			Name: row.CategoryName,
		})
	}
	return budgetLineDTO(line, categories), nil
}

func moneyString(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func parseMoney(value string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(value), 64)
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
