// Package restclient implements the standard-library client for the Voltr API.
package restclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"rdmm404/voltr-finance/internal/api"
)

const defaultTimeout = 30 * time.Second

type Config struct {
	BaseURL    string
	APIKey     string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type Client struct {
	baseURL *url.URL
	apiKey  string
	http    *http.Client
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API request failed with status %d: %s", e.StatusCode, e.Message)
}

type TransportError struct {
	Operation string
	Err       error
}

func (e *TransportError) Error() string { return fmt.Sprintf("%s: %v", e.Operation, e.Err) }
func (e *TransportError) Unwrap() error { return e.Err }

func New(config Config) (*Client, error) {
	baseURL, err := normalizeBaseURL(config.BaseURL)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, errors.New("API key is required")
	}
	if config.Timeout < 0 {
		return nil, errors.New("request timeout cannot be negative")
	}
	timeout := config.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	httpClient := &http.Client{Timeout: timeout}
	if config.HTTPClient != nil {
		clone := *config.HTTPClient
		clone.Timeout = timeout
		httpClient = &clone
	}
	return &Client{baseURL: baseURL, apiKey: config.APIKey, http: httpClient}, nil
}

func normalizeBaseURL(value string) (*url.URL, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("API base URL is required")
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("parse API base URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("API base URL must use http or https")
	}
	if parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("API base URL must contain only scheme, host, and an optional path")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed, nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, input, output any) error {
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/" + strings.TrimLeft(path, "/")
	endpoint.RawQuery = query.Encode()

	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return &TransportError{Operation: "encode request", Err: err}
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return &TransportError{Operation: "create request", Err: err}
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.http.Do(request)
	if err != nil {
		return &TransportError{Operation: "send request", Err: err}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return decodeAPIError(response)
	}
	if output == nil || response.StatusCode == http.StatusNoContent {
		_, err := io.Copy(io.Discard, response.Body)
		if err != nil {
			return &TransportError{Operation: "read response", Err: err}
		}
		return nil
	}
	if err := decodeStrict(response.Body, output); err != nil {
		return &TransportError{Operation: "decode response", Err: err}
	}
	return nil
}

func decodeAPIError(response *http.Response) error {
	var envelope api.ErrorResponse
	if err := decodeStrict(response.Body, &envelope); err != nil {
		return &APIError{StatusCode: response.StatusCode, Code: "invalid_response", Message: "API returned an invalid error response"}
	}
	if envelope.Error.Code == "" || envelope.Error.Message == "" {
		return &APIError{StatusCode: response.StatusCode, Code: "invalid_response", Message: "API returned an invalid error response"}
	}
	return &APIError{StatusCode: response.StatusCode, Code: envelope.Error.Code, Message: envelope.Error.Message}
}

func decodeStrict(reader io.Reader, output any) error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("response contains multiple JSON values")
		}
		return err
	}
	return nil
}
