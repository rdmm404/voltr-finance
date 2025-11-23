package config

import (
	"log/slog"
	"os"
	"strconv"
)

func GetEnvBool(key string, defaultValue bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
		slog.Warn("Environment variable cannot be parsed as bool, using default", "key", key, "value", value, "default", defaultValue)
	}
	return defaultValue
}

func GetEnvString(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
