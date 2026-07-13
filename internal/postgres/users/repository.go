package users

import (
	"context"
	"time"

	apperrors "rdmm404/voltr-finance/internal/app/errors"
	appusers "rdmm404/voltr-finance/internal/app/users"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/postgres"
)

type queries interface {
	CreateUser(context.Context, sqlc.CreateUserParams) (sqlc.User, error)
	UpdateUser(context.Context, sqlc.UpdateUserParams) (sqlc.User, error)
	GetUserById(context.Context, int64) (sqlc.User, error)
	GetUserByDiscordId(context.Context, *string) (sqlc.User, error)
	GetUserByTelegramId(context.Context, *string) (sqlc.User, error)
	GetUserByPhoneNumber(context.Context, *string) (sqlc.User, error)
	GetUserByWhatsappId(context.Context, *string) (sqlc.User, error)
	ListUsers(context.Context) ([]sqlc.User, error)
}

type Repository struct{ queries queries }

func NewRepository(queries queries) *Repository { return &Repository{queries: queries} }

func (r *Repository) Create(ctx context.Context, input appusers.CreateInput) (appusers.User, error) {
	row, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{DiscordID: input.DiscordID, TelegramID: input.TelegramID, PhoneNumber: input.PhoneNumber, WhatsappID: input.WhatsAppID, Name: input.Name})
	return mapUser(row), mapError(err)
}

func (r *Repository) Update(ctx context.Context, id int64, input appusers.Update) (appusers.User, error) {
	name := ""
	if input.Name != nil {
		name = *input.Name
	}
	row, err := r.queries.UpdateUser(ctx, sqlc.UpdateUserParams{
		SetDiscordID: input.SetDiscordID, DiscordID: input.DiscordID,
		SetTelegramID: input.SetTelegramID, TelegramID: input.TelegramID,
		SetPhoneNumber: input.SetPhoneNumber, PhoneNumber: input.PhoneNumber,
		SetWhatsappID: input.SetWhatsAppID, WhatsappID: input.WhatsAppID,
		SetName: input.Name != nil, Name: name, ID: id,
	})
	return mapUser(row), mapError(err)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (appusers.User, error) {
	row, err := r.queries.GetUserById(ctx, id)
	return mapUser(row), mapError(err)
}
func (r *Repository) GetByDiscordID(ctx context.Context, id string) (appusers.User, error) {
	row, err := r.queries.GetUserByDiscordId(ctx, &id)
	return mapUser(row), mapError(err)
}
func (r *Repository) GetByTelegramID(ctx context.Context, id string) (appusers.User, error) {
	row, err := r.queries.GetUserByTelegramId(ctx, &id)
	return mapUser(row), mapError(err)
}
func (r *Repository) GetByPhoneNumber(ctx context.Context, id string) (appusers.User, error) {
	row, err := r.queries.GetUserByPhoneNumber(ctx, &id)
	return mapUser(row), mapError(err)
}
func (r *Repository) GetByWhatsAppID(ctx context.Context, id string) (appusers.User, error) {
	row, err := r.queries.GetUserByWhatsappId(ctx, &id)
	return mapUser(row), mapError(err)
}
func (r *Repository) List(ctx context.Context) ([]appusers.User, error) {
	rows, err := r.queries.ListUsers(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	items := make([]appusers.User, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapUser(row))
	}
	return items, nil
}

func mapUser(row sqlc.User) appusers.User {
	return appusers.User{ID: row.ID, Name: row.Name, DiscordID: row.DiscordID, TelegramID: row.TelegramID, PhoneNumber: row.PhoneNumber, WhatsAppID: row.WhatsappID, CreatedAt: timestamp(row.CreatedAt.Time, row.CreatedAt.Valid), UpdatedAt: timestamp(row.UpdatedAt.Time, row.UpdatedAt.Valid)}
}
func timestamp(value time.Time, valid bool) *time.Time {
	if !valid {
		return nil
	}
	return &value
}
func mapError(err error) error {
	return postgres.MapError(err, postgres.ErrorMapping{NotFoundCode: apperrors.CodeUserNotFound, NotFoundMessage: "user not found", ConflictCode: apperrors.CodeUserConflict, ConflictMessage: "user identity already exists"})
}

var _ appusers.Repository = (*Repository)(nil)
