package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveConfigPathPrecedence(t *testing.T) {
	t.Setenv("VOLTR_CONFIG", "/env/config.json")
	path, err := ResolveConfigPath("/flag/config.json")
	if err != nil || path != "/flag/config.json" {
		t.Fatalf("path=%q error=%v", path, err)
	}
	path, err = ResolveConfigPath("")
	if err != nil || path != "/env/config.json" {
		t.Fatalf("path=%q error=%v", path, err)
	}
}

func TestResolveConfigPathDefaultsToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VOLTR_CONFIG", "")
	path, err := ResolveConfigPath("")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".config", "voltr-finance", "config.json")
	if path != want {
		t.Fatalf("path=%q want=%q", path, want)
	}
}

func TestLoadConfigParsesAPIFields(t *testing.T) {
	path := writeConfig(t, `{"api":{"baseUrl":"https://api.example.com","apiKey":"secret"}}`)
	config, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.API.BaseURL != "https://api.example.com" || config.API.APIKey != "secret" {
		t.Fatalf("config=%+v", config)
	}
}

func TestLoadConfigAppliesEnvironmentOverrides(t *testing.T) {
	t.Setenv("VOLTR_API_URL", "https://override.example.com/")
	t.Setenv("VOLTR_API_KEY", "override-key")
	path := writeConfig(t, `{"api":{"baseUrl":"https://file.example.com","apiKey":"file-key"}}`)
	config, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.API.BaseURL != "https://override.example.com/" || config.API.APIKey != "override-key" {
		t.Fatalf("config=%+v", config)
	}
}

func TestLoadConfigRejectsUnknownFieldsAndTrailingTokens(t *testing.T) {
	for name, content := range map[string]string{
		"unknown":  `{"api":{"baseUrl":"https://api.example.com","apiKey":"key","database":"no"}}`,
		"trailing": `{"api":{"baseUrl":"https://api.example.com","apiKey":"key"}} {}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := LoadConfig(writeConfig(t, content))
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestLoadConfigRequiresValidAPIFields(t *testing.T) {
	for name, content := range map[string]string{
		"base URL": `{"api":{"apiKey":"key"}}`, "API key": `{"api":{"baseUrl":"https://api.example.com"}}`,
		"absolute URL": `{"api":{"baseUrl":"localhost:8080","apiKey":"key"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := LoadConfig(writeConfig(t, content))
			if err == nil || !strings.Contains(err.Error(), "api.") {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
