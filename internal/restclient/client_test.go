package restclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewValidatesAndNormalizesConfiguration(t *testing.T) {
	client, err := New(Config{BaseURL: " https://example.com/root/ ", APIKey: "secret", Timeout: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if client.baseURL.String() != "https://example.com/root" || client.http.Timeout != 2*time.Second {
		t.Fatalf("client = %#v", client)
	}
	for name, config := range map[string]Config{
		"missing URL": {APIKey: "key"}, "invalid scheme": {BaseURL: "ftp://example.com", APIKey: "key"},
		"credentials in URL": {BaseURL: "https://user@example.com", APIKey: "key"}, "missing key": {BaseURL: "https://example.com"},
		"negative timeout": {BaseURL: "https://example.com", APIKey: "key", Timeout: -1},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := New(config); err == nil {
				t.Fatal("New accepted invalid config")
			}
		})
	}
}

func TestDoAuthenticatesAndStrictlyDecodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/base/v1/test" || request.URL.Query().Get("value") != "x" {
			t.Errorf("URL = %s", request.URL)
		}
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	client, err := New(Config{BaseURL: server.URL + "/base/", APIKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	var output struct {
		OK bool `json:"ok"`
	}
	if err := client.do(context.Background(), http.MethodGet, "/v1/test", mapValues("value", "x"), nil, &output); err != nil {
		t.Fatal(err)
	}
	if !output.OK {
		t.Fatal("response was not decoded")
	}
}

func TestDoReturnsTypedAPIErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"code":"duplicate","message":"already exists"}}`))
	}))
	defer server.Close()
	client, _ := New(Config{BaseURL: server.URL, APIKey: "secret"})
	err := client.do(context.Background(), http.MethodGet, "/v1/test", nil, nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict || apiErr.Code != "duplicate" {
		t.Fatalf("error = %#v", err)
	}
}

func TestDoReturnsTypedTransportAndStrictResponseErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"ok":true,"unknown":1}`)) }))
	defer server.Close()
	client, _ := New(Config{BaseURL: server.URL, APIKey: "secret"})
	var output struct {
		OK bool `json:"ok"`
	}
	err := client.do(context.Background(), http.MethodGet, "/", nil, nil, &output)
	var transportErr *TransportError
	if !errors.As(err, &transportErr) || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error = %#v", err)
	}

	client.http.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") })
	err = client.do(context.Background(), http.MethodGet, "/", nil, nil, nil)
	if !errors.As(err, &transportErr) || !strings.Contains(err.Error(), "offline") {
		t.Fatalf("error = %#v", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (function roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
func mapValues(key, value string) url.Values { return url.Values{key: []string{value}} }
