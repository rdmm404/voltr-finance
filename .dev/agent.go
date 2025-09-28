package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"rdmm404/voltr-finance/internal/ai"
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

	agent, err := ai.NewAgent(ctx, tp)

	if err != nil {
		log.Fatalf("Failed to initialize agent %v", err)
	}

	log.Println("Running. Press Ctrl+C to exit…")

	// agent.SendMessage(ctx, gai.NewUserTextMessage("What tools do you have available?"))
	res, _ := repository.GetUserDetailsByDiscordId(ctx, utils.StringPtr("263106741711929351"))

	ch, err := agent.Run(
		ctx,
		&ai.Message{
			Msg: "Please store the transactions in the image. These are personal transactions",
			Attachments: []*ai.Attachment{
				{Mimetype: "image/png", URI: "https://cdn.discordapp.com/attachments/1404637483077074984/1415541865335492769/image.png?ex=68d95658&is=68d804d8&hm=596a6a21f18dd397869ae0a7fae02ae61d81dea7926619187870d485a6ef14e7&"},
			},
			SenderInfo: &ai.MessageSenderInfo{
				User: &ai.MessageUser{
					ID: res.User.ID,
					Name: res.User.Name,
					DiscordID: res.User.DiscordID,
				},
				Household: &ai.MessageHousehold{
					ID: res.Household.ID,
					Name: res.Household.Name,
				},
			},
		},
		ai.StreamingModeComplete,
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