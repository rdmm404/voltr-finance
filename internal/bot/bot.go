package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"rdmm404/voltr-finance/internal/ai/agent"
	"rdmm404/voltr-finance/internal/config"
	database "rdmm404/voltr-finance/internal/database/repository"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var ErrInvalidBotConfig = errors.New("bot is not set up correctly")

type Bot struct {
	session *discordgo.Session
	agent agent.ChatAgent
	repository *database.Queries
}

func NewBot(a agent.ChatAgent, repository *database.Queries) (*Bot, error) {
	token := config.DISCORD_TOKEN
	if token == "" {
		return nil, fmt.Errorf("%w - DISCORD_TOKEN environment variable is not set", ErrInvalidBotConfig)
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("error creating discord session - %w", err)
	}

	bot := &Bot{
		session: dg,
		agent: a,
		repository: repository,
	}

	dg.AddHandler(bot.handlerMessageCreate)

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	return bot, nil
}

func (b *Bot) Run() error {
	err := b.session.Open()
	if err != nil {
		return fmt.Errorf("error opening ws connection - %w", err)
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	return b.session.Close()
}

func (b *Bot) handlerMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	ctx := context.Background()
	msgJson, _ := json.MarshalIndent(m, "", "  ")
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	fmt.Printf("message received %v\n", string(msgJson))

	s.ChannelTyping(m.ChannelID)

	senderInfo, err := b.getSenderIntoFromMessage(ctx, m)

	if err != nil {
		fmt.Printf("Bot: Error while getting sender info %v", err)
		return
	}

	aiMsg := &agent.Message{Msg: m.Content, SenderInfo: senderInfo}


	for _, att := range m.Attachments {
		aiMsg.Attachments = append(aiMsg.Attachments, &agent.Attachment{URI: att.URL, Mimetype: att.ContentType})
	}

	ch, err := b.agent.Run(ctx, aiMsg, agent.StreamingModeMessages)

	if err != nil {
		log.Printf("Bot: Error received from agent %v", err)
		return
	}

	for update := range ch {
		s.ChannelTyping(m.ChannelID)

		if (update.Err != nil) {
			fmt.Printf("error %v\n", err)
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Something went wrong :( - %v", err))
			// TODO: Send debug trace as spoiler or something that makes it hidden
		}

		if update.Text == "" && update.ToolCall == nil && update.ToolResponse == nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Something went wrong :( - %v", update))
		}

		// TODO: include tool call in msg
		if update.Text != "" {
			err = b.sendMessageInChunks(update.Text, nil, s, m)
		}

		if (err != nil) {
			fmt.Println(err)
		}
	}
}

func (b *Bot) getSenderIntoFromMessage(ctx context.Context, m *discordgo.MessageCreate) (*agent.MessageSenderInfo, error) {
	if m == nil || m.Author == nil {
		return nil, fmt.Errorf("message received does not have an authoor")
	}

	result, err := b.repository.GetUserDetailsByDiscordId(ctx, &m.Author.ID)

	if err != nil {
		return nil, err
	}

	return &agent.MessageSenderInfo{
		User: &agent.MessageUser{
			ID: result.User.ID,
			Name: result.User.Name,
			DiscordID: result.User.DiscordID,
		},
		Household: &agent.MessageHousehold{
			ID: result.Household.ID,
			Name: result.Household.Name,
		},
	}, nil
}

func (b *Bot) sendMessageInChunks(msg string, chunkSizePtr *int, s *discordgo.Session, m *discordgo.MessageCreate) error {
	remainder := []rune(msg)
	chunkSize := MAX_MESSAGE_LENGTH
	if chunkSizePtr != nil {
		chunkSize = *chunkSizePtr
	}
	for (len(remainder) > 0) {
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
