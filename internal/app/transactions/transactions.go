package transactions

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/cespare/xxhash"
	"github.com/jxskiss/base62"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
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

type UpdateInput struct {
	ID               int64
	Amount           *float32
	TransactionDate  *time.Time
	Description      *string
	Notes            *string
	CategoryID       *int64
	CategoryCode     *string
	HouseholdID      *int64
	Author           *IdentitySelector
	ClearDescription bool
	ClearNotes       bool
	ClearCategoryID  bool
	ClearHouseholdID bool
}

type Mutation struct {
	Hash               string
	SetAmount          bool
	Amount             float32
	SetTransactionDate bool
	TransactionDate    time.Time
	SetDescription     bool
	Description        *string
	SetNotes           bool
	Notes              *string
	SetCategoryID      bool
	CategoryID         *int64
	SetHouseholdID     bool
	HouseholdID        *int64
	SetAuthorID        bool
	AuthorID           int64
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
	result := newBulkResult(len(inputs))
	for index, input := range inputs {
		item, err := s.Create(ctx, input)
		if err != nil {
			result.Failed = append(result.Failed, Failed{Index: index, Error: err})
			continue
		}
		result.Succeeded = append(result.Succeeded, Succeeded{Index: index, ID: item.ID})
	}
	return result
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
	existing, err := s.repo.Get(ctx, input.ID, true)
	if err != nil {
		return Transaction{}, apperrors.WrapInternal("get transaction for update", err)
	}
	mutation, err := s.prepareUpdate(ctx, existing, input)
	if err != nil {
		return Transaction{}, err
	}
	item, err := s.repo.Update(ctx, input.ID, mutation)
	return item, apperrors.WrapInternal("update transaction", err)
}

func (s *Service) UpdateBatch(ctx context.Context, inputs []UpdateInput) BulkResult {
	result := newBulkResult(len(inputs))
	for index, input := range inputs {
		item, err := s.Update(ctx, input)
		if err != nil {
			result.Failed = append(result.Failed, Failed{Index: index, ID: knownID(input.ID), Error: err})
			continue
		}
		result.Succeeded = append(result.Succeeded, Succeeded{Index: index, ID: item.ID})
	}
	return result
}

func (s *Service) SoftDelete(ctx context.Context, input DeleteInput) (Transaction, error) {
	if input.ID == 0 || input.DeletedByUserID == 0 {
		return Transaction{}, apperrors.Validation("transaction id and deleted by user id are required")
	}
	item, err := s.repo.SoftDelete(ctx, input)
	return item, apperrors.WrapInternal("delete transaction", err)
}

func (s *Service) DeleteBatch(ctx context.Context, ids []int64, deletedByUserID int64, reason *string) BulkResult {
	result := newBulkResult(len(ids))
	if deletedByUserID == 0 {
		for index, id := range ids {
			result.Failed = append(result.Failed, Failed{Index: index, ID: knownID(id), Error: apperrors.Validation("deleted by user id is required")})
		}
		return result
	}
	for index, id := range ids {
		item, err := s.SoftDelete(ctx, DeleteInput{ID: id, DeletedByUserID: deletedByUserID, Reason: reason})
		if err != nil {
			result.Failed = append(result.Failed, Failed{Index: index, ID: knownID(id), Error: err})
			continue
		}
		result.Succeeded = append(result.Succeeded, Succeeded{Index: index, ID: item.ID})
	}
	return result
}

func (s *Service) Restore(ctx context.Context, input RestoreInput) (Transaction, error) {
	if input.ID == 0 || input.RestoredByUserID == 0 {
		return Transaction{}, apperrors.Validation("transaction id and restored by user id are required")
	}
	item, err := s.repo.Restore(ctx, input)
	return item, apperrors.WrapInternal("restore transaction", err)
}

func (s *Service) RestoreBatch(ctx context.Context, ids []int64, restoredByUserID int64) BulkResult {
	result := newBulkResult(len(ids))
	if restoredByUserID == 0 {
		for index, id := range ids {
			result.Failed = append(result.Failed, Failed{Index: index, ID: knownID(id), Error: apperrors.Validation("restored by user id is required")})
		}
		return result
	}
	for index, id := range ids {
		item, err := s.Restore(ctx, RestoreInput{ID: id, RestoredByUserID: restoredByUserID})
		if err != nil {
			result.Failed = append(result.Failed, Failed{Index: index, ID: knownID(id), Error: err})
			continue
		}
		result.Succeeded = append(result.Succeeded, Succeeded{Index: index, ID: item.ID})
	}
	return result
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

func (s *Service) prepareUpdate(ctx context.Context, existing Transaction, input UpdateInput) (Mutation, error) {
	mutation := Mutation{}
	if input.Amount != nil {
		if *input.Amount == 0 {
			return Mutation{}, apperrors.Validation("amount is required")
		}
		mutation.SetAmount, mutation.Amount = true, *input.Amount
	}
	if input.TransactionDate != nil {
		if input.TransactionDate.IsZero() {
			return Mutation{}, apperrors.Validation("transaction date is required")
		}
		mutation.SetTransactionDate, mutation.TransactionDate = true, *input.TransactionDate
	}
	if input.Description != nil || input.ClearDescription {
		mutation.SetDescription, mutation.Description = true, input.Description
	}
	if input.Notes != nil || input.ClearNotes {
		mutation.SetNotes, mutation.Notes = true, input.Notes
	}
	if input.CategoryID != nil || input.CategoryCode != nil {
		categoryID, err := s.categories.ResolveActiveCategoryID(ctx, input.CategoryID, input.CategoryCode)
		if err != nil {
			return Mutation{}, apperrors.Normalize(err)
		}
		mutation.SetCategoryID, mutation.CategoryID = true, categoryID
	} else if input.ClearCategoryID {
		mutation.SetCategoryID = true
	}
	if input.HouseholdID != nil || input.ClearHouseholdID {
		mutation.SetHouseholdID, mutation.HouseholdID = true, input.HouseholdID
	}
	if input.Author != nil {
		authorID, err := s.identities.ResolveUserID(ctx, *input.Author)
		if err != nil {
			return Mutation{}, apperrors.Normalize(err)
		}
		mutation.SetAuthorID, mutation.AuthorID = true, authorID
	}
	merged := applyMutation(existing, mutation)
	hash, err := Hash(merged.Description, merged.TransactionDate, merged.AuthorID, merged.HouseholdID, merged.CategoryID, merged.Amount)
	if err != nil {
		return Mutation{}, err
	}
	mutation.Hash = hash
	return mutation, nil
}

func applyMutation(item Transaction, update Mutation) Transaction {
	if update.SetAmount {
		item.Amount = update.Amount
	}
	if update.SetTransactionDate {
		item.TransactionDate = update.TransactionDate
	}
	if update.SetDescription {
		item.Description = update.Description
	}
	if update.SetNotes {
		item.Notes = update.Notes
	}
	if update.SetCategoryID {
		item.CategoryID = update.CategoryID
	}
	if update.SetHouseholdID {
		item.HouseholdID = update.HouseholdID
	}
	if update.SetAuthorID {
		item.AuthorID = update.AuthorID
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

func newBulkResult(size int) BulkResult {
	return BulkResult{Succeeded: make([]Succeeded, 0, size), Failed: make([]Failed, 0)}
}

func knownID(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}

func SortBulkResult(result *BulkResult) {
	sort.SliceStable(result.Succeeded, func(i, j int) bool { return result.Succeeded[i].Index < result.Succeeded[j].Index })
	sort.SliceStable(result.Failed, func(i, j int) bool { return result.Failed[i].Index < result.Failed[j].Index })
}
