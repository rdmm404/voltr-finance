package cli

import "rdmm404/voltr-finance/internal/api"

type BudgetsCmd struct {
	Get    BudgetGetCmd    `cmd:"" help:"Get a monthly budget."`
	Report BudgetReportCmd `cmd:"" help:"Show a budget report."`
	Lines  BudgetLinesCmd  `cmd:"" help:"Manage budget lines."`
}

type BudgetGetCmd struct {
	HouseholdID *int64 `placeholder:"INT-64" help:"Household budget owner."`
	UserID      *int64 `placeholder:"INT-64" help:"Personal budget owner."`
	Month       string `required:"" help:"Budget month in YYYY-MM format."`
	Create      bool   `help:"Create the monthly budget if missing."`
}

func (c *BudgetGetCmd) Run(ctx *runContext) error {
	year, month, err := parseBudgetMonth(c.Month)
	if err != nil {
		return err
	}
	params := api.MonthlyBudgetParams{HouseholdID: c.HouseholdID, UserID: c.UserID, Year: year, Month: month}
	var budget api.Budget
	if c.Create {
		budget, err = ctx.budgets.EnsureMonthlyBudget(ctx.Context, params)
	} else {
		budget, err = ctx.budgets.GetMonthlyBudget(ctx.Context, params)
	}
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, budget)
}

type BudgetReportCmd struct {
	ID int64 `arg:"" required:"" help:"Budget ID."`
}

func (c *BudgetReportCmd) Run(ctx *runContext) error {
	report, err := ctx.budgets.GetBudgetReport(ctx.Context, c.ID)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, report)
}

type BudgetLinesCmd struct {
	Add    BudgetLineAddCmd    `cmd:"" help:"Add a budget line."`
	Update BudgetLineUpdateCmd `cmd:"" help:"Update a budget line."`
	Delete BudgetLineDeleteCmd `cmd:"" help:"Delete a budget line."`
}

type BudgetLineAddCmd struct {
	BudgetID   int64   `required:"" placeholder:"INT-64" help:"Budget ID."`
	Name       string  `required:"" help:"Budget line name."`
	Amount     string  `required:"" help:"Allocation amount."`
	Categories *string `help:"Comma-separated category codes."`
	SortOrder  *int32  `help:"Display sort order."`
}

func (c *BudgetLineAddCmd) Run(ctx *runContext) error {
	line, err := ctx.budgets.CreateBudgetLine(ctx.Context, c.BudgetID, api.CreateBudgetLineRequest{
		Name:             c.Name,
		AllocationAmount: c.Amount,
		CategoryCodes:    parseOptionalCSV(c.Categories),
		SortOrder:        c.SortOrder,
	})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, line)
}

type BudgetLineUpdateCmd struct {
	ID         int64   `arg:"" required:"" help:"Budget line ID."`
	Name       *string `help:"Replacement budget line name."`
	Amount     *string `help:"Replacement allocation amount."`
	Categories *string `help:"Replacement comma-separated category codes."`
	SortOrder  *int32  `help:"Replacement display sort order."`
}

func (c *BudgetLineUpdateCmd) Run(ctx *runContext) error {
	var categoryCodes *[]string
	if c.Categories != nil {
		parsed := parseOptionalCSV(c.Categories)
		categoryCodes = &parsed
	}
	line, err := ctx.budgets.UpdateBudgetLine(ctx.Context, c.ID, api.UpdateBudgetLineRequest{
		Name:             c.Name,
		AllocationAmount: c.Amount,
		CategoryCodes:    categoryCodes,
		SortOrder:        c.SortOrder,
	})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, line)
}

type BudgetLineDeleteCmd struct {
	ID int64 `arg:"" required:"" help:"Budget line ID."`
}

func (c *BudgetLineDeleteCmd) Run(ctx *runContext) error {
	return ctx.budgets.DeleteBudgetLine(ctx.Context, c.ID)
}
