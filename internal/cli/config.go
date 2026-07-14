package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	API APIConfig `json:"api"`
}

type APIConfig struct {
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"`
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

	var config Config
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Config{}, errors.New("parse config: unexpected trailing JSON")
		}
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	config.applyEnvironment()
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c *Config) applyEnvironment() {
	if value := strings.TrimSpace(os.Getenv("VOLTR_API_URL")); value != "" {
		c.API.BaseURL = value
	}
	if value := os.Getenv("VOLTR_API_KEY"); value != "" {
		c.API.APIKey = value
	}
}

func (c Config) Validate() error {
	var validationErrors []error
	if strings.TrimSpace(c.API.BaseURL) == "" {
		validationErrors = append(validationErrors, errors.New("api.baseUrl is required"))
	} else {
		parsed, err := url.Parse(c.API.BaseURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			validationErrors = append(validationErrors, errors.New("api.baseUrl must be an absolute HTTP(S) URL"))
		}
	}
	if strings.TrimSpace(c.API.APIKey) == "" {
		validationErrors = append(validationErrors, errors.New("api.apiKey is required"))
	}
	return errors.Join(validationErrors...)
}
