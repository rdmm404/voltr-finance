package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"rdmm404/voltr-finance/internal/ai/agent"
	"rdmm404/voltr-finance/internal/ai/tool"
	database "rdmm404/voltr-finance/internal/database/repository"
	"rdmm404/voltr-finance/internal/transaction"
	"rdmm404/voltr-finance/internal/utils"
	"runtime/debug"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	defer func() {
        if r := recover(); r != nil {
            slog.Error("recovered from panic", "panic", r, "stack", string(debug.Stack()))
			slog.Info("Process still running. Press Ctrl+C to exit.")
        }
    }()

	slog.SetLogLoggerLevel(slog.LevelDebug)

	if err := godotenv.Load(); err != nil {
		slog.Error("Failed to load .env file", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	db := database.Init()
	defer db.Close(ctx)

	repository := database.New(db)

	ts := transaction.NewTransactionService(db, repository)

	tp := tool.NewToolProvider(&tool.ToolDependencies{Ts: ts})

	a, err := agent.NewChatAgent(ctx, tp)

	if err != nil {
		slog.Error("Failed to initialize agent", "error", err)
		os.Exit(1)
	}

	slog.Info("Running. Press Ctrl+C to exit…")

	// agent.SendMessage(ctx, gagent.NewUserTextMessage("What tools do you have available?"))
	res, _ := repository.GetUserDetailsByDiscordId(ctx, utils.StringPtr("263106741711929351"))

	ch, err := a.Run(
		ctx,
		&agent.Message{
			Msg: "Please store the transactions in the image. These are personal transactions",
			Attachments: []*agent.Attachment{
				{Mimetype: "image/png", URI: "https://cdn.discordapp.com/attachments/1404637483077074984/1415541865335492769/image.png?ex=68d95658&is=68d804d8&hm=596a6a21f18dd397869ae0a7fae02ae61d81dea7926619187870d485a6ef14e7&"},
			},
			SenderInfo: &agent.MessageSenderInfo{
				User: &agent.MessageUser{
					ID: res.User.ID,
					Name: res.User.Name,
					DiscordID: res.User.DiscordID,
				},
				Household: &agent.MessageHousehold{
					ID: res.Household.ID,
					Name: res.Household.Name,
				},
			},
		},
		agent.StreamingModeComplete,
	)

	for update := range ch {
		if update == nil {
			slog.Error("CRITICAL: nil model update received")
		}

		slog.Debug("**** BEGIN UPDATE ****")
		jsonUpdate, _ := json.Marshal(update)
		fmt.Println(string(jsonUpdate))
		slog.Debug("**** END UPDATE ****")
	}


	if err != nil {
		slog.Error("Error while sending message to agent", "error", err)
		os.Exit(1)
	}

	<-ctx.Done()
	slog.Info("Signal received, exiting.")
	stop()
}