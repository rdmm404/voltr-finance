package database

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultSearchPath = "transactions"

type Config struct {
	User        string
	Password    string
	Host        string
	Port        uint16
	Name        string
	MaxPoolSize int32
	MinPoolSize int32
}

func (c Config) Validate() error {
	var errs []error
	if strings.TrimSpace(c.User) == "" {
		errs = append(errs, errors.New("database user is required"))
	}
	if strings.TrimSpace(c.Host) == "" {
		errs = append(errs, errors.New("database host is required"))
	}
	if strings.TrimSpace(c.Name) == "" {
		errs = append(errs, errors.New("database name is required"))
	}
	if c.Port == 0 {
		errs = append(errs, errors.New("database port is required"))
	}
	if c.MaxPoolSize < 1 {
		errs = append(errs, errors.New("database max pool size must be at least 1"))
	}
	if c.MinPoolSize < 0 || c.MinPoolSize > c.MaxPoolSize {
		errs = append(errs, errors.New("database min pool size must be between 0 and max pool size"))
	}
	return errors.Join(errs...)
}

func ConfigFromStrings(user, password, host, port, name string, maxPoolSize int) (Config, error) {
	parsedPort, err := strconv.ParseUint(port, 10, 16)
	if err != nil || parsedPort == 0 {
		return Config{}, fmt.Errorf("invalid database port %q", port)
	}
	config := Config{User: user, Password: password, Host: host, Port: uint16(parsedPort), Name: name, MaxPoolSize: int32(maxPoolSize)}
	return config, config.Validate()
}

func BuildPoolConfig(config Config) (*pgxpool.Config, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	connectionURL := fmt.Sprintf("postgres://%s:%s@%s/%s", url.PathEscape(config.User), url.PathEscape(config.Password), net.JoinHostPort(config.Host, strconv.Itoa(int(config.Port))), url.PathEscape(config.Name))
	poolConfig, err := pgxpool.ParseConfig(connectionURL)
	if err != nil {
		return nil, fmt.Errorf("parse database configuration: %w", err)
	}
	poolConfig.MaxConns = config.MaxPoolSize
	poolConfig.MinConns = config.MinPoolSize
	poolConfig.ConnConfig.RuntimeParams["search_path"] = defaultSearchPath
	return poolConfig, nil
}

func NewPool(ctx context.Context, config Config) (*pgxpool.Pool, error) {
	poolConfig, err := BuildPoolConfig(config)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, poolConfig)
}

// NewPoolFromURL is retained for CLI configuration files that already store a
// complete connection URL. New servers should use the validated Config path.
func NewPoolFromURL(ctx context.Context, connectionURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connectionURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = defaultSearchPath
	return pgxpool.NewWithConfig(ctx, config)
}
