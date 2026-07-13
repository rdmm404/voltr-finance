package api

const (
	APIPrefix = "/v1"
	LivePath  = "/live"

	TransactionsPath        = APIPrefix + "/transactions"
	TransactionsBulkPath    = TransactionsPath + "/bulk"
	TransactionsRestorePath = TransactionsPath + "/restore"
	TransactionPath         = TransactionsPath + "/{id}"

	UsersPath       = APIPrefix + "/users"
	UserPath        = UsersPath + "/{id}"
	UserResolvePath = UsersPath + "/resolve"

	HouseholdsPath       = APIPrefix + "/households"
	HouseholdPath        = HouseholdsPath + "/{id}"
	HouseholdUsersPath   = HouseholdPath + "/users"
	HouseholdResolvePath = HouseholdsPath + "/resolve"

	CategoriesPath     = APIPrefix + "/categories"
	CategoryPath       = CategoriesPath + "/{code}"
	CategoryUpdatePath = CategoriesPath + "/{id}"

	MonthlyBudgetsPath = APIPrefix + "/budgets/monthly"
	BudgetReportPath   = APIPrefix + "/budgets/{id}/report"
	BudgetLinesPath    = APIPrefix + "/budgets/{id}/lines"
	BudgetLinePath     = APIPrefix + "/budget-lines/{id}"
)
