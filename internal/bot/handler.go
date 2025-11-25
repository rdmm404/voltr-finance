package bot

import (
	"context"
	"log/slog"
	"rdmm404/voltr-finance/internal/config"

	"github.com/bwmarrin/discordgo"
)

func eventHandler[E any](handler func(context.Context, *discordgo.Session, *E) error) func(*discordgo.Session, *E) {
	return func(s *discordgo.Session, e *E) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Bot: recovered from panic", "error", r)
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), config.DISCORD_EVENT_HANDLE_TIMEOUT)
		defer cancel()

		err := handler(ctx, s, e)

		if err != nil {
			slog.Error("Bot: error handling discord event", "event", e, "error", err)
		}
	}
}
