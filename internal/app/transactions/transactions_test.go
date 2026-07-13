package transactions

import (
	"context"
	"errors"
	"testing"
	"time"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
)

type fakeRepository struct {
	nextID         int64
	items          map[int64]Transaction
	hashes         map[string]int64
	createCalls    int
	failCreateCall int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{nextID: 100, items: map[int64]Transaction{}, hashes: map[string]int64{}}
}
func (f *fakeRepository) Create(_ context.Context, input NewTransaction) (Transaction, error) {
	f.createCalls++
	if f.failCreateCall == f.createCalls {
		return Transaction{}, errors.New("database unavailable")
	}
	if _, exists := f.hashes[input.Hash]; exists {
		return Transaction{}, apperrors.Conflict(apperrors.CodeDuplicateTransaction, "duplicate transaction", nil)
	}
	f.nextID++
	item := Transaction{ID: f.nextID, Hash: input.Hash, Amount: input.Amount, TransactionDate: input.TransactionDate, AuthorID: input.AuthorID, HouseholdID: input.HouseholdID, CategoryID: input.CategoryID, Description: input.Description, Notes: input.Notes}
	f.items[item.ID], f.hashes[item.Hash] = item, item.ID
	return item, nil
}
func (f *fakeRepository) Get(_ context.Context, id int64, includeDeleted bool) (Transaction, error) {
	item, ok := f.items[id]
	if !ok || (item.DeletedAt != nil && !includeDeleted) {
		return Transaction{}, apperrors.NotFound(apperrors.CodeTransactionNotFound, "transaction not found", nil)
	}
	return item, nil
}
func (*fakeRepository) List(context.Context, ListFilter) ([]Transaction, error) { return nil, nil }
func (f *fakeRepository) Update(_ context.Context, id int64, update Mutation) (Transaction, error) {
	item, ok := f.items[id]
	if !ok {
		return Transaction{}, apperrors.NotFound(apperrors.CodeTransactionNotFound, "transaction not found", nil)
	}
	item = applyMutation(item, update)
	item.Hash = update.Hash
	f.items[id] = item
	return item, nil
}
func (f *fakeRepository) SoftDelete(_ context.Context, input DeleteInput) (Transaction, error) {
	item, ok := f.items[input.ID]
	if !ok {
		return Transaction{}, apperrors.NotFound(apperrors.CodeTransactionNotFound, "transaction not found", nil)
	}
	now := time.Now()
	item.DeletedAt = &now
	item.DeletedByUserID = &input.DeletedByUserID
	f.items[input.ID] = item
	return item, nil
}
func (f *fakeRepository) Restore(_ context.Context, input RestoreInput) (Transaction, error) {
	item, ok := f.items[input.ID]
	if !ok {
		return Transaction{}, apperrors.NotFound(apperrors.CodeTransactionNotFound, "transaction not found", nil)
	}
	item.DeletedAt = nil
	f.items[input.ID] = item
	return item, nil
}

type fakeIdentities struct{}

func (fakeIdentities) ResolveUserID(context.Context, IdentitySelector) (int64, error) { return 7, nil }

type fakeCategories struct{}

func (fakeCategories) ResolveActiveCategoryID(_ context.Context, id *int64, _ *string) (*int64, error) {
	return id, nil
}

func TestSingleTransactionLifecycleAndHash(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo, fakeIdentities{}, fakeCategories{})
	householdID, categoryID := int64(2), int64(42)
	date := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	description := "Coffee"
	created, err := service.Create(context.Background(), CreateInput{Amount: 4.25, TransactionDate: date, Description: &description, HouseholdID: &householdID, CategoryID: &categoryID})
	if err != nil {
		t.Fatalf("Create error=%v", err)
	}
	wantHash, _ := Hash(&description, date, 7, &householdID, &categoryID, 4.25)
	if created.Hash != wantHash {
		t.Fatalf("hash=%q want=%q", created.Hash, wantHash)
	}
	amount := float32(5)
	updated, err := service.Update(context.Background(), UpdateInput{ID: created.ID, Amount: &amount, ClearCategoryID: true})
	if err != nil || updated.Amount != 5 || updated.CategoryID != nil {
		t.Fatalf("Update=%+v error=%v", updated, err)
	}
	if _, err := service.SoftDelete(context.Background(), DeleteInput{ID: created.ID, DeletedByUserID: 7}); err != nil {
		t.Fatalf("SoftDelete error=%v", err)
	}
	if _, err := service.Get(context.Background(), created.ID, false); !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Fatalf("Get deleted error=%v", err)
	}
	if _, err := service.Restore(context.Background(), RestoreInput{ID: created.ID, RestoredByUserID: 7}); err != nil {
		t.Fatalf("Restore error=%v", err)
	}
	items, err := service.List(context.Background(), ListFilter{})
	if err != nil || items == nil {
		t.Fatalf("List=%#v error=%v", items, err)
	}
}

func TestBatchMarksInfrastructureFailureAndContinues(t *testing.T) {
	repo := newFakeRepository()
	repo.failCreateCall = 1
	service := NewService(repo, fakeIdentities{}, fakeCategories{})
	householdID := int64(2)
	date := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	result := service.CreateBatch(context.Background(), []CreateInput{
		{Amount: 10, TransactionDate: date, HouseholdID: &householdID},
		{Amount: 20, TransactionDate: date, HouseholdID: &householdID},
	})
	if len(result.Failed) != 1 || result.Failed[0].Index != 0 || apperrors.CodeOf(result.Failed[0].Error) != apperrors.CodeInternal {
		t.Fatalf("Failed=%+v, want safe internal failure at index 0", result.Failed)
	}
	if len(result.Succeeded) != 1 || result.Succeeded[0].Index != 1 {
		t.Fatalf("Succeeded=%+v, want index 1 committed", result.Succeeded)
	}
}

func TestBatchesAccountForEveryInputInOrder(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(repo, fakeIdentities{}, fakeCategories{})
	householdID := int64(2)
	date := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	inputs := []CreateInput{
		{Amount: 10, TransactionDate: date, HouseholdID: &householdID},
		{Amount: 0, TransactionDate: date, HouseholdID: &householdID},
		{Amount: 10, TransactionDate: date, HouseholdID: &householdID},
	}
	result := service.CreateBatch(context.Background(), inputs)
	if len(result.Succeeded) != 1 || result.Succeeded[0].Index != 0 {
		t.Fatalf("Succeeded=%+v", result.Succeeded)
	}
	if len(result.Failed) != 2 || result.Failed[0].Index != 1 || result.Failed[1].Index != 2 {
		t.Fatalf("Failed=%+v", result.Failed)
	}

	deleteResult := service.DeleteBatch(context.Background(), []int64{result.Succeeded[0].ID, 999}, 7, nil)
	if len(deleteResult.Succeeded) != 1 || deleteResult.Succeeded[0].Index != 0 || len(deleteResult.Failed) != 1 || deleteResult.Failed[0].Index != 1 || deleteResult.Failed[0].ID == nil || *deleteResult.Failed[0].ID != 999 {
		t.Fatalf("DeleteBatch=%+v", deleteResult)
	}
	restoreResult := service.RestoreBatch(context.Background(), []int64{999, result.Succeeded[0].ID}, 7)
	if len(restoreResult.Failed) != 1 || restoreResult.Failed[0].Index != 0 || len(restoreResult.Succeeded) != 1 || restoreResult.Succeeded[0].Index != 1 {
		t.Fatalf("RestoreBatch=%+v", restoreResult)
	}
}
