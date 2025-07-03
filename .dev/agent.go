package main

import (
	"context"
	"io"
	"log"
	"os"
	"rdmm404/voltr-finance/internal/ai"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Failed to load .env file %v", err)
	}

	file, err := os.Open("test.jpg")

	if err != nil {
		log.Fatalf("Error opening file %v\n", err)
	}

	fileContents, err := io.ReadAll(file)

	if err != nil {
		log.Fatalf("Error reading file %v\n", err)
	}

	defer file.Close()

	ctx := context.Background()
	agent, err := ai.NewAgent(ctx, nil)

	if err != nil {
		log.Fatalf("Failed to initialize agent %v", err)
	}

	_, err = agent.SendMessage(ctx, &ai.Message{
		Attachments: []ai.Attachment{
			{ File: fileContents, Mimetype: "image/jpeg"},
		},
		Msg: "Sent by Rob",
	})

	if err != nil {
		log.Fatalf("Error while calling LLM provider %v", err)
	}
}