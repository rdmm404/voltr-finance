package cli

import "rdmm404/voltr-finance/internal/api"

type CategoriesCmd struct {
	Create     CategoryCreateCmd     `cmd:"" help:"Create a category."`
	List       CategoryListCmd       `cmd:"" help:"List categories."`
	Rename     CategoryRenameCmd     `cmd:"" help:"Rename a category by code."`
	Deactivate CategoryDeactivateCmd `cmd:"" help:"Deactivate a category by code."`
}

type CategoryCreateCmd struct {
	Name        string  `arg:"" required:"" help:"Category display name."`
	Code        *string `help:"Stable category code. Defaults to a slug generated from name."`
	Description *string `help:"Optional category description."`
}

func (c *CategoryCreateCmd) Run(ctx *runContext) error {
	category, err := ctx.categories.CreateCategory(ctx.Context, api.CreateCategoryRequest{
		Name:        c.Name,
		Code:        c.Code,
		Description: c.Description,
	})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, category)
}

type CategoryListCmd struct {
	IncludeInactive bool `help:"Include inactive categories."`
}

func (c *CategoryListCmd) Run(ctx *runContext) error {
	categories, err := ctx.categories.ListCategories(ctx.Context, api.ListCategoriesQuery{IncludeInactive: c.IncludeInactive})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, categories)
}

type CategoryRenameCmd struct {
	Code string `arg:"" required:"" help:"Existing category code."`
	Name string `arg:"" required:"" help:"New category display name."`
}

func (c *CategoryRenameCmd) Run(ctx *runContext) error {
	category, err := ctx.categories.UpdateCategory(ctx.Context, c.Code, api.UpdateCategoryRequest{Name: &c.Name})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, category)
}

type CategoryDeactivateCmd struct {
	Code string `arg:"" required:"" help:"Category code to deactivate."`
}

func (c *CategoryDeactivateCmd) Run(ctx *runContext) error {
	category, err := ctx.categories.DeactivateCategory(ctx.Context, c.Code)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, category)
}
