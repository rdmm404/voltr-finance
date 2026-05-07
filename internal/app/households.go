package app

import (
	"context"
	"time"

	"rdmm404/voltr-finance/internal/database/sqlc"
)

type HouseholdDTO struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	GuildID   string     `json:"guildId"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type GetHouseholdRequest struct {
	ID      *int64  `json:"id,omitempty"`
	GuildID *string `json:"guildId,omitempty"`
	Name    *string `json:"name,omitempty"`
}

func (s *Service) GetHousehold(ctx context.Context, req GetHouseholdRequest) (HouseholdDTO, error) {
	count := 0
	if req.ID != nil {
		count++
	}
	if req.GuildID != nil {
		count++
	}
	if req.Name != nil {
		count++
	}
	if count != 1 {
		return HouseholdDTO{}, NewError(CodeValidationError, "exactly one household selector is required", nil)
	}

	var (
		household sqlc.Household
		err       error
	)
	switch {
	case req.ID != nil:
		household, err = s.repo.GetHouseholdById(ctx, *req.ID)
	case req.GuildID != nil:
		household, err = s.repo.GetHouseholdByGuildId(ctx, *req.GuildID)
	case req.Name != nil:
		household, err = s.repo.GetHouseholdByName(ctx, *req.Name)
	}
	if err != nil {
		return HouseholdDTO{}, NewError(CodeDatabaseError, "household lookup failed", err)
	}
	return householdDTO(household), nil
}

func (s *Service) ListHouseholds(ctx context.Context) ([]HouseholdDTO, error) {
	households, err := s.repo.ListHouseholds(ctx)
	if err != nil {
		return nil, NewError(CodeDatabaseError, "household list failed", err)
	}
	dtos := make([]HouseholdDTO, 0, len(households))
	for _, household := range households {
		dtos = append(dtos, householdDTO(household))
	}
	return dtos, nil
}

func (s *Service) GetHouseholdUsers(ctx context.Context, householdID int64) ([]UserDTO, error) {
	if householdID == 0 {
		return nil, NewError(CodeValidationError, "household id is required", nil)
	}
	users, err := s.repo.GetHouseholdUsers(ctx, householdID)
	if err != nil {
		return nil, mapUserError(err)
	}
	return userDTOs(users), nil
}

func householdDTO(household sqlc.Household) HouseholdDTO {
	return HouseholdDTO{
		ID:        household.ID,
		Name:      household.Name,
		GuildID:   household.GuildID,
		CreatedAt: validTime(household.CreatedAt.Time, household.CreatedAt.Valid),
		UpdatedAt: validTime(household.UpdatedAt.Time, household.UpdatedAt.Valid),
	}
}
