package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"rdmm404/voltr-finance/internal/ai"
	"rdmm404/voltr-finance/internal/ai/tool"
	database "rdmm404/voltr-finance/internal/database/repository"
	"rdmm404/voltr-finance/internal/transaction"
	"runtime/debug"
	"syscall"

	gai "github.com/firebase/genkit/go/ai"

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

	_, err = agent.SendMessage(ctx, gai.NewUserMessage(
		gai.NewTextPart("Give me 3 stories. You must give me each story as a separate message."),
		// gai.NewTextPart("Please store the transactions in the image. These are personal transactions. user ID is 1."),
		// gai.NewMediaPart(
		// 	"image/png",
		// 	"https://cdn.discordapp.com/attachments/1404637483077074984/1415541865335492769/image.png?ex=68d804d8&is=68d6b358&hm=316029f666e46a9b77d780e13333e2446e28f9eb9537c4d2f76088501bc4f9d8&",
		// ),
	))


	if err != nil {
		log.Fatalf("Error while sending message to agent - %v", err)
	}

	<-ctx.Done()
	log.Println("Signal received, exiting.")
	stop()
}