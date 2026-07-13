package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractConfigArgRemovesConfigFlag(t *testing.T) {
	configPath, args, err := extractConfigArg([]string{"--config", "/tmp/config.json", "users", "list"})
	if err != nil {
		t.Fatalf("extractConfigArg returned error: %v", err)
	}
	if configPath != "/tmp/config.json" {
		t.Fatalf("configPath = %q, want /tmp/config.json", configPath)
	}
	if strings.Join(args, " ") != "users list" {
		t.Fatalf("args = %v, want users list", args)
	}
}

func TestExtractConfigArgSupportsEqualsForm(t *testing.T) {
	configPath, args, err := extractConfigArg([]string{"transactions", "list", "--config=/tmp/config.json"})
	if err != nil {
		t.Fatalf("extractConfigArg returned error: %v", err)
	}
	if configPath != "/tmp/config.json" {
		t.Fatalf("configPath = %q, want /tmp/config.json", configPath)
	}
	if strings.Join(args, " ") != "transactions list" {
		t.Fatalf("args = %v, want transactions list", args)
	}
}

func TestRunDelegatesHelpToKongBeforeConfig(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--help"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "transactions create") {
		t.Fatalf("stdout = %q, want Kong help", stdout.String())
	}
}

func TestRunUsesAuthenticatedAPIAndEnvironmentOverrides(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer env-key" {
			t.Errorf("authorization=%q", request.Header.Get("Authorization"))
		}
		if request.Method != http.MethodGet || request.URL.Path != "/v1/users" {
			t.Errorf("request=%s %s", request.Method, request.URL.Path)
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	t.Setenv("VOLTR_API_URL", server.URL)
	t.Setenv("VOLTR_API_KEY", "env-key")
	config := writeCLIConfig(t, "https://unused.example.com", "file-key")
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--config", config, "users", "list"}, nil, &stdout, &stderr)
	if code != 0 || stdout.String() != "[]\n" {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunRendersPartialBulkResultBeforeExitTwo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/transactions/bulk" {
			t.Errorf("path=%s", request.URL.Path)
		}
		_, _ = w.Write([]byte(`{"succeeded":[{"index":0,"id":7}],"failed":[{"index":1,"error":{"code":"validation_error","message":"bad"}}]}`))
	}))
	defer server.Close()
	config := writeCLIConfig(t, server.URL, "key")
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--config", config, "transactions", "create-bulk"}, strings.NewReader(`{"transactions":[]}`), &stdout, &stderr)
	if code != 2 || !strings.Contains(stdout.String(), `"succeeded"`) || !strings.Contains(stdout.String(), `"failed"`) {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

func TestRunAuthenticationFailureExitsOne(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"authentication failed"}}`))
	}))
	defer server.Close()
	config := writeCLIConfig(t, server.URL, "bad-key")
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--config", config, "users", "list"}, nil, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "authentication failed") {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
}

func TestRunBudgetCreateFlagSelectsEndpoint(t *testing.T) {
	methods := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		methods = append(methods, request.Method)
		_, _ = w.Write([]byte(`{"id":1,"periodStart":"2026-07-01T00:00:00Z","periodEnd":"2026-07-31T00:00:00Z","lines":[]}`))
	}))
	defer server.Close()
	config := writeCLIConfig(t, server.URL, "key")
	for _, create := range []bool{false, true} {
		args := []string{"--config", config, "budgets", "get", "--household-id=2", "--month=2026-07"}
		if create {
			args = append(args, "--create")
		}
		var stdout, stderr bytes.Buffer
		if code := run(context.Background(), args, nil, &stdout, &stderr); code != 0 {
			t.Fatalf("code=%d stderr=%s", code, stderr.String())
		}
	}
	if strings.Join(methods, ",") != "GET,POST" {
		t.Fatalf("methods=%v", methods)
	}
}

func writeCLIConfig(t *testing.T, baseURL, apiKey string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	content := fmt.Sprintf(`{"api":{"baseUrl":%q,"apiKey":%q}}`, baseURL, apiKey)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
