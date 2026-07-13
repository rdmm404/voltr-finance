package budgets

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
)

type Owner struct {
	HouseholdID *int64
	UserID      *int64
}

type MonthlyInput struct {
	Owner Owner
	Year  int
	Month int
}

type Budget struct {
	ID             int64
	Owner          Owner
	PeriodStart    time.Time
	PeriodEnd      time.Time
	SourceBudgetID *int64
	Lines          []Line
}

type Line struct {
	ID               int64
	BudgetID         int64
	Name             string
	AllocationAmount string
	SortOrder        int32
	Categories       []Category
}

type Category struct {
	ID   int64
	Code string
	Name string
}

type LineCategory struct {
	BudgetID int64
	LineID   int64
	Category Category
}

type CreateBudget struct {
	Owner          Owner
	PeriodStart    time.Time
	PeriodEnd      time.Time
	SourceBudgetID *int64
}

type CreateLineInput struct {
	BudgetID         int64
	Name             string
	AllocationAmount string
	CategoryIDs      []int64
	CategoryCodes    []string
	SortOrder        *int32
}

type UpdateLineInput struct {
	LineID           int64
	Name             *string
	AllocationAmount *string
	CategoryIDs      *[]int64
	CategoryCodes    *[]string
	SortOrder        *int32
}

type LineUpdate struct {
	Name             *string
	AllocationAmount *string
	SortOrder        *int32
}

type ReportLineData struct {
	Line
	ActualAmount string
}

type UnmappedTransaction struct {
	ID              int64
	TransactionDate time.Time
	Description     *string
	Amount          string
	Category        *Category
}

type Report struct {
	Budget               BudgetSummary
	Lines                []ReportLine
	UnmappedTransactions []UnmappedTransaction
	Totals               ReportTotals
}

type BudgetSummary struct {
	ID             int64
	Owner          Owner
	PeriodStart    time.Time
	PeriodEnd      time.Time
	SourceBudgetID *int64
}

type ReportLine struct {
	Line
	ActualAmount    string
	RemainingAmount string
}

type ReportTotals struct {
	AllocationAmount          string
	ActualAmount              string
	RemainingAmount           string
	UnmappedActualAmount      string
	UncategorizedActualAmount string
}

type EnsureResult struct {
	Budget  Budget
	Created bool
}

// Repository values and callbacks use only application-owned models. Adapters
// translate persistence errors into application errors before returning.
type Repository interface {
	FindMonthly(context.Context, Owner, time.Time, time.Time) (Budget, error)
	FindLatestPrior(context.Context, Owner, time.Time) (Budget, error)
	GetBudget(context.Context, int64) (Budget, error)
	CreateBudget(context.Context, CreateBudget) (Budget, error)
	ListLines(context.Context, int64) ([]Line, error)
	ListLineCategories(context.Context, int64) ([]LineCategory, error)
	GetLine(context.Context, int64) (Line, error)
	MaxSortOrder(context.Context, int64) (int32, error)
	CreateLine(context.Context, CreateLineInput) (Line, error)
	UpdateLine(context.Context, int64, LineUpdate) (Line, error)
	DeleteLine(context.Context, int64) error
	DeleteLineCategories(context.Context, int64) error
	CreateLineCategory(context.Context, int64, int64, int64) error
	GetActiveCategoryByID(context.Context, int64) (Category, error)
	GetActiveCategoryByCode(context.Context, string) (Category, error)
	ListReportLines(context.Context, int64) ([]ReportLineData, error)
	ListUnmappedTransactions(context.Context, int64) ([]UnmappedTransaction, error)
	SumUncategorized(context.Context, int64) (string, error)
}

type Transactor interface {
	WithinTransaction(context.Context, func(Repository) error) error
}

type Service struct {
	repo Repository
	tx   Transactor
}

func NewService(repo Repository, tx Transactor) *Service { return &Service{repo: repo, tx: tx} }

func (s *Service) GetMonthly(ctx context.Context, input MonthlyInput) (Budget, error) {
	start, end, err := validateMonthly(input)
	if err != nil {
		return Budget{}, err
	}
	budget, err := s.repo.FindMonthly(ctx, input.Owner, start, end)
	if err != nil {
		return Budget{}, apperrors.WrapInternal("get monthly budget", err)
	}
	return s.loadBudget(ctx, budget)
}

func (s *Service) EnsureMonthly(ctx context.Context, input MonthlyInput) (EnsureResult, error) {
	start, end, err := validateMonthly(input)
	if err != nil {
		return EnsureResult{}, err
	}
	existing, err := s.repo.FindMonthly(ctx, input.Owner, start, end)
	if err == nil {
		loaded, loadErr := s.loadBudget(ctx, existing)
		return EnsureResult{Budget: loaded}, loadErr
	}
	if !apperrors.IsKind(err, apperrors.KindNotFound) {
		return EnsureResult{}, apperrors.WrapInternal("find monthly budget", err)
	}

	prior, priorErr := s.repo.FindLatestPrior(ctx, input.Owner, start)
	if priorErr != nil && !apperrors.IsKind(priorErr, apperrors.KindNotFound) {
		return EnsureResult{}, apperrors.WrapInternal("find prior budget", priorErr)
	}
	var sourceID *int64
	if priorErr == nil {
		sourceID = pointer(prior.ID)
	}

	var created Budget
	create := func(repo Repository) error {
		item, createErr := repo.CreateBudget(ctx, CreateBudget{Owner: input.Owner, PeriodStart: start, PeriodEnd: end, SourceBudgetID: sourceID})
		if createErr != nil {
			return createErr
		}
		created = item
		if sourceID != nil {
			if err := copyStructure(ctx, repo, prior.ID, created.ID); err != nil {
				return err
			}
		}
		return nil
	}
	if sourceID != nil {
		if s.tx == nil {
			return EnsureResult{}, apperrors.Internal(errors.New("budget copy requires transaction support"))
		}
		err = s.tx.WithinTransaction(ctx, create)
	} else {
		err = create(s.repo)
	}
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			concurrent, findErr := s.repo.FindMonthly(ctx, input.Owner, start, end)
			if findErr == nil {
				loaded, loadErr := s.loadBudget(ctx, concurrent)
				return EnsureResult{Budget: loaded}, loadErr
			}
		}
		return EnsureResult{}, apperrors.WrapInternal("ensure monthly budget", err)
	}
	loaded, err := s.loadBudget(ctx, created)
	return EnsureResult{Budget: loaded, Created: true}, err
}

func (s *Service) CreateLine(ctx context.Context, input CreateLineInput) (Line, error) {
	if input.BudgetID == 0 {
		return Line{}, apperrors.Validation("budget id is required")
	}
	name, err := lineName(input.Name)
	if err != nil {
		return Line{}, err
	}
	amount, err := amountString(input.AllocationAmount)
	if err != nil {
		return Line{}, err
	}
	if s.tx == nil {
		return Line{}, apperrors.Internal(errors.New("budget line changes require transaction support"))
	}
	var created Line
	err = s.tx.WithinTransaction(ctx, func(repo Repository) error {
		if _, err := repo.GetBudget(ctx, input.BudgetID); err != nil {
			return err
		}
		categoryIDs, err := resolveCategoryIDs(ctx, repo, input.CategoryIDs, input.CategoryCodes)
		if err != nil {
			return err
		}
		if err := validateCategoryAvailability(ctx, repo, input.BudgetID, 0, categoryIDs); err != nil {
			return err
		}
		sortOrder := input.SortOrder
		if sortOrder == nil {
			max, err := repo.MaxSortOrder(ctx, input.BudgetID)
			if err != nil {
				return err
			}
			next := max + 1
			sortOrder = &next
		}
		created, err = repo.CreateLine(ctx, CreateLineInput{BudgetID: input.BudgetID, Name: name, AllocationAmount: amount, SortOrder: sortOrder})
		if err != nil {
			return err
		}
		return replaceCategories(ctx, repo, input.BudgetID, created.ID, categoryIDs)
	})
	if err != nil {
		return Line{}, apperrors.WrapInternal("create budget line", err)
	}
	return s.loadLine(ctx, created)
}

func (s *Service) UpdateLine(ctx context.Context, input UpdateLineInput) (Line, error) {
	if input.LineID == 0 {
		return Line{}, apperrors.Validation("budget line id is required")
	}
	update := LineUpdate{SortOrder: input.SortOrder}
	if input.Name != nil {
		name, err := lineName(*input.Name)
		if err != nil {
			return Line{}, err
		}
		update.Name = &name
	}
	if input.AllocationAmount != nil {
		amount, err := amountString(*input.AllocationAmount)
		if err != nil {
			return Line{}, err
		}
		update.AllocationAmount = &amount
	}
	if s.tx == nil {
		return Line{}, apperrors.Internal(errors.New("budget line changes require transaction support"))
	}
	var updated Line
	err := s.tx.WithinTransaction(ctx, func(repo Repository) error {
		existing, err := repo.GetLine(ctx, input.LineID)
		if err != nil {
			return err
		}
		var categoryIDs []int64
		changeCategories := input.CategoryIDs != nil || input.CategoryCodes != nil
		if changeCategories {
			var ids []int64
			var codes []string
			if input.CategoryIDs != nil {
				ids = *input.CategoryIDs
			}
			if input.CategoryCodes != nil {
				codes = *input.CategoryCodes
			}
			categoryIDs, err = resolveCategoryIDs(ctx, repo, ids, codes)
			if err != nil {
				return err
			}
			if err := validateCategoryAvailability(ctx, repo, existing.BudgetID, existing.ID, categoryIDs); err != nil {
				return err
			}
		}
		updated, err = repo.UpdateLine(ctx, input.LineID, update)
		if err != nil {
			return err
		}
		if changeCategories {
			return replaceCategories(ctx, repo, existing.BudgetID, existing.ID, categoryIDs)
		}
		return nil
	})
	if err != nil {
		return Line{}, apperrors.WrapInternal("update budget line", err)
	}
	return s.loadLine(ctx, updated)
}

func (s *Service) DeleteLine(ctx context.Context, id int64) error {
	if id == 0 {
		return apperrors.Validation("budget line id is required")
	}
	return apperrors.WrapInternal("delete budget line", s.repo.DeleteLine(ctx, id))
}

func (s *Service) Report(ctx context.Context, budgetID int64) (Report, error) {
	if budgetID == 0 {
		return Report{}, apperrors.Validation("budget id is required")
	}
	budget, err := s.repo.GetBudget(ctx, budgetID)
	if err != nil {
		return Report{}, apperrors.WrapInternal("get report budget", err)
	}
	rows, err := s.repo.ListReportLines(ctx, budgetID)
	if err != nil {
		return Report{}, apperrors.WrapInternal("list report lines", err)
	}
	mappings, err := s.repo.ListLineCategories(ctx, budgetID)
	if err != nil {
		return Report{}, apperrors.WrapInternal("list report categories", err)
	}
	uncategorized, err := s.repo.SumUncategorized(ctx, budgetID)
	if err != nil {
		return Report{}, apperrors.WrapInternal("sum uncategorized", err)
	}
	unmapped, err := s.repo.ListUnmappedTransactions(ctx, budgetID)
	if err != nil {
		return Report{}, apperrors.WrapInternal("list unmapped transactions", err)
	}

	categories := map[int64][]Category{}
	for _, mapping := range mappings {
		categories[mapping.LineID] = append(categories[mapping.LineID], mapping.Category)
	}
	lines := make([]ReportLine, 0, len(rows))
	totalAllocation, totalActual := int64(0), int64(0)
	for _, row := range rows {
		allocation, err := cents(row.AllocationAmount)
		if err != nil {
			return Report{}, apperrors.Internal(fmt.Errorf("invalid allocation amount: %w", err))
		}
		actual, err := cents(row.ActualAmount)
		if err != nil {
			return Report{}, apperrors.Internal(fmt.Errorf("invalid actual amount: %w", err))
		}
		totalAllocation += allocation
		totalActual += actual
		row.Line.Categories = nonNilCategories(categories[row.ID])
		lines = append(lines, ReportLine{Line: row.Line, ActualAmount: formatCents(actual), RemainingAmount: formatCents(allocation - actual)})
	}
	unmappedTotal := int64(0)
	for i := range unmapped {
		value, err := cents(unmapped[i].Amount)
		if err != nil {
			return Report{}, apperrors.Internal(fmt.Errorf("invalid unmapped amount: %w", err))
		}
		unmapped[i].Amount = formatCents(value)
		unmappedTotal += value
	}
	uncategorizedCents, err := cents(uncategorized)
	if err != nil {
		return Report{}, apperrors.Internal(fmt.Errorf("invalid uncategorized amount: %w", err))
	}
	return Report{
		Budget: BudgetSummary{ID: budget.ID, Owner: budget.Owner, PeriodStart: budget.PeriodStart, PeriodEnd: budget.PeriodEnd, SourceBudgetID: budget.SourceBudgetID},
		Lines:  lines, UnmappedTransactions: nonNilUnmapped(unmapped),
		Totals: ReportTotals{AllocationAmount: formatCents(totalAllocation), ActualAmount: formatCents(totalActual), RemainingAmount: formatCents(totalAllocation - totalActual), UnmappedActualAmount: formatCents(unmappedTotal), UncategorizedActualAmount: formatCents(uncategorizedCents)},
	}, nil
}

func validateMonthly(input MonthlyInput) (time.Time, time.Time, error) {
	if (input.Owner.HouseholdID == nil) == (input.Owner.UserID == nil) {
		return time.Time{}, time.Time{}, apperrors.Validation("exactly one budget owner is required")
	}
	if input.Year < 1 {
		return time.Time{}, time.Time{}, apperrors.Validation("year must be greater than 0")
	}
	if input.Month < 1 || input.Month > 12 {
		return time.Time{}, time.Time{}, apperrors.Validation("month must be between 1 and 12")
	}
	start := time.Date(input.Year, time.Month(input.Month), 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(0, 1, -1), nil
}

func copyStructure(ctx context.Context, repo Repository, sourceID, targetID int64) error {
	lines, err := repo.ListLines(ctx, sourceID)
	if err != nil {
		return err
	}
	mappings, err := repo.ListLineCategories(ctx, sourceID)
	if err != nil {
		return err
	}
	byLine := map[int64][]int64{}
	for _, mapping := range mappings {
		byLine[mapping.LineID] = append(byLine[mapping.LineID], mapping.Category.ID)
	}
	for _, source := range lines {
		sortOrder := source.SortOrder
		created, err := repo.CreateLine(ctx, CreateLineInput{BudgetID: targetID, Name: source.Name, AllocationAmount: source.AllocationAmount, SortOrder: &sortOrder})
		if err != nil {
			return err
		}
		for _, categoryID := range byLine[source.ID] {
			if err := repo.CreateLineCategory(ctx, targetID, created.ID, categoryID); err != nil {
				return err
			}
		}
	}
	return nil
}

func resolveCategoryIDs(ctx context.Context, repo Repository, ids []int64, codes []string) ([]int64, error) {
	seen := map[int64]struct{}{}
	resolved := make([]int64, 0, len(ids)+len(codes))
	for _, id := range ids {
		category, err := repo.GetActiveCategoryByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[category.ID]; !ok {
			seen[category.ID] = struct{}{}
			resolved = append(resolved, category.ID)
		}
	}
	for _, code := range codes {
		category, err := repo.GetActiveCategoryByCode(ctx, strings.TrimSpace(code))
		if err != nil {
			return nil, err
		}
		if _, ok := seen[category.ID]; !ok {
			seen[category.ID] = struct{}{}
			resolved = append(resolved, category.ID)
		}
	}
	return resolved, nil
}

func validateCategoryAvailability(ctx context.Context, repo Repository, budgetID, currentLineID int64, ids []int64) error {
	mappings, err := repo.ListLineCategories(ctx, budgetID)
	if err != nil {
		return err
	}
	wanted := map[int64]struct{}{}
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	for _, mapping := range mappings {
		if mapping.LineID != currentLineID {
			if _, exists := wanted[mapping.Category.ID]; exists {
				return apperrors.Conflict(apperrors.CodeBudgetConflict, "category already mapped to another budget line", nil)
			}
		}
	}
	return nil
}

func replaceCategories(ctx context.Context, repo Repository, budgetID, lineID int64, ids []int64) error {
	if err := repo.DeleteLineCategories(ctx, lineID); err != nil {
		return err
	}
	for _, id := range ids {
		if err := repo.CreateLineCategory(ctx, budgetID, lineID, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) loadBudget(ctx context.Context, budget Budget) (Budget, error) {
	lines, err := s.repo.ListLines(ctx, budget.ID)
	if err != nil {
		return Budget{}, apperrors.WrapInternal("list budget lines", err)
	}
	mappings, err := s.repo.ListLineCategories(ctx, budget.ID)
	if err != nil {
		return Budget{}, apperrors.WrapInternal("list budget categories", err)
	}
	byLine := map[int64][]Category{}
	for _, mapping := range mappings {
		byLine[mapping.LineID] = append(byLine[mapping.LineID], mapping.Category)
	}
	for i := range lines {
		lines[i].Categories = nonNilCategories(byLine[lines[i].ID])
	}
	budget.Lines = nonNilLines(lines)
	return budget, nil
}

func (s *Service) loadLine(ctx context.Context, line Line) (Line, error) {
	mappings, err := s.repo.ListLineCategories(ctx, line.BudgetID)
	if err != nil {
		return Line{}, apperrors.WrapInternal("list line categories", err)
	}
	line.Categories = []Category{}
	for _, mapping := range mappings {
		if mapping.LineID == line.ID {
			line.Categories = append(line.Categories, mapping.Category)
		}
	}
	return line, nil
}

func lineName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", apperrors.Validation("budget line name is required")
	}
	return value, nil
}

func amountString(value string) (string, error) {
	parsed, err := cents(value)
	if err != nil || parsed < 0 {
		return "", apperrors.Validation("allocation amount must be a non-negative number with at most two decimal places")
	}
	return formatCents(parsed), nil
}

func cents(value string) (int64, error) {
	ratio, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	if !ok {
		return 0, errors.New("invalid decimal")
	}
	ratio.Mul(ratio, big.NewRat(100, 1))
	if !ratio.IsInt() || !ratio.Num().IsInt64() {
		return 0, errors.New("amount has more than two decimal places or is out of range")
	}
	return ratio.Num().Int64(), nil
}

func formatCents(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	return fmt.Sprintf("%s%d.%02d", sign, value/100, value%100)
}

func pointer(value int64) *int64 { return &value }
func nonNilCategories(items []Category) []Category {
	if items == nil {
		return []Category{}
	}
	return items
}
func nonNilLines(items []Line) []Line {
	if items == nil {
		return []Line{}
	}
	return items
}
func nonNilUnmapped(items []UnmappedTransaction) []UnmappedTransaction {
	if items == nil {
		return []UnmappedTransaction{}
	}
	return items
}
