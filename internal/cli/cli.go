// Package cli is mcpify's command surface: parse arguments, load an OpenAPI
// spec, and either serve it as an MCP server or preview the tools it would
// expose.
package cli

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// options are the parsed flags shared by the serve and ls commands.
type options struct {
	base     string
	headers  map[string]string
	http     string
	readOnly bool
	stdio    bool
	timeout  time.Duration
	spec     string
}

// Run dispatches a command and returns a process exit code.
func Run(version string, args []string) int {
	if len(args) == 0 {
		welcome(version)
		return 0
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Println("mcpify", version)
		return 0
	case "help", "--help", "-h":
		usage(os.Stdout)
		return 0
	case "ls":
		return ls(args[1:])
	case "serve":
		return serve(version, args[1:])
	case "upgrade":
		return Upgrade(version)
	default:
		// A bare spec path or URL serves it directly.
		return serve(version, args)
	}
}

// parse hand-parses flags so they may appear before or after the positional
// spec argument. It supports "--flag value", "--flag=value", and repeatable
// -H / --header.
func parse(args []string) (options, error) {
	o := options{headers: map[string]string{}, timeout: 30 * time.Second}
	for i := 0; i < len(args); i++ {
		a := args[i]
		name, inlineVal, hasInline := splitFlag(a)
		next := func() (string, error) {
			if hasInline {
				return inlineVal, nil
			}
			if i+1 >= len(args) {
				return "", fmt.Errorf("flag %s needs a value", a)
			}
			i++
			return args[i], nil
		}
		switch name {
		case "--base":
			v, err := next()
			if err != nil {
				return o, err
			}
			o.base = v
		case "-H", "--header":
			v, err := next()
			if err != nil {
				return o, err
			}
			if err := addHeader(o.headers, v); err != nil {
				return o, err
			}
		case "--http":
			v, err := next()
			if err != nil {
				return o, err
			}
			o.http = v
		case "--read-only":
			o.readOnly = true
		case "--stdio":
			o.stdio = true
		case "--timeout":
			v, err := next()
			if err != nil {
				return o, err
			}
			d, err := time.ParseDuration(v)
			if err != nil {
				return o, fmt.Errorf("invalid --timeout %q: %w", v, err)
			}
			o.timeout = d
		default:
			if strings.HasPrefix(a, "-") {
				return o, fmt.Errorf("unknown flag %q", a)
			}
			if o.spec != "" {
				return o, fmt.Errorf("unexpected argument %q (spec already set to %q)", a, o.spec)
			}
			o.spec = a
		}
	}
	if o.spec == "" {
		return o, fmt.Errorf("missing spec (a file path or http(s) URL to an OpenAPI document)")
	}
	return o, nil
}

// splitFlag separates "--flag=value" into its parts. Short combined forms are
// not used, so a leading "-H" is returned whole.
func splitFlag(a string) (name, val string, hasVal bool) {
	if !strings.HasPrefix(a, "-") {
		return a, "", false
	}
	if eq := strings.IndexByte(a, '='); eq >= 0 {
		return a[:eq], a[eq+1:], true
	}
	return a, "", false
}

func addHeader(into map[string]string, raw string) error {
	idx := strings.IndexByte(raw, ':')
	if idx < 0 {
		return fmt.Errorf("header %q must be in \"Name: value\" form", raw)
	}
	name := strings.TrimSpace(raw[:idx])
	value := strings.TrimSpace(raw[idx+1:])
	if name == "" {
		return fmt.Errorf("header %q has an empty name", raw)
	}
	into[name] = value
	return nil
}
