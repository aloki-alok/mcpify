// Command mcpify turns any OpenAPI 3.x spec into a working MCP server in one
// command: every API operation becomes an MCP tool that proxies the real HTTP
// call. See DESIGN.md.
package main

import (
	"os"

	"github.com/aloki-alok/mcpify/internal/cli"
)

// version is overridden at release time via -ldflags.
var version = "0.0.0-dev"

func main() {
	os.Exit(cli.Run(version, os.Args[1:]))
}
