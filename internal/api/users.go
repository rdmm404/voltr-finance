package api

import "time"

type User struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	DiscordID   *string    `json:"discordId,omitempty"`
	TelegramID  *string    `json:"telegramId,omitempty"`
	PhoneNumber *string    `json:"phoneNumber,omitempty"`
	WhatsAppID  *string    `json:"whatsappId,omitempty"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
}

type CreateUserRequest struct {
	Name        string  `json:"name"`
	DiscordID   *string `json:"discordId,omitempty"`
	TelegramID  *string `json:"telegramId,omitempty"`
	PhoneNumber *string `json:"phoneNumber,omitempty"`
	WhatsAppID  *string `json:"whatsappId,omitempty"`
}

type UpdateUserRequest struct {
	Name        *string `json:"name,omitempty"`
	DiscordID   *string `json:"discordId,omitempty"`
	TelegramID  *string `json:"telegramId,omitempty"`
	PhoneNumber *string `json:"phoneNumber,omitempty"`
	WhatsAppID  *string `json:"whatsappId,omitempty"`

	ClearDiscordID   bool `json:"clearDiscordId,omitempty"`
	ClearTelegramID  bool `json:"clearTelegramId,omitempty"`
	ClearPhoneNumber bool `json:"clearPhoneNumber,omitempty"`
	ClearWhatsAppID  bool `json:"clearWhatsappId,omitempty"`
}

type ResolveUserRequest struct {
	IdentitySelector
}
