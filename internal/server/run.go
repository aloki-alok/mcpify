package server

import (
	"context"
	"io"
	"net/http"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// RunStdio serves the MCP server over stdio, the transport MCP clients use when
// they spawn a server as a subprocess. It blocks until the client disconnects
// or ctx is cancelled.
func RunStdio(ctx context.Context, srv *sdk.Server) error {
	return srv.Run(ctx, &sdk.StdioTransport{})
}

// RunHTTP serves the MCP server over the streamable-HTTP transport. The protocol
// endpoint is mounted at /mcp; any other path returns infoText as plain text, so
// a human who opens the root URL in a browser gets an explanation instead of a
// protocol error. The same server instance is shared across sessions. It blocks
// until ctx is cancelled or the listener fails.
func RunHTTP(ctx context.Context, srv *sdk.Server, addr, infoText string) error {
	mcpHandler := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return srv }, nil)
	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, infoText)
	})
	httpSrv := &http.Server{Addr: addr, Handler: mux}

	errc := make(chan error, 1)
	go func() { errc <- httpSrv.ListenAndServe() }()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
		return ctx.Err()
	case err := <-errc:
		return err
	}
}
