package app

import (
	"context"
	"strings"
	"time"

	"rdmm404/voltr-finance/internal/database/sqlc"
)

type UserDTO struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	DiscordID   *string    `json:"discordId,omitempty"`
	TelegramID  *string    `json:"telegramId,omitempty"`
	PhoneNumber *string    `json:"phoneNumber,omitempty"`
	WhatsappID  *string    `json:"whatsappId,omitempty"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
}

type CreateUserRequest struct {
	Name        string  `json:"name"`
	DiscordID   *string `json:"discordId,omitempty"`
	TelegramID  *string `json:"telegramId,omitempty"`
	PhoneNumber *string `json:"phoneNumber,omitempty"`
	WhatsappID  *string `json:"whatsappId,omitempty"`
}

type UpdateUserRequest struct {
	ID int64 `json:"id"`

	Name        *string `json:"name,omitempty"`
	DiscordID   *string `json:"discordId,omitempty"`
	TelegramID  *string `json:"telegramId,omitempty"`
	PhoneNumber *string `json:"phoneNumber,omitempty"`
	WhatsappID  *string `json:"whatsappId,omitempty"`

	ClearDiscordID   bool `json:"clearDiscordId,omitempty"`
	ClearTelegramID  bool `json:"clearTelegramId,omitempty"`
	ClearPhoneNumber bool `json:"clearPhoneNumber,omitempty"`
	ClearWhatsappID  bool `json:"clearWhatsappId,omitempty"`
}

func (s *Service) CreateUser(ctx context.Context, req CreateUserRequest) (UserDTO, error) {
	if strings.TrimSpace(req.Name) == "" {
		return UserDTO{}, NewError(CodeValidationError, "user name is required", nil)
	}

	user, err := s.repo.CreateUser(ctx, sqlc.CreateUserParams{
		DiscordID:   req.DiscordID,
		TelegramID:  req.TelegramID,
		PhoneNumber: req.PhoneNumber,
		WhatsappID:  req.WhatsappID,
		Name:        req.Name,
	})
	if err != nil {
		return UserDTO{}, mapUserError(err)
	}
	return userDTO(user), nil
}

func (s *Service) UpdateUser(ctx context.Context, req UpdateUserRequest) (UserDTO, error) {
	if req.ID == 0 {
		return UserDTO{}, NewError(CodeValidationError, "user id is required", nil)
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return UserDTO{}, NewError(CodeValidationError, "user name cannot be empty", nil)
	}

	params := sqlc.UpdateUserParams{ID: req.ID}
	if req.Name != nil {
		params.SetName = true
		params.Name = *req.Name
	}
	params.SetDiscordID = req.DiscordID != nil || req.ClearDiscordID
	params.DiscordID = req.DiscordID
	params.SetTelegramID = req.TelegramID != nil || req.ClearTelegramID
	params.TelegramID = req.TelegramID
	params.SetPhoneNumber = req.PhoneNumber != nil || req.ClearPhoneNumber
	params.PhoneNumber = req.PhoneNumber
	params.SetWhatsappID = req.WhatsappID != nil || req.ClearWhatsappID
	params.WhatsappID = req.WhatsappID

	user, err := s.repo.UpdateUser(ctx, params)
	if err != nil {
		return UserDTO{}, mapUserError(err)
	}
	return userDTO(user), nil
}

func (s *Service) GetUser(ctx context.Context, id int64) (UserDTO, error) {
	if id == 0 {
		return UserDTO{}, NewError(CodeValidationError, "user id is required", nil)
	}
	user, err := s.repo.GetUserById(ctx, id)
	if err != nil {
		return UserDTO{}, mapUserError(err)
	}
	return userDTO(user), nil
}

func (s *Service) ResolveUser(ctx context.Context, selector IdentitySelector) (UserDTO, error) {
	selector = selector.Normalized()
	if err := selector.ValidateExactlyOne(); err != nil {
		return UserDTO{}, err
	}

	var (
		user sqlc.User
		err  error
	)
	switch {
	case selector.AuthorID != nil:
		user, err = s.repo.GetUserById(ctx, *selector.AuthorID)
	case selector.DiscordID != nil:
		user, err = s.repo.GetUserByDiscordId(ctx, selector.DiscordID)
	case selector.TelegramID != nil:
		user, err = s.repo.GetUserByTelegramId(ctx, selector.TelegramID)
	case selector.PhoneNumber != nil:
		user, err = s.repo.GetUserByPhoneNumber(ctx, selector.PhoneNumber)
	case selector.WhatsappID != nil:
		user, err = s.repo.GetUserByWhatsappId(ctx, selector.WhatsappID)
	}
	if err != nil {
		return UserDTO{}, mapUserError(err)
	}
	return userDTO(user), nil
}

func (s *Service) ListUsers(ctx context.Context) ([]UserDTO, error) {
	users, err := s.repo.ListUsers(ctx)
	if err != nil {
		return nil, mapUserError(err)
	}
	return userDTOs(users), nil
}

func userDTOs(users []sqlc.User) []UserDTO {
	dtos := make([]UserDTO, 0, len(users))
	for _, user := range users {
		dtos = append(dtos, userDTO(user))
	}
	return dtos
}

func userDTO(user sqlc.User) UserDTO {
	return UserDTO{
		ID:          user.ID,
		Name:        user.Name,
		DiscordID:   user.DiscordID,
		TelegramID:  user.TelegramID,
		PhoneNumber: user.PhoneNumber,
		WhatsappID:  user.WhatsappID,
		CreatedAt:   validTime(user.CreatedAt.Time, user.CreatedAt.Valid),
		UpdatedAt:   validTime(user.UpdatedAt.Time, user.UpdatedAt.Valid),
	}
}

func validTime(value time.Time, valid bool) *time.Time {
	if !valid {
		return nil
	}
	return &value
}
