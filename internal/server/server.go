// Package server wires a parsed OpenAPI document into a running MCP server:
// every tool definition gets a handler that proxies the call to the upstream
// API and formats the response back into an MCP result.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aloki-alok/mcpify/internal/openapi"
	"github.com/aloki-alok/mcpify/internal/tool"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxRespBytes caps how much of an upstream response we read into a tool result,
// so a huge or unbounded body cannot exhaust memory or overwhelm a client.
const maxRespBytes = 1 << 20 // 1 MiB

// Config describes one mcpify server instance.
type Config struct {
	Doc     *openapi.Document
	Defs    []tool.Def
	BaseURL string            // resolved upstream API root
	Headers map[string]string // injected on every upstream request (e.g. auth)
	Timeout time.Duration     // per-request upstream timeout
	Version string            // mcpify version, reported to clients
}

// New builds an MCP server exposing every tool in cfg.Defs.
func New(cfg Config) *sdk.Server {
	name := cfg.Doc.Title
	if name == "" {
		name = "mcpify"
	}
	impl := &sdk.Implementation{
		Name:    name,
		Title:   cfg.Doc.Title,
		Version: cfg.Version,
	}
	srv := sdk.NewServer(impl, nil)

	client := &http.Client{Timeout: cfg.Timeout}
	for _, d := range cfg.Defs {
		srv.AddTool(&sdk.Tool{
			Name:        d.Name,
			Description: d.Description,
			InputSchema: d.InputSchema,
		}, handlerFor(client, cfg.BaseURL, cfg.Headers, d))
	}
	return srv
}

func handlerFor(client *http.Client, base string, headers map[string]string, d tool.Def) sdk.ToolHandler {
	return func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		args := map[string]any{}
		if len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return toolError("invalid arguments: " + err.Error()), nil
			}
		}

		hreq, err := tool.BuildRequest(ctx, d, base, args)
		if err != nil {
			return toolError(err.Error()), nil
		}
		for k, v := range headers {
			hreq.Header.Set(k, v)
		}
		if hreq.Header.Get("Accept") == "" {
			hreq.Header.Set("Accept", "application/json")
		}

		resp, err := client.Do(hreq)
		if err != nil {
			return toolError("upstream request failed: " + err.Error()), nil
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(io.LimitReader(resp.Body, maxRespBytes))
		if err != nil {
			return toolError("read upstream response: " + err.Error()), nil
		}
		return formatResponse(resp, data), nil
	}
}

// formatResponse renders an upstream HTTP response as an MCP result. On success
// the content is the response body (pretty-printed when JSON) so it parses
// cleanly for clients that render structured output; the JSON object, if any,
// is also attached as structured content. Non-2xx responses are marked as
// errors with the status line for visibility.
func formatResponse(resp *http.Response, data []byte) *sdk.CallToolResult {
	isErr := resp.StatusCode >= 400

	pretty, obj := prettyJSON(data)
	text := pretty
	if isErr {
		text = fmt.Sprintf("HTTP %s\n\n%s", resp.Status, pretty)
	}

	res := &sdk.CallToolResult{
		Content: []sdk.Content{&sdk.TextContent{Text: text}},
		IsError: isErr,
	}
	if obj != nil {
		res.StructuredContent = obj
	}
	return res
}

// prettyJSON returns an indented form of body when it is valid JSON (and the
// decoded value if it is a JSON object, for structured content). Non-JSON
// bodies are returned unchanged with a nil object.
func prettyJSON(body []byte) (string, map[string]any) {
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return string(body), nil
	}
	indented, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(body), nil
	}
	obj, _ := v.(map[string]any)
	return string(indented), obj
}

func toolError(msg string) *sdk.CallToolResult {
	return &sdk.CallToolResult{
		Content: []sdk.Content{&sdk.TextContent{Text: msg}},
		IsError: true,
	}
}
