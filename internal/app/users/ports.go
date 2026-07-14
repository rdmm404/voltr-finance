package users

import "context"

type Repository interface {
	Create(context.Context, CreateInput) (User, error)
	Update(context.Context, int64, Update) (User, error)
	GetByID(context.Context, int64) (User, error)
	GetByDiscordID(context.Context, string) (User, error)
	GetByTelegramID(context.Context, string) (User, error)
	GetByPhoneNumber(context.Context, string) (User, error)
	GetByWhatsAppID(context.Context, string) (User, error)
	List(context.Context) ([]User, error)
}
