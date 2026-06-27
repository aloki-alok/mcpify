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
		srv := doc.Servers[0]
		raw := srv.URL
		if strings.Contains(raw, "{") {
			filled, ok := fillTemplate(raw, srv.Variables)
			if !ok {
				return "", fmt.Errorf("server URL %q has variables without defaults; pass --base <url>", raw)
			}
			raw = filled
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

// fillTemplate substitutes {var} placeholders in a server URL with their default
// values. It returns ok=false if any placeholder has no default, so the caller
// can ask for an explicit --base.
func fillTemplate(raw string, vars map[string]string) (string, bool) {
	for {
		open := strings.IndexByte(raw, '{')
		if open < 0 {
			return raw, true
		}
		close := strings.IndexByte(raw[open:], '}')
		if close < 0 {
			return "", false
		}
		close += open
		name := raw[open+1 : close]
		val, ok := vars[name]
		if !ok {
			return "", false
		}
		raw = raw[:open] + val + raw[close+1:]
	}
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
