package categories

import (
	"context"

	appcategories "rdmm404/voltr-finance/internal/app/categories"
	apperrors "rdmm404/voltr-finance/internal/app/errors"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/postgres"
)

type queries interface {
	CreateCategory(context.Context, sqlc.CreateCategoryParams) (sqlc.Category, error)
	ListCategories(context.Context, bool) ([]sqlc.Category, error)
	GetCategoryByCode(context.Context, string) (sqlc.Category, error)
	GetActiveCategoryById(context.Context, int64) (sqlc.Category, error)
	GetActiveCategoryByCode(context.Context, string) (sqlc.Category, error)
	UpdateCategory(context.Context, sqlc.UpdateCategoryParams) (sqlc.Category, error)
	DeactivateCategory(context.Context, string) (sqlc.Category, error)
}

type Repository struct{ queries queries }

func NewRepository(queries queries) *Repository { return &Repository{queries: queries} }

func (r *Repository) Create(ctx context.Context, input appcategories.CreateInput) (appcategories.Category, error) {
	code := ""
	if input.Code != nil {
		code = *input.Code
	}
	row, err := r.queries.CreateCategory(ctx, sqlc.CreateCategoryParams{Code: code, Name: input.Name, Description: input.Description})
	return mapCategory(row), mapError(err)
}
func (r *Repository) List(ctx context.Context, includeInactive bool) ([]appcategories.Category, error) {
	rows, err := r.queries.ListCategories(ctx, includeInactive)
	if err != nil {
		return nil, mapError(err)
	}
	items := make([]appcategories.Category, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapCategory(row))
	}
	return items, nil
}
func (r *Repository) GetByCode(ctx context.Context, code string) (appcategories.Category, error) {
	row, err := r.queries.GetCategoryByCode(ctx, code)
	return mapCategory(row), mapError(err)
}
func (r *Repository) GetActiveByID(ctx context.Context, id int64) (appcategories.Category, error) {
	row, err := r.queries.GetActiveCategoryById(ctx, id)
	return mapCategory(row), mapError(err)
}
func (r *Repository) GetActiveByCode(ctx context.Context, code string) (appcategories.Category, error) {
	row, err := r.queries.GetActiveCategoryByCode(ctx, code)
	return mapCategory(row), mapError(err)
}
func (r *Repository) Update(ctx context.Context, id int64, input appcategories.Update) (appcategories.Category, error) {
	name := ""
	if input.Name != nil {
		name = *input.Name
	}
	row, err := r.queries.UpdateCategory(ctx, sqlc.UpdateCategoryParams{SetName: input.Name != nil, Name: name, SetDescription: input.SetDescription, Description: input.Description, ID: id})
	return mapCategory(row), mapError(err)
}
func (r *Repository) Deactivate(ctx context.Context, code string) (appcategories.Category, error) {
	row, err := r.queries.DeactivateCategory(ctx, code)
	return mapCategory(row), mapError(err)
}

func mapCategory(row sqlc.Category) appcategories.Category {
	return appcategories.Category{ID: row.ID, Code: row.Code, Name: row.Name, Description: row.Description, IsActive: row.IsActive}
}
func mapError(err error) error {
	return postgres.MapError(err, postgres.ErrorMapping{NotFoundCode: apperrors.CodeCategoryNotFound, NotFoundMessage: "category not found", ConflictCode: apperrors.CodeCategoryConflict, ConflictMessage: "category already exists or violates an invariant"})
}

var _ appcategories.Repository = (*Repository)(nil)
