package main

import (
	"context"
	"fmt"
	"rdmm404/voltr-finance/internal/ai"

	"github.com/joho/godotenv"
)

func PanicOnErr(err error) {
	if (err != nil) {
		fmt.Println(err)
		panic(err)
	}
}

func main() {
	err := godotenv.Load()
	PanicOnErr(err)

	ctx := context.Background()
	agent, err := ai.NewAgent(ctx, nil)

	PanicOnErr(err)

	_, err = agent.SendMessage(ctx, &ai.Message{
		Msg: "I am testing your tool calling capabilities. can you call the SaveTransactions tool with some dummy data?",
	})

	PanicOnErr(err)
}