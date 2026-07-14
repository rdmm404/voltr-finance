package households

import "context"

type Repository interface {
	List(context.Context) ([]Household, error)
	GetByID(context.Context, int64) (Household, error)
	GetByName(context.Context, string) (Household, error)
	GetByGuildID(context.Context, string) (Household, error)
	ListUsers(context.Context, int64) ([]User, error)
}
