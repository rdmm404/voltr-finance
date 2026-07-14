package webui

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	appbudgets "rdmm404/voltr-finance/internal/app/budgets"
	apperrors "rdmm404/voltr-finance/internal/app/errors"
	apphouseholds "rdmm404/voltr-finance/internal/app/households"
	appusers "rdmm404/voltr-finance/internal/app/users"
)

type Config struct {
	DefaultUserID      int64
	DefaultHouseholdID int64
}

func (c Config) Validate() error {
	var errs []error
	if c.DefaultUserID <= 0 {
		errs = append(errs, errors.New("default UI user ID must be positive"))
	}
	if c.DefaultHouseholdID <= 0 {
		errs = append(errs, errors.New("default UI household ID must be positive"))
	}
	return errors.Join(errs...)
}

type BudgetReader interface {
	DetailedMonthlyReport(context.Context, appbudgets.MonthlyInput) (appbudgets.DetailedReport, error)
}
type UserReader interface {
	List(context.Context) ([]appusers.User, error)
	Get(context.Context, int64) (appusers.User, error)
}
type HouseholdReader interface {
	List(context.Context) ([]apphouseholds.Household, error)
	Get(context.Context, int64) (apphouseholds.Household, error)
}

type Services struct {
	Budgets    BudgetReader
	Users      UserReader
	Households HouseholdReader
}

type RequestState struct {
	Month       time.Time
	UserID      int64
	HouseholdID int64
}

func ParseRequestState(values url.Values, config Config, now time.Time) (RequestState, bool, error) {
	state := RequestState{UserID: config.DefaultUserID, HouseholdID: config.DefaultHouseholdID}
	month := values.Get("month")
	if month == "" {
		state.Month = time.Date(now.In(time.Local).Year(), now.In(time.Local).Month(), 1, 0, 0, 0, 0, time.Local)
		return state, true, nil
	}
	if len(month) != 7 {
		return RequestState{}, false, fmt.Errorf("month must use YYYY-MM format")
	}
	parsed, err := time.ParseInLocation("2006-01", month, time.Local)
	if err != nil || parsed.Format("2006-01") != month {
		return RequestState{}, false, fmt.Errorf("month must be a valid calendar month in YYYY-MM format")
	}
	state.Month = parsed
	for name, target := range map[string]*int64{"userId": &state.UserID, "householdId": &state.HouseholdID} {
		if raw, exists := values[name]; exists {
			if len(raw) != 1 || strings.TrimSpace(raw[0]) == "" {
				return RequestState{}, false, fmt.Errorf("%s must be a positive integer", name)
			}
			value, err := strconv.ParseInt(raw[0], 10, 64)
			if err != nil || value <= 0 {
				return RequestState{}, false, fmt.Errorf("%s must be a positive integer", name)
			}
			*target = value
		}
	}
	return state, false, nil
}

func StateURL(state RequestState) string {
	values := url.Values{}
	values.Set("month", state.Month.Format("2006-01"))
	values.Set("userId", strconv.FormatInt(state.UserID, 10))
	values.Set("householdId", strconv.FormatInt(state.HouseholdID, 10))
	return "/?" + values.Encode()
}

type Dashboard struct{ services Services }

func NewDashboard(services Services) *Dashboard { return &Dashboard{services: services} }

func (d *Dashboard) Assemble(ctx context.Context, state RequestState) (PageView, error) {
	users, err := d.services.Users.List(ctx)
	if err != nil {
		return PageView{}, err
	}
	households, err := d.services.Households.List(ctx)
	if err != nil {
		return PageView{}, err
	}
	user, err := d.services.Users.Get(ctx, state.UserID)
	if err != nil {
		return PageView{}, err
	}
	household, err := d.services.Households.Get(ctx, state.HouseholdID)
	if err != nil {
		return PageView{}, err
	}
	personal, personalMissing, err := d.report(ctx, appbudgets.Owner{UserID: &state.UserID}, state, "Personal", user.Name)
	if err != nil {
		return PageView{}, err
	}
	householdReport, householdMissing, err := d.report(ctx, appbudgets.Owner{HouseholdID: &state.HouseholdID}, state, "Household", household.Name)
	if err != nil {
		return PageView{}, err
	}
	previous, next := state, state
	previous.Month = state.Month.AddDate(0, -1, 0)
	next.Month = state.Month.AddDate(0, 1, 0)
	view := PageView{
		Month: state.Month.Format("January 2006"), MonthValue: state.Month.Format("2006-01"),
		PreviousURL: StateURL(previous), NextURL: StateURL(next),
		UserID: state.UserID, HouseholdID: state.HouseholdID,
		Users: users, Households: households, Personal: personal, Household: householdReport,
		AllEmpty: personalMissing && householdMissing,
	}
	view.Combined = combineScopes(personal, householdReport)
	return view, nil
}

func (d *Dashboard) report(ctx context.Context, owner appbudgets.Owner, state RequestState, label, name string) (ScopeView, bool, error) {
	report, err := d.services.Budgets.DetailedMonthlyReport(ctx, appbudgets.MonthlyInput{Owner: owner, Year: state.Month.Year(), Month: int(state.Month.Month())})
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return ScopeView{Label: label, OwnerName: name, Empty: true}, true, nil
		}
		return ScopeView{}, false, err
	}
	view, err := mapScope(report, label, name)
	return view, false, err
}
