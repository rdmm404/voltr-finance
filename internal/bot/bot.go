package bot

import (
	"context"
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
	// createdCommands, err := dg.ApplicationCommandBulkOverwrite(config.DISCORD_APP_ID, "", commands)
	// if err != nil {
	// 	return nil, fmt.Errorf("error creating application commands %w", err)
	// }

	// jsonCommands, _ := json.Marshal(createdCommands)
	// slog.Info("created commands", "commands", string(jsonCommands))

	bot := &Bot{
		session:    dg,
		agent:      a,
		repository: repository,
	}

	dg.AddHandler(bot.handlerMessageCreate)
	dg.AddHandler(bot.handlerInteractionCreate)

	dg.Identify.Intents = discordgo.IntentsGuildMessages

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
	slog.Info("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	return b.session.Close()
}

func (b *Bot) handlerMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	ctx := context.Background()
	msgJson, _ := json.MarshalIndent(m, "", "  ")
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	slog.Debug("message received", "message", string(msgJson))

	s.ChannelTyping(m.ChannelID)

	senderInfo, err := b.getSenderInfoFromMessage(ctx, m)

	if err != nil {
		slog.Error("Bot: Error while getting sender info", "error", err)
		return
	}

	aiMsg := &agent.Message{Msg: m.Content, SenderInfo: senderInfo}

	for _, att := range m.Attachments {
		aiMsg.Attachments = append(aiMsg.Attachments, &agent.Attachment{URI: att.URL, Mimetype: att.ContentType})
	}

	ch, err := b.agent.Run(ctx, aiMsg, agent.StreamingModeMessages)

	if err != nil {
		slog.Error("Bot: Error received from agent", "error", err)
		return
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
}

func (b *Bot) getSenderInfoFromMessage(ctx context.Context, m *discordgo.MessageCreate) (*agent.MessageSenderInfo, error) {
	if m == nil || m.Author == nil {
		return nil, fmt.Errorf("message received does not have an author")
	}

	result, err := b.repository.GetUserDetailsByDiscordId(ctx, m.Author.ID)

	if err != nil {
		return nil, err
	}

	return &agent.MessageSenderInfo{
		User: &agent.MessageUser{
			ID:        result.User.ID,
			Name:      result.User.Name,
			DiscordID: result.User.DiscordID,
		},
		Household: &agent.MessageHousehold{
			ID:   result.Household.ID,
			Name: result.Household.Name,
		},
		ChannelID: m.ChannelID,
	}, nil
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

func (b *Bot) handlerInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	jsonEvent, _ := json.MarshalIndent(i, "", "  ")
	slog.Debug("Interaction event received", "event", string(jsonEvent))

	if i.Type != discordgo.InteractionApplicationCommand {
		return
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
			slog.Error("Bot: error getting user details", "error", err)
			s.InteractionResponseEdit(
				i.Interaction,
				&discordgo.WebhookEdit{Content: utils.StringPtr("Something went wrong while creating session :(")},
			)
			return
		}

		params := sqlc.CreateLlmSessionParams{UserID: user.ID, SourceID: i.ChannelID}

		if _, err = b.repository.CreateLlmSession(ctx, params); err != nil {
			slog.Error("Bot: error creating new session", "error", err)
			s.InteractionResponseEdit(
				i.Interaction,
				&discordgo.WebhookEdit{Content: utils.StringPtr("Something went wrong while creating session :(")},
			)
			return
		}

		s.InteractionResponseEdit(
			i.Interaction,
			&discordgo.WebhookEdit{Content: utils.StringPtr("Session created successfully!")},
		)

	default:
		slog.Debug("Unrecognized command received", "command", data.Name)
	}
}
