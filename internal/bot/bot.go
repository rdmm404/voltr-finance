package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"rdmm404/voltr-finance/internal/ai"
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
	token := os.Getenv("DISCORD_TOKEN")
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
	fmt.Printf("message received %v\n", string(msgJson))
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	resp, err := b.agent.SendMessage(context.TODO(), &ai.Message{Msg: m.Content})

	if (err != nil || len(resp.Candidates) < 1) {
		s.ChannelMessageSend(m.ChannelID, "Something went wrong :(")
		fmt.Printf("error %v", err)
		// TODO: Send debug trace as spoiler or something that makes it hidden
		return
	}

	_, err = s.ChannelMessageSend(m.ChannelID, resp.Candidates[0].Content.Parts[0].Text)
	if (err != nil) {
		fmt.Println(err)
	}
}
