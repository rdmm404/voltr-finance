package main

import (
	"strings"
	"testing"
)

func TestLoadConfigAndValidate(t *testing.T) {
	t.Setenv("VOLTR_API_ADDRESS", ":9090")
	t.Setenv("VOLTR_API_KEY", "secret")
	t.Setenv("DB_USER", "voltr")
	t.Setenv("DB_PASSWORD", "password")
	t.Setenv("DB_HOST", "database")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_NAME", "finance")
	t.Setenv("DB_POOL_SIZE", "8")
	t.Setenv("DB_MIN_POOL_SIZE", "2")
	config := loadConfig()
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}
	if config.API.Address != ":9090" || config.Database.Port != 5433 || config.Database.MaxPoolSize != 8 || config.Database.MinPoolSize != 2 {
		t.Fatalf("config=%+v", config)
	}
}

func TestConfigurationRejectsEmptyAPIKeyBeforeStartup(t *testing.T) {
	t.Setenv("VOLTR_API_KEY", "")
	t.Setenv("DB_USER", "voltr")
	t.Setenv("DB_HOST", "database")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_NAME", "finance")
	err := loadConfig().Validate()
	if err == nil || !strings.Contains(err.Error(), "API key") {
		t.Fatalf("error=%v", err)
	}
}

func TestInvalidDatabaseNumbersFailValidation(t *testing.T) {
	t.Setenv("VOLTR_API_KEY", "secret")
	t.Setenv("DB_USER", "voltr")
	t.Setenv("DB_HOST", "database")
	t.Setenv("DB_PORT", "invalid")
	t.Setenv("DB_NAME", "finance")
	err := loadConfig().Validate()
	if err == nil || !strings.Contains(err.Error(), "database port") {
		t.Fatalf("error=%v", err)
	}
}
