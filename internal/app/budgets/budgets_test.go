package budgets

import (
	"context"
	"testing"
	"time"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
)

type fakeRepository struct {
	monthly        Budget
	monthlyMisses  int
	prior          Budget
	budgets        map[int64]Budget
	lines          map[int64][]Line
	mappings       map[int64][]LineCategory
	categories     map[int64]Category
	nextBudgetID   int64
	nextLineID     int64
	createConflict bool
	reportLines    []ReportLineData
	unmapped       []UnmappedTransaction
	uncategorized  string
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{budgets: map[int64]Budget{}, lines: map[int64][]Line{}, mappings: map[int64][]LineCategory{}, categories: map[int64]Category{}, nextBudgetID: 20, nextLineID: 100}
}
func (f *fakeRepository) FindMonthly(context.Context, Owner, time.Time, time.Time) (Budget, error) {
	if f.monthlyMisses > 0 {
		f.monthlyMisses--
		return Budget{}, apperrors.NotFound(apperrors.CodeBudgetNotFound, "budget not found", nil)
	}
	if f.monthly.ID == 0 {
		return Budget{}, apperrors.NotFound(apperrors.CodeBudgetNotFound, "budget not found", nil)
	}
	return f.monthly, nil
}
func (f *fakeRepository) FindLatestPrior(context.Context, Owner, time.Time) (Budget, error) {
	if f.prior.ID == 0 {
		return Budget{}, apperrors.NotFound(apperrors.CodeBudgetNotFound, "budget not found", nil)
	}
	return f.prior, nil
}
func (f *fakeRepository) GetBudget(_ context.Context, id int64) (Budget, error) {
	item, ok := f.budgets[id]
	if !ok {
		return Budget{}, apperrors.NotFound(apperrors.CodeBudgetNotFound, "budget not found", nil)
	}
	return item, nil
}
func (f *fakeRepository) CreateBudget(_ context.Context, input CreateBudget) (Budget, error) {
	if f.createConflict {
		return Budget{}, apperrors.Conflict(apperrors.CodeBudgetConflict, "budget exists", nil)
	}
	f.nextBudgetID++
	item := Budget{ID: f.nextBudgetID, Owner: input.Owner, PeriodStart: input.PeriodStart, PeriodEnd: input.PeriodEnd, SourceBudgetID: input.SourceBudgetID}
	f.budgets[item.ID] = item
	return item, nil
}
func (f *fakeRepository) ListLines(_ context.Context, id int64) ([]Line, error) {
	return append([]Line(nil), f.lines[id]...), nil
}
func (f *fakeRepository) ListLineCategories(_ context.Context, id int64) ([]LineCategory, error) {
	return append([]LineCategory(nil), f.mappings[id]...), nil
}
func (f *fakeRepository) GetLine(_ context.Context, id int64) (Line, error) {
	for _, lines := range f.lines {
		for _, line := range lines {
			if line.ID == id {
				return line, nil
			}
		}
	}
	return Line{}, apperrors.NotFound(apperrors.CodeBudgetLineNotFound, "line not found", nil)
}
func (f *fakeRepository) MaxSortOrder(_ context.Context, budgetID int64) (int32, error) {
	var max int32
	for _, line := range f.lines[budgetID] {
		if line.SortOrder > max {
			max = line.SortOrder
		}
	}
	return max, nil
}
func (f *fakeRepository) CreateLine(_ context.Context, input CreateLineInput) (Line, error) {
	f.nextLineID++
	order := int32(0)
	if input.SortOrder != nil {
		order = *input.SortOrder
	}
	line := Line{ID: f.nextLineID, BudgetID: input.BudgetID, Name: input.Name, AllocationAmount: input.AllocationAmount, SortOrder: order}
	f.lines[input.BudgetID] = append(f.lines[input.BudgetID], line)
	return line, nil
}
func (f *fakeRepository) UpdateLine(_ context.Context, id int64, update LineUpdate) (Line, error) {
	for budgetID, lines := range f.lines {
		for index, line := range lines {
			if line.ID == id {
				if update.Name != nil {
					line.Name = *update.Name
				}
				if update.AllocationAmount != nil {
					line.AllocationAmount = *update.AllocationAmount
				}
				if update.SortOrder != nil {
					line.SortOrder = *update.SortOrder
				}
				f.lines[budgetID][index] = line
				return line, nil
			}
		}
	}
	return Line{}, apperrors.NotFound(apperrors.CodeBudgetLineNotFound, "line not found", nil)
}
func (f *fakeRepository) DeleteLine(_ context.Context, id int64) error {
	for budgetID, lines := range f.lines {
		for i, line := range lines {
			if line.ID == id {
				f.lines[budgetID] = append(lines[:i], lines[i+1:]...)
				return nil
			}
		}
	}
	return apperrors.NotFound(apperrors.CodeBudgetLineNotFound, "line not found", nil)
}
func (f *fakeRepository) DeleteLineCategories(_ context.Context, lineID int64) error {
	for budgetID, mappings := range f.mappings {
		kept := mappings[:0]
		for _, mapping := range mappings {
			if mapping.LineID != lineID {
				kept = append(kept, mapping)
			}
		}
		f.mappings[budgetID] = kept
	}
	return nil
}
func (f *fakeRepository) CreateLineCategory(_ context.Context, budgetID, lineID, categoryID int64) error {
	f.mappings[budgetID] = append(f.mappings[budgetID], LineCategory{BudgetID: budgetID, LineID: lineID, Category: f.categories[categoryID]})
	return nil
}
func (f *fakeRepository) GetActiveCategoryByID(_ context.Context, id int64) (Category, error) {
	item, ok := f.categories[id]
	if !ok {
		return Category{}, apperrors.NotFound(apperrors.CodeCategoryNotFound, "category not found", nil)
	}
	return item, nil
}
func (f *fakeRepository) GetActiveCategoryByCode(_ context.Context, code string) (Category, error) {
	for _, item := range f.categories {
		if item.Code == code {
			return item, nil
		}
	}
	return Category{}, apperrors.NotFound(apperrors.CodeCategoryNotFound, "category not found", nil)
}
func (f *fakeRepository) ListReportLines(context.Context, int64) ([]ReportLineData, error) {
	return f.reportLines, nil
}
func (f *fakeRepository) ListUnmappedTransactions(context.Context, int64) ([]UnmappedTransaction, error) {
	return f.unmapped, nil
}
func (f *fakeRepository) SumUncategorized(context.Context, int64) (string, error) {
	if f.uncategorized == "" {
		return "0.00", nil
	}
	return f.uncategorized, nil
}

type fakeTransactor struct {
	repo  Repository
	calls int
}

func (f *fakeTransactor) WithinTransaction(ctx context.Context, callback func(Repository) error) error {
	f.calls++
	return callback(f.repo)
}

func TestMonthlyReadAndEnsurePriorCopy(t *testing.T) {
	householdID := int64(7)
	repo := newFakeRepository()
	repo.prior = Budget{ID: 10, Owner: Owner{HouseholdID: &householdID}}
	repo.budgets[10] = repo.prior
	repo.lines[10] = []Line{{ID: 90, BudgetID: 10, Name: "Food", AllocationAmount: "100.00", SortOrder: 1}}
	repo.categories[3] = Category{ID: 3, Code: "food", Name: "Food"}
	repo.mappings[10] = []LineCategory{{BudgetID: 10, LineID: 90, Category: repo.categories[3]}}
	tx := &fakeTransactor{repo: repo}
	service := NewService(repo, tx)
	input := MonthlyInput{Owner: Owner{HouseholdID: &householdID}, Year: 2026, Month: 7}
	if _, err := service.GetMonthly(context.Background(), input); !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Fatalf("GetMonthly error=%v", err)
	}
	result, err := service.EnsureMonthly(context.Background(), input)
	if err != nil || !result.Created || result.Budget.SourceBudgetID == nil || *result.Budget.SourceBudgetID != 10 || len(result.Budget.Lines) != 1 || len(result.Budget.Lines[0].Categories) != 1 || tx.calls != 1 {
		t.Fatalf("EnsureMonthly=%+v calls=%d error=%v", result, tx.calls, err)
	}
}

func TestEnsureMonthlyRecoversConcurrentCreation(t *testing.T) {
	userID := int64(8)
	repo := newFakeRepository()
	repo.monthlyMisses = 1
	repo.monthly = Budget{ID: 55, Owner: Owner{UserID: &userID}}
	repo.createConflict = true
	result, err := NewService(repo, &fakeTransactor{repo: repo}).EnsureMonthly(context.Background(), MonthlyInput{Owner: Owner{UserID: &userID}, Year: 2026, Month: 7})
	if err != nil || result.Created || result.Budget.ID != 55 {
		t.Fatalf("EnsureMonthly=%+v error=%v", result, err)
	}
}

func TestLineChangesUseTransactionAndEnforceCategoryInvariant(t *testing.T) {
	repo := newFakeRepository()
	repo.budgets[12] = Budget{ID: 12}
	repo.categories[3] = Category{ID: 3, Code: "food"}
	tx := &fakeTransactor{repo: repo}
	service := NewService(repo, tx)
	line, err := service.CreateLine(context.Background(), CreateLineInput{BudgetID: 12, Name: " Food ", AllocationAmount: "100", CategoryIDs: []int64{3}})
	if err != nil || line.Name != "Food" || line.AllocationAmount != "100.00" || len(line.Categories) != 1 || tx.calls != 1 {
		t.Fatalf("CreateLine=%+v calls=%d error=%v", line, tx.calls, err)
	}
	_, err = service.CreateLine(context.Background(), CreateLineInput{BudgetID: 12, Name: "Other", AllocationAmount: "20", CategoryIDs: []int64{3}})
	if !apperrors.IsKind(err, apperrors.KindConflict) {
		t.Fatalf("duplicate category error=%v", err)
	}
	empty := []int64{}
	updated, err := service.UpdateLine(context.Background(), UpdateLineInput{LineID: line.ID, CategoryIDs: &empty})
	if err != nil || len(updated.Categories) != 0 || tx.calls != 3 {
		t.Fatalf("UpdateLine=%+v calls=%d error=%v", updated, tx.calls, err)
	}
}

func TestReportAssemblesTotalsAndUnmappedRequirements(t *testing.T) {
	householdID := int64(1)
	repo := newFakeRepository()
	repo.budgets[12] = Budget{ID: 12, Owner: Owner{HouseholdID: &householdID}}
	repo.reportLines = []ReportLineData{{Line: Line{ID: 1, BudgetID: 12, Name: "Food", AllocationAmount: "100.00"}, ActualAmount: "60.25"}}
	repo.unmapped = []UnmappedTransaction{{ID: 9, Amount: "12.50"}, {ID: 10, Amount: "3.25", Category: &Category{ID: 4, Code: "other"}}}
	repo.uncategorized = "12.50"
	report, err := NewService(repo, nil).Report(context.Background(), 12)
	if err != nil || report.Lines[0].RemainingAmount != "39.75" || report.Totals.UnmappedActualAmount != "15.75" || report.Totals.UncategorizedActualAmount != "12.50" || len(report.UnmappedTransactions) != 2 {
		t.Fatalf("Report=%+v error=%v", report, err)
	}
}
