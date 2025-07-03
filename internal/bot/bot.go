package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var ErrInvalidBotConfig = errors.New("bot is not set up correctly")

type Bot struct {
	session *discordgo.Session
}

func NewBot() (*Bot, error) {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("%w - DISCORD_TOKEN environment variable is not set", ErrInvalidBotConfig)
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("error creating discord session - %w", err)
	}

	dg.AddHandler(messageCreate)

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	return &Bot{
		session: dg,
	}, nil
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

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	msgJson, _ := json.MarshalIndent(m, "", "  ")
	fmt.Printf("message received %v\n", string(msgJson))
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}
	// If the message is "ping" reply with "Pong!"
	if m.Content == "ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}

	// If the message is "pong" reply with "Ping!"
	if m.Content == "pong" {
		s.ChannelMessageSend(m.ChannelID, "Ping!")
	}
}
