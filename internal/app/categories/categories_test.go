package categories

import (
	"context"
	"testing"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
	"rdmm404/voltr-finance/internal/app/patch"
)

type fakeRepository struct {
	create          CreateInput
	update          Update
	includeInactive bool
}

func (f *fakeRepository) Create(_ context.Context, input CreateInput) (Category, error) {
	f.create = input
	return Category{ID: 1, Code: *input.Code, Name: input.Name}, nil
}
func (f *fakeRepository) List(_ context.Context, include bool) ([]Category, error) {
	f.includeInactive = include
	return nil, nil
}
func (*fakeRepository) GetByCode(_ context.Context, code string) (Category, error) {
	return Category{ID: 1, Code: code}, nil
}
func (*fakeRepository) GetActiveByID(_ context.Context, id int64) (Category, error) {
	return Category{ID: id, Code: "food"}, nil
}
func (*fakeRepository) GetActiveByCode(_ context.Context, code string) (Category, error) {
	return Category{ID: 3, Code: code}, nil
}
func (f *fakeRepository) Update(_ context.Context, code string, update Update) (Category, error) {
	f.update = update
	return Category{ID: 1, Code: code}, nil
}
func (*fakeRepository) Deactivate(_ context.Context, code string) (Category, error) {
	return Category{ID: 1, Code: code, IsActive: false}, nil
}

func TestServiceCategoryLifecycleAndErrorContract(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)
	item, err := service.Create(context.Background(), CreateInput{Name: "Restaurants & Takeout"})
	if err != nil || item.Code != "restaurants-takeout" {
		t.Fatalf("Create=%+v error=%v", item, err)
	}
	if _, err := service.GetByCode(context.Background(), "INVALID"); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("GetByCode error=%v", err)
	}
	id, otherCode := int64(2), "food"
	if _, err := service.ResolveActive(context.Background(), &id, &otherCode); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("ResolveActive error=%v", err)
	}
	if _, err := service.Update(context.Background(), UpdateInput{Code: "food", Description: patch.Clear[string]()}); err != nil || !repo.update.Description.Present() || repo.update.Description.Value() != nil {
		t.Fatalf("Update error=%v update=%+v", err, repo.update)
	}
	items, err := service.List(context.Background(), true)
	if err != nil || items == nil || !repo.includeInactive {
		t.Fatalf("List=%#v include=%v error=%v", items, repo.includeInactive, err)
	}
	if item, err := service.Deactivate(context.Background(), "food"); err != nil || item.IsActive {
		t.Fatalf("Deactivate=%+v error=%v", item, err)
	}
}
