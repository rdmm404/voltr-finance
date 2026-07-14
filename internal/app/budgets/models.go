package budgets

import "time"

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
