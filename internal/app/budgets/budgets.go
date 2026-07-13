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

type CreateMonthlyFromTemplateInput struct {
	Owner       Owner
	PeriodStart time.Time
	PeriodEnd   time.Time
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

type ReportSnapshot struct {
	Budget               Budget
	Lines                []ReportLineData
	UnmappedTransactions []UnmappedTransaction
	UncategorizedAmount  string
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

// Repository exposes use-case-level persistence operations. Implementations own
// transaction boundaries, locking, aggregate loading, and join-table mechanics.
type Repository interface {
	FindMonthly(context.Context, Owner, time.Time, time.Time) (Budget, error)
	CreateMonthlyFromTemplate(context.Context, CreateMonthlyFromTemplateInput) (Budget, error)
	CreateLineWithCategories(context.Context, CreateLineInput) (Line, error)
	UpdateLineWithCategories(context.Context, UpdateLineInput) (Line, error)
	DeleteLine(context.Context, int64) error
	LoadReportSnapshot(context.Context, int64) (ReportSnapshot, error)
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) GetMonthly(ctx context.Context, input MonthlyInput) (Budget, error) {
	start, end, err := validateMonthly(input)
	if err != nil {
		return Budget{}, err
	}
	budget, err := s.repo.FindMonthly(ctx, input.Owner, start, end)
	if err != nil {
		return Budget{}, apperrors.WrapInternal("get monthly budget", err)
	}
	return normalizeBudget(budget), nil
}

func (s *Service) EnsureMonthly(ctx context.Context, input MonthlyInput) (EnsureResult, error) {
	start, end, err := validateMonthly(input)
	if err != nil {
		return EnsureResult{}, err
	}
	existing, err := s.repo.FindMonthly(ctx, input.Owner, start, end)
	if err == nil {
		return EnsureResult{Budget: normalizeBudget(existing)}, nil
	}
	if !apperrors.IsKind(err, apperrors.KindNotFound) {
		return EnsureResult{}, apperrors.WrapInternal("find monthly budget", err)
	}

	created, err := s.repo.CreateMonthlyFromTemplate(ctx, CreateMonthlyFromTemplateInput{Owner: input.Owner, PeriodStart: start, PeriodEnd: end})
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			concurrent, findErr := s.repo.FindMonthly(ctx, input.Owner, start, end)
			if findErr == nil {
				return EnsureResult{Budget: normalizeBudget(concurrent)}, nil
			}
		}
		return EnsureResult{}, apperrors.WrapInternal("ensure monthly budget", err)
	}
	return EnsureResult{Budget: normalizeBudget(created), Created: true}, nil
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
	input.Name, input.AllocationAmount = name, amount
	line, err := s.repo.CreateLineWithCategories(ctx, input)
	if err != nil {
		return Line{}, apperrors.WrapInternal("create budget line", err)
	}
	line.Categories = nonNilCategories(line.Categories)
	return line, nil
}

func (s *Service) UpdateLine(ctx context.Context, input UpdateLineInput) (Line, error) {
	if input.LineID == 0 {
		return Line{}, apperrors.Validation("budget line id is required")
	}
	if input.Name != nil {
		name, err := lineName(*input.Name)
		if err != nil {
			return Line{}, err
		}
		input.Name = &name
	}
	if input.AllocationAmount != nil {
		amount, err := amountString(*input.AllocationAmount)
		if err != nil {
			return Line{}, err
		}
		input.AllocationAmount = &amount
	}
	line, err := s.repo.UpdateLineWithCategories(ctx, input)
	if err != nil {
		return Line{}, apperrors.WrapInternal("update budget line", err)
	}
	line.Categories = nonNilCategories(line.Categories)
	return line, nil
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
	snapshot, err := s.repo.LoadReportSnapshot(ctx, budgetID)
	if err != nil {
		return Report{}, apperrors.WrapInternal("load budget report snapshot", err)
	}

	lines := make([]ReportLine, 0, len(snapshot.Lines))
	totalAllocation, totalActual := int64(0), int64(0)
	for _, row := range snapshot.Lines {
		allocation, err := cents(row.AllocationAmount)
		if err != nil {
			return Report{}, apperrors.WrapInternal("calculate budget report", fmt.Errorf("invalid allocation amount: %w", err))
		}
		actual, err := cents(row.ActualAmount)
		if err != nil {
			return Report{}, apperrors.WrapInternal("calculate budget report", fmt.Errorf("invalid actual amount: %w", err))
		}
		totalAllocation += allocation
		totalActual += actual
		row.Line.Categories = nonNilCategories(row.Line.Categories)
		lines = append(lines, ReportLine{Line: row.Line, ActualAmount: formatCents(actual), RemainingAmount: formatCents(allocation - actual)})
	}
	unmapped := nonNilUnmapped(snapshot.UnmappedTransactions)
	unmappedTotal := int64(0)
	for i := range unmapped {
		value, err := cents(unmapped[i].Amount)
		if err != nil {
			return Report{}, apperrors.WrapInternal("calculate budget report", fmt.Errorf("invalid unmapped amount: %w", err))
		}
		unmapped[i].Amount = formatCents(value)
		unmappedTotal += value
	}
	uncategorized, err := cents(snapshot.UncategorizedAmount)
	if err != nil {
		return Report{}, apperrors.WrapInternal("calculate budget report", fmt.Errorf("invalid uncategorized amount: %w", err))
	}
	budget := snapshot.Budget
	return Report{
		Budget: BudgetSummary{ID: budget.ID, Owner: budget.Owner, PeriodStart: budget.PeriodStart, PeriodEnd: budget.PeriodEnd, SourceBudgetID: budget.SourceBudgetID},
		Lines:  lines, UnmappedTransactions: unmapped,
		Totals: ReportTotals{AllocationAmount: formatCents(totalAllocation), ActualAmount: formatCents(totalActual), RemainingAmount: formatCents(totalAllocation - totalActual), UnmappedActualAmount: formatCents(unmappedTotal), UncategorizedActualAmount: formatCents(uncategorized)},
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

func normalizeBudget(budget Budget) Budget {
	budget.Lines = nonNilLines(budget.Lines)
	for i := range budget.Lines {
		budget.Lines[i].Categories = nonNilCategories(budget.Lines[i].Categories)
	}
	return budget
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
