package main

import (
	"context"
	"log"
	"rdmm404/voltr-finance/internal/ai"
	"rdmm404/voltr-finance/internal/bot"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Failed to load .env file %v", err)
	}

	ctx := context.Background()
	agentCfg := ai.AgentConfig{MaxTokens: 400}
	agent, err := ai.NewAgent(ctx, &agentCfg)

	if err != nil {
		log.Fatalf("Failed to initialize agent %v", err)
	}

	bot, err := bot.NewBot(agent)
	if err != nil {
		log.Panicf("Error creating bot %w", err)
	}

	err = bot.Run()
	if err != nil {
		log.Panicf("Error running bot %w", err)
	}
}
