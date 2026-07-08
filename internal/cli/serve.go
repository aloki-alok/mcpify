package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aloki-alok/mcpify/internal/server"
	"github.com/aloki-alok/mcpify/internal/tool"
	"github.com/aloki-alok/mcpify/internal/ui"
)

func serve(version string, args []string) int {
	o, err := parse(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mcpify:", err)
		return 2
	}

	doc, specURL, err := loadSpec(o.spec)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mcpify:", err)
		return 1
	}

	base, err := server.ResolveBase(doc, o.base, specURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mcpify:", err)
		return 1
	}

	defs := tool.Defs(doc, o.readOnly)
	if len(defs) == 0 {
		fmt.Fprintln(os.Stderr, "mcpify: no operations to expose (is the spec empty, or --read-only with no GET/HEAD?)")
		return 1
	}

	srv := server.New(server.Config{
		Doc:     doc,
		Defs:    defs,
		BaseURL: base,
		Headers: o.headers,
		Timeout: o.timeout,
		Version: version,
	})

	name := doc.Title
	if name == "" {
		name = o.spec
	}

	// Explicit HTTP server.
	if o.http != "" {
		if !isPortFree(o.http) {
			fmt.Fprintf(os.Stderr, "mcpify: %s is already in use; try --http %s\n", o.http, nextFreePort(o.http))
			return 1
		}
		return serveHTTP(srv, o.http, name, base, len(defs))
	}

	// stdio: a client spawned us over a pipe, or --stdio forces it. Serve quietly;
	// stdout is the MCP transport so all diagnostics go to stderr.
	if o.stdio || !ui.IsTTY(os.Stdin) {
		fmt.Fprintf(os.Stderr, "mcpify: %s — %d tools → %s (stdio)\n", name, len(defs), base)
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := server.RunStdio(ctx, srv); err != nil && err != context.Canceled {
			fmt.Fprintln(os.Stderr, "mcpify:", err)
			return 1
		}
		return 0
	}

	// A human ran us in a terminal: guide them interactively instead of hanging
	// on stdin waiting for a client that will never speak.
	return interact(o, srv, name, base, defs)
}
