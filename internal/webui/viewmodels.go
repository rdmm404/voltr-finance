package webui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	appbudgets "rdmm404/voltr-finance/internal/app/budgets"
	apphouseholds "rdmm404/voltr-finance/internal/app/households"
	appusers "rdmm404/voltr-finance/internal/app/users"
)

type SemanticState string

const (
	StateNormal  SemanticState = "normal"
	StateWarning SemanticState = "warning"
	StateDanger  SemanticState = "danger"
)

type PageView struct {
	Month, MonthValue, PreviousURL, NextURL string
	UserID, HouseholdID                     int64
	Users                                   []appusers.User
	Households                              []apphouseholds.Household
	Combined                                SummaryView
	Personal, Household                     ScopeView
	AllEmpty                                bool
}

type SummaryView struct {
	Allocation, Spent, Remaining, Unmapped string
	Progress                               string
	State                                  SemanticState
}

type ScopeView struct {
	Label, OwnerName string
	Empty            bool
	Summary          SummaryView
	Lines            []LineView
	Unmapped         []TransactionView
}

type LineView struct {
	Name, Allocation, Actual, Remaining, Progress string
	State                                         SemanticState
	Categories                                    string
	Transactions                                  []TransactionView
}

type TransactionView struct {
	ID                                                     int64
	Date, Amount, Description, Notes, Category, AuthorName string
}

func mapScope(report appbudgets.DetailedReport, label, ownerName string) (ScopeView, error) {
	allocation, err := moneyCents(report.Totals.AllocationAmount)
	if err != nil {
		return ScopeView{}, err
	}
	mapped, err := moneyCents(report.Totals.ActualAmount)
	if err != nil {
		return ScopeView{}, err
	}
	unmapped, err := moneyCents(report.Totals.UnmappedActualAmount)
	if err != nil {
		return ScopeView{}, err
	}
	spent, remaining := mapped+unmapped, allocation-mapped-unmapped
	view := ScopeView{Label: label, OwnerName: ownerName, Lines: make([]LineView, 0, len(report.Lines)), Unmapped: mapTransactions(report.UnmappedTransactions)}
	view.Summary = summary(allocation, spent, remaining, unmapped)
	for _, line := range report.Lines {
		lineAllocation, err := moneyCents(line.AllocationAmount)
		if err != nil {
			return ScopeView{}, err
		}
		actual, err := moneyCents(line.ActualAmount)
		if err != nil {
			return ScopeView{}, err
		}
		remaining := lineAllocation - actual
		percentage := int64(0)
		if lineAllocation > 0 {
			percentage = actual * 100 / lineAllocation
		}
		if percentage < 0 {
			percentage = 0
		}
		categories := make([]string, 0, len(line.Categories))
		for _, category := range line.Categories {
			categories = append(categories, category.Name)
		}
		view.Lines = append(view.Lines, LineView{
			Name: line.Name, Allocation: formatCAD(lineAllocation), Actual: formatCAD(actual), Remaining: formatCAD(remaining),
			Progress: strconv.FormatInt(percentage, 10), State: varianceState(remaining, actual, lineAllocation),
			Categories: strings.Join(categories, ", "), Transactions: mapTransactions(line.Transactions),
		})
	}
	return view, nil
}

func combineScopes(scopes ...ScopeView) SummaryView {
	var allocation, spent, remaining, unmapped int64
	for _, scope := range scopes {
		if scope.Empty {
			continue
		}
		allocation += mustMoneyCents(scope.Summary.Allocation)
		spent += mustMoneyCents(scope.Summary.Spent)
		remaining += mustMoneyCents(scope.Summary.Remaining)
		unmapped += mustMoneyCents(scope.Summary.Unmapped)
	}
	return summary(allocation, spent, remaining, unmapped)
}

func summary(allocation, spent, remaining, unmapped int64) SummaryView {
	percentage := int64(0)
	if allocation > 0 {
		percentage = spent * 100 / allocation
	}
	if percentage < 0 {
		percentage = 0
	}
	return SummaryView{
		Allocation: formatCAD(allocation), Spent: formatCAD(spent), Remaining: formatCAD(remaining), Unmapped: formatCAD(unmapped),
		Progress: strconv.FormatInt(percentage, 10), State: varianceState(remaining, spent, allocation),
	}
}

func varianceState(remaining, spent, allocation int64) SemanticState {
	if remaining < 0 {
		return StateDanger
	}
	if allocation > 0 && spent*100 >= allocation*80 {
		return StateWarning
	}
	return StateNormal
}

func mapTransactions(items []appbudgets.DetailedTransaction) []TransactionView {
	result := make([]TransactionView, 0, len(items))
	for _, item := range items {
		view := TransactionView{ID: item.ID, Date: item.TransactionDate.Format("Jan 2, 2006"), Amount: formatCAD(mustMoneyCents(item.Amount)), AuthorName: item.Author.Name, Description: "No description", Category: "Uncategorized"}
		if item.Description != nil && strings.TrimSpace(*item.Description) != "" {
			view.Description = *item.Description
		}
		if item.Notes != nil {
			view.Notes = *item.Notes
		}
		if item.Category != nil {
			view.Category = item.Category.Name
		}
		result = append(result, view)
	}
	return result
}

func moneyCents(value string) (int64, error) {
	value = strings.TrimSpace(strings.TrimSuffix(value, " CAD"))
	value = strings.ReplaceAll(value, ",", "")
	negative := strings.HasPrefix(value, "-")
	value = strings.TrimPrefix(value, "-")
	value = strings.TrimPrefix(value, "$")
	parts := strings.Split(value, ".")
	if len(parts) > 2 || len(parts) == 0 {
		return 0, fmt.Errorf("invalid money %q", value)
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid money %q: %w", value, err)
	}
	fraction := int64(0)
	if len(parts) == 2 {
		if len(parts[1]) > 2 {
			return 0, fmt.Errorf("invalid money precision %q", value)
		}
		fractionText := parts[1] + strings.Repeat("0", 2-len(parts[1]))
		fraction, err = strconv.ParseInt(fractionText, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid money %q: %w", value, err)
		}
	}
	result := whole*100 + fraction
	if negative {
		result = -result
	}
	return result, nil
}

func mustMoneyCents(value string) int64 { result, _ := moneyCents(value); return result }

func formatCAD(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	whole := strconv.FormatInt(cents/100, 10)
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	return fmt.Sprintf("%s$%s.%02d", sign, whole, cents%100)
}

var _ = time.Local
