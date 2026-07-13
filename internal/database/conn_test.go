package database

import (
	"strings"
	"testing"
)

func TestBuildPoolConfigValidatesBoundsAndSearchPath(t *testing.T) {
	config := Config{User: "user", Password: "p@ss", Host: "127.0.0.1", Port: 5432, Name: "finance", MaxPoolSize: 8, MinPoolSize: 2}
	poolConfig, err := BuildPoolConfig(config)
	if err != nil {
		t.Fatalf("BuildPoolConfig error=%v", err)
	}
	if poolConfig.MaxConns != 8 || poolConfig.MinConns != 2 {
		t.Fatalf("pool bounds=%d/%d", poolConfig.MinConns, poolConfig.MaxConns)
	}
	if poolConfig.ConnConfig.RuntimeParams["search_path"] != "transactions" {
		t.Fatalf("search_path=%q", poolConfig.ConnConfig.RuntimeParams["search_path"])
	}
	if poolConfig.ConnConfig.Password != "p@ss" {
		t.Fatalf("password did not round trip")
	}
}

func TestConfigValidationReturnsAllRelevantFailures(t *testing.T) {
	err := (Config{MinPoolSize: 2}).Validate()
	if err == nil {
		t.Fatal("Validate returned nil")
	}
	for _, message := range []string{"user is required", "host is required", "name is required", "port is required", "max pool size", "min pool size"} {
		if !strings.Contains(err.Error(), message) {
			t.Errorf("error %q missing %q", err, message)
		}
	}
	if _, err := ConfigFromStrings("user", "pass", "host", "not-a-port", "db", 5); err == nil {
		t.Fatal("ConfigFromStrings accepted invalid port")
	}
}
