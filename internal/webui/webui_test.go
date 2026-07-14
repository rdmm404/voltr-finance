package webui

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	appbudgets "rdmm404/voltr-finance/internal/app/budgets"
	apperrors "rdmm404/voltr-finance/internal/app/errors"
	apphouseholds "rdmm404/voltr-finance/internal/app/households"
	appusers "rdmm404/voltr-finance/internal/app/users"
)

type budgetStub struct {
	reports map[bool]appbudgets.DetailedReport
	errs    map[bool]error
	calls   int
}

func (s *budgetStub) DetailedMonthlyReport(_ context.Context, input appbudgets.MonthlyInput) (appbudgets.DetailedReport, error) {
	s.calls++
	household := input.Owner.HouseholdID != nil
	return s.reports[household], s.errs[household]
}

type userStub struct {
	users []appusers.User
	err   error
}

func (s userStub) List(context.Context) ([]appusers.User, error) { return s.users, s.err }
func (s userStub) Get(_ context.Context, id int64) (appusers.User, error) {
	for _, item := range s.users {
		if item.ID == id {
			return item, nil
		}
	}
	return appusers.User{}, apperrors.NotFound("user_not_found", "user not found", nil)
}

type householdStub struct {
	items []apphouseholds.Household
	err   error
}

func (s householdStub) List(context.Context) ([]apphouseholds.Household, error) {
	return s.items, s.err
}
func (s householdStub) Get(_ context.Context, id int64) (apphouseholds.Household, error) {
	for _, item := range s.items {
		if item.ID == id {
			return item, nil
		}
	}
	return apphouseholds.Household{}, apperrors.NotFound("household_not_found", "household not found", nil)
}

func TestParseRequestStateCanonicalAndOverrides(t *testing.T) {
	original := time.Local
	location, err := time.LoadLocation("America/Toronto")
	if err != nil {
		t.Fatal(err)
	}
	time.Local = location
	t.Cleanup(func() { time.Local = original })
	config := Config{DefaultUserID: 1, DefaultHouseholdID: 2}
	state, redirect, err := ParseRequestState(url.Values{}, config, time.Date(2026, 8, 1, 2, 0, 0, 0, time.UTC))
	if err != nil || !redirect || state.Month.Format("2006-01") != "2026-07" {
		t.Fatalf("state=%+v redirect=%v error=%v", state, redirect, err)
	}
	state, redirect, err = ParseRequestState(url.Values{"month": {"2026-02"}, "userId": {"3"}, "householdId": {"4"}}, config, time.Now())
	if err != nil || redirect || state.UserID != 3 || state.HouseholdID != 4 || !strings.Contains(StateURL(state), "month=2026-02") {
		t.Fatalf("state=%+v redirect=%v error=%v", state, redirect, err)
	}
	for _, values := range []url.Values{{"month": {"2026-2"}}, {"month": {"2026-13"}}, {"month": {"2026-02"}, "userId": {"0"}}, {"month": {"2026-02"}, "householdId": {"nope"}}} {
		if _, _, err := ParseRequestState(values, config, time.Now()); err == nil {
			t.Fatalf("expected validation error for %v", values)
		}
	}
}

func TestMapScopeCurrencyAndCombinedTotals(t *testing.T) {
	report := appbudgets.DetailedReport{Totals: appbudgets.ReportTotals{AllocationAmount: "1000.00", ActualAmount: "700.00", UnmappedActualAmount: "50.25"}, Lines: []appbudgets.DetailedReportLine{{ReportLine: appbudgets.ReportLine{Line: appbudgets.Line{Name: "Food", AllocationAmount: "100.00"}, ActualAmount: "90", RemainingAmount: "10"}}}}
	scope, err := mapScope(report, "Personal", "Alex")
	if err != nil {
		t.Fatal(err)
	}
	if scope.Summary.Spent != "$750.25" || scope.Summary.Remaining != "$249.75" || scope.Summary.Progress != "75" || scope.Summary.State != StateNormal {
		t.Fatalf("summary=%+v", scope.Summary)
	}
	combined := combineScopes(scope, ScopeView{Summary: SummaryView{Allocation: "$100.00", Spent: "$125.00", Remaining: "-$25.00", Unmapped: "$5.00", State: StateDanger}})
	if combined.Allocation != "$1,100.00" || combined.Spent != "$875.25" || combined.Remaining != "$224.75" {
		t.Fatalf("combined=%+v", combined)
	}
}

func TestHandlerRedirectRenderAssetsAndErrors(t *testing.T) {
	userID, householdID := int64(1), int64(2)
	report := appbudgets.DetailedReport{Budget: appbudgets.BudgetSummary{ID: 10}, Totals: appbudgets.ReportTotals{AllocationAmount: "100", ActualAmount: "25", UnmappedActualAmount: "5", UncategorizedActualAmount: "5"}, Lines: []appbudgets.DetailedReportLine{}, UnmappedTransactions: []appbudgets.DetailedTransaction{}}
	budgets := &budgetStub{reports: map[bool]appbudgets.DetailedReport{false: report, true: report}, errs: map[bool]error{}}
	handler, err := New(Config{DefaultUserID: userID, DefaultHouseholdID: householdID}, Services{Budgets: budgets, Users: userStub{users: []appusers.User{{ID: userID, Name: "Alex"}}}, Households: householdStub{items: []apphouseholds.Household{{ID: householdID, Name: "Home"}}}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	handler.now = func() time.Time { return time.Date(2026, 7, 14, 12, 0, 0, 0, time.Local) }
	mux := http.NewServeMux()
	handler.Register(mux)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || !strings.Contains(response.Header().Get("Location"), "month=2026-07") {
		t.Fatalf("status=%d location=%s", response.Code, response.Header().Get("Location"))
	}
	request = httptest.NewRequest(http.MethodGet, "/?month=2026-07", nil)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, "Combined monthly summary") || !strings.Contains(body, "&lt;") && strings.Contains(body, "<script") || budgets.calls != 2 {
		t.Fatalf("status=%d calls=%d body=%s", response.Code, budgets.calls, body)
	}
	request = httptest.NewRequest(http.MethodGet, "/assets/app.css", nil)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("Content-Type"), "text/css") || response.Header().Get("Cache-Control") == "" {
		t.Fatalf("asset status=%d headers=%v", response.Code, response.Header())
	}
	request = httptest.NewRequest(http.MethodGet, "/?month=bad", nil)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("validation status=%d", response.Code)
	}
	budgets.errs[false] = errors.New("database unavailable")
	request = httptest.NewRequest(http.MethodGet, "/?month=2026-07", nil)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "database unavailable") {
		t.Fatalf("internal response=%d %s", response.Code, response.Body.String())
	}
}

func TestMissingBudgetsRenderSuccessfulEmptyState(t *testing.T) {
	notFound := apperrors.NotFound(apperrors.CodeBudgetNotFound, "budget not found", nil)
	budgets := &budgetStub{reports: map[bool]appbudgets.DetailedReport{}, errs: map[bool]error{false: notFound, true: notFound}}
	handler, err := New(Config{DefaultUserID: 1, DefaultHouseholdID: 2}, Services{Budgets: budgets, Users: userStub{users: []appusers.User{{ID: 1, Name: "Alex"}}}, Households: householdStub{items: []apphouseholds.Household{{ID: 2, Name: "Home"}}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	handler.Register(mux)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/?month=2026-07", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "No budgets this month") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
