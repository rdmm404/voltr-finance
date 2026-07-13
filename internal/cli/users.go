package cli

import "rdmm404/voltr-finance/internal/api"

type UsersCmd struct {
	Create  UserCreateCmd  `cmd:"" help:"Create a user with optional external identities."`
	Update  UserUpdateCmd  `cmd:"" help:"Update a user and optional external identities."`
	Get     UserGetCmd     `cmd:"" help:"Get a user by internal ID."`
	Resolve UserResolveCmd `cmd:"" help:"Resolve a user from exactly one identity selector."`
	List    UserListCmd    `cmd:"" help:"List all users."`
}

type UserCreateCmd struct {
	Name        string  `required:"" help:"Display name for the user."`
	DiscordID   *string `help:"Discord user ID."`
	TelegramID  *string `help:"Telegram user ID."`
	PhoneNumber *string `help:"Phone number."`
	WhatsappID  *string `help:"WhatsApp ID."`
}

func (c *UserCreateCmd) Run(ctx *runContext) error {
	user, err := ctx.users.CreateUser(ctx.Context, api.CreateUserRequest{Name: c.Name, DiscordID: c.DiscordID, TelegramID: c.TelegramID, PhoneNumber: c.PhoneNumber, WhatsAppID: c.WhatsappID})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, user)
}

type UserUpdateCmd struct {
	ID               int64   `required:"" help:"Internal user ID."`
	Name             *string `help:"Replacement display name."`
	DiscordID        *string `help:"Replacement Discord user ID."`
	TelegramID       *string `help:"Replacement Telegram user ID."`
	PhoneNumber      *string `help:"Replacement phone number."`
	WhatsappID       *string `help:"Replacement WhatsApp ID."`
	ClearDiscordID   bool    `help:"Clear the Discord ID."`
	ClearTelegramID  bool    `help:"Clear the Telegram ID."`
	ClearPhoneNumber bool    `help:"Clear the phone number."`
	ClearWhatsappID  bool    `help:"Clear the WhatsApp ID."`
}

func (c *UserUpdateCmd) Run(ctx *runContext) error {
	user, err := ctx.users.UpdateUser(ctx.Context, c.ID, api.UpdateUserRequest{
		Name: c.Name, DiscordID: c.DiscordID, TelegramID: c.TelegramID, PhoneNumber: c.PhoneNumber, WhatsAppID: c.WhatsappID,
		ClearDiscordID: c.ClearDiscordID, ClearTelegramID: c.ClearTelegramID,
		ClearPhoneNumber: c.ClearPhoneNumber, ClearWhatsAppID: c.ClearWhatsappID,
	})
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, user)
}

type UserGetCmd struct {
	ID int64 `required:"" help:"Internal user ID."`
}

func (c *UserGetCmd) Run(ctx *runContext) error {
	user, err := ctx.users.GetUser(ctx.Context, c.ID)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, user)
}

type UserResolveCmd struct {
	AuthorID    *int64  `placeholder:"INT-64" help:"Internal user ID. Exactly one identity selector is required."`
	DiscordID   *string `help:"Discord user ID. Exactly one identity selector is required."`
	TelegramID  *string `help:"Telegram user ID. Exactly one identity selector is required."`
	PhoneNumber *string `help:"Phone number. Exactly one identity selector is required."`
	WhatsappID  *string `help:"WhatsApp ID. Exactly one identity selector is required."`
}

func (c *UserResolveCmd) Run(ctx *runContext) error {
	user, err := ctx.users.ResolveUser(ctx.Context, identity(c.AuthorID, c.DiscordID, c.TelegramID, c.PhoneNumber, c.WhatsappID))
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, user)
}

type UserListCmd struct{}

func (c *UserListCmd) Run(ctx *runContext) error {
	users, err := ctx.users.ListUsers(ctx.Context)
	if err != nil {
		return err
	}
	return RenderJSON(ctx.stdout, users)
}
