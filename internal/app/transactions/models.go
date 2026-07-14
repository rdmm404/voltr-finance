package transactions

import (
	"time"

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
