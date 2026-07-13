package budgets

import (
	"context"
	"reflect"
	"testing"
	"time"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
)

type fakeRepository struct {
	monthly       Budget
	monthlyMisses int
	findErr       error
	created       Budget
	createErr     error
	createInput   CreateMonthlyFromTemplateInput
	createdLine   Line
	createLineErr error
	createLine    CreateLineInput
	updatedLine   Line
	updateLineErr error
	updateLine    UpdateLineInput
	deleteErr     error
	deletedID     int64
	snapshot      ReportSnapshot
	reportErr     error
}

func (f *fakeRepository) FindMonthly(context.Context, Owner, time.Time, time.Time) (Budget, error) {
	if f.monthlyMisses > 0 {
		f.monthlyMisses--
		return Budget{}, apperrors.NotFound(apperrors.CodeBudgetNotFound, "budget not found", nil)
	}
	if f.findErr != nil {
		return Budget{}, f.findErr
	}
	if f.monthly.ID == 0 {
		return Budget{}, apperrors.NotFound(apperrors.CodeBudgetNotFound, "budget not found", nil)
	}
	return f.monthly, nil
}
func (f *fakeRepository) CreateMonthlyFromTemplate(_ context.Context, input CreateMonthlyFromTemplateInput) (Budget, error) {
	f.createInput = input
	return f.created, f.createErr
}
func (f *fakeRepository) CreateLineWithCategories(_ context.Context, input CreateLineInput) (Line, error) {
	f.createLine = input
	return f.createdLine, f.createLineErr
}
func (f *fakeRepository) UpdateLineWithCategories(_ context.Context, input UpdateLineInput) (Line, error) {
	f.updateLine = input
	return f.updatedLine, f.updateLineErr
}
func (f *fakeRepository) DeleteLine(_ context.Context, id int64) error {
	f.deletedID = id
	return f.deleteErr
}
func (f *fakeRepository) LoadReportSnapshot(context.Context, int64) (ReportSnapshot, error) {
	return f.snapshot, f.reportErr
}

func TestRepositoryPortExposesOnlyCohesiveOperations(t *testing.T) {
	port := reflect.TypeOf((*Repository)(nil)).Elem()
	got := make([]string, port.NumMethod())
	for i := range port.NumMethod() {
		got[i] = port.Method(i).Name
	}
	want := []string{"CreateLineWithCategories", "CreateMonthlyFromTemplate", "DeleteLine", "FindMonthly", "LoadReportSnapshot", "UpdateLineWithCategories"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("repository methods=%v want=%v", got, want)
	}
}

func TestMonthlyReadAndEnsureUseAggregateRepositoryOperations(t *testing.T) {
	householdID := int64(7)
	repo := &fakeRepository{
		monthlyMisses: 1,
		created:       Budget{ID: 20, Owner: Owner{HouseholdID: &householdID}, SourceBudgetID: int64Pointer(10), Lines: []Line{{ID: 2, Categories: []Category{{ID: 3, Code: "food"}}}}},
	}
	service := NewService(repo)
	input := MonthlyInput{Owner: Owner{HouseholdID: &householdID}, Year: 2026, Month: 7}
	if _, err := service.GetMonthly(context.Background(), input); !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Fatalf("GetMonthly error=%v", err)
	}
	result, err := service.EnsureMonthly(context.Background(), input)
	if err != nil || !result.Created || result.Budget.ID != 20 || len(result.Budget.Lines[0].Categories) != 1 {
		t.Fatalf("EnsureMonthly=%+v error=%v", result, err)
	}
	if repo.createInput.PeriodStart != time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC) || repo.createInput.PeriodEnd != time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("create input=%+v", repo.createInput)
	}
}

func TestEnsureMonthlyRecoversConcurrentCreation(t *testing.T) {
	userID := int64(8)
	repo := &fakeRepository{
		monthlyMisses: 1,
		monthly:       Budget{ID: 55, Owner: Owner{UserID: &userID}},
		createErr:     apperrors.Conflict(apperrors.CodeBudgetConflict, "budget exists", nil),
	}
	result, err := NewService(repo).EnsureMonthly(context.Background(), MonthlyInput{Owner: Owner{UserID: &userID}, Year: 2026, Month: 7})
	if err != nil || result.Created || result.Budget.ID != 55 || result.Budget.Lines == nil {
		t.Fatalf("EnsureMonthly=%+v error=%v", result, err)
	}
}

func TestLineCreateNormalizesAndDelegatesCategoryCodes(t *testing.T) {
	repo := &fakeRepository{createdLine: Line{ID: 100, BudgetID: 12, Name: "Essentials", AllocationAmount: "100.00", Categories: []Category{{ID: 3, Code: "food"}, {ID: 4, Code: "rent"}}}}
	service := NewService(repo)
	line, err := service.CreateLine(context.Background(), CreateLineInput{
		BudgetID: 12, Name: " Essentials ", AllocationAmount: "100", CategoryIDs: []int64{3}, CategoryCodes: []string{" food ", "rent"},
	})
	if err != nil || line.Name != "Essentials" || line.AllocationAmount != "100.00" || len(line.Categories) != 2 {
		t.Fatalf("line=%+v error=%v", line, err)
	}
	if repo.createLine.Name != "Essentials" || repo.createLine.AllocationAmount != "100.00" || len(repo.createLine.CategoryCodes) != 2 {
		t.Fatalf("repository input=%+v", repo.createLine)
	}
}

func TestLineUpdateNormalizesAndDelegatesCategoryReplacement(t *testing.T) {
	codes := []string{"rent"}
	name, amount := " Essentials ", "25.5"
	repo := &fakeRepository{updatedLine: Line{ID: 100, BudgetID: 12, Categories: []Category{{ID: 4, Code: "rent"}}}}
	updated, err := NewService(repo).UpdateLine(context.Background(), UpdateLineInput{LineID: 100, Name: &name, AllocationAmount: &amount, CategoryCodes: &codes})
	if err != nil || len(updated.Categories) != 1 || updated.Categories[0].Code != "rent" {
		t.Fatalf("updated=%+v error=%v", updated, err)
	}
	if *repo.updateLine.Name != "Essentials" || *repo.updateLine.AllocationAmount != "25.50" || repo.updateLine.CategoryCodes == nil {
		t.Fatalf("repository input=%+v", repo.updateLine)
	}
}

func TestLineRepositoryErrorsPreserveSafeKinds(t *testing.T) {
	repo := &fakeRepository{createLineErr: apperrors.Conflict(apperrors.CodeBudgetConflict, "category already mapped to another budget line", nil)}
	_, err := NewService(repo).CreateLine(context.Background(), CreateLineInput{BudgetID: 12, Name: "Food", AllocationAmount: "1", CategoryCodes: []string{"food"}})
	if !apperrors.IsKind(err, apperrors.KindConflict) || apperrors.MessageOf(err) != "category already mapped to another budget line" {
		t.Fatalf("error=%v", err)
	}
	repo.updateLineErr = apperrors.NotFound(apperrors.CodeCategoryNotFound, "category not found", nil)
	codes := []string{"missing"}
	if _, err := NewService(repo).UpdateLine(context.Background(), UpdateLineInput{LineID: 100, CategoryCodes: &codes}); !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Fatalf("missing category error=%v", err)
	}
}

func TestMonthlyValidation(t *testing.T) {
	householdID, userID := int64(1), int64(2)
	for name, input := range map[string]MonthlyInput{
		"missing owner":     {Year: 2026, Month: 7},
		"multiple owners":   {Owner: Owner{HouseholdID: &householdID, UserID: &userID}, Year: 2026, Month: 7},
		"invalid year":      {Owner: Owner{HouseholdID: &householdID}, Year: 0, Month: 7},
		"month below range": {Owner: Owner{HouseholdID: &householdID}, Year: 2026, Month: 0},
		"month above range": {Owner: Owner{HouseholdID: &householdID}, Year: 2026, Month: 13},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewService(&fakeRepository{}).GetMonthly(context.Background(), input); !apperrors.IsKind(err, apperrors.KindValidation) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestLineInputValidationAndDelete(t *testing.T) {
	repo := &fakeRepository{createdLine: Line{ID: 1, BudgetID: 12}}
	service := NewService(repo)
	for name, amount := range map[string]string{"negative": "-1", "too precise": "1.001", "not numeric": "one"} {
		t.Run(name, func(t *testing.T) {
			if _, err := service.CreateLine(context.Background(), CreateLineInput{BudgetID: 12, Name: "Food", AllocationAmount: amount}); !apperrors.IsKind(err, apperrors.KindValidation) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	if err := service.DeleteLine(context.Background(), 9); err != nil || repo.deletedID != 9 {
		t.Fatalf("deletedID=%d error=%v", repo.deletedID, err)
	}
}

func TestReportAssemblesTotalsAndUnmappedRequirements(t *testing.T) {
	householdID := int64(1)
	repo := &fakeRepository{snapshot: ReportSnapshot{
		Budget:               Budget{ID: 12, Owner: Owner{HouseholdID: &householdID}},
		Lines:                []ReportLineData{{Line: Line{ID: 1, BudgetID: 12, Name: "Food", AllocationAmount: "100.00", Categories: []Category{{ID: 3, Code: "food"}}}, ActualAmount: "60.25"}},
		UnmappedTransactions: []UnmappedTransaction{{ID: 9, Amount: "12.50"}, {ID: 10, Amount: "3.25", Category: &Category{ID: 4, Code: "other"}}},
		UncategorizedAmount:  "12.50",
	}}
	report, err := NewService(repo).Report(context.Background(), 12)
	if err != nil || report.Lines[0].RemainingAmount != "39.75" || report.Totals.UnmappedActualAmount != "15.75" || report.Totals.UncategorizedActualAmount != "12.50" || len(report.UnmappedTransactions) != 2 || len(report.Lines[0].Categories) != 1 {
		t.Fatalf("Report=%+v error=%v", report, err)
	}
}

func TestReportSupportsNegativeValuesAndEmptyCollections(t *testing.T) {
	repo := &fakeRepository{snapshot: ReportSnapshot{Budget: Budget{ID: 12}, Lines: []ReportLineData{{Line: Line{ID: 1, BudgetID: 12, AllocationAmount: "0.00"}, ActualAmount: "-5.25"}}, UncategorizedAmount: "0"}}
	report, err := NewService(repo).Report(context.Background(), 12)
	if err != nil || report.Totals.ActualAmount != "-5.25" || report.Lines[0].RemainingAmount != "5.25" || report.UnmappedTransactions == nil || report.Lines[0].Categories == nil {
		t.Fatalf("report=%+v error=%v", report, err)
	}

	repo.snapshot.Lines = nil
	empty, err := NewService(repo).Report(context.Background(), 12)
	if err != nil || empty.Lines == nil || empty.UnmappedTransactions == nil || len(empty.Lines) != 0 {
		t.Fatalf("empty=%+v error=%v", empty, err)
	}
}

func int64Pointer(value int64) *int64 { return &value }
