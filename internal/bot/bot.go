package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"rdmm404/voltr-finance/internal/ai"
	"rdmm404/voltr-finance/internal/config"
	"rdmm404/voltr-finance/internal/utils"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var ErrInvalidBotConfig = errors.New("bot is not set up correctly")

type Bot struct {
	session *discordgo.Session
	agent Agent
}

type Agent interface {
	SendMessage(ctx context.Context, msg *ai.Message) (*ai.AgentResponse, error)
}

func NewBot(agent Agent) (*Bot, error) {
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
		agent: agent,
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
	msgJson, _ := json.MarshalIndent(m, "", "  ")
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	fmt.Printf("message received %v\n", string(msgJson))

	s.ChannelTyping(m.ChannelID)

	aiMsg := &ai.Message{Msg: m.Content}

		for _, att := range m.Attachments {
			bytes, err := utils.DownloadFileBytes(att.URL)
			if err != nil {
				fmt.Printf("Error downloading attachment %+v - %v\n", att, err)
				return
			}
			aiMsg.Attachments = append(aiMsg.Attachments, &ai.Attachment{File: bytes, Mimetype: att.ContentType})
		}

	resp, err := b.agent.SendMessage(context.TODO(), aiMsg)

	if (err != nil || len(resp.Candidates) < 1) {
		fmt.Printf("error %v\n", err)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Something went wrong :( - %v", err))
		// TODO: Send debug trace as spoiler or something that makes it hidden
		return
	}

	err = sendMessageInChunks(resp.Candidates[0].Content.Parts[0].Text, nil, s, m)

	if (err != nil) {
		fmt.Println(err)
	}
}


func sendMessageInChunks(msg string, chunkSizePtr *int, s *discordgo.Session, m *discordgo.MessageCreate) error {
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