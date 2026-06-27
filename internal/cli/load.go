package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aloki-alok/mcpify/internal/openapi"
)

// loadSpec reads an OpenAPI document from a file path or http(s) URL. The second
// return value is the URL it was fetched from (empty for a file), used later to
// resolve relative server URLs.
func loadSpec(spec string) (*openapi.Document, string, error) {
	var data []byte
	var specURL string

	if strings.HasPrefix(spec, "http://") || strings.HasPrefix(spec, "https://") {
		specURL = spec
		b, err := fetch(spec)
		if err != nil {
			return nil, "", err
		}
		data = b
	} else {
		b, err := os.ReadFile(spec)
		if err != nil {
			return nil, "", fmt.Errorf("read spec: %w", err)
		}
		data = b
	}

	doc, err := openapi.Load(data)
	if err != nil {
		return nil, "", err
	}
	return doc, specURL, nil
}

func fetch(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch spec: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch spec: %s", resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}
	return b, nil
}
