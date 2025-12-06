package config

import (
	"log/slog"
	"time"

	"github.com/joho/godotenv"
)

var (
	DEBUG     bool
	LOG_LEVEL LogLevel

	DISCORD_TOKEN                string
	DISCORD_APP_ID               string
	DISCORD_MAX_MESSAGE_LENGTH   int
	DISCORD_EVENT_HANDLE_TIMEOUT time.Duration
	DISCORD_CREATE_COMMANDS      bool

	DB_USER      string
	DB_PASSWORD  string
	DB_NAME      string
	DB_HOST      string
	DB_PORT      string
	DB_POOL_SIZE int

	AGENT_MAX_TURNS int
)

func init() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("Failed to load .env file. Using environment variables instead", "error", err)
	}
	// general
	DEBUG = GetEnvBool("DEBUG", true)

	//discord
	DISCORD_TOKEN = GetEnvString("DISCORD_TOKEN", "")
	DISCORD_APP_ID = GetEnvString("DISCORD_APP_ID", "")
	DISCORD_MAX_MESSAGE_LENGTH = GetEnvInt("DISCORD_MAX_MESSAGE_LENGTH", 2000)
	DISCORD_EVENT_HANDLE_TIMEOUT = time.Duration(GetEnvInt("DISCORD_EVENT_HANDLE_TIMEOUT", 300)) * time.Second
	DISCORD_CREATE_COMMANDS = GetEnvBool("DISCORD_CREATE_COMMANDS", true)

	logLevel := LogLevel(GetEnvString("LOG_LEVEL", "INFO"))
	LOG_LEVEL = logLevel
	if !logLevel.Valid() {
		slog.Warn("Invalid log level received, defaulting to INFO", "log_level", logLevel)
		LOG_LEVEL = LogLevelInfo
	}

	// database
	DB_USER = GetEnvString("DB_USER", "")
	DB_PASSWORD = GetEnvString("DB_PASSWORD", "")
	DB_HOST = GetEnvString("DB_HOST", "")
	DB_NAME = GetEnvString("DB_NAME", "")
	DB_PORT = GetEnvString("DB_PORT", "")
	DB_POOL_SIZE = GetEnvInt("DB_POOL_SIZE", 5)

	// agent
	AGENT_MAX_TURNS = GetEnvInt("AGENT_MAX_TURNS", 10)
}
