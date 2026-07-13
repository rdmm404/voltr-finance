package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	appbudgets "rdmm404/voltr-finance/internal/app/budgets"
	appcategories "rdmm404/voltr-finance/internal/app/categories"
	apphouseholds "rdmm404/voltr-finance/internal/app/households"
	apptransactions "rdmm404/voltr-finance/internal/app/transactions"
	appusers "rdmm404/voltr-finance/internal/app/users"
	"rdmm404/voltr-finance/internal/database"
	"rdmm404/voltr-finance/internal/database/sqlc"
	"rdmm404/voltr-finance/internal/httpapi"
	budgetpostgres "rdmm404/voltr-finance/internal/postgres/budgets"
	categorypostgres "rdmm404/voltr-finance/internal/postgres/categories"
	householdpostgres "rdmm404/voltr-finance/internal/postgres/households"
	transactionpostgres "rdmm404/voltr-finance/internal/postgres/transactions"
	userpostgres "rdmm404/voltr-finance/internal/postgres/users"
	"rdmm404/voltr-finance/internal/server"
)

type config struct {
	API      httpapi.Config
	Database database.Config
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, loadConfig()); err != nil {
		slog.Error("API server stopped", "error", err)
		os.Exit(1)
	}
}

func loadConfig() config {
	return config{
		API: httpapi.Config{Address: env("VOLTR_API_ADDRESS", ":8080"), APIKey: os.Getenv("VOLTR_API_KEY")},
		Database: database.Config{
			User: os.Getenv("DB_USER"), Password: os.Getenv("DB_PASSWORD"), Host: os.Getenv("DB_HOST"),
			Port: uint16(envInt("DB_PORT", 5432)), Name: os.Getenv("DB_NAME"),
			MaxPoolSize: int32(envInt("DB_POOL_SIZE", 5)), MinPoolSize: int32(envInt("DB_MIN_POOL_SIZE", 0)),
		},
	}
}

func (c config) Validate() error {
	return errors.Join(c.API.Validate(), c.Database.Validate())
}

func run(ctx context.Context, cfg config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate configuration: %w", err)
	}
	pool, err := database.NewPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("open database pool: %w", err)
	}
	defer pool.Close()

	queries := sqlc.New(pool)
	userService := appusers.NewService(userpostgres.NewRepository(queries))
	categoryService := appcategories.NewService(categorypostgres.NewRepository(queries))
	householdService := apphouseholds.NewService(householdpostgres.NewRepository(queries))
	transactionService := apptransactions.NewService(
		transactionpostgres.NewRepository(pool),
		identityResolver{users: userService},
		categoryResolver{categories: categoryService},
	)
	budgetService := appbudgets.NewService(budgetpostgres.NewRepository(queries), budgetpostgres.NewTransactor(pool))

	httpServer, err := server.New(cfg.API, transactionService, userService, householdService, categoryService, budgetService)
	if err != nil {
		return fmt.Errorf("configure HTTP server: %w", err)
	}

	result := make(chan error, 1)
	go func() {
		slog.Info("API server listening", "address", httpServer.Addr)
		result <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("shut down HTTP server: %w", err)
		}
		err := <-result
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

type identityResolver struct{ users *appusers.Service }

func (r identityResolver) ResolveUserID(ctx context.Context, selector apptransactions.IdentitySelector) (int64, error) {
	user, err := r.users.Resolve(ctx, appusers.Selector{
		UserID: selector.UserID, DiscordID: selector.DiscordID, TelegramID: selector.TelegramID,
		PhoneNumber: selector.PhoneNumber, WhatsAppID: selector.WhatsAppID,
	})
	return user.ID, err
}

type categoryResolver struct{ categories *appcategories.Service }

func (r categoryResolver) ResolveActiveCategoryID(ctx context.Context, id *int64, code *string) (*int64, error) {
	if id == nil && code == nil {
		return nil, nil
	}
	category, err := r.categories.ResolveActive(ctx, id, code)
	if err != nil {
		return nil, err
	}
	return &category.ID, nil
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}
