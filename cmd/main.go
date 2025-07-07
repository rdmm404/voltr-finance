package main

import (
	"context"
	"log"
	"rdmm404/voltr-finance/internal/ai"
	"rdmm404/voltr-finance/internal/bot"
)

func main() {
	ctx := context.Background()
	agentCfg := ai.AgentConfig{MaxTokens: 400}
	agent, err := ai.NewAgent(ctx, &agentCfg)

	if err != nil {
		log.Fatalf("Failed to initialize agent %v", err)
	}

	bot, err := bot.NewBot(agent)
	if err != nil {
		log.Panicf("Error creating bot %v", err)
	}

	err = bot.Run()
	if err != nil {
		log.Panicf("Error running bot %v", err)
	}
}
