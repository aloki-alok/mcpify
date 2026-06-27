// Command mcpify turns any OpenAPI 3.x spec into a working MCP server in one
// command: every API operation becomes an MCP tool that proxies the real HTTP
// call. See DESIGN.md.
package main

import (
	"os"
	"runtime/debug"

	"github.com/aloki-alok/mcpify/internal/cli"
)

// version is overridden at release time via -ldflags. For "go install" builds,
// which carry no ldflags, it falls back to the module version from build info.
var version = "0.0.0-dev"

func main() {
	os.Exit(cli.Run(resolveVersion(), os.Args[1:]))
}

func resolveVersion() string {
	if version != "0.0.0-dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}
