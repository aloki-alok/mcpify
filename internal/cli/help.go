package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/aloki-alok/mcpify/internal/ui"
)

func welcome(version string) {
	s := ui.For(os.Stdout)
	fmt.Println(s.Banner())
	fmt.Printf("\n%s  %s  %s\n\n", s.Dim("turn any OpenAPI spec into an MCP server"), s.Dim("·"), s.Dim(version))
	fmt.Println(s.Accent("quickstart"))
	fmt.Println(quick(s, "mcpify ls openapi.yaml", "preview the tools a spec exposes"))
	fmt.Println(quick(s, "mcpify openapi.yaml", "serve it as an MCP server over stdio"))
	fmt.Println(quick(s, "mcpify --http :8080 openapi.yaml", "serve over HTTP instead"))
	fmt.Println(quick(s, "mcpify https://api.example.com/openapi.json", "load a spec straight from a URL"))
	fmt.Printf("\nrun %s for all options.\n", s.Bold("mcpify help"))
}

func quick(s ui.Style, command, desc string) string {
	const width = 42
	gap := width - len(command)
	if gap < 1 {
		gap = 1
	}
	return "  " + s.Bold(command) + strings.Repeat(" ", gap) + s.Dim(desc)
}

func usage(w *os.File) {
	s := ui.For(w)
	fmt.Fprintln(w, s.Banner())
	fmt.Fprint(w, `
Usage:
  mcpify <spec>              serve an OpenAPI spec as an MCP server (stdio)
  mcpify serve <spec>        same, explicit
  mcpify ls <spec>           preview the tools the spec would expose
  mcpify upgrade             update mcpify to the latest release
  mcpify version             print the version

A <spec> is a path to an OpenAPI 3.x file or an http(s):// URL.

Options:
  --base <url>               upstream base URL (overrides the spec's servers)
  -H, --header "Name: val"   header sent on every upstream request (repeatable)
  --header-env "Name=VAR"    like -H, but the value is read from environment
                             variable VAR, so secrets stay off the command line
                             and out of printed client configs (repeatable)
  --http <addr>              serve over HTTP at addr (e.g. :8080) instead of stdio
  --read-only                only expose GET and HEAD operations
  --include <pattern>        keep operations whose tool name, operationId, or path
                             match (substring, or a glob when it has *; repeatable)
  --tag <tag>                keep operations carrying this OpenAPI tag (repeatable)
  --stdio                    force stdio serving even in a terminal (skip the menu)
  --timeout <dur>            upstream request timeout (default 30s)

Run in a terminal, mcpify <spec> opens a short menu (run a server, print a
client config, or list the tools). An MCP client that launches mcpify over
stdio gets the server directly.

Examples:
  mcpify ls ./petstore.yaml
  mcpify --base https://api.example.com --header-env "Authorization=API_TOKEN" spec.json
  mcpify --http :8080 https://api.example.com/openapi.json
  mcpify ls --tag pet --include "*Order*" ./petstore.yaml
`)
}
