package households

import (
	"context"
	"strings"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context) ([]Household, error) {
	items, err := s.repo.List(ctx)
	if items == nil && err == nil {
		items = []Household{}
	}
	return items, apperrors.WrapInternal("list households", err)
}

func (s *Service) Get(ctx context.Context, id int64) (Household, error) {
	if id == 0 {
		return Household{}, apperrors.Validation("household id is required")
	}
	item, err := s.repo.GetByID(ctx, id)
	return item, apperrors.WrapInternal("get household", err)
}

func (s *Service) Resolve(ctx context.Context, selector Selector) (Household, error) {
	count := 0
	if selector.Name != nil {
		count++
	}
	if selector.GuildID != nil {
		count++
	}
	if count != 1 {
		return Household{}, apperrors.Validation("exactly one household selector is required")
	}
	var item Household
	var err error
	if selector.Name != nil {
		name := strings.TrimSpace(*selector.Name)
		if name == "" {
			return Household{}, apperrors.Validation("household name is required")
		}
		item, err = s.repo.GetByName(ctx, name)
	} else {
		guildID := strings.TrimSpace(*selector.GuildID)
		if guildID == "" {
			return Household{}, apperrors.Validation("guild id is required")
		}
		item, err = s.repo.GetByGuildID(ctx, guildID)
	}
	return item, apperrors.WrapInternal("resolve household", err)
}

func (s *Service) ListUsers(ctx context.Context, householdID int64) ([]User, error) {
	if householdID == 0 {
		return nil, apperrors.Validation("household id is required")
	}
	items, err := s.repo.ListUsers(ctx, householdID)
	if items == nil && err == nil {
		items = []User{}
	}
	return items, apperrors.WrapInternal("list household users", err)
}
