package categories

import "context"

// Repository implementations translate missing rows and unique violations to
// application not-found and conflict errors respectively.
type Repository interface {
	Create(context.Context, CreateInput) (Category, error)
	List(context.Context, bool) ([]Category, error)
	GetByCode(context.Context, string) (Category, error)
	GetActiveByID(context.Context, int64) (Category, error)
	GetActiveByCode(context.Context, string) (Category, error)
	Update(context.Context, string, Update) (Category, error)
	Deactivate(context.Context, string) (Category, error)
}
