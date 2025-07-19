package config

import (
	"log"

	"github.com/joho/godotenv"
)

var (
	DEBUG         bool
	DISCORD_TOKEN string
	DB_USER string
	DB_PASSWORD string
	DB_NAME string
	DB_HOST string
	DB_PORT string
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Failed to load .env file %v. Using environment variables instead", err)
	}

	DISCORD_TOKEN = GetEnvString("DISCORD_TOKEN", "")
	DEBUG = GetEnvBool("DEBUG", true)
	DB_USER = GetEnvString("DB_USER", "")
	DB_PASSWORD = GetEnvString("DB_PASSWORD", "")
	DB_HOST = GetEnvString("DB_HOST", "")
	DB_NAME = GetEnvString("DB_NAME", "")
	DB_PORT = GetEnvString("DB_PORT", "")
}
