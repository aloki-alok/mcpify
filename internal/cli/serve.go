package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aloki-alok/mcpify/internal/server"
	"github.com/aloki-alok/mcpify/internal/tool"
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

	// Diagnostics go to stderr; in stdio mode stdout is the MCP transport.
	title := doc.Title
	if title == "" {
		title = o.spec
	}
	fmt.Fprintf(os.Stderr, "mcpify: %s — %d tools → %s\n", title, len(defs), base)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if o.http != "" {
		fmt.Fprintf(os.Stderr, "mcpify: serving MCP over HTTP at %s\n", o.http)
		if err := server.RunHTTP(ctx, srv, o.http); err != nil && err != context.Canceled {
			fmt.Fprintln(os.Stderr, "mcpify:", err)
			return 1
		}
		return 0
	}

	if err := server.RunStdio(ctx, srv); err != nil && err != context.Canceled {
		fmt.Fprintln(os.Stderr, "mcpify:", err)
		return 1
	}
	return 0
}
