package config

import (
	"log"

	"github.com/joho/godotenv"
)

var (
	DEBUG         bool
	DISCORD_TOKEN string
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Failed to load .env file %v", err)
	}

	DISCORD_TOKEN = GetEnvString("DISCORD_TOKEN", "")
	DEBUG = GetEnvBool("DEBUG", true)
}
