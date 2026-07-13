package cli

import "rdmm404/voltr-finance/internal/api"

type HouseholdsCmd struct {
	Get   HouseholdGetCmd   `cmd:"" help:"Get a household from exactly one selector."`
	List  HouseholdListCmd  `cmd:"" help:"List all households."`
	Users HouseholdUsersCmd `cmd:"" help:"List users in a household."`
}

type HouseholdGetCmd struct {
	ID      *int64  `placeholder:"INT-64" help:"Internal household ID. Exactly one household selector is required."`
	GuildID *string `help:"Discord guild/server ID. Exactly one household selector is required."`
	Name    *string `help:"Household name. Exactly one household selector is required."`
}

func (c *HouseholdGetCmd) Run(ctx *runContext) error {
	var household api.Household
	var err error
	if c.ID != nil {
		household, err = ctx.households.GetHousehold(ctx.Context, *c.ID)
	} else {
		household, err = ctx.households.ResolveHousehold(ctx.Context, api.ResolveHouseholdQuery{Name: c.Name, GuildID: c.GuildID})
	}
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, household)
}

type HouseholdListCmd struct{}

func (c *HouseholdListCmd) Run(ctx *runContext) error {
	households, err := ctx.households.ListHouseholds(ctx.Context)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, households)
}

type HouseholdUsersCmd struct {
	HouseholdID int64 `required:"" help:"Internal household ID."`
}

func (c *HouseholdUsersCmd) Run(ctx *runContext) error {
	users, err := ctx.households.ListHouseholdUsers(ctx.Context, c.HouseholdID)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, users)
}
