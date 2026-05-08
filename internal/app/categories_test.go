package app

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"rdmm404/voltr-finance/internal/database/sqlc"
)

func TestCreateCategoryGeneratesCode(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeTransactionService{})

	category, err := svc.CreateCategory(context.Background(), CreateCategoryRequest{
		Name: "Restaurants & Takeout",
	})

	if err != nil {
		t.Fatalf("CreateCategory returned error: %v", err)
	}
	if category.Code != "restaurants-takeout" {
		t.Fatalf("Code = %q, want restaurants-takeout", category.Code)
	}
	if repo.lastCreateCategory.Code != "restaurants-takeout" || repo.lastCreateCategory.Name != "Restaurants & Takeout" {
		t.Fatalf("CreateCategoryParams = %+v, want generated code and name", repo.lastCreateCategory)
	}
}

func TestCreateCategoryGeneratesASCIICodeFromUnicodeName(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeTransactionService{})

	category, err := svc.CreateCategory(context.Background(), CreateCategoryRequest{
		Name: "Café au lait",
	})

	if err != nil {
		t.Fatalf("CreateCategory returned error: %v", err)
	}
	if category.Code != "caf-au-lait" {
		t.Fatalf("Code = %q, want caf-au-lait", category.Code)
	}
	if repo.lastCreateCategory.Code != "caf-au-lait" {
		t.Fatalf("CreateCategoryParams.Code = %q, want caf-au-lait", repo.lastCreateCategory.Code)
	}
}

func TestCreateCategoryAcceptsExplicitCode(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, &fakeTransactionService{})

	category, err := svc.CreateCategory(context.Background(), CreateCategoryRequest{
		Name: "Restaurants & Takeout",
		Code: strPtr("restaurants"),
	})

	if err != nil {
		t.Fatalf("CreateCategory returned error: %v", err)
	}
	if category.Code != "restaurants" {
		t.Fatalf("Code = %q, want restaurants", category.Code)
	}
}

func TestCreateCategoryRejectsInvalidCode(t *testing.T) {
	svc := NewService(&fakeRepo{}, &fakeTransactionService{})

	_, err := svc.CreateCategory(context.Background(), CreateCategoryRequest{
		Name: "Groceries",
		Code: strPtr("Groceries!"),
	})

	if appErr, ok := err.(*AppError); !ok || appErr.Code != CodeValidationError {
		t.Fatalf("err = %v, want validation error", err)
	}
}

func TestListCategoriesMapsRows(t *testing.T) {
	repo := &fakeRepo{listCategoriesResult: []sqlc.Category{
		{ID: 1, Code: "groceries", Name: "Groceries", IsActive: true},
		{ID: 2, Code: "utilities", Name: "Utilities", IsActive: true},
	}}
	svc := NewService(repo, &fakeTransactionService{})

	categories, err := svc.ListCategories(context.Background(), ListCategoriesRequest{
		IncludeInactive: true,
	})

	if err != nil {
		t.Fatalf("ListCategories returned error: %v", err)
	}
	if len(categories) != 2 || categories[0].Code != "groceries" || categories[1].Code != "utilities" {
		t.Fatalf("categories = %+v, want mapped category DTOs", categories)
	}
	if !repo.lastListCategoriesIncludeInactive {
		t.Fatalf("ListCategories includeInactive = false, want true")
	}
}

func TestMapCategoryErrorMapsUniqueViolationToValidationError(t *testing.T) {
	err := mapCategoryError(&pgconn.PgError{Code: "23505"})

	appErr, ok := err.(*AppError)
	if !ok {
		t.Fatalf("err = %T, want *AppError", err)
	}
	if appErr.Code != CodeValidationError {
		t.Fatalf("Code = %q, want %q", appErr.Code, CodeValidationError)
	}
	if appErr.Message != "category code already exists" {
		t.Fatalf("Message = %q, want category code already exists", appErr.Message)
	}
}
