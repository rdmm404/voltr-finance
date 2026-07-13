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

	databaseConfig, err := database.ConfigFromStrings(config.DB_USER, config.DB_PASSWORD, config.DB_HOST, config.DB_PORT, config.DB_NAME, config.DB_POOL_SIZE)
	if err != nil {
		slog.Error("Invalid database configuration", "error", err)
		os.Exit(1)
	}
	db, err := database.NewPool(ctx, databaseConfig)
	if err != nil {
		slog.Error("Failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	readOnlyConfig := databaseConfig
	readOnlyConfig.User = config.DB_RO_USER
	readOnlyConfig.Password = config.DB_RO_PASSWORD
	readOnlyDB, err := database.NewPool(ctx, readOnlyConfig)
	if err != nil {
		slog.Error("Failed to create read-only database pool", "error", err)
		os.Exit(1)
	}
	defer readOnlyDB.Close()

	repository := sqlc.New(db)

	ts := transaction.NewTransactionService(db, repository)

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

	tp := tool.NewToolProvider(&tool.ToolDependencies{Ts: ts, ReadOnlyDB: readOnlyDB, Genkit: g})
	tp.Init()

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
