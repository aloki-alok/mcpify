package server

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/aloki-alok/mcpify/internal/openapi"
)

// ResolveBase determines the upstream API root to send requests to. An explicit
// override always wins. Otherwise the first server URL in the spec is used:
// absolute URLs as-is, relative URLs resolved against the URL the spec was
// fetched from. A templated server URL or an unresolvable relative URL is an
// error that asks the operator for --base.
func ResolveBase(doc *openapi.Document, override, specURL string) (string, error) {
	if override != "" {
		if !hasScheme(override) {
			return "", fmt.Errorf("--base %q must be an absolute http(s) URL", override)
		}
		return override, nil
	}

	if len(doc.Servers) > 0 {
		raw := doc.Servers[0].URL
		if strings.Contains(raw, "{") {
			return "", fmt.Errorf("server URL %q is templated; pass --base <url>", raw)
		}
		if hasScheme(raw) {
			return raw, nil
		}
		if specURL != "" {
			return resolveAgainst(specURL, raw)
		}
		return "", fmt.Errorf("server URL %q is relative; pass --base <url>", raw)
	}

	// No servers listed: a spec fetched over HTTP defaults to its own origin.
	if specURL != "" {
		return originOf(specURL)
	}
	return "", fmt.Errorf("spec has no servers; pass --base <url>")
}

func hasScheme(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func resolveAgainst(specURL, rel string) (string, error) {
	base, err := url.Parse(specURL)
	if err != nil {
		return "", fmt.Errorf("parse spec URL: %w", err)
	}
	ref, err := url.Parse(rel)
	if err != nil {
		return "", fmt.Errorf("parse server URL: %w", err)
	}
	return base.ResolveReference(ref).String(), nil
}

func originOf(specURL string) (string, error) {
	u, err := url.Parse(specURL)
	if err != nil {
		return "", fmt.Errorf("parse spec URL: %w", err)
	}
	return u.Scheme + "://" + u.Host, nil
}
