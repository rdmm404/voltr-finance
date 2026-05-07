package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"rdmm404/voltr-finance/internal/app"
	"rdmm404/voltr-finance/internal/cli"
	"rdmm404/voltr-finance/internal/database"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/transaction"
)

func main() {
	configPath := flag.String("config", "", "path to config.json")
	flag.CommandLine.SetOutput(os.Stderr)
	flag.Parse()

	path, err := cli.ResolveConfigPath(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cfg, err := cli.LoadConfig(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg.Database.ConnString())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := sqlc.New(pool)
	txSvc := transaction.NewTransactionService(pool, repo)
	appSvc := app.NewService(repo, txSvc)

	os.Exit(cli.Run(ctx, flag.Args(), os.Stdin, os.Stdout, os.Stderr, appSvc))
}
