package config

import (
	"fmt"
	"os"
	"strconv"
)

func GetEnvBool(key string, defaultValue bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
		fmt.Printf("Warning: Environment variable %s='%s' cannot be parsed as bool. Using default %t.\n", key, value, defaultValue)
	}
	return defaultValue
}

func GetEnvString(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
