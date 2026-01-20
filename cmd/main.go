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
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

func main() {
	slog.SetLogLoggerLevel(config.LOG_LEVEL.ToSlog())
	ctx := context.Background()

	db := database.Init(ctx)
	defer db.Close()

	readOnlyDB := database.InitReadOnly(ctx)
	defer readOnlyDB.Close()

	repository := sqlc.New(db)

	ts := transaction.NewTransactionService(db, repository)

	tp := tool.NewToolProvider(&tool.ToolDependencies{Ts: ts, ReadOnlyDB: readOnlyDB})

	client, err := storage.NewClient(ctx)
	defer func() {
		if err := client.Close(); err != nil {
			slog.Error("Failed to close storage client", "error", err)
		}
	}()

	if err != nil {
		slog.Error("Failed to initialize bucket client", "error", err)
		panic(err)
	}

	sm, err := agent.NewSessionManager(db, repository, client)

	if err != nil {
		slog.Error("Failed to initialize session manager", "error", err)
		os.Exit(1)
	}

	g := genkit.Init(
		ctx,
		genkit.WithPlugins(&googlegenai.VertexAI{Location: "global"}),
		genkit.WithDefaultModel("vertexai/gemini-2.5-flash"),
	)

	tp.Init(g)

	a, err := agent.NewChatAgent(ctx, tp, sm, repository, g)

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
