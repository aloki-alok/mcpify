// Package cli is mcpify's command surface: parse arguments, load an OpenAPI
// spec, and either serve it as an MCP server or preview the tools it would
// expose.
package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aloki-alok/mcpify/internal/openapi"
	"github.com/aloki-alok/mcpify/internal/tool"
)

// options are the parsed flags shared by the serve and ls commands.
type options struct {
	base    string
	headers map[string]string
	// headerEnv records which headers came from --header-env (header name ->
	// environment variable name), so printed client configs can carry the
	// reference instead of the resolved secret.
	headerEnv map[string]string
	http      string
	readOnly  bool
	include   []string
	tags      []string
	stdio     bool
	timeout   time.Duration
	spec      string
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

// filter builds the operation filter these options describe.
func (o options) filter() tool.Filter {
	return tool.Filter{Include: o.include, Tags: o.tags}
}

// noToolsError explains an empty tool set, naming whichever flags caused it so
// the user does not have to guess between an empty spec, --read-only, and
// --include/--tag matching nothing. When a filter is the cause it also lists a
// few tags and tool names the spec does offer, so a mistyped pattern is easy to
// correct.
func (o options) noToolsError(doc *openapi.Document) error {
	if !o.filter().Empty() {
		tags, names := available(doc, o.readOnly)
		return fmt.Errorf("no operations matched --include/--tag (include=%v tag=%v); available tags: %s; tool names: %s", o.include, o.tags, sample(tags), sample(names))
	}
	if o.readOnly {
		return fmt.Errorf("no operations to expose (--read-only with no GET/HEAD in the spec)")
	}
	return fmt.Errorf("no operations to expose (is the spec empty?)")
}

// available lists the tags and tool names the spec exposes under the current
// --read-only setting, drawn from an unfiltered build so the read-only rule
// stays in tool.Defs and the names match what `ls` would print.
func available(doc *openapi.Document, readOnly bool) (tags, names []string) {
	seen := map[string]bool{}
	for _, d := range tool.Defs(doc, readOnly, tool.Filter{}) {
		names = append(names, d.Name)
		for _, t := range d.Tags() {
			if key := strings.ToLower(t); !seen[key] {
				seen[key] = true
				tags = append(tags, t)
			}
		}
	}
	return tags, names
}

// sample renders up to five items as a comma-separated list, marking a longer
// list with a trailing "...", and says "none" when there are none.
func sample(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	if len(items) > 5 {
		return strings.Join(items[:5], ", ") + ", ..."
	}
	return strings.Join(items, ", ")
}

// parse hand-parses flags so they may appear before or after the positional
// spec argument. It supports "--flag value", "--flag=value", and repeatable
// -H / --header.
func parse(args []string) (options, error) {
	o := options{headers: map[string]string{}, headerEnv: map[string]string{}, timeout: 30 * time.Second}
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
			hname, err := addHeader(o.headers, v)
			if err != nil {
				return o, err
			}
			delete(o.headerEnv, hname)
		case "--header-env":
			v, err := next()
			if err != nil {
				return o, err
			}
			if err := addHeaderEnv(o.headers, o.headerEnv, v); err != nil {
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
		case "--include":
			v, err := next()
			if err != nil {
				return o, err
			}
			o.include = append(o.include, v)
		case "--tag":
			v, err := next()
			if err != nil {
				return o, err
			}
			o.tags = append(o.tags, v)
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

func addHeader(into map[string]string, raw string) (string, error) {
	idx := strings.IndexByte(raw, ':')
	if idx < 0 {
		return "", fmt.Errorf("header %q must be in \"Name: value\" form", raw)
	}
	name := strings.TrimSpace(raw[:idx])
	value := strings.TrimSpace(raw[idx+1:])
	if name == "" {
		return "", fmt.Errorf("header %q has an empty name", raw)
	}
	into[name] = value
	return name, nil
}

// addHeaderEnv resolves a "Name=ENV_VAR" pair against the environment, so the
// secret never appears on the command line or in a printed client config.
func addHeaderEnv(headers, headerEnv map[string]string, raw string) error {
	idx := strings.IndexByte(raw, '=')
	if idx < 0 {
		return fmt.Errorf("--header-env %q must be in \"Name=ENV_VAR\" form", raw)
	}
	name := strings.TrimSpace(raw[:idx])
	envVar := strings.TrimSpace(raw[idx+1:])
	if name == "" || envVar == "" {
		return fmt.Errorf("--header-env %q must be in \"Name=ENV_VAR\" form", raw)
	}
	value := os.Getenv(envVar)
	if value == "" {
		return fmt.Errorf("--header-env %s: environment variable %s is not set", raw, envVar)
	}
	headers[name] = value
	headerEnv[name] = envVar
	return nil
}
