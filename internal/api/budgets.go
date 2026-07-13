package api

import "time"

type MonthlyBudgetParams struct {
	HouseholdID *int64 `json:"householdId,omitempty"`
	UserID      *int64 `json:"userId,omitempty"`
	Year        int    `json:"year"`
	Month       int    `json:"month"`
}

type Budget struct {
	ID             int64        `json:"id"`
	HouseholdID    *int64       `json:"householdId,omitempty"`
	UserID         *int64       `json:"userId,omitempty"`
	PeriodStart    time.Time    `json:"periodStart"`
	PeriodEnd      time.Time    `json:"periodEnd"`
	SourceBudgetID *int64       `json:"sourceBudgetId,omitempty"`
	Lines          []BudgetLine `json:"lines"`
}

type BudgetLine struct {
	ID               int64         `json:"id"`
	BudgetID         int64         `json:"budgetId"`
	Name             string        `json:"name"`
	AllocationAmount string        `json:"allocationAmount"`
	SortOrder        int32         `json:"sortOrder"`
	Categories       []CategoryRef `json:"categories"`
}

type CreateBudgetLineRequest struct {
	Name             string   `json:"name"`
	AllocationAmount string   `json:"allocationAmount"`
	CategoryIDs      []int64  `json:"categoryIds,omitempty"`
	CategoryCodes    []string `json:"categoryCodes,omitempty"`
	SortOrder        *int32   `json:"sortOrder,omitempty"`
}

type UpdateBudgetLineRequest struct {
	Name             *string   `json:"name,omitempty"`
	AllocationAmount *string   `json:"allocationAmount,omitempty"`
	CategoryIDs      *[]int64  `json:"categoryIds,omitempty"`
	CategoryCodes    *[]string `json:"categoryCodes,omitempty"`
	SortOrder        *int32    `json:"sortOrder,omitempty"`
}

type BudgetReport struct {
	Budget               BudgetSummary               `json:"budget"`
	Lines                []BudgetReportLine          `json:"lines"`
	UnmappedTransactions []BudgetUnmappedTransaction `json:"unmappedTransactions"`
	Totals               BudgetReportTotals          `json:"totals"`
}

type BudgetSummary struct {
	ID             int64     `json:"id"`
	HouseholdID    *int64    `json:"householdId,omitempty"`
	UserID         *int64    `json:"userId,omitempty"`
	PeriodStart    time.Time `json:"periodStart"`
	PeriodEnd      time.Time `json:"periodEnd"`
	SourceBudgetID *int64    `json:"sourceBudgetId,omitempty"`
}

type BudgetReportLine struct {
	ID               int64         `json:"id"`
	BudgetID         int64         `json:"budgetId"`
	Name             string        `json:"name"`
	AllocationAmount string        `json:"allocationAmount"`
	ActualAmount     string        `json:"actualAmount"`
	RemainingAmount  string        `json:"remainingAmount"`
	SortOrder        int32         `json:"sortOrder"`
	Categories       []CategoryRef `json:"categories"`
}

type BudgetUnmappedTransaction struct {
	ID              int64        `json:"id"`
	TransactionDate time.Time    `json:"transactionDate"`
	Description     *string      `json:"description,omitempty"`
	Amount          string       `json:"amount"`
	Category        *CategoryRef `json:"category,omitempty"`
}

type BudgetReportTotals struct {
	AllocationAmount          string `json:"allocationAmount"`
	ActualAmount              string `json:"actualAmount"`
	RemainingAmount           string `json:"remainingAmount"`
	UnmappedActualAmount      string `json:"unmappedActualAmount"`
	UncategorizedActualAmount string `json:"uncategorizedActualAmount"`
}
