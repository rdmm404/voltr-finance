package users

import (
	"context"
	"strings"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, input CreateInput) (User, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return User{}, apperrors.Validation("user name is required")
	}
	user, err := s.repo.Create(ctx, input)
	return user, apperrors.WrapInternal("create user", err)
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (User, error) {
	if input.ID == 0 {
		return User{}, apperrors.Validation("user id is required")
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return User{}, apperrors.Validation("user name cannot be empty")
		}
		input.Name = &name
	}
	user, err := s.repo.Update(ctx, input.ID, Update{
		Name: input.Name, DiscordID: input.DiscordID, TelegramID: input.TelegramID,
		PhoneNumber: input.PhoneNumber, WhatsAppID: input.WhatsAppID,
	})
	return user, apperrors.WrapInternal("update user", err)
}

func (s *Service) Get(ctx context.Context, id int64) (User, error) {
	if id == 0 {
		return User{}, apperrors.Validation("user id is required")
	}
	user, err := s.repo.GetByID(ctx, id)
	return user, apperrors.WrapInternal("get user", err)
}

func (s *Service) Resolve(ctx context.Context, selector Selector) (User, error) {
	selector = selector.normalized()
	if err := selector.validate(); err != nil {
		return User{}, err
	}
	var user User
	var err error
	switch {
	case selector.UserID != nil:
		user, err = s.repo.GetByID(ctx, *selector.UserID)
	case selector.DiscordID != nil:
		user, err = s.repo.GetByDiscordID(ctx, *selector.DiscordID)
	case selector.TelegramID != nil:
		user, err = s.repo.GetByTelegramID(ctx, *selector.TelegramID)
	case selector.PhoneNumber != nil:
		user, err = s.repo.GetByPhoneNumber(ctx, *selector.PhoneNumber)
	case selector.WhatsAppID != nil:
		user, err = s.repo.GetByWhatsAppID(ctx, *selector.WhatsAppID)
	}
	return user, apperrors.WrapInternal("resolve user", err)
}

func (s *Service) List(ctx context.Context) ([]User, error) {
	items, err := s.repo.List(ctx)
	if items == nil && err == nil {
		items = []User{}
	}
	return items, apperrors.WrapInternal("list users", err)
}

func (s Selector) normalized() Selector {
	if s.TelegramID != nil {
		value, _, _ := strings.Cut(*s.TelegramID, "|")
		s.TelegramID = &value
	}
	return s
}

func (s Selector) validate() error {
	count := 0
	for _, set := range []bool{s.UserID != nil, s.DiscordID != nil, s.TelegramID != nil, s.PhoneNumber != nil, s.WhatsAppID != nil} {
		if set {
			count++
		}
	}
	if count != 1 {
		return apperrors.Validation("exactly one identity selector is required")
	}
	return nil
}
