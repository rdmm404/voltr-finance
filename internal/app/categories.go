package app

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"rdmm404/voltr-finance/internal/database"
	"rdmm404/voltr-finance/internal/database/sqlc"
)

var categoryCodePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type CategoryDTO struct {
	ID          int64   `json:"id"`
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	IsActive    bool    `json:"isActive"`
}

type CategoryRefDTO struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type CreateCategoryRequest struct {
	Name        string  `json:"name"`
	Code        *string `json:"code,omitempty"`
	Description *string `json:"description,omitempty"`
}

type ListCategoriesRequest struct {
	IncludeInactive bool `json:"includeInactive,omitempty"`
}

type UpdateCategoryRequest struct {
	ID               int64   `json:"id"`
	Name             *string `json:"name,omitempty"`
	Description      *string `json:"description,omitempty"`
	ClearDescription bool    `json:"clearDescription,omitempty"`
}

func (s *Service) CreateCategory(ctx context.Context, req CreateCategoryRequest) (CategoryDTO, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return CategoryDTO{}, NewError(CodeValidationError, "category name is required", nil)
	}

	code := categoryCodeFromName(name)
	if req.Code != nil {
		code = strings.TrimSpace(*req.Code)
	}
	if !categoryCodePattern.MatchString(code) {
		return CategoryDTO{}, NewError(CodeValidationError, "category code must be a lowercase slug", nil)
	}

	category, err := s.repo.CreateCategory(ctx, sqlc.CreateCategoryParams{
		Code:        code,
		Name:        name,
		Description: req.Description,
	})
	if err != nil {
		return CategoryDTO{}, mapCategoryError(err)
	}
	return categoryDTO(category), nil
}

func (s *Service) ListCategories(ctx context.Context, req ListCategoriesRequest) ([]CategoryDTO, error) {
	rows, err := s.repo.ListCategories(ctx, req.IncludeInactive)
	if err != nil {
		return nil, mapCategoryError(err)
	}

	categories := make([]CategoryDTO, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, categoryDTO(row))
	}
	return categories, nil
}

func (s *Service) GetCategoryByCode(ctx context.Context, code string) (CategoryDTO, error) {
	code = strings.TrimSpace(code)
	if !categoryCodePattern.MatchString(code) {
		return CategoryDTO{}, NewError(CodeValidationError, "category code must be a lowercase slug", nil)
	}

	category, err := s.repo.GetCategoryByCode(ctx, code)
	if err != nil {
		return CategoryDTO{}, mapCategoryError(err)
	}
	return categoryDTO(category), nil
}

func (s *Service) UpdateCategory(ctx context.Context, req UpdateCategoryRequest) (CategoryDTO, error) {
	if req.ID == 0 {
		return CategoryDTO{}, NewError(CodeValidationError, "category id is required", nil)
	}
	if req.Name == nil && req.Description == nil && !req.ClearDescription {
		return CategoryDTO{}, NewError(CodeValidationError, "at least one category field is required", nil)
	}

	name := ""
	setName := false
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
		if name == "" {
			return CategoryDTO{}, NewError(CodeValidationError, "category name is required", nil)
		}
		setName = true
	}

	description := req.Description
	if req.ClearDescription {
		description = nil
	}

	category, err := s.repo.UpdateCategory(ctx, sqlc.UpdateCategoryParams{
		ID:             req.ID,
		SetName:        setName,
		Name:           name,
		SetDescription: req.Description != nil || req.ClearDescription,
		Description:    description,
	})
	if err != nil {
		return CategoryDTO{}, mapCategoryError(err)
	}
	return categoryDTO(category), nil
}

func (s *Service) DeactivateCategory(ctx context.Context, code string) (CategoryDTO, error) {
	code = strings.TrimSpace(code)
	if !categoryCodePattern.MatchString(code) {
		return CategoryDTO{}, NewError(CodeValidationError, "category code must be a lowercase slug", nil)
	}

	category, err := s.repo.DeactivateCategory(ctx, code)
	if err != nil {
		return CategoryDTO{}, mapCategoryError(err)
	}
	return categoryDTO(category), nil
}

func categoryCodeFromName(name string) string {
	var b strings.Builder
	lastHyphen := true
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
			lastHyphen = false
		default:
			if !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func categoryDTO(category sqlc.Category) CategoryDTO {
	return CategoryDTO{
		ID:          category.ID,
		Code:        category.Code,
		Name:        category.Name,
		Description: category.Description,
		IsActive:    category.IsActive,
	}
}

func mapCategoryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return NewError(CodeValidationError, "category not found", err)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && database.PgErrorCode(pgErr.Code) == database.ErrorCodeUniqueViolation {
		return NewError(CodeValidationError, "category code already exists", err)
	}
	return NewError(CodeDatabaseError, "database error", err)
}
