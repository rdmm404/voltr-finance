package users

import (
	"time"

	"rdmm404/voltr-finance/internal/app/patch"
)

type User struct {
	ID          int64
	Name        string
	DiscordID   *string
	TelegramID  *string
	PhoneNumber *string
	WhatsAppID  *string
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

type Selector struct {
	UserID      *int64
	DiscordID   *string
	TelegramID  *string
	PhoneNumber *string
	WhatsAppID  *string
}

type CreateInput struct {
	Name        string
	DiscordID   *string
	TelegramID  *string
	PhoneNumber *string
	WhatsAppID  *string
}

type UpdateInput struct {
	ID          int64
	Name        *string
	DiscordID   patch.Field[string]
	TelegramID  patch.Field[string]
	PhoneNumber patch.Field[string]
	WhatsAppID  patch.Field[string]
}

type Update struct {
	Name        *string
	DiscordID   patch.Field[string]
	TelegramID  patch.Field[string]
	PhoneNumber patch.Field[string]
	WhatsAppID  patch.Field[string]
}
