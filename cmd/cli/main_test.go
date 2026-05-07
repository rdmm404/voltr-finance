package main

import (
	"bytes"
	"context"
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
