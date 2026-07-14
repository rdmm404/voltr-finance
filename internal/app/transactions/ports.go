package transactions

import "context"

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
