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
)

func main() {
	slog.SetLogLoggerLevel(config.LOG_LEVEL.ToSlog())
	ctx := context.Background()

	db := database.Init()
	defer db.Close(ctx)

	repository := sqlc.New(db)

	ts := transaction.NewTransactionService(db, repository)

	tp := tool.NewToolProvider(&tool.ToolDependencies{Ts: ts})

	sm, err := agent.NewSessionManager(db, repository)

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
