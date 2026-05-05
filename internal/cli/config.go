package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

const defaultPoolSize = 5

type Config struct {
	Database DBConfig `json:"database"`
}

type DBConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
	PoolSize int    `json:"poolSize"`
}

func ResolveConfigPath(flagPath string) (string, error) {
	if flagPath != "" {
		return flagPath, nil
	}
	if envPath := os.Getenv("VOLTR_CONFIG"); envPath != "" {
		return envPath, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "voltr-finance", "config.json"), nil
}

func LoadConfig(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	return c.Database.Validate()
}

func (c DBConfig) Validate() error {
	missing := make([]error, 0)
	if c.Host == "" {
		missing = append(missing, errors.New("database.host is required"))
	}
	if c.Port == "" {
		missing = append(missing, errors.New("database.port is required"))
	}
	if c.Name == "" {
		missing = append(missing, errors.New("database.name is required"))
	}
	if c.User == "" {
		missing = append(missing, errors.New("database.user is required"))
	}
	if c.Password == "" {
		missing = append(missing, errors.New("database.password is required"))
	}
	return errors.Join(missing...)
}

func (c DBConfig) ConnString() string {
	poolSize := c.PoolSize
	if poolSize == 0 {
		poolSize = defaultPoolSize
	}

	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   c.Host + ":" + c.Port,
		Path:   c.Name,
	}
	q := u.Query()
	q.Set("pool_max_conns", fmt.Sprintf("%d", poolSize))
	q.Set("search_path", "transactions")
	u.RawQuery = q.Encode()
	return u.String()
}
