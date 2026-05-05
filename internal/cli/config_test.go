package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveConfigPathPrefersFlag(t *testing.T) {
	t.Setenv("VOLTR_CONFIG", "/env/config.json")

	path, err := ResolveConfigPath("/flag/config.json")
	if err != nil {
		t.Fatalf("ResolveConfigPath returned error: %v", err)
	}
	if path != "/flag/config.json" {
		t.Fatalf("path = %q, want flag path", path)
	}
}

func TestResolveConfigPathFallsBackToEnv(t *testing.T) {
	t.Setenv("VOLTR_CONFIG", "/env/config.json")

	path, err := ResolveConfigPath("")
	if err != nil {
		t.Fatalf("ResolveConfigPath returned error: %v", err)
	}
	if path != "/env/config.json" {
		t.Fatalf("path = %q, want env path", path)
	}
}

func TestResolveConfigPathDefaultsToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VOLTR_CONFIG", "")

	path, err := ResolveConfigPath("")
	if err != nil {
		t.Fatalf("ResolveConfigPath returned error: %v", err)
	}

	want := filepath.Join(home, ".config", "voltr-finance", "config.json")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestLoadConfigParsesDatabaseFields(t *testing.T) {
	path := writeConfig(t, `{
		"database": {
			"host": "localhost",
			"port": "5432",
			"name": "voltr_finance",
			"user": "voltr",
			"password": "secret",
			"poolSize": 9
		}
	}`)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if cfg.Database.Host != "localhost" || cfg.Database.Port != "5432" || cfg.Database.Name != "voltr_finance" || cfg.Database.User != "voltr" || cfg.Database.Password != "secret" || cfg.Database.PoolSize != 9 {
		t.Fatalf("unexpected database config: %+v", cfg.Database)
	}
}

func TestLoadConfigRejectsUnknownFields(t *testing.T) {
	path := writeConfig(t, `{
		"database": {
			"host": "localhost",
			"port": "5432",
			"name": "voltr_finance",
			"user": "voltr",
			"password": "secret",
			"extra": true
		}
	}`)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("LoadConfig returned nil error")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error = %q, want unknown field validation", err)
	}
}

func TestLoadConfigRequiresDatabaseFields(t *testing.T) {
	required := map[string]string{
		"host":     `"port":"5432","name":"voltr_finance","user":"voltr","password":"secret"`,
		"port":     `"host":"localhost","name":"voltr_finance","user":"voltr","password":"secret"`,
		"name":     `"host":"localhost","port":"5432","user":"voltr","password":"secret"`,
		"user":     `"host":"localhost","port":"5432","name":"voltr_finance","password":"secret"`,
		"password": `"host":"localhost","port":"5432","name":"voltr_finance","user":"voltr"`,
	}

	for field, body := range required {
		t.Run(field, func(t *testing.T) {
			path := writeConfig(t, `{"database":{`+body+`}}`)

			_, err := LoadConfig(path)
			if err == nil {
				t.Fatal("LoadConfig returned nil error")
			}
			if !strings.Contains(err.Error(), "database."+field) {
				t.Fatalf("error = %q, want missing database.%s", err, field)
			}
		})
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
