package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
            log.Printf("recovered from panic: %v\n%s", r, debug.Stack())
			log.Println("Process still running. Press Ctrl+C to exit.")
        }
    }()

	if err := godotenv.Load(); err != nil {
		log.Fatalf("Failed to load .env file %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	db := database.Init()
	defer db.Close(ctx)

	repository := database.New(db)

	ts := transaction.NewTransactionService(db, repository)

	tp := tool.NewToolProvider(&tool.ToolDependencies{Ts: ts})

	a, err := agent.NewChatAgent(ctx, tp)

	if err != nil {
		log.Fatalf("Failed to initialize agent %v", err)
	}

	log.Println("Running. Press Ctrl+C to exit…")

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
			log.Println("CRITICAL: nil model update received")
		}

		log.Println("**** BEGIN UPDATE ****")
		jsonUpdate, _ := json.Marshal(update)
		fmt.Println(string(jsonUpdate))
		log.Println("**** END UPDATE ****")
	}


	if err != nil {
		log.Fatalf("Error while sending message to agent - %v", err)
	}

	<-ctx.Done()
	log.Println("Signal received, exiting.")
	stop()
}