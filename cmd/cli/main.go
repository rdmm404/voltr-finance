package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"rdmm404/voltr-finance/internal/app"
	"rdmm404/voltr-finance/internal/cli"
	"rdmm404/voltr-finance/internal/database"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/transaction"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	configFlag, cliArgs, err := extractConfigArg(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if shouldDelegateBeforeConfig(cliArgs) {
		return cli.Run(ctx, cliArgs, stdin, stdout, stderr, nil)
	}
	path, err := cli.ResolveConfigPath(configFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cfg, err := cli.LoadConfig(path)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	pool, err := database.NewPool(ctx, cfg.Database.ConnString())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer pool.Close()

	repo := sqlc.New(pool)
	txSvc := transaction.NewTransactionService(pool, repo)
	appSvc := app.NewServiceWithTransactor(repo, txSvc, app.NewSQLCTransactor(pool, repo))

	return cli.Run(ctx, cliArgs, stdin, stdout, stderr, appSvc)
}

func extractConfigArg(args []string) (string, []string, error) {
	cliArgs := make([]string, 0, len(args))
	configPath := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--config":
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--config requires a path")
			}
			i++
			configPath = args[i]
		case strings.HasPrefix(arg, "--config="):
			configPath = strings.TrimPrefix(arg, "--config=")
			if configPath == "" {
				return "", nil, fmt.Errorf("--config requires a path")
			}
		default:
			cliArgs = append(cliArgs, arg)
		}
	}
	return configPath, cliArgs, nil
}

func shouldDelegateBeforeConfig(args []string) bool {
	if len(args) == 0 {
		return true
	}
	for _, arg := range args {
		if arg == "--help" || arg == "-h" || arg == "help" {
			return true
		}
	}
	return strings.HasPrefix(args[0], "-")
}
