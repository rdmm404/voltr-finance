package main

import (
	"log"
	"rdmm404/voltr-finance/internal/bot"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Failed to load .env file %v", err)
	}

	bot, err := bot.NewBot()
	if err != nil {
		log.Panicf("Error creating bot %w", err)
	}

	err = bot.Run()
	if err != nil {
		log.Panicf("Error running bot %w", err)
	}
}
