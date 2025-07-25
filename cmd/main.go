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

	dbConn := database.Init()
	defer dbConn.Close(ctx)

	db := database.New(dbConn)

	ts := transaction.NewTransactionService(db)

	tp := tool.NewToolProvider(&tool.ToolDependencies{Ts: ts})

	agentCfg := ai.AgentConfig{MaxTokens: 400}
	agent, err := ai.NewAgent(ctx, &agentCfg, tp)

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
