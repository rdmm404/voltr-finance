package households

import "time"

type Household struct {
	ID        int64
	Name      string
	GuildID   string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

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
	Name    *string
	GuildID *string
}
