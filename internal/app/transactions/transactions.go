package transactions

import (
	"context"
	"fmt"
	"time"

	"github.com/cespare/xxhash"
	"github.com/jxskiss/base62"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
	"rdmm404/voltr-finance/internal/app/patch"
)

type CategoryRef struct {
	ID   int64
	Code string
	Name string
}

type Transaction struct {
	ID              int64
	Hash            string
	Amount          float32
	TransactionDate time.Time
	AuthorID        int64
	AuthorName      string
	HouseholdID     *int64
	HouseholdName   *string
	CategoryID      *int64
	Category        *CategoryRef
	Description     *string
	Notes           *string
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
	DeletedAt       *time.Time
	DeletedByUserID *int64
	DeleteReason    *string
}

type IdentitySelector struct {
	UserID      *int64
	DiscordID   *string
	TelegramID  *string
	PhoneNumber *string
	WhatsAppID  *string
}

type CreateInput struct {
	Amount          float32
	TransactionDate time.Time
	Description     *string
	Notes           *string
	CategoryID      *int64
	CategoryCode    *string
	HouseholdID     *int64
	Author          IdentitySelector
}

type NewTransaction struct {
	Hash            string
	Amount          float32
	TransactionDate time.Time
	Description     *string
	Notes           *string
	CategoryID      *int64
	HouseholdID     *int64
	AuthorID        int64
}

type CategorySelector struct {
	ID   *int64
	Code *string
}

type UpdateInput struct {
	ID              int64
	Amount          *float32
	TransactionDate *time.Time
	Description     patch.Field[string]
	Notes           patch.Field[string]
	Category        patch.Field[CategorySelector]
	HouseholdID     patch.Field[int64]
	Author          *IdentitySelector
}

type Mutation struct {
	Amount          *float32
	TransactionDate *time.Time
	Description     patch.Field[string]
	Notes           patch.Field[string]
	CategoryID      patch.Field[int64]
	HouseholdID     patch.Field[int64]
	AuthorID        *int64
}

type ListFilter struct {
	AuthorID       *int64
	HouseholdID    *int64
	FromDate       *time.Time
	ToDate         *time.Time
	Search         *string
	Sort           string
	SortOrder      string
	Limit          int32
	Offset         int32
	IncludeDeleted bool
	OnlyDeleted    bool
}

type DeleteInput struct {
	ID              int64
	DeletedByUserID int64
	Reason          *string
}

type RestoreInput struct {
	ID               int64
	RestoredByUserID int64
}

type Repository interface {
	Create(context.Context, NewTransaction) (Transaction, error)
	Get(context.Context, int64, bool) (Transaction, error)
	List(context.Context, ListFilter) ([]Transaction, error)
	Update(context.Context, int64, Mutation) (Transaction, error)
	SoftDelete(context.Context, DeleteInput) (Transaction, error)
	Restore(context.Context, RestoreInput) (Transaction, error)
}

type IdentityResolver interface {
	ResolveUserID(context.Context, IdentitySelector) (int64, error)
}

type CategoryResolver interface {
	ResolveActiveCategoryID(context.Context, *int64, *string) (*int64, error)
}

type Succeeded struct {
	Index int
	ID    int64
}

type Failed struct {
	Index int
	ID    *int64
	Error error
}

type BulkResult struct {
	Succeeded []Succeeded
	Failed    []Failed
}

type Service struct {
	repo       Repository
	identities IdentityResolver
	categories CategoryResolver
}

func NewService(repo Repository, identities IdentityResolver, categories CategoryResolver) *Service {
	return &Service{repo: repo, identities: identities, categories: categories}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Transaction, error) {
	newTransaction, err := s.prepareCreate(ctx, input)
	if err != nil {
		return Transaction{}, err
	}
	item, err := s.repo.Create(ctx, newTransaction)
	return item, apperrors.WrapInternal("create transaction", err)
}

func (s *Service) CreateBatch(ctx context.Context, inputs []CreateInput) BulkResult {
	return runBulk(inputs, func(CreateInput) *int64 { return nil }, func(input CreateInput) (int64, error) {
		item, err := s.Create(ctx, input)
		return item.ID, err
	})
}

func (s *Service) Get(ctx context.Context, id int64, includeDeleted bool) (Transaction, error) {
	if id == 0 {
		return Transaction{}, apperrors.Validation("transaction id is required")
	}
	item, err := s.repo.Get(ctx, id, includeDeleted)
	return item, apperrors.WrapInternal("get transaction", err)
}

func (s *Service) GetMany(ctx context.Context, ids []int64, includeDeleted bool) ([]Transaction, error) {
	items := make([]Transaction, 0, len(ids))
	for _, id := range ids {
		item, err := s.Get(ctx, id, includeDeleted)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]Transaction, error) {
	if filter.Sort == "" {
		filter.Sort = "transaction_date"
	}
	if filter.SortOrder == "" {
		filter.SortOrder = "desc"
	}
	if filter.Limit == 0 {
		filter.Limit = 100
	}
	items, err := s.repo.List(ctx, filter)
	if items == nil && err == nil {
		items = []Transaction{}
	}
	return items, apperrors.WrapInternal("list transactions", err)
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (Transaction, error) {
	if input.ID == 0 {
		return Transaction{}, apperrors.Validation("transaction id is required")
	}
	mutation, err := s.prepareUpdate(ctx, input)
	if err != nil {
		return Transaction{}, err
	}
	item, err := s.repo.Update(ctx, input.ID, mutation)
	return item, apperrors.WrapInternal("update transaction", err)
}

func (s *Service) UpdateBatch(ctx context.Context, inputs []UpdateInput) BulkResult {
	return runBulk(inputs, func(input UpdateInput) *int64 { return knownID(input.ID) }, func(input UpdateInput) (int64, error) {
		item, err := s.Update(ctx, input)
		return item.ID, err
	})
}

func (s *Service) SoftDelete(ctx context.Context, input DeleteInput) (Transaction, error) {
	if input.ID == 0 || input.DeletedByUserID == 0 {
		return Transaction{}, apperrors.Validation("transaction id and deleted by user id are required")
	}
	item, err := s.repo.SoftDelete(ctx, input)
	return item, apperrors.WrapInternal("delete transaction", err)
}

func (s *Service) DeleteBatch(ctx context.Context, ids []int64, deletedByUserID int64, reason *string) BulkResult {
	return runBulk(ids, knownID, func(id int64) (int64, error) {
		item, err := s.SoftDelete(ctx, DeleteInput{ID: id, DeletedByUserID: deletedByUserID, Reason: reason})
		return item.ID, err
	})
}

func (s *Service) Restore(ctx context.Context, input RestoreInput) (Transaction, error) {
	if input.ID == 0 || input.RestoredByUserID == 0 {
		return Transaction{}, apperrors.Validation("transaction id and restored by user id are required")
	}
	item, err := s.repo.Restore(ctx, input)
	return item, apperrors.WrapInternal("restore transaction", err)
}

func (s *Service) RestoreBatch(ctx context.Context, ids []int64, restoredByUserID int64) BulkResult {
	return runBulk(ids, knownID, func(id int64) (int64, error) {
		item, err := s.Restore(ctx, RestoreInput{ID: id, RestoredByUserID: restoredByUserID})
		return item.ID, err
	})
}

func (s *Service) prepareCreate(ctx context.Context, input CreateInput) (NewTransaction, error) {
	if input.Amount == 0 {
		return NewTransaction{}, apperrors.Validation("amount is required")
	}
	if input.TransactionDate.IsZero() {
		return NewTransaction{}, apperrors.Validation("transaction date is required")
	}
	if input.HouseholdID == nil {
		return NewTransaction{}, apperrors.Validation("household id is required")
	}
	authorID, err := s.identities.ResolveUserID(ctx, input.Author)
	if err != nil {
		return NewTransaction{}, apperrors.Normalize(err)
	}
	categoryID, err := s.categories.ResolveActiveCategoryID(ctx, input.CategoryID, input.CategoryCode)
	if err != nil {
		return NewTransaction{}, apperrors.Normalize(err)
	}
	hash, err := Hash(input.Description, input.TransactionDate, authorID, input.HouseholdID, categoryID, input.Amount)
	if err != nil {
		return NewTransaction{}, err
	}
	return NewTransaction{Hash: hash, Amount: input.Amount, TransactionDate: input.TransactionDate, Description: input.Description, Notes: input.Notes, CategoryID: categoryID, HouseholdID: input.HouseholdID, AuthorID: authorID}, nil
}

func (s *Service) prepareUpdate(ctx context.Context, input UpdateInput) (Mutation, error) {
	mutation := Mutation{Amount: input.Amount, TransactionDate: input.TransactionDate, Description: input.Description, Notes: input.Notes, HouseholdID: input.HouseholdID}
	if input.Amount != nil && *input.Amount == 0 {
		return Mutation{}, apperrors.Validation("amount is required")
	}
	if input.TransactionDate != nil && input.TransactionDate.IsZero() {
		return Mutation{}, apperrors.Validation("transaction date is required")
	}
	if input.Category.Present() {
		selector := input.Category.Value()
		if selector == nil {
			mutation.CategoryID = patch.Clear[int64]()
		} else {
			categoryID, err := s.categories.ResolveActiveCategoryID(ctx, selector.ID, selector.Code)
			if err != nil {
				return Mutation{}, apperrors.Normalize(err)
			}
			if categoryID == nil {
				mutation.CategoryID = patch.Clear[int64]()
			} else {
				mutation.CategoryID = patch.Set(*categoryID)
			}
		}
	}
	if input.Author != nil {
		authorID, err := s.identities.ResolveUserID(ctx, *input.Author)
		if err != nil {
			return Mutation{}, apperrors.Normalize(err)
		}
		mutation.AuthorID = &authorID
	}
	return mutation, nil
}

// Apply merges a validated mutation into the row locked by the persistence
// adapter. Keeping this operation and Hash in the application package ensures
// the adapter cannot invent domain merge or identity semantics.
func (update Mutation) Apply(item Transaction) Transaction {
	if update.Amount != nil {
		item.Amount = *update.Amount
	}
	if update.TransactionDate != nil {
		item.TransactionDate = *update.TransactionDate
	}
	if update.Description.Present() {
		item.Description = update.Description.Value()
	}
	if update.Notes.Present() {
		item.Notes = update.Notes.Value()
	}
	if update.CategoryID.Present() {
		item.CategoryID = update.CategoryID.Value()
	}
	if update.HouseholdID.Present() {
		item.HouseholdID = update.HouseholdID.Value()
	}
	if update.AuthorID != nil {
		item.AuthorID = *update.AuthorID
	}
	return item
}

func Hash(description *string, transactionDate time.Time, authorID int64, householdID, categoryID *int64, amount float32) (string, error) {
	if authorID == 0 && (householdID == nil || *householdID == 0) {
		return "", apperrors.Validation("either author id or household id must be set")
	}
	descriptionValue := ""
	if description != nil {
		descriptionValue = *description
	}
	householdValue := int64(0)
	if householdID != nil {
		householdValue = *householdID
	}
	h := xxhash.New()
	if categoryID == nil {
		fmt.Fprintf(h, "%s|%d|%d|%d|%.2f", descriptionValue, transactionDate.Unix(), authorID, householdValue, amount)
	} else {
		fmt.Fprintf(h, "%s|%d|%d|%d|%d|%.2f", descriptionValue, transactionDate.Unix(), authorID, householdValue, *categoryID, amount)
	}
	return base62.EncodeToString(h.Sum(nil)), nil
}

func runBulk[T any](inputs []T, known func(T) *int64, action func(T) (int64, error)) BulkResult {
	result := BulkResult{Succeeded: make([]Succeeded, 0, len(inputs)), Failed: make([]Failed, 0)}
	for index, input := range inputs {
		id, err := action(input)
		if err != nil {
			result.Failed = append(result.Failed, Failed{Index: index, ID: known(input), Error: err})
			continue
		}
		result.Succeeded = append(result.Succeeded, Succeeded{Index: index, ID: id})
	}
	return result
}

func knownID(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}
