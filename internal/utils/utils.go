package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

func BoolPtr(b bool) *bool {
	return &b
}

func StringPtr(s string) *string {
	return &s
}

var ErrDownload = errors.New("error downloading the file")

func DownloadFileBytes(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("%w - failed to make HTTP request to %s: %w", ErrDownload, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w - bad status code: %d %s", ErrDownload, resp.StatusCode, resp.Status)
	}

	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response contents %w", err)
	}

	slog.Info("Downloaded file", "bytes", len(out))
	return out, nil
}

func MapKeys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for key := range m {
		out = append(out, key)
	}

	return out
}

func JsonMarshalIgnore(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
