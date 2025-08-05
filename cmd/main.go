package main

import (
	"context"
	"log"
	"rdmm404/voltr-finance/internal/ai"
	"rdmm404/voltr-finance/internal/ai/tool"
	"rdmm404/voltr-finance/internal/bot"
	database "rdmm404/voltr-finance/internal/database/repository"
	"rdmm404/voltr-finance/internal/transaction"
)

func main() {
	ctx := context.Background()

	db := database.Init()
	defer db.Close(ctx)

	repository := database.New(db)

	ts := transaction.NewTransactionService(db, repository)

	tp := tool.NewToolProvider(&tool.ToolDependencies{Ts: ts})

	agentCfg := ai.AgentConfig{MaxTokens: 5000}
	agent, err := ai.NewAgent(ctx, &agentCfg, tp)

	if err != nil {
		log.Fatalf("Failed to initialize agent %v", err)
	}

	bot, err := bot.NewBot(agent, repository)
	if err != nil {
		log.Panicf("Error creating bot %v", err)
	}

	err = bot.Run()
	if err != nil {
		log.Panicf("Error running bot %v", err)
	}
}
