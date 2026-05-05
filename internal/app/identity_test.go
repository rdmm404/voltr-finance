package app

import (
	"errors"
	"testing"
)

func TestIdentitySelectorRequiresExactlyOneField(t *testing.T) {
	tests := []struct {
		name     string
		selector IdentitySelector
	}{
		{name: "zero fields", selector: IdentitySelector{}},
		{
			name: "two fields",
			selector: IdentitySelector{
				DiscordID:  strPtr("discord"),
				TelegramID: strPtr("telegram"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.selector.ValidateExactlyOne()
			if err == nil {
				t.Fatal("ValidateExactlyOne returned nil error")
			}

			var appErr *AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("error = %T, want *AppError", err)
			}
			if appErr.Code != CodeValidationError {
				t.Fatalf("code = %q, want %q", appErr.Code, CodeValidationError)
			}
		})
	}
}

func TestIdentitySelectorAcceptsAuthorID(t *testing.T) {
	authorID := int64(42)
	selector := IdentitySelector{AuthorID: &authorID}

	if err := selector.ValidateExactlyOne(); err != nil {
		t.Fatalf("ValidateExactlyOne returned error: %v", err)
	}
}

func TestIdentitySelectorNormalizesTelegramSenderID(t *testing.T) {
	selector := IdentitySelector{TelegramID: strPtr("123456|rafael")}

	normalized := selector.Normalized()

	if normalized.TelegramID == nil {
		t.Fatal("TelegramID is nil")
	}
	if *normalized.TelegramID != "123456" {
		t.Fatalf("TelegramID = %q, want %q", *normalized.TelegramID, "123456")
	}
}

func strPtr(value string) *string {
	return &value
}
