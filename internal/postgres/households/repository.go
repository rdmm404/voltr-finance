package households

import (
	"context"
	"time"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
	apphouseholds "rdmm404/voltr-finance/internal/app/households"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/postgres"
)

type queries interface {
	GetHouseholdById(context.Context, int64) (sqlc.Household, error)
	GetHouseholdByGuildId(context.Context, string) (sqlc.Household, error)
	GetHouseholdByName(context.Context, string) (sqlc.Household, error)
	ListHouseholds(context.Context) ([]sqlc.Household, error)
	GetHouseholdUsers(context.Context, int64) ([]sqlc.User, error)
}

type Repository struct{ queries queries }

func NewRepository(queries queries) *Repository { return &Repository{queries: queries} }

func (r *Repository) List(ctx context.Context) ([]apphouseholds.Household, error) {
	rows, err := r.queries.ListHouseholds(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	items := make([]apphouseholds.Household, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapHousehold(row))
	}
	return items, nil
}
func (r *Repository) GetByID(ctx context.Context, id int64) (apphouseholds.Household, error) {
	row, err := r.queries.GetHouseholdById(ctx, id)
	return mapHousehold(row), mapError(err)
}
func (r *Repository) GetByName(ctx context.Context, name string) (apphouseholds.Household, error) {
	row, err := r.queries.GetHouseholdByName(ctx, name)
	return mapHousehold(row), mapError(err)
}
func (r *Repository) GetByGuildID(ctx context.Context, id string) (apphouseholds.Household, error) {
	row, err := r.queries.GetHouseholdByGuildId(ctx, id)
	return mapHousehold(row), mapError(err)
}
func (r *Repository) ListUsers(ctx context.Context, householdID int64) ([]apphouseholds.User, error) {
	rows, err := r.queries.GetHouseholdUsers(ctx, householdID)
	if err != nil {
		return nil, mapError(err)
	}
	items := make([]apphouseholds.User, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapUser(row))
	}
	return items, nil
}

func mapHousehold(row sqlc.Household) apphouseholds.Household {
	return apphouseholds.Household{ID: row.ID, Name: row.Name, GuildID: row.GuildID, CreatedAt: timestamp(row.CreatedAt.Time, row.CreatedAt.Valid), UpdatedAt: timestamp(row.UpdatedAt.Time, row.UpdatedAt.Valid)}
}
func mapUser(row sqlc.User) apphouseholds.User {
	return apphouseholds.User{ID: row.ID, Name: row.Name, DiscordID: row.DiscordID, TelegramID: row.TelegramID, PhoneNumber: row.PhoneNumber, WhatsAppID: row.WhatsappID, CreatedAt: timestamp(row.CreatedAt.Time, row.CreatedAt.Valid), UpdatedAt: timestamp(row.UpdatedAt.Time, row.UpdatedAt.Valid)}
}
func timestamp(value time.Time, valid bool) *time.Time {
	if !valid {
		return nil
	}
	return &value
}
func mapError(err error) error {
	return postgres.MapError(err, postgres.ErrorMapping{NotFoundCode: apperrors.CodeHouseholdNotFound, NotFoundMessage: "household not found", ConflictCode: apperrors.CodeHouseholdConflict, ConflictMessage: "household already exists"})
}

var _ apphouseholds.Repository = (*Repository)(nil)
