package main

import (
	"context"
	"log/slog"
	"os"
	"rdmm404/voltr-finance/internal/ai/agent"
	"rdmm404/voltr-finance/internal/ai/tool"
	"rdmm404/voltr-finance/internal/bot"
	"rdmm404/voltr-finance/internal/config"
	"rdmm404/voltr-finance/internal/database"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/transaction"

	"cloud.google.com/go/storage"
)

func main() {
	slog.SetLogLoggerLevel(config.LOG_LEVEL.ToSlog())
	ctx := context.Background()

	db := database.Init(ctx)
	defer db.Close()

	repository := sqlc.New(db)

	ts := transaction.NewTransactionService(db, repository)

	tp := tool.NewToolProvider(&tool.ToolDependencies{Ts: ts})

	client, err := storage.NewClient(ctx)
	defer client.Close()
	if err != nil {
		slog.Error("Failed to initialize bucket client", "error", err)
		panic(err)
	}

	sm, err := agent.NewSessionManager(db, repository, client)

	if err != nil {
		slog.Error("Failed to initialize session manager", "error", err)
		os.Exit(1)
	}

	a, err := agent.NewChatAgent(ctx, tp, sm)

	if err != nil {
		slog.Error("Failed to initialize agent", "error", err)
		os.Exit(1)
	}

	bot, err := bot.NewBot(a, repository)
	if err != nil {
		slog.Error("Error creating bot", "error", err)
		panic(err)
	}

	err = bot.Run()
	if err != nil {
		slog.Error("Error running bot", "error", err)
		panic(err)
	}
}
