package server

import (
	"context"
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

// RunHTTP serves the MCP server over the streamable-HTTP transport at addr. The
// same server instance is shared across sessions. It blocks until ctx is
// cancelled or the listener fails.
func RunHTTP(ctx context.Context, srv *sdk.Server, addr string) error {
	handler := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return srv }, nil)
	httpSrv := &http.Server{Addr: addr, Handler: handler}

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
