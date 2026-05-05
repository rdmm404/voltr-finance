package app

import (
	"context"
	"errors"
	"testing"

	"rdmm404/voltr-finance/internal/database/sqlc"
)

func TestCreateUserRejectsEmptyName(t *testing.T) {
	svc := NewService(&fakeRepo{}, &fakeTransactionService{})

	_, err := svc.CreateUser(context.Background(), CreateUserRequest{Name: "  "})
	if err == nil {
		t.Fatal("CreateUser returned nil error")
	}

	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != CodeValidationError {
		t.Fatalf("error = %v, want validation_error", err)
	}
}

func TestUpdateUserCanClearPhoneNumber(t *testing.T) {
	repo := &fakeRepo{
		updateUser: sqlc.User{ID: 10, Name: "Rafael"},
	}
	svc := NewService(repo, &fakeTransactionService{})

	_, err := svc.UpdateUser(context.Background(), UpdateUserRequest{
		ID:               10,
		ClearPhoneNumber: true,
	})
	if err != nil {
		t.Fatalf("UpdateUser returned error: %v", err)
	}
	if !repo.lastUpdateUser.SetPhoneNumber {
		t.Fatal("SetPhoneNumber = false, want true")
	}
	if repo.lastUpdateUser.PhoneNumber != nil {
		t.Fatalf("PhoneNumber = %v, want nil", repo.lastUpdateUser.PhoneNumber)
	}
}

func TestResolveUserByTelegramID(t *testing.T) {
	telegramID := "123456"
	repo := &fakeRepo{
		userByTelegram: sqlc.User{ID: 11, Name: "Rafael", TelegramID: &telegramID},
	}
	svc := NewService(repo, &fakeTransactionService{})

	user, err := svc.ResolveUser(context.Background(), IdentitySelector{TelegramID: strPtr("123456|rafael")})
	if err != nil {
		t.Fatalf("ResolveUser returned error: %v", err)
	}
	if user.ID != 11 {
		t.Fatalf("ID = %d, want 11", user.ID)
	}
	if repo.lastTelegramID == nil || *repo.lastTelegramID != "123456" {
		t.Fatalf("telegram lookup = %v, want normalized 123456", repo.lastTelegramID)
	}
}
