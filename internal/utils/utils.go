package utils

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

func BoolPtr(b bool) *bool {
	return &b
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

	fmt.Printf("Downloaded %d bytes\n", len(out))
	return out, nil
}