package app

import "strings"

type IdentitySelector struct {
	AuthorID    *int64
	DiscordID   *string
	TelegramID  *string
	PhoneNumber *string
	WhatsappID  *string
}

func (s IdentitySelector) ValidateExactlyOne() error {
	count := 0
	if s.AuthorID != nil {
		count++
	}
	if s.DiscordID != nil {
		count++
	}
	if s.TelegramID != nil {
		count++
	}
	if s.PhoneNumber != nil {
		count++
	}
	if s.WhatsappID != nil {
		count++
	}

	if count != 1 {
		return NewError(CodeValidationError, "exactly one identity selector is required", nil)
	}
	return nil
}

func (s IdentitySelector) Normalized() IdentitySelector {
	normalized := s
	if normalized.TelegramID != nil {
		value := *normalized.TelegramID
		// Nanobot can fall back to sender IDs like 123456789|rafael; the database stores the stable numeric Telegram user ID.
		if id, _, ok := strings.Cut(value, "|"); ok {
			normalized.TelegramID = &id
		}
	}
	return normalized
}
