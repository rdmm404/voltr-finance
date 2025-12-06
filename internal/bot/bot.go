package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"rdmm404/voltr-finance/internal/ai/agent"
	"rdmm404/voltr-finance/internal/config"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/utils"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var ErrInvalidBotConfig = errors.New("bot is not set up correctly")

type Bot struct {
	session    *discordgo.Session
	agent      agent.ChatAgent
	repository *sqlc.Queries
}

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "session",
		Description: "Create a brand new session for chatting with Voltio.",
		DescriptionLocalizations: &map[discordgo.Locale]string{
			discordgo.SpanishES:    "Crear una nueva sesión para hablar con Voltio.",
			discordgo.SpanishLATAM: "Crear una nueva sesión para hablar con Voltio.",
		},
		Contexts: &[]discordgo.InteractionContextType{
			discordgo.InteractionContextBotDM, discordgo.InteractionContextGuild,
		},
		Type: discordgo.ChatApplicationCommand,
	},
}

func NewBot(a agent.ChatAgent, repository *sqlc.Queries) (*Bot, error) {
	if err := validateDiscordConfig(); err != nil {
		return nil, err
	}

	dg, err := discordgo.New("Bot " + config.DISCORD_TOKEN)
	if err != nil {
		return nil, fmt.Errorf("error creating discord session - %w", err)
	}

	if config.DISCORD_CREATE_COMMANDS {
		createdCommands, err := dg.ApplicationCommandBulkOverwrite(config.DISCORD_APP_ID, "", commands)
		if err != nil {
			return nil, fmt.Errorf("error creating application commands %w", err)
		}

		jsonCommands, _ := json.Marshal(createdCommands)
		slog.Info("created commands", "commands", string(jsonCommands))
	}

	bot := &Bot{
		session:    dg,
		agent:      a,
		repository: repository,
	}

	dg.AddHandler(eventHandler(bot.handlerMessageCreate))
	dg.AddHandler(eventHandler(bot.handlerInteractionCreate))

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages

	return bot, nil
}

func validateDiscordConfig() error {
	if config.DISCORD_TOKEN == "" {
		return fmt.Errorf("%w - DISCORD_TOKEN environment variable is not set", ErrInvalidBotConfig)
	}

	if config.DISCORD_APP_ID == "" {
		return fmt.Errorf("%w - DISCORD_APP_ID environment variable is not set", ErrInvalidBotConfig)
	}
	return nil
}

func (b *Bot) Run() error {
	err := b.session.Open()
	if err != nil {
		return fmt.Errorf("error opening ws connection - %w", err)
	}

	// TODO handle interrupts in main instead of here
	// receive ctx and wait for cancel (or done)
	slog.Info("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	return b.session.Close()
}

func (b *Bot) handlerMessageCreate(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate) error {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return nil
	}

	slog.Debug("message received", "message", utils.JsonMarshalIgnore(m))

	s.ChannelTyping(m.ChannelID)

	senderInfo, err := b.getSenderInfoFromMessage(ctx, m)

	if err != nil {
		return fmt.Errorf("error while getting sender info - %w", err)
	}

	aiMsg := &agent.Message{Msg: m.Content, SenderInfo: senderInfo}

	for _, att := range m.Attachments {
		aiMsg.Attachments = append(aiMsg.Attachments, &agent.Attachment{URI: att.URL, Mimetype: att.ContentType})
	}

	ch, err := b.agent.Run(ctx, aiMsg, agent.StreamingModeMessages)

	if err != nil {
		return fmt.Errorf("error received from agent - %w", err)
	}

	for update := range ch {
		s.ChannelTyping(m.ChannelID)

		updateJson, err := json.Marshal(update)
		slog.Debug("update received", "update", string(updateJson))

		if update.Err != nil {
			slog.Error("update error", "error", update.Err)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Something went wrong :( - %v", update.Err))
			continue
			// TODO: Send debug trace as spoiler or something that makes it hidden
		}

		if update.Text == "" && update.ToolCall == nil && update.ToolResponse == nil {
			s.ChannelMessageSend(m.ChannelID, "Something went wrong :( - Empty update received")
			continue
		}

		// TODO: include tool call in msg
		if update.Text != "" {
			err = b.sendMessageInChunks(update.Text, nil, s, m)
		}

		if err != nil {
			slog.Error("error sending message", "error", err)
		}
	}

	return nil
}

func (b *Bot) getSenderInfoFromMessage(ctx context.Context, m *discordgo.MessageCreate) (*agent.MessageSenderInfo, error) {
	if m == nil || m.Author == nil {
		return nil, fmt.Errorf("message received does not have an author")
	}

	senderInfo := &agent.MessageSenderInfo{ChannelID: m.ChannelID}

	var user sqlc.User
	var err error

	if m.GuildID != "" {
		var household sqlc.Household
		household, err = b.repository.GetHouseholdByGuildId(ctx, m.GuildID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("household with guild id %q not found", m.GuildID)
			}
			return nil, fmt.Errorf("error while getting household by guild id %q: %w", m.GuildID, err)
		}

		senderInfo.Household = &agent.MessageHousehold{ID: household.ID, Name: household.Name, GuildID: household.GuildID}

		user, err = b.repository.GetUserByDiscordAndHouseholdId(ctx,
			sqlc.GetUserByDiscordAndHouseholdIdParams{DiscordID: m.Author.ID, HouseholdID: household.ID},
		)
	} else {
		user, err = b.repository.GetUserByDiscordId(ctx, m.Author.ID)
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user with discord id %q not found", user.DiscordID)
		}
		return nil, fmt.Errorf("error while getting user by discord id %q: %w", m.Author.ID, err)
	}

	senderInfo.User = agent.MessageUser{ID: user.ID, Name: user.Name, DiscordID: user.DiscordID}

	return senderInfo, nil

}

func (b *Bot) sendMessageInChunks(msg string, chunkSizePtr *int, s *discordgo.Session, m *discordgo.MessageCreate) error {
	remainder := []rune(msg)
	chunkSize := config.DISCORD_MAX_MESSAGE_LENGTH
	if chunkSizePtr != nil {
		chunkSize = *chunkSizePtr
	}
	for len(remainder) > 0 {
		var currMessage []rune
		if len(remainder) > int(chunkSize) {
			currMessage = remainder[:chunkSize]
		} else {
			currMessage = remainder
		}

		remainder = remainder[len(currMessage):]

		_, err := s.ChannelMessageSend(m.ChannelID, string(currMessage))

		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Bot) handlerInteractionCreate(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	jsonEvent, _ := json.MarshalIndent(i, "", "  ")
	slog.Debug("Interaction event received", "event", string(jsonEvent))

	if i.Type != discordgo.InteractionApplicationCommand {
		return nil
	}

	data := i.ApplicationCommandData()
	// TODO turn this into a map or something if adding more commands
	switch data.Name {
	case "session":
		var discordUser *discordgo.User
		if i.Member != nil {
			discordUser = i.Member.User
		} else {
			discordUser = i.User
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Creating session...",
			},
		})

		user, err := b.repository.GetUserByDiscordId(ctx, discordUser.ID)

		if err != nil {
			s.InteractionResponseEdit(
				i.Interaction,
				&discordgo.WebhookEdit{Content: utils.StringPtr("Something went wrong while creating session :(")},
			)
			return fmt.Errorf("error getting user details: %w", err)
		}

		params := sqlc.CreateLlmSessionParams{UserID: user.ID, SourceID: i.ChannelID}

		if _, err = b.repository.CreateLlmSession(ctx, params); err != nil {
			s.InteractionResponseEdit(
				i.Interaction,
				&discordgo.WebhookEdit{Content: utils.StringPtr("Something went wrong while creating session :(")},
			)
			return fmt.Errorf("error creating new session: %w", err)
		}

		s.InteractionResponseEdit(
			i.Interaction,
			&discordgo.WebhookEdit{Content: utils.StringPtr("Session created successfully!")},
		)

	default:
		slog.Warn("Unrecognized command received", "command", data.Name)
	}

	return nil
}
