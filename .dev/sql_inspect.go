package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"rdmm404/voltr-finance/internal/database"
	"rdmm404/voltr-finance/internal/database/sqlc"
)

func main() {
	ctx := context.Background()
	db := database.Init(ctx)
	queries := sqlc.New(db)

	metadata, err := database.InspectTables(ctx, queries, []string{"household", "household_user", "transaction", "users"})
	if err != nil {
		slog.Error("error inspecting database", "error", err)
		os.Exit(1)
	}

	jsonMetadata, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		slog.Error("error marshalling metadata", "error", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonMetadata))
}
